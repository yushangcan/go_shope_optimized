# 优化版秒杀压测入口

本目录对应 `codex/optimized-seckill`，用于和普通版 `baseline-v1.0.0` 做同条件单机对比。优化版的秒杀接口只负责快速受理：请求先由 Redis Lua 原子校验并扣减预热库存，再发布到 RabbitMQ 持久化队列；单个 Worker 异步执行现有 MySQL 事务，最终状态通过查询接口确认。

## 启动

```powershell
docker compose -f compose.optimized.yaml up --build -d
$env:BASE_URL = 'http://localhost:8081'
$env:ADMIN_USERNAME = 'your-admin'
$env:ADMIN_PASSWORD = 'your-password'
.\benchmark\scripts\prepare-optimized.ps1
```

RabbitMQ 管理页面默认在 `http://localhost:15672`，账号为 `seckill`，密码为 `seckillpass`（仅用于本地实验）。API 端口为 `8081`，Prometheus 端口为 `9090`。

脚本输出的 `ACTIVITY_ID` 可用于后续 k6 场景。普通版使用 8080，优化版使用 8081；两边应使用独立数据库和同样的商品/活动库存参数。

## 场景

- `optimized-admission.js`：只测 `POST /api/seckill/activities/:id/requests` 的受理延迟；202 仅代表消息已发布到 RabbitMQ，不代表订单已落库。
- `optimized-read.js`：测请求状态查询和商品/活动只读链路。
- `optimized-duplicate.js`：重复 `request_id` 的幂等受理。
- `optimized-sold-out.js`：售罄后的快速失败。

压测结束后应等待 Worker 消化 RabbitMQ 队列，再统计 `GET /api/seckill/requests/:request_id` 的最终 `SUCCEEDED`/`FAILED`，不要把 202 当作业务成功。`results/` 只保存真实 k6 JSON、commit SHA、配置和监控截图；仓库不包含任何虚构 QPS、p95、p99 或可用性数据。正式压测和故障演练尚未在本阶段执行。

当前 Compose 只使用单 Redis、单 MySQL、单 API、单 Worker，目标是单机压测；不把它描述为分布式或生产级高可用项目。
