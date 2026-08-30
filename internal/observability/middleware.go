package observability

import (
	"github.com/gin-gonic/gin"
	"time"
)

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		Observe(c.FullPath(), c.Request.Method, c.Writer.Status(), started)
	}
}
