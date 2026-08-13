package dao

import (
	"errors"

	"go_shope/model"
	"gorm.io/gorm"
)

var ErrOutOfStock = errors.New("out of stock")

func (r *Repository) CreateSeckillOrder(order *model.Order) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		activityResult := tx.Model(&model.SeckillActivity{}).
			Where("id = ? AND available_stock > 0", order.ActivityID).
			UpdateColumn("available_stock", gorm.Expr("available_stock - 1"))
		if activityResult.Error != nil {
			return activityResult.Error
		}
		if activityResult.RowsAffected != 1 {
			return ErrOutOfStock
		}

		productResult := tx.Model(&model.Product{}).
			Where("id = ? AND stock > 0", order.ProductID).
			UpdateColumn("stock", gorm.Expr("stock - 1"))
		if productResult.Error != nil {
			return productResult.Error
		}
		if productResult.RowsAffected != 1 {
			return ErrOutOfStock
		}

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
	result := r.DB.Model(&model.Order{}).Where("id = ? AND user_id = ? AND status = ?", id, userID, from).Update("status", to)
	return result.RowsAffected, result.Error
}

func (r *Repository) CancelOrderAndRestoreStock(id, userID uint64) (int64, error) {
	var changed int64
	err := r.DB.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Where("id = ? AND user_id = ? AND status = ?", id, userID, "PENDING").First(&order).Error; err != nil {
			return err
		}
		result := tx.Model(&model.Order{}).Where("id = ? AND status = ?", id, "PENDING").Update("status", "CANCELLED")
		if result.Error != nil {
			return result.Error
		}
		changed = result.RowsAffected
		if changed != 1 {
			return nil
		}
		if err := tx.Model(&model.SeckillActivity{}).Where("id = ?", order.ActivityID).UpdateColumn("available_stock", gorm.Expr("available_stock + 1")).Error; err != nil {
			return err
		}
		return tx.Model(&model.Product{}).Where("id = ?", order.ProductID).UpdateColumn("stock", gorm.Expr("stock + 1")).Error
	})
	return changed, err
}
