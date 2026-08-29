// package service 定义业务层：这里负责业务规则，而不是 HTTP 或 SQL 细节。
package service

// errors 用来创建可被 router 层识别的固定业务错误。
import "errors"

// 这一组错误是 Service 与 Router 之间的约定；Router 会将它们转换为对应的 HTTP 状态码。
var (
	// ErrInvalidInput 表示调用方传入的数据不符合业务要求。
	ErrInvalidInput = errors.New("invalid input")
	// ErrConflict 表示本应唯一的数据已经存在，例如重复注册的用户名。
	ErrConflict = errors.New("conflict")
	// ErrUnauthorized 表示用户尚未通过身份验证，或用户名/密码不正确。
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden 表示已经登录，但无权操作不属于自己的资源。
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound 表示所需的用户、商品、活动或订单不存在。
	ErrNotFound = errors.New("not found")
	// ErrActivityUnavailable 表示秒杀活动未开始、已结束或当前不可用。
	ErrActivityUnavailable = errors.New("activity unavailable")
	// ErrProductUnavailable 表示普通商品已下架，当前不能购买。
	ErrProductUnavailable = errors.New("product unavailable")
	// ErrOutOfStock 表示活动库存或商品库存已经不足。
	ErrOutOfStock = errors.New("out of stock")
	// ErrInvalidOrderStatus 表示订单当前状态不允许进行支付或取消。
	ErrInvalidOrderStatus = errors.New("invalid order status")
)
