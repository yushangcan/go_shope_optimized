// package service 表示秒杀活动规则在业务层实现。
package service

import (
	"strings"
	// time 提供活动开始、结束时间的数据类型和比较方法。
	"time"

	// dao 封装活动和商品的数据库读写。
	"go_shope/dao"
	// model 定义 SeckillActivity 数据库模型。
	"go_shope/model"
)

func normalizeActivityStatus(status string) (string, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		return "NOT_STARTED", nil
	}
	if status != "NOT_STARTED" && status != "ACTIVE" && status != "ENDED" {
		return "", ErrInvalidInput
	}
	return status, nil
}

// ActivityInput 是管理员创建或更新秒杀活动时提交的请求数据。
type ActivityInput struct {
	// ProductID 表示本次秒杀绑定的商品 ID。
	ProductID uint64 `json:"product_id"`
	// SeckillPrice 是秒杀价，单位为分。
	SeckillPrice int64 `json:"seckill_price"`
	// TotalStock 是分配给本次活动的总库存。
	TotalStock int `json:"total_stock"`
	// StartTime 是活动开始时间，由 Gin 从 RFC3339 JSON 解析为 time.Time。
	StartTime time.Time `json:"start_time"`
	// EndTime 是活动结束时间，必须晚于 StartTime。
	EndTime time.Time `json:"end_time"`
	// Status 是活动状态，例如 NOT_STARTED 或 ACTIVE。
	Status string `json:"status"`
}

// ActivityService 封装秒杀活动的创建、查询、更新与删除规则。
type ActivityService struct {
	// repo 是访问商品表和秒杀活动表的 Repository。
	repo *dao.Repository
}

// NewActivityService 将现有 Repository 注入活动服务。
func NewActivityService(repo *dao.Repository) *ActivityService {
	// 返回保存了数据访问依赖的服务实例。
	return &ActivityService{repo: repo}
}

// Create 校验活动输入、确认关联商品可用，并创建一条秒杀活动。
func (s *ActivityService) Create(input ActivityInput) (*model.SeckillActivity, error) {
	// 商品 ID 不能为零，价格不能为负，库存至少一件，并且结束时间必须晚于开始时间。
	if input.ProductID == 0 || input.SeckillPrice < 0 || input.TotalStock <= 0 || !input.EndTime.After(input.StartTime) {
		// 基础字段不合法时直接拒绝创建。
		return nil, ErrInvalidInput
	}

	// 查询活动要绑定的商品，确认它真实存在。
	product, err := s.repo.FindProductByID(input.ProductID)
	// DAO 的未找到错误转换为业务层 404 语义。
	if dao.IsNotFound(err) {
		// 不允许为不存在的商品创建活动。
		return nil, ErrNotFound
	}
	// 其余错误，例如数据库连接失败，不应被伪装成参数错误。
	if err != nil {
		// 原样返回给上层。
		return nil, err
	}
	// 只有上架商品可以做秒杀，且活动分配库存不能超过商品现有库存。
	if product.Status != "ON_SALE" || input.TotalStock > product.Stock {
		// 两种情况都会使这次活动配置无效。
		return nil, ErrInvalidInput
	}

	// 读取管理员可选提交的活动状态。
	status, err := normalizeActivityStatus(input.Status)
	if err != nil {
		return nil, err
	}
	// 将请求对象转换为持久化模型。
	activity := &model.SeckillActivity{
		// 记录活动所属商品。
		ProductID: input.ProductID,
		// 保存秒杀价格。
		SeckillPrice: input.SeckillPrice,
		// 保存活动总库存上限。
		TotalStock: input.TotalStock,
		// 新活动尚未卖出商品，因此可用库存等于总库存。
		AvailableStock: input.TotalStock,
		// 保存活动开始时间。
		StartTime: input.StartTime,
		// 保存活动结束时间。
		EndTime: input.EndTime,
		// 保存最终确定的活动状态。
		Status: status,
	}
	// 调用 DAO 执行 INSERT。
	if err := s.repo.CreateActivity(activity); err != nil {
		// 插入失败时不返回伪造的成功结果。
		return nil, err
	}
	// 创建成功后返回数据库回填 ID 的活动。
	return activity, nil
}

// List 返回所有秒杀活动，用于前台展示和管理端列表。
func (s *ActivityService) List() ([]model.SeckillActivity, error) {
	// DAO 会负责查询活动及其需要展示的关联商品。
	return s.repo.ListActivities()
}

// Get 按活动 ID 查询一条秒杀活动。
func (s *ActivityService) Get(id uint64) (*model.SeckillActivity, error) {
	// 委托 DAO 查找活动记录。
	activity, err := s.repo.FindActivityByID(id)
	// 统一将“无记录”转换为业务层错误。
	if dao.IsNotFound(err) {
		// Handler 可据此返回 404。
		return nil, ErrNotFound
	}
	// 成功时返回 activity；其他数据库错误原样返回。
	return activity, err
}

// Update 修改一个已存在活动的可编辑配置。
func (s *ActivityService) Update(id uint64, input ActivityInput) (*model.SeckillActivity, error) {
	// 先读取旧活动，既确认存在，也需要用旧总库存执行保护规则。
	activity, err := s.Get(id)
	// 查询失败时不修改任何数据。
	if err != nil {
		// 返回不存在或数据库错误。
		return nil, err
	}
	if input.ProductID != activity.ProductID || input.SeckillPrice < 0 || input.TotalStock <= 0 || !input.EndTime.After(input.StartTime) {
		return nil, ErrInvalidInput
	}
	status := activity.Status
	if input.Status != "" {
		status, err = normalizeActivityStatus(input.Status)
		if err != nil {
			return nil, err
		}
	}
	product, err := s.repo.FindProductByID(activity.ProductID)
	if dao.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	// Already sold units cannot be removed. Increasing the activity allocation
	// also increases available stock by the same amount, keeping the two stock
	// counters internally consistent.
	soldStock := activity.TotalStock - activity.AvailableStock
	if input.TotalStock < soldStock || product.Status != "ON_SALE" || input.TotalStock > product.Stock+soldStock {
		return nil, ErrInvalidInput
	}
	activity.AvailableStock = input.TotalStock - soldStock
	// 覆盖秒杀价格。
	activity.SeckillPrice = input.SeckillPrice
	// 覆盖活动总库存；AvailableStock 已按已售数量同步调整。
	activity.TotalStock = input.TotalStock
	// 覆盖开始时间。
	activity.StartTime = input.StartTime
	// 覆盖结束时间。
	activity.EndTime = input.EndTime
	// 只有调用方确实提交状态时才更新状态，空字符串表示保持原状态。
	activity.Status = status
	// 交由 DAO 执行 UPDATE。
	if err := s.repo.UpdateActivity(activity); err != nil {
		// 更新失败时向上传递错误。
		return nil, err
	}
	// 返回更新后的活动数据。
	return activity, nil
}

// Delete 删除指定秒杀活动。
func (s *ActivityService) Delete(id uint64) error {
	// 先查询活动，保证删除不存在的 ID 会得到明确的 404 业务错误。
	_, err := s.Get(id)
	// 查询失败时停止删除流程。
	if err != nil {
		// 返回不存在或真实数据库错误。
		return err
	}
	// 活动存在后交由 DAO 执行 DELETE。
	return s.repo.DeleteActivity(id)
}
