// Package model 定义数据库表对应的 Go 结构体。
package model

import "time"

// User 对应 MySQL 中的 users 表。
type User struct {
	// primaryKey 表示主键；GORM 会让它成为自增 ID。
	ID uint64 `gorm:"primaryKey" json:"id"`
	// uniqueIndex 保证数据库中用户名不能重复。
	Username string `gorm:"size:100;not null;uniqueIndex" json:"username"`
	// PasswordHash 只保存 bcrypt 哈希；json:"-" 表示绝不返回给前端。
	PasswordHash string `gorm:"size:255;not null" json:"-"`
	// Role 当前只有 USER 和 ADMIN；管理员才能修改商品和活动。
	Role string `gorm:"size:20;not null;default:USER" json:"role"`
	// GORM 会自动维护创建和更新时间。
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
