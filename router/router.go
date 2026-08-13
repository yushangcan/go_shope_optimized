package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go_shope/middleware"
	"go_shope/service"
)

// New 集中注册所有 HTTP 路由。
// 参数传入的都是已创建好的 Service，router 不直接访问数据库。
func New(users *service.UserService, products *service.ProductService, activities *service.ActivityService, orders *service.OrderService, jwtSecret string) *gin.Engine {
	// gin.Default 自带日志和 panic 恢复中间件，适合当前学习版。
	r := gin.Default()
	// 健康检查不依赖登录，访问 GET /health 就能确认 HTTP 服务是否启动。
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	// 每种 Handler 只持有自己需要的 Service。
	authHandler := NewAuthHandler(users, jwtSecret)
	productHandler := NewProductHandler(products)
	activityHandler := NewActivityHandler(activities)
	orderHandler := NewOrderHandler(orders)

	api := r.Group("/api")
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.GET("/products", productHandler.List)
	api.GET("/products/:id", productHandler.Get)
	api.GET("/seckill/activities", activityHandler.List)
	api.GET("/seckill/activities/:id", activityHandler.Get)

	protected := api.Group("")
	protected.Use(middleware.Auth(jwtSecret))
	protected.GET("/users/me", authHandler.Me)
	protected.GET("/orders", orderHandler.List)
	protected.GET("/orders/:id", orderHandler.Get)
	protected.POST("/orders/:id/pay", orderHandler.Pay)
	protected.POST("/orders/:id/cancel", orderHandler.Cancel)
	protected.POST("/seckill/activities/:id/orders", orderHandler.CreateSeckillOrder)

	admin := protected.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	admin.POST("/products", productHandler.Create)
	admin.PUT("/products/:id", productHandler.Update)
	admin.DELETE("/products/:id", productHandler.Delete)
	admin.POST("/seckill/activities", activityHandler.Create)
	admin.PUT("/seckill/activities/:id", activityHandler.Update)
	admin.DELETE("/seckill/activities/:id", activityHandler.Delete)

	return r
}

// /api 是所有业务接口的公共前缀。
// 以下接口允许未登录访问：注册、登录、商品浏览和活动浏览。
// protected 组中的每一个路由都会先经过 Auth JWT 中间件。
// 与当前用户有关的接口：个人资料、自己的订单、下单、支付、取消。
// admin 组在已登录基础上额外检查角色必须为 ADMIN。
