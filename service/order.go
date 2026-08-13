package service

import (
	"fmt"
	"time"

	"go_shope/dao"
	"go_shope/model"
)

type OrderService struct{ repo *dao.Repository }

func NewOrderService(repo *dao.Repository) *OrderService { return &OrderService{repo: repo} }

func (s *OrderService) CreateSeckillOrder(userID, activityID uint64, requestID string) (*model.Order, error) {
	if userID == 0 || activityID == 0 || requestID == "" {
		return nil, ErrInvalidInput
	}
	activity, err := s.repo.FindActivityByID(activityID)
	if dao.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if activity.Status != "ACTIVE" || now.Before(activity.StartTime) || !now.Before(activity.EndTime) {
		return nil, ErrActivityUnavailable
	}
	if activity.Product.Status != "ON_SALE" {
		return nil, ErrActivityUnavailable
	}
	order := &model.Order{OrderNo: fmt.Sprintf("%d%d", now.UnixNano(), userID), RequestID: requestID, UserID: userID, ActivityID: activity.ID, ProductID: activity.ProductID, ProductName: activity.Product.Name, UnitPrice: activity.SeckillPrice, Quantity: 1, TotalAmount: activity.SeckillPrice, Status: "PENDING"}
	if err := s.repo.CreateSeckillOrder(order); err != nil {
		if err == dao.ErrOutOfStock {
			return nil, ErrOutOfStock
		}
		return nil, err
	}
	return order, nil
}

func (s *OrderService) ListByUserID(userID uint64) ([]model.Order, error) {
	return s.repo.ListOrdersByUserID(userID)
}

func (s *OrderService) GetForUser(id, userID uint64) (*model.Order, error) {
	order, err := s.repo.FindOrderByID(id)
	if dao.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, ErrForbidden
	}
	return order, nil
}

func (s *OrderService) Pay(id, userID uint64) error { return s.changeStatus(id, userID, "PAID") }

func (s *OrderService) Cancel(id, userID uint64) error {
	rows, err := s.repo.CancelOrderAndRestoreStock(id, userID)
	if dao.IsNotFound(err) {
		return ErrInvalidOrderStatus
	}
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrInvalidOrderStatus
	}
	return nil
}

func (s *OrderService) changeStatus(id, userID uint64, to string) error {
	if _, err := s.GetForUser(id, userID); err != nil {
		return err
	}
	rows, err := s.repo.UpdateOrderStatus(id, userID, "PENDING", to)
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrInvalidOrderStatus
	}
	return nil
}
