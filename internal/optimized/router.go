package optimized

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go_shope/internal/redisstore"
	"go_shope/middleware"
	"net/http"
	"strings"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Admit(c *gin.Context) {
	activityID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}
	userID, err := currentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var request struct {
		RequestID string `json:"request_id"`
	}
	if c.ShouldBindJSON(&request) != nil || strings.TrimSpace(request.RequestID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request_id is required"})
		return
	}
	result, err := h.service.Admit(c, userID, activityID, strings.TrimSpace(request.RequestID))
	if err != nil {
		status := http.StatusConflict
		if errors.Is(err, redisstore.ErrUnavailable) {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"error": err.Error(), "request_id": result.RequestID, "status": result.Status})
		return
	}
	c.JSON(http.StatusAccepted, result)
}

func (h *Handler) RequestStatus(c *gin.Context) {
	status, err := h.service.RequestStatus(c, c.Param("request_id"))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	if len(status) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handler) PublishActivity(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid activity id"})
		return
	}
	if err := h.service.PublishActivityByID(c, id); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "published", "activity_id": id})
}

func parseID(value string) (uint64, error) {
	var id uint64
	_, err := fmt.Sscan(value, &id)
	return id, err
}
func currentUser(c *gin.Context) (uint64, error) {
	value, ok := c.Get(middleware.UserIDKey)
	if !ok {
		return 0, errors.New("missing user")
	}
	return parseID(fmt.Sprint(value))
}
