package dao

import (
	"errors"

	"go_shope/model"
	"gorm.io/gorm"
)

// ErrOutOfStock 用于区分“没有库存”与普通数据库错误。
var ErrOutOfStock = errors.New("out of stock")

func (r *Repository) CreateSeckillOrder(order *model.Order) error {
	// 两次扣库和创建订单必须全部成功或全部失败，因此放在同一个 MySQL 事务。
	return r.DB.Transaction(func(tx *gorm.DB) error {
		// SQL 等价形式：UPDATE seckill_activities SET available_stock = available_stock - 1
		// WHERE id = ? AND available_stock > 0。
		// 受影响行数为 0 代表库存已被其他请求抢完。
		activityResult := tx.Model(&model.SeckillActivity{}).
			Where("id = ? AND available_stock > 0", order.ActivityID).
			UpdateColumn("available_stock", gorm.Expr("available_stock - 1"))
		if activityResult.Error != nil {
			return activityResult.Error
		}
		if activityResult.RowsAffected != 1 {
			return ErrOutOfStock
		}

		// 此处扣减商品总库存。如果失败，事务会回滚前面的活动库存扣减。
		productResult := tx.Model(&model.Product{}).
			Where("id = ? AND stock > 0", order.ProductID).
			UpdateColumn("stock", gorm.Expr("stock - 1"))
		if productResult.Error != nil {
			return productResult.Error
		}
		if productResult.RowsAffected != 1 {
			return ErrOutOfStock
		}

		// 两次扣库都成功后才插入订单；插入失败同样回滚两次扣库。
		return tx.Create(order).Error
	})
}

func (r *Repository) FindOrderByID(id uint64) (*model.Order, error) {
	var order model.Order
	if err := r.DB.First(&order, id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *Repository) ListOrdersByUserID(userID uint64) ([]model.Order, error) {
	var orders []model.Order
	if err := r.DB.Where("user_id = ?", userID).Order("id desc").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *Repository) UpdateOrderStatus(id, userID uint64, from, to string) (int64, error) {
	// 只有状态仍是 from 时才能改为 to，避免重复支付等重复操作。
	result := r.DB.Model(&model.Order{}).Where("id = ? AND user_id = ? AND status = ?", id, userID, from).Update("status", to)
	return result.RowsAffected, result.Error
}

func (r *Repository) CancelOrderAndRestoreStock(id, userID uint64) (int64, error) {
	// 取消订单、关闭状态、恢复两份库存是一个不可分割的业务动作。
	var changed int64
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		// 只允许订单本人取消 PENDING（待支付）订单。
		if err := tx.Where("id = ? AND user_id = ? AND status = ?", id, userID, "PENDING").First(&order).Error; err != nil {
			return err
		}

		// 再做一次条件更新，确保并发下只有一个请求能真正取消成功。
		result := tx.Model(&model.Order{}).Where("id = ? AND status = ?", id, "PENDING").Update("status", "CANCELLED")
		if result.Error != nil {
			return result.Error
		}
		changed = result.RowsAffected
		if changed != 1 {
			// 状态已被其他请求改变时不再恢复库存。
			return nil
		}

		// 订单确实取消后，才恢复活动库存和商品总库存。
		if err := tx.Model(&model.SeckillActivity{}).Where("id = ?", order.ActivityID).UpdateColumn("available_stock", gorm.Expr("available_stock + 1")).Error; err != nil {
			return err
		}
		return tx.Model(&model.Product{}).Where("id = ?", order.ProductID).UpdateColumn("stock", gorm.Expr("stock + 1")).Error
	})
	return changed, err
}
