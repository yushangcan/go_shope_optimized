package dao

import (
	"go_shope/model"
	"gorm.io/gorm"
)

func (r *Repository) CreateUser(user *model.User) error {
	// INSERT INTO users (...) VALUES (...)
	return r.DB.Create(user).Error
}

func (r *Repository) FindUserByUsername(username string) (*model.User, error) {
	// 用 ? 绑定 username，GORM 会安全处理值，不要自己拼接 SQL 字符串。
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

// IsNotFound 将 GORM 的“查无数据”错误转换成业务层容易理解的判断。
func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}
