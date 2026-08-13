package model

import "time"

type Order struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	OrderNo     string    `gorm:"size:32;not null;uniqueIndex" json:"order_no"`
	RequestID   string    `gorm:"size:64;not null;uniqueIndex" json:"request_id"`
	UserID      uint64    `gorm:"not null;uniqueIndex:uk_activity_user;index" json:"user_id"`
	ActivityID  uint64    `gorm:"not null;uniqueIndex:uk_activity_user;index" json:"activity_id"`
	ProductID   uint64    `gorm:"not null;index" json:"product_id"`
	ProductName string    `gorm:"size:100;not null" json:"product_name"`
	UnitPrice   int64     `gorm:"not null" json:"unit_price"`
	Quantity    int       `gorm:"not null;default:1" json:"quantity"`
	TotalAmount int64     `gorm:"not null" json:"total_amount"`
	Status      string    `gorm:"size:20;not null;default:PENDING;index" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
