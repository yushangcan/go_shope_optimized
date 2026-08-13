package dao

import (
	"go_shope/model"
	"gorm.io/gorm"
)

func (r *Repository) CreateUser(user *model.User) error { return r.DB.Create(user).Error }

func (r *Repository) FindUserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := r.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindUserByID(id uint64) (*model.User, error) {
	var user model.User
	if err := r.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func IsNotFound(err error) bool { return err == gorm.ErrRecordNotFound }
