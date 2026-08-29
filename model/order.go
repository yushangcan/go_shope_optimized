package model

import "time"

// 订单类型常量用于区分普通购买与秒杀购买，取消订单时会据此决定恢复哪一份库存。
const (
	OrderTypeNormal  = "NORMAL"
	OrderTypeSeckill = "SECKILL"
)

// Order 同时保存普通商品订单和秒杀订单。
type Order struct {
	ID        uint64 `gorm:"primaryKey" json:"id"`
	OrderNo   string `gorm:"size:32;not null;uniqueIndex" json:"order_no"`   // 对外展示的订单号。
	RequestID string `gorm:"size:64;not null;uniqueIndex" json:"request_id"` // 客户端生成的请求 ID，防止重复点击建单。
	// 同名 uniqueIndex 让 (UserID, ActivityID) 形成联合唯一索引，限制一人一场秒杀只能下一单。
	UserID uint64 `gorm:"not null;uniqueIndex:uk_activity_user;index" json:"user_id"`
	// 普通订单没有秒杀活动，因此 ActivityID 使用指针：nil 会在 MySQL 中保存成 NULL。
	// MySQL 联合唯一索引允许多条 NULL，用户便可以多次购买普通商品，同时秒杀订单仍受唯一索引保护。
	ActivityID  *uint64   `gorm:"uniqueIndex:uk_activity_user;index" json:"activity_id,omitempty"`
	OrderType   string    `gorm:"size:20;not null;default:SECKILL;index" json:"order_type"` // NORMAL 或 SECKILL；默认值兼容迁移前已有的秒杀订单。
	ProductID   uint64    `gorm:"not null;index" json:"product_id"`
	ProductName string    `gorm:"size:100;not null" json:"product_name"`                // 商品名称快照，商品改名后历史订单不变。
	UnitPrice   int64     `gorm:"not null" json:"unit_price"`                           // 下单时的成交单价快照，单位为分。
	Quantity    int       `gorm:"not null;default:1" json:"quantity"`                   // 购买数量；秒杀固定为 1，普通商品由用户选择。
	TotalAmount int64     `gorm:"not null" json:"total_amount"`                         // 订单总额，等于 UnitPrice * Quantity。
	Status      string    `gorm:"size:20;not null;default:PENDING;index" json:"status"` // PENDING、PAID、CANCELLED。
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
