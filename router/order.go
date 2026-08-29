package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go_shope/service"
)

// OrderHandler 处理“当前登录用户”的订单接口。
type OrderHandler struct{ orders *service.OrderService }

func NewOrderHandler(orders *service.OrderService) *OrderHandler {
	return &OrderHandler{orders: orders}
}

type createOrderRequest struct {
	// request_id 由客户端每次点击下单时生成；数据库唯一索引可防止重复建单。
	RequestID string `json:"request_id"`
}

// createProductOrderRequest 是普通商品下单接口接收的 JSON 请求体。
type createProductOrderRequest struct {
	// Quantity 是购买数量，必须大于 0。
	Quantity int `json:"quantity"`
	// RequestID 标识用户的一次下单动作，重复提交同一个值不会重复创建订单。
	RequestID string `json:"request_id"`
}

// CreateProductOrder 处理 POST /api/products/:id/orders 普通商品下单请求。
func (h *OrderHandler) CreateProductOrder(c *gin.Context) {
	// productID 来自 URL 中的 :id，代表用户要购买的商品。
	productID, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	// userID 由 JWT 中间件写入上下文，保证订单属于当前登录用户。
	userID, err := currentUserID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	// 将 JSON 中的购买数量和请求唯一 ID 绑定到请求结构体。
	var req createProductOrderRequest
	if c.ShouldBindJSON(&req) != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	// Service 校验商品并调用 DAO 事务执行扣库存和创建订单。
	order, err := h.orders.CreateProductOrder(userID, productID, req.Quantity, req.RequestID)
	if err != nil {
		writeError(c, err)
		return
	}
	// 首次创建和幂等重试都返回完整订单，方便前端展示订单号。
	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) CreateSeckillOrder(c *gin.Context) {
	// activityID 来自 /api/seckill/activities/:id/orders 中的 :id。
	activityID, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	// userID 来自 JWT，不能由客户端在请求体随意指定。
	userID, err := currentUserID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	var req createOrderRequest
	if c.ShouldBindJSON(&req) != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	// Service 会检查活动时间和状态，DAO 再在事务中扣库存与创建订单。
	order, err := h.orders.CreateSeckillOrder(userID, activityID, req.RequestID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) List(c *gin.Context) {
	// 不接收 user_id 参数，避免用户通过修改参数查看其他人的订单。
	userID, err := currentUserID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	orders, err := h.orders.ListByUserID(userID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) ListAll(c *gin.Context) {
	orders, err := h.orders.ListAll()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) Get(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	userID, err := currentUserID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	order, err := h.orders.GetForUser(id, userID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) Pay(c *gin.Context) {
	// 当前仅模拟支付成功，不接入任何第三方支付平台。
	id, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	userID, err := currentUserID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	if err := h.orders.Pay(id, userID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "order paid"})
}

func (h *OrderHandler) Cancel(c *gin.Context) {
	// 取消成功后，Service/DAO 会在事务里恢复库存。
	id, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	userID, err := currentUserID(c)
	if err != nil {
		writeError(c, err)
		return
	}
	if err := h.orders.Cancel(id, userID); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
}
