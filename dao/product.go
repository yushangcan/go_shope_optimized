package dao

import "go_shope/model"

func (r *Repository) CreateProduct(product *model.Product) error {
	// INSERT 新商品。
	return r.DB.Create(product).Error
}

func (r *Repository) ListOnSaleProducts() ([]model.Product, error) {
	// 普通用户只查询上架商品；管理员编辑接口不走这个查询。
	var products []model.Product
	if err := r.DB.Where("status = ?", "ON_SALE").Order("id desc").Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// ListProducts returns every product for the merchant dashboard, including
// products that are currently off sale.
func (r *Repository) ListProducts() ([]model.Product, error) {
	var products []model.Product
	if err := r.DB.Order("id desc").Find(&products).Error; err != nil {
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

func (r *Repository) UpdateProduct(product *model.Product) error {
	// Save 根据 product.ID 执行 UPDATE。
	return r.DB.Save(product).Error
}

func (r *Repository) DeleteProduct(id uint64) error {
	// GORM 默认物理删除；当前基础版未引入软删除字段。
	return r.DB.Delete(&model.Product{}, id).Error
}
