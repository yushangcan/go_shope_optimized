package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go_shope/service"
)

type AuthHandler struct {
	users  *service.UserService
	secret string
}

func NewAuthHandler(users *service.UserService, secret string) *AuthHandler {
	return &AuthHandler{users: users, secret: secret}
}

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	user, err := h.users.Register(req.Username, req.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "username": user.Username, "role": user.Role})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	user, err := h.users.Login(req.Username, req.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	claims := jwt.MapClaims{"sub": strconv.FormatUint(user.ID, 10), "role": user.Role, "exp": time.Now().Add(24 * time.Hour).Unix()}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.secret))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "token_type": "Bearer"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	id, err := currentUserID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	user, err := h.users.GetProfile(id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "username": user.Username, "role": user.Role})
}
