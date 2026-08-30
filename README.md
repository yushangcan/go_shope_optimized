# Go Shop: ordinary MySQL baseline

This repository is the ordinary comparison baseline for later load-testing and optimization work. It is a synchronous MySQL-only monolith and deliberately contains no Redis, RabbitMQ, caching, rate limiting, sharding, asynchronous order pipeline, or distributed components.

The baseline still keeps correctness requirements such as password hashing, JWT authorization, database transactions, conditional stock deduction, request idempotency, and integer-cent prices. Those rules prevent invalid business data; they are not performance optimizations.

## Included business flow

1. Register and log in with bcrypt password hashes and a JWT.
2. Admin CRUD for products and seckill activities.
3. Public product and activity queries.
4. Authenticated synchronous ordinary-product and seckill ordering.
5. Pay or cancel an order. Cancelling a pending order restores MySQL inventory in the same transaction.
6. Buyer storefront and merchant dashboard served directly by Gin.

## Run with Docker Compose

This project now uses Docker Compose. You do **not** need to start the local Windows `MYSQL80` service.

```powershell
docker compose up --build
```

Compose starts two containers:

```text
go-shope-app    Go + Gin API       http://localhost:8080
go-shope-mysql  MySQL 8.0          localhost:3307 (optional GUI access)
```

The API waits for MySQL's health check. On first startup, GORM automatically creates the four basic tables: `users`, `products`, `seckill_activities`, and `orders`.

Configuration is read by Viper. `config.yaml` provides defaults, while `MYSQL_DSN` and `JWT_SECRET` from Docker Compose or PowerShell override those values.

Verify that the API is running in a second PowerShell window:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Expected result:

```json
{"status":"ok"}
```

### Open the frontend pages

After the service is running, open these addresses in a browser:

| Page | Address | What it provides |
| --- | --- | --- |
| Buyer storefront | `http://localhost:8080/` | Product browsing, a top-of-page seckill area, login/registration, and seckill ordering |
| Merchant dashboard | `http://localhost:8080/admin` | Product/activity CRUD and the store-wide order overview |

The merchant dashboard uses the same JWT login as the buyer page. Before testing product or activity creation, change that user's `role` to `ADMIN` in MySQL; this basic project intentionally does not provide a public administrator-registration endpoint.

Stop containers while retaining database data:

```powershell
docker compose down
```

Reset the local Docker database completely (this deletes all test users, products, activities, and orders):

```powershell
docker compose down -v
```

### Why the MySQL address is `mysql:3306`

The Go application runs inside its own container. From there, `127.0.0.1` means the application container itself, not the database container. Docker Compose provides an internal DNS name equal to the service name, so the Go DSN uses:

```text
tcp(mysql:3306)
```

The host mapping `3307:3306` is only for tools running on Windows, such as a database GUI. It is not used by the Go container.

### Run Go directly instead of Docker

If you later want to run `go run .` directly on Windows while the Docker MySQL container is up, use host port `3307`:

```powershell
$env:MYSQL_DSN='root:root123456@tcp(127.0.0.1:3307)/go_shope?charset=utf8mb4&parseTime=True&loc=Local'
$env:JWT_SECRET='change-this-local-development-jwt-secret-2026'
go run .
```

## Main endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | Service health check |
| POST | `/api/auth/register` | Register a user |
| POST | `/api/auth/login` | Obtain a JWT |
| GET | `/api/products` | List on-sale products |
| GET | `/api/products/:id` | Get product detail |
| POST | `/api/products/:id/orders` | Create an ordinary product order |
| GET | `/api/seckill/activities` | List activities |
| POST | `/api/seckill/activities/:id/orders` | Create one synchronous seckill order |
| GET | `/api/orders` | List the current user's orders |
| POST | `/api/orders/:id/pay` | Simulate payment |
| POST | `/api/orders/:id/cancel` | Cancel a pending order and restore inventory |

Product and activity write endpoints are under `/api/admin`. The admin dashboard also uses `GET /api/admin/products` and `GET /api/admin/orders` for full management lists. A newly registered user has the `USER` role. To test admin endpoints, change the required user's `role` to `ADMIN` directly in MySQL during this basic development stage.

## Verification

Run the code-level checks:

```powershell
go test ./...
go vet ./...
go build ./...
node --check web/assets/app.js
docker compose config
```

The MySQL-backed smoke test is opt-in because it starts a complete HTTP flow and writes temporary records. It removes those records before returning:

```powershell
$env:SHOPE_SMOKE='1'
$env:MYSQL_DSN='root:root123456@tcp(127.0.0.1:3307)/go_shope?charset=utf8mb4&parseTime=True&loc=Local'
$env:JWT_SECRET='change-this-local-development-jwt-secret-2026'
go test -run TestBaselineHTTPFlow -v .
```

The flow verifies page serving, registration/login, admin product and activity creation, ordinary ordering and cancellation, seckill ordering and payment, retry idempotency, stock results, and the admin order list.

## Baseline boundary for later comparison

Keep benchmark results for this version separate from optimized branches. Record at least the commit hash, test data size, concurrency, request duration, success/error counts, throughput, latency percentiles, database state, and machine/container resources. Do not compare later Redis or queue results against an unrecorded or differently configured baseline.

This ordinary version intentionally performs synchronous request processing and uses MySQL as its only stateful service. Future optimizations should be added on a separate branch after this baseline commit is preserved.

## Important baseline guarantees

- Product prices are stored as integer cents, never `float`.
- Passwords are stored only as bcrypt hashes.
- `orders.request_id` prevents the same request from creating more than one order.
- `(activity_id, user_id)` prevents one user from buying the same activity twice.
- Creating an order and deducting both MySQL stock counters occurs in one transaction.
- Cancelling a pending order and restoring both MySQL stock counters occurs in one transaction.
# Go Shope：高并发秒杀对比版

这是普通同步 MySQL 版本 `go_shope` 的独立优化版本，用于同一套场景的单机压测对比。普通基线保持在 `baseline-v1.0.0`（commit `1a85a43`）；本仓库只承载优化实现，不修改基线项目。

## 已实现的链路

请求进入 API 后，由 Redis Lua 在一个原子脚本内完成活动窗口、一人一单、request_id 幂等、预扣库存、写入用户集合和 Redis Stream。API 返回 `202 Accepted` 只表示受理成功。两个 Worker 通过 Consumer Group 消费 Stream，调用原有 MySQL 事务创建订单；MySQL 唯一索引和事务是最终持久化兜底。失败时 Worker 标记 `FAILED` 并用 Lua 回补 Redis 库存和用户集合，重复投递则按 request_id 查找已有订单。

API 与 Worker 是同一台机器上的两个进程。API 暴露 `/livez`、`/readyz`、`/metrics`，并配置请求超时和 SIGTERM 优雅停机；Prometheus 抓取本机 API 指标。

## 启动

```powershell
docker compose -f compose.optimized.yaml up --build -d
```

默认入口为 `http://localhost:8081`。Compose 只启动一个 API、一个 Worker、Redis、MySQL 和 Prometheus，适合在一台开发机上压测。容器内 API/Worker 使用 `cmd/api` 和 `cmd/worker`，本地开发也可分别运行 `go run ./cmd/api`、`go run ./cmd/worker`。

## 当前边界

- 本阶段只准备代码、配置和压测入口，尚未执行正式压测、故障演练或填写任何性能数据。
- Compose 使用单 Redis、单 MySQL、单 API 和单 Worker，目标是单机可压测，不讨论分布式高可用。
- 活动创建后需调用管理端 publish 接口把活动预热到 Redis，`benchmark/scripts/prepare-optimized.ps1` 已自动完成。
- 最终订单结果必须通过 `GET /api/seckill/requests/:request_id` 查询并结合 MySQL 核对；不能用受理 202 代替成功率。

统一压测入口见 [`benchmark/README.md`](benchmark/README.md)。结果目录只接收真实 k6 输出，严禁把模板阈值或示例值写成简历指标。
