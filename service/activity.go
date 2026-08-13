package service

import (
	"time"

	"go_shope/dao"
	"go_shope/model"
)

type ActivityInput struct {
	ProductID    uint64    `json:"product_id"`
	SeckillPrice int64     `json:"seckill_price"`
	TotalStock   int       `json:"total_stock"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Status       string    `json:"status"`
}

type ActivityService struct{ repo *dao.Repository }

func NewActivityService(repo *dao.Repository) *ActivityService { return &ActivityService{repo: repo} }

func (s *ActivityService) Create(input ActivityInput) (*model.SeckillActivity, error) {
	if input.ProductID == 0 || input.SeckillPrice < 0 || input.TotalStock <= 0 || !input.EndTime.After(input.StartTime) {
		return nil, ErrInvalidInput
	}
	product, err := s.repo.FindProductByID(input.ProductID)
	if dao.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if product.Status != "ON_SALE" || input.TotalStock > product.Stock {
		return nil, ErrInvalidInput
	}
	status := input.Status
	if status == "" {
		status = "NOT_STARTED"
	}
	activity := &model.SeckillActivity{ProductID: input.ProductID, SeckillPrice: input.SeckillPrice, TotalStock: input.TotalStock, AvailableStock: input.TotalStock, StartTime: input.StartTime, EndTime: input.EndTime, Status: status}
	if err := s.repo.CreateActivity(activity); err != nil {
		return nil, err
	}
	return activity, nil
}

func (s *ActivityService) List() ([]model.SeckillActivity, error) { return s.repo.ListActivities() }
func (s *ActivityService) Get(id uint64) (*model.SeckillActivity, error) {
	activity, err := s.repo.FindActivityByID(id)
	if dao.IsNotFound(err) {
		return nil, ErrNotFound
	}
	return activity, err
}

func (s *ActivityService) Update(id uint64, input ActivityInput) (*model.SeckillActivity, error) {
	activity, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if input.SeckillPrice < 0 || input.TotalStock < activity.TotalStock || !input.EndTime.After(input.StartTime) {
		return nil, ErrInvalidInput
	}
	activity.SeckillPrice, activity.TotalStock, activity.StartTime, activity.EndTime = input.SeckillPrice, input.TotalStock, input.StartTime, input.EndTime
	if input.Status != "" {
		activity.Status = input.Status
	}
	if err := s.repo.UpdateActivity(activity); err != nil {
		return nil, err
	}
	return activity, nil
}

func (s *ActivityService) Delete(id uint64) error {
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.repo.DeleteActivity(id)
}
