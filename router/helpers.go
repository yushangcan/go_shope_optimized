package router

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go_shope/middleware"
	"go_shope/service"
)

func writeError(c *gin.Context, err error) {
	// Service 不需要知道 HTTP，统一由 router 把业务错误翻译成状态码。
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, service.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, service.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, service.ErrOutOfStock), errors.Is(err, service.ErrActivityUnavailable), errors.Is(err, service.ErrProductUnavailable), errors.Is(err, service.ErrInvalidOrderStatus):
		status = http.StatusConflict
	}
	// 统一使用 {"error": "..."} 格式返回错误给客户端。
	c.JSON(status, gin.H{"error": err.Error()})
}

func pathID(c *gin.Context, name string) (uint64, error) {
	// 例如请求 /api/products/12 时，c.Param("id") 是字符串 "12"；这里转成 uint64。
	return strconv.ParseUint(c.Param(name), 10, 64)
}

func currentUserID(c *gin.Context) (uint64, error) {
	// Auth 中间件已把 JWT 里的 sub 写入 Context；这里将字符串 ID 再转为 uint64。
	value, exists := c.Get(middleware.UserIDKey)
	if !exists {
		return 0, service.ErrUnauthorized
	}
	return strconv.ParseUint(value.(string), 10, 64)
}
