package service

import (
	"strings"

	"go_shope/dao"
	"go_shope/model"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct{ repo *dao.Repository }

func NewUserService(repo *dao.Repository) *UserService { return &UserService{repo: repo} }

func (s *UserService) Register(username, password string) (*model.User, error) {
	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 100 || len(password) < 6 || len(password) > 72 {
		return nil, ErrInvalidInput
	}
	if _, err := s.repo.FindUserByUsername(username); err == nil {
		return nil, ErrConflict
	} else if !dao.IsNotFound(err) {
		return nil, err
	}
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
	user, err := s.repo.FindUserByUsername(strings.TrimSpace(username))
	if err != nil {
		if dao.IsNotFound(err) {
			return nil, ErrUnauthorized
		}
		return nil, err
	}
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
