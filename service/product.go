package service

import (
	"strings"

	"go_shope/dao"
	"go_shope/model"
)

type ProductInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Stock       int    `json:"stock"`
	Status      string `json:"status"`
}

type ProductService struct{ repo *dao.Repository }

func NewProductService(repo *dao.Repository) *ProductService { return &ProductService{repo: repo} }

func normalizeProductInput(input ProductInput) (ProductInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	if input.Status == "" {
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
	product := &model.Product{Name: input.Name, Description: input.Description, Price: input.Price, Stock: input.Stock, Status: input.Status}
	if err := s.repo.CreateProduct(product); err != nil {
		return nil, err
	}
	return product, nil
}

func (s *ProductService) List() ([]model.Product, error) { return s.repo.ListOnSaleProducts() }

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
	if _, err := s.Get(id); err != nil {
		return err
	}
	return s.repo.DeleteProduct(id)
}
