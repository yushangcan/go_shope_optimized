package service

import (
	"strings"

	"go_shope/dao"
	"go_shope/model"
)

// ProductInput 是管理员创建/修改商品时前端提交的 JSON 结构。
// 它与 Product 分开，避免客户端直接提交 ID、创建时间等数据库字段。
type ProductInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"` // 单位为分。
	Stock       int    `json:"stock"`
	Status      string `json:"status"`
}

type ProductService struct{ repo *dao.Repository }

func NewProductService(repo *dao.Repository) *ProductService {
	return &ProductService{repo: repo}
}

func normalizeProductInput(input ProductInput) (ProductInput, error) {
	// 规范化后，下面的 Create 和 Update 都不必重复这些输入处理代码。
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	if input.Status == "" {
		// 管理员没填写状态时，默认让商品上架。
		input.Status = "ON_SALE"
	}
	if input.Name == "" || input.Price < 0 || input.Stock < 0 || (input.Status != "ON_SALE" && input.Status != "OFF_SALE") {
		return input, ErrInvalidInput
	}
	return input, nil
}

func (s *ProductService) Create(input ProductInput) (*model.Product, error) {
	input, err := normalizeProductInput(input)
	if err != nil {
		return nil, err
	}
	// 将“请求结构”转换成要写入数据库的“模型结构”。
	product := &model.Product{Name: input.Name, Description: input.Description, Price: input.Price, Stock: input.Stock, Status: input.Status}
	if err := s.repo.CreateProduct(product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) List() ([]model.Product, error) {
	return s.repo.ListOnSaleProducts()
}

func (s *ProductService) Get(id uint64) (*model.Product, error) {
	product, err := s.repo.FindProductByID(id)
	if dao.IsNotFound(err) {
		return nil, ErrNotFound
	}
	return product, err
}

func (s *ProductService) Update(id uint64, input ProductInput) (*model.Product, error) {
	input, err := normalizeProductInput(input)
	if err != nil {
		return nil, err
	}
	// 先查旧对象，确保 ID 存在，再只替换允许修改的业务字段。
	product, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	product.Name, product.Description, product.Price, product.Stock, product.Status = input.Name, input.Description, input.Price, input.Stock, input.Status
	if err := s.repo.UpdateProduct(product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) Delete(id uint64) error {
	// 先查询使“不存在”和“删除失败”能返回不同业务结果。
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.repo.DeleteProduct(id)
}
