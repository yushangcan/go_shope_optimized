# Go Shope 单机高并发秒杀项目介绍

## 1. 项目概述

Go Shope 是一个基于 Go 实现的电商秒杀示例项目。本项目包含一个普通同步 MySQL 基线版本和一个面向单机高并发压测的优化版本，用于对比不同下单链路在并发场景下的行为和性能。

当前文档介绍的是 `go_shope_optimized` 优化版。项目定位是适合实习和校招面试展示的单机高并发项目，重点体现以下工程能力：

- 高并发请求下的库存安全控制；
- 基于 `request_id` 的接口幂等；
- Redis Lua 脚本保证多个检查和扣库存操作的原子性；
- Redis Stream 将快速受理和慢速落库解耦；
- Worker 异步创建订单；
- MySQL 事务作为最终数据一致性保障；
- 失败场景下的库存补偿和状态记录；
- 可观测的 HTTP、受理和异步消费指标；
- 可重复执行的单机压测场景。

项目不以分布式部署或生产级高可用为目标。当前运行环境是单机上的一个 Go API、一个 Go Worker、一个 Redis、一个 MySQL 和一个 Prometheus，重点是把秒杀核心流程做正确、做清楚，并为后续基线对比压测提供稳定代码。

## 2. 项目背景与问题

普通同步下单流程通常由一个 HTTP 请求完成以下操作：查询活动、判断活动时间、判断用户是否重复购买、扣减活动库存、扣减商品库存、创建订单。这种实现逻辑直观，但在瞬时并发较高时会出现几个典型问题：

1. 大量请求同时访问 MySQL，数据库连接、事务和行锁成为瓶颈；
2. 如果库存读取和库存扣减分成多个步骤，可能出现超卖；
3. 用户重复点击或客户端重试可能重复创建订单；
4. 请求一直等待订单事务完成，接口响应时间受 MySQL 写入耗时影响；
5. 数据库异常、进程中断或消息重复消费时，容易出现库存和订单状态不一致。

本项目没有简单地增加缓存，而是把“快速判断和受理”和“最终持久化”拆成两个阶段：

- Redis 负责高并发入口的原子校验和预扣库存；
- Redis Stream 负责保存待处理订单事件；
- Worker 负责调用原有 MySQL 事务完成最终订单创建；
- 失败时执行补偿，恢复 Redis 中的预扣状态。

## 3. 技术栈

| 层次 | 技术 | 用途 |
| --- | --- | --- |
| Web 服务 | Go、Gin | HTTP 路由、参数解析和 JSON 响应 |
| 业务层 | Go Service | 用户、商品、活动、订单和优化秒杀业务 |
| 数据访问 | GORM | MySQL 查询、事务和条件更新 |
| 关系数据库 | MySQL 8.0 | 用户、商品、活动、订单最终持久化 |
| 高并发入口 | Redis 7 | 活动状态、预扣库存、用户购买集合和请求状态 |
| 原子逻辑 | Redis Lua | 一次完成幂等、一人一单、库存扣减和事件投递 |
| 异步队列 | Redis Streams | 保存待落库的秒杀订单事件 |
| 监控 | Prometheus client | HTTP、受理结果和 Worker 结果指标 |
| 鉴权 | JWT、bcrypt | 登录身份验证和密码安全存储 |
| 压测 | k6 | 受理、只读、重复请求和售罄场景压测 |
| 部署 | Docker Compose | 在一台机器上启动完整对比环境 |

## 4. 整体架构

```text
                         k6 压测
                            |
                            v
                    +----------------+
                    | Go API :8081  |
                    +----------------+
                       |           |
             普通业务 |           | 秒杀受理
                       v           v
                 +---------+   +-------------------+
                 | MySQL   |   | Redis Lua Script  |
                 | GORM    |   | 原子校验/预扣库存  |
                 +---------+   +-------------------+
                                      |
                                      v
                              Redis Stream
                                      |
                                      v
                              +---------------+
                              | Go Worker     |
                              | 异步订单落库   |
                              +---------------+
                                      |
                                      v
                              MySQL Transaction
```

单机版本不使用 Nginx、服务发现、分片、Sentinel 或多实例负载均衡。API 和 Worker 是同一个项目中的两个独立进程，可以在同一台机器上分别启动，也可以由 Compose 启动。

## 5. 核心业务流程

### 5.1 活动准备

管理员通过已有活动管理接口创建商品和秒杀活动。API 启动时会读取 MySQL 中已有活动，并将活动状态、开始时间、结束时间和可用库存写入 Redis。活动创建后也可以调用管理端发布接口主动预热 Redis。

Redis 中的主要 Key 结构如下：

```text
seckill:activity:{activity_id}   Hash
  status                         活动状态
  start_at                       Unix 时间戳
  end_at                         Unix 时间戳
  stock                          Redis 预扣库存

seckill:buyers:{activity_id}     Set
  user_id                        已成功受理的用户

seckill:request:{request_id}     Hash
  status                         ACCEPTED/PROCESSING/SUCCEEDED/FAILED
  user_id                        请求所属用户
  activity_id                    秒杀活动
  order_id                       最终订单 ID（成功后写入）
  reason                         失败原因（失败后写入）

seckill:stream:orders            Stream
  request_id
  user_id
  activity_id
```

### 5.2 秒杀请求受理

接口：

```http
POST /api/seckill/activities/:id/requests
Authorization: Bearer <JWT>
Content-Type: application/json

{"request_id":"client-generated-id"}
```

请求进入 API 后，Redis Lua 脚本按以下顺序执行：

1. 根据 `request_id` 查询请求状态；
2. 如果请求已存在，校验原请求所属用户；同一用户重试返回原状态，不重复扣库存；不同用户复用同一 ID 则返回冲突；
3. 判断活动状态和时间窗口；
4. 判断用户是否已经在活动购买集合中；
5. 判断 Redis 预扣库存是否大于 0；
6. 原子扣减 Redis 库存；
7. 将用户加入已购买集合；
8. 写入 `ACCEPTED` 请求状态；
9. 将订单事件写入 Redis Stream。

上述操作在同一个 Lua 脚本中完成。即使多个请求同时到达，也不会在“检查库存”和“扣减库存”之间被其他请求插入，从而避免 Redis 层面的超卖和重复受理。

成功响应示例：

```json
{
  "request_id": "client-generated-id",
  "status": "ACCEPTED",
  "activity_id": 1
}
```

HTTP 状态码为 `202 Accepted`。这表示请求已经进入异步处理链路，不表示 MySQL 订单已经创建成功。

### 5.3 Worker 异步落库

Worker 启动后创建或复用 Redis Stream Consumer Group，持续读取订单事件：

1. 将请求状态更新为 `PROCESSING`；
2. 根据活动 ID 查询活动和商品快照；
3. 调用 MySQL 事务创建秒杀订单；
4. 事务内条件扣减活动库存；
5. 事务内条件扣减商品总库存；
6. 两次扣减都成功后插入订单；
7. 将请求状态更新为 `SUCCEEDED` 并记录订单 ID；
8. ACK Stream 消息。

MySQL 事务中的库存扣减使用条件更新，例如：

```sql
UPDATE seckill_activities
SET available_stock = available_stock - 1
WHERE id = ? AND available_stock > 0;
```

如果受影响行数不是 1，则认为没有库存。商品总库存也使用相同的条件更新。任意一步失败，事务回滚前面的数据库修改。

## 6. 幂等性设计

### 6.1 接口幂等

客户端每次业务尝试生成唯一 `request_id`，并在网络超时或用户重复点击时复用该 ID。Redis Lua 先查询请求状态：

- 原用户重复提交：返回原请求状态，不再次扣库存、不再次写入 Stream；
- 其他用户使用同一 ID：返回 `REQUEST_ID_CONFLICT`，防止跨用户篡改请求；
- 请求不存在：执行一次新的受理流程。

### 6.2 数据库幂等

订单表仍然保留数据库唯一约束：

- `orders.request_id` 唯一，避免同一请求创建多笔订单；
- `(user_id, activity_id)` 联合唯一，保证一人一场活动只能有一笔秒杀订单。

Redis 负责快速挡住大多数重复请求，MySQL 唯一索引负责最终兜底。即使 Worker 因为重启或消息重复投递再次处理同一事件，也会先按 `request_id` 查找已有订单；如果数据库已经成功创建订单，则直接把请求标记为 `SUCCEEDED`，不会再次补库存。

## 7. 防止超卖设计

项目采用两层库存保护：

### 第一层：Redis Lua 预扣

高并发请求首先竞争 Redis 中的活动库存。库存判断、扣减、用户集合写入和 Stream 投递在一个 Lua 脚本内执行，避免并发交错导致库存变负。

### 第二层：MySQL 条件更新

Worker 落库时再次对活动库存和商品库存做 `stock > 0` 条件扣减，并将扣库存与订单创建放在同一个事务中。Redis 层的预扣不是最终事实，MySQL 仍然会对异常情况进行最后校验。

### 失败补偿

如果 MySQL 不可用、活动不存在或最终库存不足，Worker 会：

1. 将请求状态标记为 `FAILED`；
2. 通过 Redis Lua 将活动预扣库存加回；
3. 从活动购买集合移除用户；
4. ACK 当前 Stream 消息，避免无限重复消费。

这样可以把 Redis 中已经占用但没有形成订单的临时状态释放出来。

## 8. 状态模型与接口语义

| 状态 | 含义 |
| --- | --- |
| `ACCEPTED` | Redis 已完成原子受理，事件已写入 Stream |
| `PROCESSING` | Worker 已领取事件，正在执行 MySQL 事务 |
| `SUCCEEDED` | MySQL 订单已创建，并记录订单 ID |
| `FAILED` | 最终落库失败，已执行 Redis 补偿或记录失败原因 |

最终状态查询接口：

```http
GET /api/seckill/requests/:request_id
Authorization: Bearer <JWT>
```

压测时应分别统计：

- API 受理延迟：请求到 Redis Lua 完成的时间；
- 异步处理延迟：请求从 `ACCEPTED` 到 `SUCCEEDED` 或 `FAILED` 的时间；
- 最终业务成功率：MySQL 订单和请求状态核对后的结果。

不能把 HTTP 202 直接当成订单成功。

## 9. 项目目录

```text
cmd/
├── api/main.go                 # API 进程入口
└── worker/main.go              # 异步 Worker 入口

config/                         # Viper 配置加载
dao/                            # GORM 数据访问和事务
model/                          # 用户、商品、活动、订单模型
middleware/                     # JWT 和管理员鉴权
router/                         # 普通业务路由和 Handler
service/                        # 普通业务 Service

internal/redisstore/
└── store.go                    # Redis Key、Lua、Stream 操作
internal/optimized/
├── router.go                   # 优化秒杀接口
├── service.go                  # 受理、发布、状态和落库业务
└── worker.go                   # Stream 消费和失败补偿
internal/observability/
├── metrics.go                  # Prometheus 指标
└── middleware.go               # HTTP 指标中间件

benchmark/
├── k6/                         # 单机压测场景
├── scripts/                    # 活动准备和结果核对脚本
└── results/                    # 保存真实压测结果，不提交虚构数据

compose.optimized.yaml          # 单机 Compose 环境
Dockerfile                      # API/Worker 镜像构建
```

## 10. 启动与压测准备

启动单机优化版：

```powershell
cd D:\GO_study\GoLand\go_shope_optimized
docker compose -f compose.optimized.yaml up --build -d
```

准备管理员账号、商品和秒杀活动：

```powershell
$env:BASE_URL = 'http://localhost:8081'
$env:ADMIN_USERNAME = 'your-admin'
$env:ADMIN_PASSWORD = 'your-password'
.\benchmark\scripts\prepare-optimized.ps1
```

脚本会创建活动并调用发布接口，将活动预热到 Redis。然后使用输出的 `ACTIVITY_ID` 运行 k6：

```powershell
$env:ACTIVITY_ID = '1'
$env:K6_VUS = '100'
$env:K6_DURATION = '60s'
k6 run .\benchmark\k6\optimized-admission.js
```

当前仓库只准备代码和压测入口，正式压测数据应在统一实验条件下生成并保存，包括 commit SHA、库存量、VU 数、持续时间、吞吐量、延迟分位数、错误率、最终订单数和机器资源使用情况。

## 11. 与普通基线版的对比边界

普通版位于原仓库 `yushangcan/go_shope`，基线标签为 `baseline-v1.0.0`；优化版位于 `yushangcan/go_shope_optimized`，代码标签为 `optimized-code-v1`。

| 对比项 | 普通基线版 | 单机优化版 |
| --- | --- | --- |
| 下单方式 | HTTP 请求内同步执行 MySQL 事务 | Redis 快速受理，Worker 异步执行 MySQL |
| 库存入口 | MySQL 条件扣减 | Redis Lua 预扣 + MySQL 条件扣减 |
| 重复请求 | MySQL request_id 唯一约束 | Redis 幂等 + MySQL 唯一约束 |
| 一人一单 | MySQL 联合唯一约束 | Redis Set 快速判断 + MySQL 联合唯一约束 |
| 请求响应 | 等待订单事务结果 | 先返回 202，再查询最终状态 |
| 压测关注点 | 同步接口整体延迟 | 受理延迟、异步完成延迟和最终成功率 |

两版压测必须使用相同的商品库存、活动库存、用户规模、并发数、持续时间和机器环境，不能只比较不同口径的接口耗时。

## 12. 项目亮点（简历表述）

可以在简历中这样描述：

> 基于 Go、Gin、GORM、MySQL 和 Redis 实现单机高并发秒杀系统。使用 Redis Lua 将活动校验、request_id 幂等、一人一单和库存预扣合并为原子操作，避免并发超卖；通过 Redis Streams 解耦请求受理与订单落库，由 Worker 异步执行 MySQL 事务，并结合数据库唯一索引、重复消费检查和失败补偿保证订单一致性；设计 `ACCEPTED/PROCESSING/SUCCEEDED/FAILED` 状态机及 Prometheus 指标，使用 k6 对同步基线版和异步优化版进行统一压测对比。

如果还没有完成正式压测，不应在简历中填写具体 QPS、p95、p99 或成功率。完成真实压测后，再根据实验记录补充数据和机器配置。

## 13. 面试讲解重点

### 为什么需要 Redis Lua？

因为库存判断、库存扣减、重复购买判断和请求状态写入必须具备原子性。使用多个普通 Redis 命令时，命令之间可能被其他请求插入；Lua 脚本可以让这些操作在 Redis 内连续执行。

### Redis 扣库存后 MySQL 失败怎么办？

Worker 将请求标记为失败，并执行补偿 Lua：恢复活动库存、移除用户购买标记。MySQL 订单和库存写入仍然在事务中完成，避免只成功一半。

### 为什么还需要 MySQL 唯一索引？

Redis 是高并发入口状态，MySQL 是最终持久化事实。唯一索引可以在 Redis 状态丢失、Worker 重试或消息重复投递时提供最后一道幂等保护。

### 202 返回是不是下单成功？

不是。202 只表示请求被 Redis 接受并进入 Stream。必须查询 request status，并核对 MySQL 订单，才能确认最终成功或失败。

### 这个项目的高可用做到什么程度？

当前只面向单机压测，重点是并发控制和数据一致性，不宣称分布式高可用。API、Worker、Redis 和 MySQL 都是单实例，后续如果要扩展到生产环境，还需要独立设计多实例、故障切换、持久化恢复和监控告警。

## 14. 当前状态

- 单机 API、Worker、Redis、MySQL、Prometheus Compose 已准备；
- Redis Lua 幂等、防重复购买和预扣库存逻辑已实现；
- MySQL 事务、唯一索引和条件库存扣减已保留；
- Worker 异步落库、重复消费处理和失败补偿已实现；
- k6 受理、查询、重复请求和售罄场景已准备；
- `go build ./...` 和 Compose 配置检查已完成；
- 正式压测、故障注入和性能数据采集尚未执行。

