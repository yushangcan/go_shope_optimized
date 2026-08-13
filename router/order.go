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
