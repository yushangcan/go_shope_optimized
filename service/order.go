// package service 表示订单编排逻辑属于业务层。
package service

import (
	// fmt 用于把时间戳和用户 ID 组合成学习版订单号。
	"fmt"
	// strings 用于去掉请求唯一 ID 首尾可能出现的空格。
	"strings"
	// time 用于判断秒杀活动是否处在可下单的时间范围内。
	"time"

	// dao 执行订单、活动库存和商品库存的数据库事务。
	"go_shope/dao"
	// model 定义订单数据库模型。
	"go_shope/model"
)

// OrderService 负责普通下单、秒杀下单、订单归属校验和状态流转。
type OrderService struct {
	// repo 是所有订单数据库操作的唯一入口。
	repo *dao.Repository
}

// NewOrderService 创建一个使用指定 Repository 的订单服务。
func NewOrderService(repo *dao.Repository) *OrderService {
	// 记录数据访问依赖并返回服务。
	return &OrderService{repo: repo}
}

// CreateProductOrder 创建普通商品订单，数量由用户选择，价格使用下单时的商品价格快照。
func (s *OrderService) CreateProductOrder(userID, productID uint64, quantity int, requestID string) (*model.Order, error) {
	// 去掉请求 ID 首尾空格，避免空格字符串绕过必填校验。
	requestID = strings.TrimSpace(requestID)
	// 用户、商品、数量和请求 ID 都是创建普通订单所必需的输入。
	if userID == 0 || productID == 0 || quantity <= 0 || requestID == "" {
		return nil, ErrInvalidInput
	}

	// 相同 request_id 再次到达时直接返回第一次创建的订单，不重复扣库存。
	if existing, err := s.repo.FindOrderByRequestID(requestID); err == nil {
		// 只有同一用户对同一普通商品的重复请求才属于幂等重试。
		if existing.UserID == userID && existing.ProductID == productID && existing.OrderType == model.OrderTypeNormal {
			return existing, nil
		}
		// 请求 ID 被其他业务占用时返回冲突，避免把别人的订单返回给当前用户。
		return nil, ErrConflict
	} else if !dao.IsNotFound(err) {
		// 查询幂等记录时数据库出错，不能继续执行可能重复扣库存的操作。
		return nil, err
	}

	// 查询商品是为了验证上架状态，并取得名称和普通售价快照。
	product, err := s.repo.FindProductByID(productID)
	if dao.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	// 下架商品不能通过普通购买接口继续创建订单。
	if product.Status != "ON_SALE" {
		return nil, ErrProductUnavailable
	}
	// 提前返回清晰的库存不足错误；DAO 事务中的条件 UPDATE 仍会负责并发兜底。
	if product.Stock < quantity {
		return nil, ErrOutOfStock
	}

	// 使用当前时间和用户 ID 生成学习版订单号。
	now := time.Now()
	order := &model.Order{
		// OrderNo 是给用户查看的订单编号。
		OrderNo: fmt.Sprintf("%d%d", now.UnixNano(), userID),
		// RequestID 用于识别客户端对同一次点击发起的重试。
		RequestID: requestID,
		// UserID 来自 JWT，不能由请求体冒充。
		UserID: userID,
		// 普通订单没有 ActivityID，该字段保持 nil 并在数据库保存为 NULL。
		ActivityID: nil,
		// OrderType 明确标记这是普通商品订单。
		OrderType: model.OrderTypeNormal,
		// ProductID 记录购买的是哪件商品。
		ProductID: product.ID,
		// ProductName 保存下单时的名称，之后商品改名不会影响历史订单。
		ProductName: product.Name,
		// UnitPrice 保存下单时的普通售价，单位为分。
		UnitPrice: product.Price,
		// Quantity 保存本次购买数量。
		Quantity: quantity,
		// TotalAmount 由单价乘数量得出，全程使用整数分避免浮点误差。
		TotalAmount: product.Price * int64(quantity),
		// 新订单进入待支付状态。
		Status: "PENDING",
	}

	// DAO 使用 MySQL 事务完成条件扣库存和创建订单。
	err = s.repo.CreateProductOrder(order)
	if err == nil {
		return order, nil
	}
	if err == dao.ErrOutOfStock {
		return nil, ErrOutOfStock
	}
	// 两个完全并发的相同请求可能都通过前置查询；数据库唯一索引只会让一个创建成功。
	if existing, findErr := s.repo.FindOrderByRequestID(requestID); findErr == nil && existing.UserID == userID && existing.ProductID == productID && existing.OrderType == model.OrderTypeNormal {
		// 失败的事务已经自动回滚库存，因此安全地返回成功请求创建的同一订单。
		return existing, nil
	}
	// 其余数据库异常原样向上传递。
	return nil, err
}

// CreateSeckillOrder 为一个用户创建一笔单件秒杀订单。
func (s *OrderService) CreateSeckillOrder(userID, activityID uint64, requestID string) (*model.Order, error) {
	// 去掉请求 ID 首尾空格后再验证。
	requestID = strings.TrimSpace(requestID)
	// userID 来自 JWT，activityID 来自 URL，requestID 来自请求体；三者都是下单必需信息。
	if userID == 0 || activityID == 0 || requestID == "" {
		// 缺少任何一个信息都无法正确创建归属明确且可幂等的订单。
		return nil, ErrInvalidInput
	}
	if existing, err := s.repo.FindOrderByRequestID(requestID); err == nil {
		if existing.UserID == userID && existing.ActivityID != nil && *existing.ActivityID == activityID && existing.OrderType == model.OrderTypeSeckill {
			return existing, nil
		}
		return nil, ErrConflict
	} else if !dao.IsNotFound(err) {
		return nil, err
	}
	if _, err := s.repo.FindSeckillOrderByUserAndActivity(userID, activityID); err == nil {
		return nil, ErrConflict
	} else if !dao.IsNotFound(err) {
		return nil, err
	}

	// 查询活动；DAO 会预加载关联商品，后面可直接读取商品状态和名称。
	activity, err := s.repo.FindActivityByID(activityID)
	// 活动不存在时返回业务层 404。
	if dao.IsNotFound(err) {
		// 不允许对不存在的活动下单。
		return nil, ErrNotFound
	}
	// 其他错误可能代表数据库故障，应按原错误处理。
	if err != nil {
		// 终止下单流程。
		return nil, err
	}

	// 获取当前服务器时间，用它与活动时间窗口比较。
	now := time.Now()
	// 只有 ACTIVE 状态并且当前时间处于 [StartTime, EndTime) 的活动才允许下单。
	if activity.Status != "ACTIVE" || now.Before(activity.StartTime) || !now.Before(activity.EndTime) {
		// 未开始、已结束和非活动状态统一返回“活动不可用”。
		return nil, ErrActivityUnavailable
	}
	// 即便活动记录存在，关联商品已下架时也不应再售卖。
	if activity.Product.Status != "ON_SALE" {
		// 用相同业务错误避免暴露不必要实现细节。
		return nil, ErrActivityUnavailable
	}

	// 生成一个学习项目可用的订单号；生产环境通常会替换为雪花算法或专用号段服务。
	orderNo := fmt.Sprintf("%d%d", now.UnixNano(), userID)
	// 先复制活动 ID，再取地址保存；普通订单则会让此字段保持 nil。
	seckillActivityID := activity.ID
	// 构造写入数据库的订单快照，避免商品后续改名/改价影响历史订单。
	order := &model.Order{
		// 保存刚生成的订单号。
		OrderNo: orderNo,
		// 保存客户端请求唯一 ID，数据库会用它阻止同一请求重复建单。
		RequestID: requestID,
		// 保存下单用户，后续按它做订单权限校验。
		UserID: userID,
		// 保存关联秒杀活动。
		ActivityID: &seckillActivityID,
		// 明确标记这是一笔秒杀订单，取消时需要恢复活动库存。
		OrderType: model.OrderTypeSeckill,
		// 保存关联商品 ID。
		ProductID: activity.ProductID,
		// 保存下单时商品名称快照。
		ProductName: activity.Product.Name,
		// 保存下单时秒杀单价快照（单位为分）。
		UnitPrice: activity.SeckillPrice,
		// 当前项目每次秒杀订单固定购买一件。
		Quantity: 1,
		// 一件商品的应付总额等于单价。
		TotalAmount: activity.SeckillPrice,
		// 新创建的订单等待支付。
		Status: "PENDING",
	}
	// DAO 在同一事务中扣减活动库存、商品库存并插入订单；任一步失败都会回滚。
	err = s.repo.CreateSeckillOrder(order)
	// 数据库事务失败时需要转换部分可预期错误。
	if err != nil {
		// 条件 UPDATE 没有更新到行时，DAO 返回库存不足错误。
		if err == dao.ErrOutOfStock {
			// 对外使用稳定的库存不足业务错误。
			return nil, ErrOutOfStock
		}
		// Concurrent retries can arrive after both pre-checks. Resolve them back
		// to the original order instead of leaking a MySQL duplicate-key error.
		if existing, findErr := s.repo.FindOrderByRequestID(requestID); findErr == nil && existing.UserID == userID && existing.ActivityID != nil && *existing.ActivityID == activityID && existing.OrderType == model.OrderTypeSeckill {
			return existing, nil
		}
		if _, findErr := s.repo.FindSeckillOrderByUserAndActivity(userID, activityID); findErr == nil {
			return nil, ErrConflict
		}
		return nil, err
	}
	// 事务成功后返回包含 ID、订单号等数据的订单。
	return order, nil
}

// ListByUserID 只列出指定用户自己的订单。
func (s *OrderService) ListByUserID(userID uint64) ([]model.Order, error) {
	// 将用户 ID 作为查询条件，防止普通用户看到全站订单。
	return s.repo.ListOrdersByUserID(userID)
}

// ListAll returns all store orders to the admin-only HTTP route.
func (s *OrderService) ListAll() ([]model.Order, error) {
	return s.repo.ListOrders()
}

// GetForUser 查询一笔订单，并确认它属于当前登录用户。
func (s *OrderService) GetForUser(id, userID uint64) (*model.Order, error) {
	// 先按订单主键读取订单。
	order, err := s.repo.FindOrderByID(id)
	// 没有该订单时转换为 404 业务错误。
	if dao.IsNotFound(err) {
		// 不返回数据库层错误。
		return nil, ErrNotFound
	}
	// 数据库其他错误时无法继续做权限判断。
	if err != nil {
		// 原样返回错误。
		return nil, err
	}
	// 校验订单所属用户和 JWT 中的当前用户是否一致。
	if order.UserID != userID {
		// 订单存在但不属于当前人，应返回 403 而不是订单内容。
		return nil, ErrForbidden
	}
	// 归属校验通过后返回订单。
	return order, nil
}

// Pay 模拟支付成功：只允许把待支付订单改为已支付。
func (s *OrderService) Pay(id, userID uint64) error {
	// 复用统一状态机方法，目标状态设为 PAID。
	return s.changeStatus(id, userID, "PAID")
}

// Cancel 取消待支付订单，并由 DAO 事务按订单类型恢复库存。
func (s *OrderService) Cancel(id, userID uint64) error {
	// 秒杀订单恢复活动库存和商品库存，普通订单只恢复商品库存。
	rows, err := s.repo.CancelOrderAndRestoreStock(id, userID)
	// DAO 未找到可取消订单时，订单通常已支付或已取消。
	if dao.IsNotFound(err) {
		// 对外表达为不允许的状态流转。
		return ErrInvalidOrderStatus
	}
	// 数据库事务出错时交给上层处理。
	if err != nil {
		// 返回原始错误。
		return err
	}
	// 正常取消必须精确影响一行订单；0 行表示状态已改变或用户无权取消。
	if rows != 1 {
		// 防止重复取消造成重复回补库存。
		return ErrInvalidOrderStatus
	}
	// 订单取消与库存恢复均成功。
	return nil
}

// changeStatus 执行不涉及库存回补的订单状态转换。
func (s *OrderService) changeStatus(id, userID uint64, to string) error {
	// 先确认订单存在且归当前用户所有。
	_, err := s.GetForUser(id, userID)
	// 不存在、无权或查询失败时停止更新。
	if err != nil {
		// 返回 GetForUser 已定义的业务错误。
		return err
	}
	// DAO 使用 WHERE status = PENDING 的条件更新，只允许待支付订单流转。
	rows, err := s.repo.UpdateOrderStatus(id, userID, "PENDING", to)
	// SQL 执行异常时向上传递错误。
	if err != nil {
		// 返回原始数据库错误。
		return err
	}
	// 更新不到一行说明订单已不是 PENDING，不能重复支付或从其他状态跳转。
	if rows != 1 {
		// 返回统一的非法状态错误。
		return ErrInvalidOrderStatus
	}
	// 状态更新成功。
	return nil
}
