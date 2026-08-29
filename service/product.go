// package service 表示商品业务逻辑也属于业务层。
package service

import (
	// strings 用于标准化商品名称、描述和状态值。
	"strings"

	// dao 负责实际的数据库读写。
	"go_shope/dao"
	// model 定义数据库中的 Product 结构。
	"go_shope/model"
)

// ProductInput 是管理员创建或更新商品时提交的 JSON 数据，而不是数据库实体本身。
type ProductInput struct {
	// Name 是商品展示名称。
	Name string `json:"name"`
	// Description 是可选的商品描述。
	Description string `json:"description"`
	// Price 使用“分”保存金额，避免 float 的精度问题。
	Price int64 `json:"price"`
	// Stock 是当前商品库存，不能是负数。
	Stock int `json:"stock"`
	// Status 只允许 ON_SALE（上架）或 OFF_SALE（下架）。
	Status string `json:"status"`
}

// ProductService 封装商品 CRUD 的业务校验和数据转换。
type ProductService struct {
	// repo 提供商品表的数据库操作。
	repo *dao.Repository
}

// NewProductService 创建一个使用指定 Repository 的商品服务。
func NewProductService(repo *dao.Repository) *ProductService {
	// 保存依赖，避免 Service 自己创建数据库连接。
	return &ProductService{repo: repo}
}

// normalizeProductInput 清理输入并验证公共规则，供 Create 和 Update 共同使用。
func normalizeProductInput(input ProductInput) (ProductInput, error) {
	// 删除商品名首尾空格，避免只包含空格的名称。
	input.Name = strings.TrimSpace(input.Name)
	// 删除描述首尾空格，使保存的数据更整洁。
	input.Description = strings.TrimSpace(input.Description)
	// 统一状态大小写，保证后续比较稳定。
	input.Status = strings.ToUpper(strings.TrimSpace(input.Status))
	// 未填写状态时，创建或更新后默认让商品处于上架状态。
	if input.Status == "" {
		// 写入明确的默认状态。
		input.Status = "ON_SALE"
	}
	// 名称不能为空，价格/库存不能小于零，状态必须在白名单中。
	if input.Name == "" || input.Price < 0 || input.Stock < 0 || (input.Status != "ON_SALE" && input.Status != "OFF_SALE") {
		// 返回已经规范化的 input，方便必要时排查，同时返回统一参数错误。
		return input, ErrInvalidInput
	}
	// 校验通过后把规范化的数据交给调用者。
	return input, nil
}

// Create 校验商品输入并写入一条新商品记录。
func (s *ProductService) Create(input ProductInput) (*model.Product, error) {
	// 先复用公共清理和校验逻辑。
	input, err := normalizeProductInput(input)
	// 无效输入不能继续写数据库。
	if err != nil {
		// 直接返回业务错误。
		return nil, err
	}
	// 将 HTTP 输入对象转换为数据库模型；ID 和时间字段由数据库/GORM 管理。
	product := &model.Product{
		// 写入已校验名称。
		Name: input.Name,
		// 写入已清理描述。
		Description: input.Description,
		// 写入以分为单位的价格。
		Price: input.Price,
		// 写入非负库存。
		Stock: input.Stock,
		// 写入合法上架状态。
		Status: input.Status,
	}
	// 交由 DAO 执行 INSERT。
	if err := s.repo.CreateProduct(product); err != nil {
		// INSERT 失败则不返回商品。
		return nil, err
	}
	// 创建成功后返回包含数据库回填 ID 的商品。
	return product, nil
}

// List 返回买家可以浏览的上架商品列表。
func (s *ProductService) List() ([]model.Product, error) {
	// DAO 内部只查询 ON_SALE 状态的数据，避免下架商品出现在前台。
	return s.repo.ListOnSaleProducts()
}

// ListAll returns both on-sale and off-sale products to an authenticated admin.
func (s *ProductService) ListAll() ([]model.Product, error) {
	return s.repo.ListProducts()
}

// Get 按商品 ID 查询单个商品。
func (s *ProductService) Get(id uint64) (*model.Product, error) {
	// 委托 DAO 查找数据库记录。
	product, err := s.repo.FindProductByID(id)
	// 统一处理不存在的商品。
	if dao.IsNotFound(err) {
		// Handler 会将此错误映射为 404。
		return nil, ErrNotFound
	}
	// 成功时返回商品；数据库错误原样向上传递。
	return product, err
}

// Update 用新的合法字段覆盖已有商品。
func (s *ProductService) Update(id uint64, input ProductInput) (*model.Product, error) {
	// 先清理并校验管理员提交的新字段。
	input, err := normalizeProductInput(input)
	// 参数错误时不读取也不修改数据库。
	if err != nil {
		// 直接将业务错误交给 Router。
		return nil, err
	}
	// 先查旧商品，确保目标 ID 存在。
	product, err := s.Get(id)
	// 不存在或查询失败时不能继续更新。
	if err != nil {
		// 保留 Get 已转换的错误类型。
		return nil, err
	}
	// 覆盖允许管理员修改的全部业务字段。
	product.Name = input.Name
	// 覆盖描述字段。
	product.Description = input.Description
	// 覆盖以分为单位的价格。
	product.Price = input.Price
	// 覆盖库存；库存的并发扣减兜底在 DAO 的下单事务中完成。
	product.Stock = input.Stock
	// 覆盖上架状态。
	product.Status = input.Status
	// 调用 DAO 进行 UPDATE。
	if err := s.repo.UpdateProduct(product); err != nil {
		// 更新失败时向上传递数据库错误。
		return nil, err
	}
	// 返回更新后的商品。
	return product, nil
}

// Delete 删除指定商品。
func (s *ProductService) Delete(id uint64) error {
	// 先确认商品存在，这样“不存在”和“删除 SQL 执行失败”能有不同业务含义。
	_, err := s.Get(id)
	// 查询失败时无需继续删除。
	if err != nil {
		// 返回 404 或真实数据库错误。
		return err
	}
	// 商品存在后交给 DAO 执行 DELETE。
	return s.repo.DeleteProduct(id)
}
