package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go_shope/service"
)

type ProductHandler struct{ products *service.ProductService }

func NewProductHandler(products *service.ProductService) *ProductHandler {
	return &ProductHandler{products: products}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var req service.ProductInput
	if c.ShouldBindJSON(&req) != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	product, err := h.products.Create(req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, product)
}
func (h *ProductHandler) List(c *gin.Context) {
	products, err := h.products.List()
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, products)
}
func (h *ProductHandler) Get(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	product, err := h.products.Get(id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}
func (h *ProductHandler) Update(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	var req service.ProductInput
	if c.ShouldBindJSON(&req) != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	product, err := h.products.Update(id, req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, product)
}
func (h *ProductHandler) Delete(c *gin.Context) {
	id, err := pathID(c, "id")
	if err != nil {
		writeError(c, service.ErrInvalidInput)
		return
	}
	if err := h.products.Delete(id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
