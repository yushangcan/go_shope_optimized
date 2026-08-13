package service

import (
	"strings"

	"go_shope/dao"
	"go_shope/model"
	"golang.org/x/crypto/bcrypt"
)

// UserService 负责注册、登录等用户业务规则。
// repo 只做数据读写，不在这里直接写 GORM 查询。
type UserService struct{ repo *dao.Repository }

func NewUserService(repo *dao.Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(username, password string) (*model.User, error) {
	// 去除用户名前后空格，避免 "alice" 和 " alice " 被当成两个用户名。
	username = strings.TrimSpace(username)
	// bcrypt 最多只处理 72 字节密码，因此这里也限制长度。
	if len(username) < 3 || len(username) > 100 || len(password) < 6 || len(password) > 72 {
		return nil, ErrInvalidInput
	}

	// 先查用户名是否存在。查到用户说明重复注册；查无记录则可继续。
	if _, err := s.repo.FindUserByUsername(username); err == nil {
		return nil, ErrConflict
	} else if !dao.IsNotFound(err) {
		return nil, err
	}

	// 只将 bcrypt 哈希写入数据库，明文 password 只存在于这一次请求内。
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{Username: username, PasswordHash: string(hash), Role: "USER"}
	if err := s.repo.CreateUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Login(username, password string) (*model.User, error) {
	// 登录时同样忽略用户名首尾空格。
	user, err := s.repo.FindUserByUsername(strings.TrimSpace(username))
	if err != nil {
		// 为了不暴露用户名是否存在，用户不存在和密码不对都返回未授权。
		if dao.IsNotFound(err) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}

	// CompareHashAndPassword 用明文密码与已保存的哈希作比较。
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrUnauthorized
	}
	return user, nil
}

func (s *UserService) GetProfile(id uint64) (*model.User, error) {
	user, err := s.repo.FindUserByID(id)
	if dao.IsNotFound(err) {
		return nil, ErrNotFound
	}
	return user, err
}
