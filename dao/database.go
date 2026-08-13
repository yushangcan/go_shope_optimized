package dao

import (
	"fmt"

	"go_shope/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

func New(dsn string) (*Repository, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Product{}, &model.SeckillActivity{}, &model.Order{}); err != nil {
		return nil, fmt.Errorf("migrate tables: %w", err)
	}
	return &Repository{DB: db}, nil
}
