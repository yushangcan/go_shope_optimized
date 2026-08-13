package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go_shope/service"
)

// ProductHandler 只做 HTTP 输入输出，商品规则在 ProductService。
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
	// 管理员提交的 JSON 已绑定为 ProductInput，继续交给 Service 校验并写库。
	product, err := h.products.Create(req)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, product)
}
func (h *ProductHandler) List(c *gin.Context) {
	// List 会返回 ON_SALE 商品。
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
	// URL 里的 id 决定修改哪条记录；请求体决定新的业务字段。
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
	// 删除成功使用 204，响应体为空。
	c.Status(http.StatusNoContent)
}
