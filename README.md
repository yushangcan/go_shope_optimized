# Go Shope：RabbitMQ 单机高并发秒杀版

这是普通同步 MySQL 基线项目 `go_shope` 的独立优化版本，用于实习项目展示和统一压测对比。项目只面向单机，不引入 Nginx、服务发现、分片或多实例部署；重点实现 Redis Lua 原子校验、RabbitMQ 异步削峰、幂等、防超卖、MySQL 事务和失败补偿。

详细的项目介绍、架构、流程、面试讲解和简历表述见 [`docs/PROJECT-INTRODUCTION.md`](docs/PROJECT-INTRODUCTION.md)。

云服务器部署、机器规格、端口和压测切换步骤见 [`docs/CLOUD-DEPLOYMENT.md`](docs/CLOUD-DEPLOYMENT.md)。

## 核心流程

```text
k6 -> Go API
       -> Redis Lua：活动校验、幂等、一人一单、预扣库存
       -> RabbitMQ durable queue
       -> Go Worker：MySQL 事务扣减库存并创建订单
       -> Redis request status：查询最终状态
```

API 返回 `202 Accepted` 只表示消息已经发布到 RabbitMQ，不代表订单已经落库。最终状态通过 `GET /api/seckill/requests/:request_id` 查询。

## 启动单机环境

```powershell
cd D:\GO_study\GoLand\go_shope_optimized
docker compose -f compose.optimized.yaml up --build -d
```

服务端口：

| 服务 | 地址 |
| --- | --- |
| Go API | `http://localhost:8081` |
| MySQL | `localhost:3308` |
| Redis | `localhost:6379` |
| RabbitMQ AMQP | `localhost:5672` |
| RabbitMQ 管理页面 | `http://localhost:15672` |
| Prometheus | `http://localhost:9090` |

RabbitMQ 本地实验账号为 `seckill/seckillpass`，不要直接用于公网部署。

## 准备活动并压测

```powershell
$env:BASE_URL = 'http://localhost:8081'
$env:ADMIN_USERNAME = 'your-admin'
$env:ADMIN_PASSWORD = 'your-password'
.\benchmark\scripts\prepare-optimized.ps1

$env:ACTIVITY_ID = '1'
$env:K6_VUS = '100'
$env:K6_DURATION = '60s'
k6 run .\benchmark\k6\optimized-admission.js
```

压测场景位于 [`benchmark/`](benchmark/)：受理延迟、状态查询、重复 request_id 和售罄风暴。优化版应与普通版使用相同库存、VU 数、持续时间和机器环境，最后再核对 MySQL 订单数量与请求状态。

## 主要接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/auth/register` | 注册用户 |
| `POST` | `/api/auth/login` | 登录并获取 JWT |
| `GET` | `/api/products` | 商品列表 |
| `GET` | `/api/seckill/activities` | 秒杀活动列表 |
| `POST` | `/api/seckill/activities/:id/requests` | Redis 原子受理并发布 RabbitMQ 消息 |
| `GET` | `/api/seckill/requests/:request_id` | 查询异步处理状态 |
| `POST` | `/api/admin/seckill/activities/:id/publish` | 将活动库存预热到 Redis |
| `GET` | `/livez` | 进程存活检查 |
| `GET` | `/readyz` | Redis 就绪检查 |
| `GET` | `/metrics` | Prometheus 指标 |

## 已实现的并发与一致性措施

- Redis Lua 将活动时间、活动状态、request_id、一人一单和预扣库存放在原子脚本中执行；
- RabbitMQ 队列使用 durable queue、persistent message、publisher confirm 和手动 ACK；
- API 发布 MQ 失败时执行 Redis 库存和用户标记补偿并返回 `503`；
- Worker 使用 MySQL 事务同时处理活动库存、商品库存和订单；
- MySQL 条件更新和唯一索引防止最终超卖及重复订单；
- Worker 重复收到同一消息时按 request_id 查找已有订单，不重复扣库存；
- Worker 处理失败时记录 `FAILED` 并执行 Redis 补偿；
- Redis 重启后 API 启动会从 MySQL 重新预热活动状态；
- Prometheus 记录 HTTP 延迟、受理结果和 Worker 成功/失败数量。

## 与普通基线版的边界

普通版本位于 [yushangcan/go_shope](https://github.com/yushangcan/go_shope)，基线标签为 `baseline-v1.0.0`；本仓库为 RabbitMQ 优化版本，标签为 `optimized-code-v1`。当前项目是单机对比实验，不宣称分布式高可用。正式压测尚未执行，仓库不包含虚构的 QPS、p95、p99 或成功率数据。

## 代码检查

```powershell
go build ./...
docker compose -f compose.optimized.yaml config --quiet
```

正式压测前，请额外记录 commit SHA、活动库存、VU、持续时间、请求吞吐、延迟分位数、错误率、最终订单数以及 API、Redis、RabbitMQ、MySQL 和 Worker 的资源使用情况。
