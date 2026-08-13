package model

import "time"

type SeckillActivity struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	ProductID      uint64    `gorm:"not null;index" json:"product_id"`
	SeckillPrice   int64     `gorm:"not null" json:"seckill_price"`
	TotalStock     int       `gorm:"not null" json:"total_stock"`
	AvailableStock int       `gorm:"not null" json:"available_stock"`
	StartTime      time.Time `gorm:"not null;index" json:"start_time"`
	EndTime        time.Time `gorm:"not null;index" json:"end_time"`
	Status         string    `gorm:"size:20;not null;default:NOT_STARTED;index" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Product        Product   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}
