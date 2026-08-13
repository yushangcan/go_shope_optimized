package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// 这两个 key 是当前请求上下文 c 中保存登录用户信息的位置。
const UserIDKey = "userID"
const RoleKey = "role"

// Auth 返回一个 Gin 中间件。它在真正的 Handler 之前验证 JWT。
func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 HTTP 请求头读取：Authorization: Bearer <token>。
		header := c.GetHeader("Authorization")
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"})
			c.Abort() // 终止后续 Handler，避免未登录用户进入业务代码。
			return
		}

		// Parse 会验证签名、过期时间等 JWT 标准字段。
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (any, error) {
			// 只接受 HMAC 系列签名算法，避免接收意外算法。
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// 登录接口写入的是 MapClaims，因此这里读取成同一个类型。
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}
		// sub（subject）字段保存用户 ID；role 字段保存用户角色。
		userID, err := claims.GetSubject()
		if err != nil || userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			c.Abort()
			return
		}
		role, _ := claims["role"].(string)

		// 写入 Context 后，后面的订单 Handler 可以取得当前用户身份。
		c.Set(UserIDKey, userID)
		c.Set(RoleKey, role)
		c.Next() // 令请求继续进入真正的业务 Handler。
	}
}

// RequireAdmin 必须放在 Auth 后面使用，因为它依赖 Auth 写入的 role。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(RoleKey)
		if role != "ADMIN" {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
