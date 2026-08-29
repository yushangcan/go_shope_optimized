package router

import (
	"net/http"
	"path/filepath"

	"go_shope/middleware"
	"go_shope/service"

	"github.com/gin-gonic/gin"
)

// New 集中注册所有 HTTP 路由。
// 参数传入的都是已创建好的 Service，router 不直接访问数据库。
func New(users *service.UserService, products *service.ProductService, activities *service.ActivityService, orders *service.OrderService, jwtSecret string) *gin.Engine {
	// gin.Default 自带日志和 panic 恢复中间件，适合当前学习版。
	r := gin.Default()
	// 健康检查不依赖登录，访问 GET /health 就能确认 HTTP 服务是否启动。
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	// Serve the lightweight storefront and admin pages. API routes stay under /api.
	r.Static("/assets", "./web/assets")
	r.GET("/", func(c *gin.Context) { c.File(filepath.Join("web", "index.html")) })
	r.GET("/admin", func(c *gin.Context) { c.File(filepath.Join("web", "admin.html")) })

	// 每种 Handler 只持有自己需要的 Service。
	authHandler := NewAuthHandler(users, jwtSecret)   //用户认证处理
	productHandler := NewProductHandler(products)     //商品处理
	activityHandler := NewActivityHandler(activities) //秒杀活动处理
	orderHandler := NewOrderHandler(orders)           //订单处理
	//通过路由分组实现不同业务组
	//公共接口，不需要进行校验和登录就可以访问
	api := r.Group("/api")
	api.POST("/auth/register", authHandler.Register)        //用户注册
	api.POST("/auth/login", authHandler.Login)              //用户登录
	api.GET("/products", productHandler.List)               //获取商品列表
	api.GET("/products/:id", productHandler.Get)            //获取单个商品的详情页面
	api.GET("/seckill/activities", activityHandler.List)    //获取秒杀活动列表
	api.GET("/seckill/activities/:id", activityHandler.Get) //单个秒杀活动的详情页面

	protected := api.Group("")
	//创建保护路由，访问之前都要通过鉴权中间件，http请求头里都需要携带JWT token校验是否合法
	protected.Use(middleware.Auth(jwtSecret))
	protected.GET("/users/me", authHandler.Me)                                        //获取用户自己的个人信息
	protected.GET("/orders", orderHandler.List)                                       //查看我的订单列表
	protected.GET("/orders/:id", orderHandler.Get)                                    //查看某一条订单详情
	protected.POST("/orders/:id/pay", orderHandler.Pay)                               //支付订单
	protected.POST("/orders/:id/cancel", orderHandler.Cancel)                         //取消订单
	protected.POST("/products/:id/orders", orderHandler.CreateProductOrder)           //购买普通商品
	protected.POST("/seckill/activities/:id/orders", orderHandler.CreateSeckillOrder) //创建秒杀订单

	//本质是一个父子中间件，执行之后的路由必须先执行前面的父路由，必须是管理员token才可以访问后面的admin
	admin := protected.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	admin.GET("/products", productHandler.ListAll)                  //管理员查看全部商品
	admin.GET("/orders", orderHandler.ListAll)                      //管理员查看全站订单
	admin.POST("/products", productHandler.Create)                  //管理员新增商品
	admin.PUT("/products/:id", productHandler.Update)               //管理员修改商品
	admin.DELETE("/products/:id", productHandler.Delete)            //管理员删除商品
	admin.POST("/seckill/activities", activityHandler.Create)       //创建秒杀活动
	admin.PUT("/seckill/activities/:id", activityHandler.Update)    //修改秒杀活动
	admin.DELETE("/seckill/activities/:id", activityHandler.Delete) //删除秒杀活动

	return r
}

// /api 是所有业务接口的公共前缀。
// 以下接口允许未登录访问：注册、登录、商品浏览和活动浏览。
// protected 组中的每一个路由都会先经过 Auth JWT 中间件。
// 与当前用户有关的接口：个人资料、自己的订单、下单、支付、取消。
// admin 组在已登录基础上额外检查角色必须为 ADMIN。
