package service

import (
	"fmt"
	"time"

	"go_shope/dao"
	"go_shope/model"
)

// OrderService 负责秒杀下单和订单状态流转。
type OrderService struct{ repo *dao.Repository }

func NewOrderService(repo *dao.Repository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) CreateSeckillOrder(userID, activityID uint64, requestID string) (*model.Order, error) {
	// userID 来自 JWT，activityID 来自 URL，requestID 来自 JSON 请求体。
	if userID == 0 || activityID == 0 || requestID == "" {
		return nil, ErrInvalidInput
	}

	// 查询活动时 DAO 会同时 Preload 商品，后面可以直接读取活动商品快照。
	activity, err := s.repo.FindActivityByID(activityID)
	if dao.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// 只有状态 ACTIVE，且当前时间落在 [StartTime, EndTime) 的活动才能下单。
	now := time.Now()
	if activity.Status != "ACTIVE" || now.Before(activity.StartTime) || !now.Before(activity.EndTime) {
		return nil, ErrActivityUnavailable
	}
	if activity.Product.Status != "ON_SALE" {
		return nil, ErrActivityUnavailable
	}

	// 下单时保存商品名和秒杀价快照，避免商品后来修改影响已有订单。
	// UnixNano + userID 只是学习版订单号生成方式，后续可以替换为雪花算法等方案。
	order := &model.Order{
		OrderNo: fmt.Sprintf("%d%d", now.UnixNano(), userID), RequestID: requestID,
		UserID: userID, ActivityID: activity.ID, ProductID: activity.ProductID,
		ProductName: activity.Product.Name, UnitPrice: activity.SeckillPrice,
		Quantity: 1, TotalAmount: activity.SeckillPrice, Status: "PENDING",
	}

	// DAO 内部的事务会扣减库存并插入订单；任意一步失败都会回滚。
	if err := s.repo.CreateSeckillOrder(order); err != nil {
		if err == dao.ErrOutOfStock {
			return nil, ErrOutOfStock
		}
		return nil, err
	}
	return order, nil
}

func (s *OrderService) ListByUserID(userID uint64) ([]model.Order, error) {
	// 普通用户只看自己的订单，不能查看全站订单。
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
		// 订单存在但不属于当前登录用户，返回无权限而不是订单内容。
		return nil, ErrForbidden
	}
	return order, nil
}

func (s *OrderService) Pay(id, userID uint64) error {
	// 基础版不接真实支付渠道，只模拟 PENDING -> PAID 状态转换。
	return s.changeStatus(id, userID, "PAID")
}

func (s *OrderService) Cancel(id, userID uint64) error {
	// 取消与恢复库存必须在 DAO 的同一个事务中完成。
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
	// 先校验订单是否存在且属于当前用户。
	if _, err := s.GetForUser(id, userID); err != nil {
		return err
	}
	// 只允许待支付订单改变状态；RowsAffected=0 说明已支付或已取消。
	rows, err := s.repo.UpdateOrderStatus(id, userID, "PENDING", to)
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrInvalidOrderStatus
	}
	return nil
}
