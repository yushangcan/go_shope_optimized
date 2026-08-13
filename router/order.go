package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go_shope/service"
)

type OrderHandler struct{ orders *service.OrderService }

func NewOrderHandler(orders *service.OrderService) *OrderHandler {
	return &OrderHandler{orders: orders}
}

type createOrderRequest struct {
	RequestID string `json:"request_id"`
}

func (h *OrderHandler) CreateSeckillOrder(c *gin.Context) {
	activityID, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
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
	order, err := h.orders.CreateSeckillOrder(userID, activityID, req.RequestID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, order)
}

func (h *OrderHandler) List(c *gin.Context) {
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
