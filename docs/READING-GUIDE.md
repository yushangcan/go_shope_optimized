# 基础版阅读顺序

不要从所有文件同时开始看。按下面四条完整调用链阅读，每读完一条就运行对应接口。

## 0. 程序如何启动

1. `main.go`：读取配置、连接 MySQL、创建所有 Service、注册路由、启动 Gin。
2. `config/config.go`：了解 `config.yaml` 与环境变量如何组合成配置。
3. `dao/database.go`：了解 GORM 如何连接 MySQL，并根据 model 自动建表。
4. `router/router.go`：查看每个 URL 最终绑定到哪个 Handler。

## 1. 注册链路

请求：`POST /api/auth/register`

```text
router/auth.go Register
    -> service/user.go Register
    -> dao/user.go FindUserByUsername / CreateUser
    -> MySQL users 表
```

先看 `router` 如何拿到 JSON，再看 `service` 如何校验和生成 bcrypt 哈希，最后看 `dao` 如何写入数据库。

## 2. 商品 CRUD 链路

管理员请求：`POST|PUT|DELETE /api/admin/products...`

```text
middleware/auth.go Auth + RequireAdmin
    -> router/product.go
    -> service/product.go
    -> dao/product.go
    -> MySQL products 表
```

普通用户只走 `GET /api/products` 和 `GET /api/products/:id`，不需要 JWT。

## 3. 秒杀活动 CRUD 链路

管理员请求：`POST|PUT|DELETE /api/admin/seckill/activities...`

```text
router/activity.go
    -> service/activity.go
    -> dao/activity.go
    -> MySQL seckill_activities 表
```

创建活动前会检查活动对应商品存在、上架，且活动库存不超过商品库存。

## 4. 下单与取消订单链路

用户下单：`POST /api/seckill/activities/:id/orders`

```text
middleware/auth.go
    -> router/order.go
    -> service/order.go
    -> dao/order.go Transaction
         -> 条件扣活动库存
         -> 条件扣商品库存
         -> 创建订单
```

取消订单也在 `dao/order.go` 的事务中完成：订单状态改为 `CANCELLED` 后，再恢复活动库存和商品库存。

## 常用缩写

- `c *gin.Context`：一次 HTTP 请求的上下文，包含路径参数、JSON 请求体、请求头和响应方法。
- `Service`：业务规则所在层，例如注册检查、活动时间判断。
- `DAO`：Data Access Object，数据访问层，只读写 MySQL。
- `GORM`：Go 的 ORM 库，用 Go 方法生成 SQL。
- `JWT`：登录后客户端保存的签名令牌；每次受保护请求通过 `Authorization: Bearer <token>` 带回。
- `Transaction`：事务。里面任意一步返回错误，前面已经写入的数据库操作会一起回滚。
