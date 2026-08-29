// package service 表示本文件属于业务层。
package service

import (
	// strings 用于删除用户名前后的无意义空格。
	"strings"

	// dao 提供数据库访问方法和“记录不存在”的判断工具。
	"go_shope/dao"
	// model 定义要保存到数据库的 User 结构体。
	"go_shope/model"
	// bcrypt 用于安全地生成和校验密码哈希。
	"golang.org/x/crypto/bcrypt"
)

// UserService 封装注册、登录和个人资料查询的业务规则。
type UserService struct {
	// repo 是数据访问对象；Service 不直接编写 GORM 查询。
	repo *dao.Repository
}

// NewUserService 接收已创建的 Repository，并返回可被 Handler 使用的用户服务。
func NewUserService(repo *dao.Repository) *UserService {
	// 将同一个 Repository 注入服务，后续所有用户操作都通过它访问数据库。
	return &UserService{repo: repo}
}

// Register 校验输入、检查用户名唯一性、加密密码，并创建普通用户。
func (s *UserService) Register(username, password string) (*model.User, error) {
	// TrimSpace 避免 “alice” 和 “ alice ” 被当作两个不同用户名。
	username = strings.TrimSpace(username)

	// 用户名至少 3 位、最多 100 位；bcrypt 最多只处理 72 字节密码，因此密码也限制在该范围内。
	if len(username) < 3 || len(username) > 100 || len(password) < 6 || len(password) > 72 {
		// 输入不合格时不访问数据库，直接交给 Router 返回参数错误。
		return nil, ErrInvalidInput
	}

	// 查询同名用户，防止数据库中出现重复的用户名。
	_, err := s.repo.FindUserByUsername(username)
	// 查询成功说明用户名已经被占用。
	if err == nil {
		// 用业务错误表达“重复注册”，而不是暴露底层数据库细节。
		return nil, ErrConflict
	}
	// 只有“未找到记录”才是注册流程允许继续的查询结果。
	if !dao.IsNotFound(err) {
		// 例如数据库连接异常时，原样向上返回，不能误认为用户名可用。
		return nil, err
	}

	// 使用 bcrypt 默认成本从明文密码生成不可逆哈希。
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	// 哈希计算失败时不创建用户。
	if err != nil {
		// 将错误交给上层统一处理。
		return nil, err
	}

	// 组装数据库模型；只保存哈希，不保存明文密码；新注册账号默认是普通用户。
	user := &model.User{
		// 写入已经清理过空格的用户名。
		Username: username,
		// 将 bcrypt 的字节切片转换为数据库可保存的字符串。
		PasswordHash: string(hash),
		// 不开放注册管理员，默认角色固定为 USER。
		Role: "USER",
	}
	// 调用 DAO 执行真正的 INSERT 操作。
	if err := s.repo.CreateUser(user); err != nil {
		// 数据库写入失败时不返回一个看似创建成功的用户。
		return nil, err
	}
	// GORM 创建成功后会回填 user.ID 等字段，将完整用户返回 Handler。
	return user, nil
}

// Login 根据用户名查找用户，并用 bcrypt 校验密码。
func (s *UserService) Login(username, password string) (*model.User, error) {
	// 登录查询也使用清理过空格的用户名，和注册规则保持一致。
	username = strings.TrimSpace(username)
	// 通过 DAO 查询包含密码哈希的用户记录。
	user, err := s.repo.FindUserByUsername(username)
	// 查询失败需要区分“用户不存在”和系统错误。
	if err != nil {
		// 不向客户端透露用户名是否存在，避免被用于枚举账号。
		if dao.IsNotFound(err) {
			// 用户不存在统一表现为“未认证”。
			return nil, ErrUnauthorized
		}
		// 非未找到错误通常是数据库故障，应保留原始错误。
		return nil, err
	}

	// 将用户输入的明文密码与数据库中保存的 bcrypt 哈希进行安全比较。
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	// 比较失败说明密码错误。
	if err != nil {
		// 同样统一返回未认证错误，避免泄露更多信息。
		return nil, ErrUnauthorized
	}
	// 校验成功后由 Handler 使用该用户生成 JWT。
	return user, nil
}

// GetProfile 按用户 ID 查询个人资料。
func (s *UserService) GetProfile(id uint64) (*model.User, error) {
	// 委托 DAO 从数据库读取用户。
	user, err := s.repo.FindUserByID(id)
	// 将 DAO 的“未找到”语义转换为业务层统一错误。
	if dao.IsNotFound(err) {
		// Handler 据此返回 404。
		return nil, ErrNotFound
	}
	// 查询成功时 err 为 nil；其他数据库错误会保留并向上传递。
	return user, err
}
