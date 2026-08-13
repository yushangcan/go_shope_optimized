package model

import "time"

// Order 是“一个用户抢到一件秒杀商品”产生的订单。
type Order struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	OrderNo   string `gorm:"size:32;not null;uniqueIndex" json:"order_no"`   // 对外展示的订单号。
	RequestID string `gorm:"size:64;not null;uniqueIndex" json:"request_id"` // 客户端生成的请求 ID，防止重复点击建单。
	// 同名 uniqueIndex 让 (UserID, ActivityID) 形成联合唯一索引，即一人一场只能下一单。
	UserID      uint64    `gorm:"not null;uniqueIndex:uk_activity_user;index" json:"user_id"`
	ActivityID  uint64    `gorm:"not null;uniqueIndex:uk_activity_user;index" json:"activity_id"`
	ProductID   uint64    `gorm:"not null;index" json:"product_id"`
	ProductName string    `gorm:"size:100;not null" json:"product_name"`                // 商品名称快照，商品改名后历史订单不变。
	UnitPrice   int64     `gorm:"not null" json:"unit_price"`                           // 秒杀成交单价，单位为分。
	Quantity    int       `gorm:"not null;default:1" json:"quantity"`                   // 当前项目一单固定一件。
	TotalAmount int64     `gorm:"not null" json:"total_amount"`                         // 此版本等于 UnitPrice * Quantity。
	Status      string    `gorm:"size:20;not null;default:PENDING;index" json:"status"` // PENDING、PAID、CANCELLED。
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
