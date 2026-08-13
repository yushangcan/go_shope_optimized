package dao

import "go_shope/model"

func (r *Repository) CreateProduct(product *model.Product) error { return r.DB.Create(product).Error }

func (r *Repository) ListOnSaleProducts() ([]model.Product, error) {
	var products []model.Product
	if err := r.DB.Where("status = ?", "ON_SALE").Order("id desc").Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (r *Repository) FindProductByID(id uint64) (*model.Product, error) {
	var product model.Product
	if err := r.DB.First(&product, id).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *Repository) UpdateProduct(product *model.Product) error { return r.DB.Save(product).Error }
func (r *Repository) DeleteProduct(id uint64) error              { return r.DB.Delete(&model.Product{}, id).Error }
