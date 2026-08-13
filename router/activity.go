package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go_shope/service"
)

// ActivityHandler 对应秒杀活动的 HTTP CRUD。
type ActivityHandler struct{ activities *service.ActivityService }

func NewActivityHandler(activities *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{activities: activities}
}

func (h *ActivityHandler) Create(c *gin.Context) {
	var req service.ActivityInput
	if c.ShouldBindJSON(&req) != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	// Service 会检查商品存在、商品上架、库存数量和活动时间是否合法。
	activity, err := h.activities.Create(req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, activity)
}

func (h *ActivityHandler) List(c *gin.Context) {
	activities, err := h.activities.List()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, activities)
}

func (h *ActivityHandler) Get(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	activity, err := h.activities.Get(id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, activity)
}

func (h *ActivityHandler) Update(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	var req service.ActivityInput
	if c.ShouldBindJSON(&req) != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	activity, err := h.activities.Update(id, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, activity)
}

func (h *ActivityHandler) Delete(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	if err := h.activities.Delete(id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
