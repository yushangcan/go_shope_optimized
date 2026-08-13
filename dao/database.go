// Package dao 只执行数据读写；不放 HTTP 参数解析和业务规则。
package dao

import (
	"fmt"

	"go_shope/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Repository 把 GORM 的 DB 包一层，后续各 DAO 方法都通过它访问数据库。
type Repository struct {
	DB *gorm.DB
}

func New(dsn string) (*Repository, error) {
	// mysql.Open 创建 MySQL 方言配置；gorm.Open 初始化连接池。
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}

	// AutoMigrate 根据结构体和 gorm 标签创建或补充表、索引。
	// 学习版使用它减少建表步骤；生产环境通常改用版本化 SQL migration。
	if err := db.AutoMigrate(&model.User{}, &model.Product{}, &model.SeckillActivity{}, &model.Order{}); err != nil {
		return nil, fmt.Errorf("migrate tables: %w", err)
	}

	// 返回的 Repository 会被注入给每个 Service。
	return &Repository{DB: db}, nil
}
