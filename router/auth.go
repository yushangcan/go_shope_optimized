package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go_shope/service"
)

// AuthHandler 负责注册、登录、读取当前用户资料这三个 HTTP 入口。
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
	// ShouldBindJSON 把请求体 JSON 绑定到 req，例如 {"username":"alice","password":"secret"}。
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	// 具体校验、查重、密码哈希都交给 UserService。
	user, err := h.users.Register(req.Username, req.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	// 不返回 PasswordHash；只返回登录前端真正需要的公开用户信息。
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
	// sub 是 JWT 约定的“主体”，本项目用它保存用户 ID。
	// exp 是过期时间；这里设置为当前时间后的 24 小时。
	claims := jwt.MapClaims{"sub": strconv.FormatUint(user.ID, 10), "role": user.Role, "exp": time.Now().Add(24 * time.Hour).Unix()}
	// 使用同一份 secret 和 HS256 算法为 claims 签名，得到客户端要保存的 token。
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.secret))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "token_type": "Bearer"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	// 当前用户 ID 不是客户端传的，而是从已验证 JWT 的 Context 中读取。
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
