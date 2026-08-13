package model

import "time"

// SeckillActivity 对应一场秒杀活动，而不是商品本身。
// 同一商品可以创建多场活动，每场有独立的时间、价格和活动库存。
type SeckillActivity struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	ProductID      uint64    `gorm:"not null;index" json:"product_id"` // 关联的商品 ID。
	SeckillPrice   int64     `gorm:"not null" json:"seckill_price"`    // 秒杀价，单位为分。
	TotalStock     int       `gorm:"not null" json:"total_stock"`      // 创建活动时设置的总量。
	AvailableStock int       `gorm:"not null" json:"available_stock"`  // 成功下单后递减的剩余活动库存。
	StartTime      time.Time `gorm:"not null;index" json:"start_time"`
	EndTime        time.Time `gorm:"not null;index" json:"end_time"`
	Status         string    `gorm:"size:20;not null;default:NOT_STARTED;index" json:"status"` // NOT_STARTED、ACTIVE、ENDED。
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	// Preload("Product") 时，GORM 根据 ProductID 填充此字段，便于接口返回商品信息。
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}
