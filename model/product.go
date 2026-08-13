package model

import "time"

// Product 对应商品表。价格统一用“分”保存，避免 float 金额精度问题。
type Product struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"size:1000;not null;default:''" json:"description"`
	Price       int64     `gorm:"not null" json:"price"`                                // 例如 19900 表示 199.00 元。
	Stock       int       `gorm:"not null;default:0" json:"stock"`                      // 商品总库存，由基础版 MySQL 直接维护。
	Status      string    `gorm:"size:20;not null;default:ON_SALE;index" json:"status"` // ON_SALE 或 OFF_SALE。
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
