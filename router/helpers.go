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
	case errors.Is(err, service.ErrOutOfStock), errors.Is(err, service.ErrActivityUnavailable), errors.Is(err, service.ErrInvalidOrderStatus):
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func pathID(c *gin.Context, name string) (uint64, error) {
	return strconv.ParseUint(c.Param(name), 10, 64)
}

func currentUserID(c *gin.Context) (uint64, error) {
	value, exists := c.Get(middleware.UserIDKey)
	if !exists {
		return 0, service.ErrUnauthorized
	}
	return strconv.ParseUint(value.(string), 10, 64)
}
