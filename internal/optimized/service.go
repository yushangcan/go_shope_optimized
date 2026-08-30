package optimized

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go_shope/dao"
	"go_shope/internal/observability"
	"go_shope/internal/redisstore"
	"go_shope/model"
	"gorm.io/gorm"
)

var (
	ErrSoldOut          = errors.New("sold out")
	ErrDuplicateBuyer   = errors.New("duplicate buyer")
	ErrActivityInactive = errors.New("activity inactive")
	ErrRequestFailed    = errors.New("request failed")
	ErrRequestConflict  = errors.New("request id belongs to another user")
)

const (
	RequestAccepted   = "ACCEPTED"
	RequestProcessing = "PROCESSING"
	RequestSucceeded  = "SUCCEEDED"
	RequestFailed     = "FAILED"
)

type Service struct {
	repo  *dao.Repository
	store *redisstore.Store
}

func New(repo *dao.Repository, store *redisstore.Store) *Service {
	return &Service{repo: repo, store: store}
}

type Admission struct {
	RequestID  string `json:"request_id"`
	Status     string `json:"status"`
	ActivityID uint64 `json:"activity_id"`
}

func (s *Service) Admit(ctx context.Context, userID, activityID uint64, requestID string) (Admission, error) {
	code, status, err := s.store.Admit(ctx, activityID, userID, requestID, time.Now())
	if err != nil {
		return Admission{}, err
	}
	result := Admission{RequestID: requestID, Status: status, ActivityID: activityID}
	switch code {
	case 0, 1:
		observability.Admissions.WithLabelValues(status).Inc()
		return result, nil
	case 2:
		observability.Admissions.WithLabelValues("SOLD_OUT").Inc()
		return result, ErrSoldOut
	case 3:
		observability.Admissions.WithLabelValues("DUPLICATE_BUYER").Inc()
		return result, ErrDuplicateBuyer
	case 4:
		observability.Admissions.WithLabelValues("ACTIVITY_NOT_ACTIVE").Inc()
		return result, ErrActivityInactive
	case 5:
		observability.Admissions.WithLabelValues("REQUEST_ID_CONFLICT").Inc()
		return result, ErrRequestConflict
	default:
		return result, ErrRequestFailed
	}
}

func (s *Service) PublishActivity(ctx context.Context, activity *model.SeckillActivity) error {
	return s.store.PublishActivity(ctx, activity.ID, activity.Status, activity.StartTime, activity.EndTime, activity.AvailableStock)
}

func (s *Service) PublishActivityByID(ctx context.Context, activityID uint64) error {
	activity, err := s.repo.FindActivityByID(activityID)
	if err != nil {
		return err
	}
	return s.PublishActivity(ctx, activity)
}

func (s *Service) RequestStatus(ctx context.Context, requestID string) (map[string]string, error) {
	return s.store.RequestStatus(ctx, requestID)
}

func (s *Service) ProcessEvent(ctx context.Context, event redisstore.OrderEvent) error {
	// A redelivered stream entry may already have a durable order. Treat it as
	// success so compensation never returns inventory for an existing order.
	if existing, err := s.repo.FindOrderByRequestID(event.RequestID); err == nil && existing != nil {
		return s.store.Mark(ctx, event.RequestID, RequestSucceeded, map[string]any{"order_id": existing.ID})
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	activity, err := s.repo.FindActivityByID(event.ActivityID)
	if err != nil {
		return err
	}
	// The consumer keeps the baseline MySQL transaction as the final durable guard.
	order := &model.Order{OrderNo: fmt.Sprintf("%d%d", time.Now().UnixNano(), event.UserID), RequestID: event.RequestID, UserID: event.UserID, ActivityID: &event.ActivityID, OrderType: model.OrderTypeSeckill, ProductID: activity.ProductID, ProductName: activity.Product.Name, UnitPrice: activity.SeckillPrice, Quantity: 1, TotalAmount: activity.SeckillPrice, Status: "PENDING"}
	if err := s.repo.CreateSeckillOrder(order); err != nil {
		if existing, lookupErr := s.repo.FindOrderByRequestID(event.RequestID); lookupErr == nil && existing != nil {
			return s.store.Mark(ctx, event.RequestID, RequestSucceeded, map[string]any{"order_id": existing.ID})
		}
		if errors.Is(err, dao.ErrOutOfStock) {
			return ErrSoldOut
		}
		return err
	}
	return s.store.Mark(ctx, event.RequestID, "SUCCEEDED", map[string]any{"order_id": order.ID})
}
