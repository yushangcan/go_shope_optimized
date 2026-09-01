# 云服务器单机部署与压测方案

本文档针对当前项目的真实结构：一个 Go API、一个 Go Worker、单 Redis、单 RabbitMQ、单 MySQL 和 Prometheus。目标是实习项目的单机压测，不引入 Nginx、服务发现、分片或多机高可用。

## 1. 需要几台云服务器

最低配置是一台云服务器，同时运行 Docker Compose 中的 API、Worker、MySQL、Redis、RabbitMQ 和 Prometheus，并在该机运行 k6。这样适合启动验证和小规模压测，但 k6 会与被测服务争抢资源。

推荐配置是两台同地域云服务器：

| 机器 | 作用 | 运行内容 |
| --- | --- | --- |
| 被测机 | 运行项目 | API、Worker、MySQL、Redis、RabbitMQ、Prometheus |
| 压测机 | 只发请求 | k6、结果保存脚本 |

普通基线版和 RabbitMQ 优化版应轮流部署在同一台被测机上，使用相同规格和系统环境。当前不需要第三台机器。

## 2. 推荐规格

被测机建议 4 vCPU、8 GB RAM、40 GB 以上 SSD、Ubuntu 22.04 LTS、5 Mbps 以上带宽。最低可以使用 2 vCPU、4 GB RAM，但多个容器同时运行时余量较小。

压测机建议 2 vCPU、4 GB RAM、20 GB SSD、Ubuntu 22.04 LTS。VU 数提高到数千时，应优先提升压测机规格或增加压测机，不要把压测机瓶颈误判成 API 瓶颈。

## 3. 安全组与端口

| 端口 | 用途 | 建议 |
| --- | --- | --- |
| 22 | SSH | 只允许自己的固定 IP |
| 8081 | Go API | 只允许压测机 IP 和办公 IP |
| 9090 | Prometheus | 不开放公网 |
| 15672 | RabbitMQ 管理台 | 不开放公网，必要时用 SSH 隧道 |
| 3308 | MySQL 映射 | 不开放公网 |
| 6379 | Redis 映射 | 不开放公网 |
| 5672 | RabbitMQ AMQP 映射 | 不开放公网 |

API 直接监听 8081，不需要 Nginx。RabbitMQ 管理台建议用 SSH 隧道：

```bash
ssh -L 15672:127.0.0.1:15672 ubuntu@<被测机公网IP>
```

## 4. 初始化被测机

```bash
sudo apt-get update
sudo apt-get install -y git ca-certificates curl
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker "$USER"
newgrp docker
docker compose version
git clone https://github.com/yushangcan/go_shope_optimized.git
cd go_shope_optimized
git checkout optimized-code-v1
```

部署前记录版本和机器信息：

```bash
git rev-parse HEAD
uname -a
docker version
docker compose version
nproc
free -h
df -h
```

## 5. 设置云环境变量

不要把云服务器真实密码提交到 Git。在项目目录创建只保存在服务器上的 `.env`：

```dotenv
MYSQL_ROOT_PASSWORD=替换为随机长密码
MQ_USER=seckill
MQ_PASSWORD=替换为随机长密码
JWT_SECRET=替换为随机长密钥
```

确认 `.env` 已被 `.gitignore` 忽略。Compose 会使用这些变量生成 MySQL DSN 和 RabbitMQ URL。

## 6. 启动优化版

```bash
docker compose --env-file .env -f compose.optimized.yaml up --build -d
docker compose --env-file .env -f compose.optimized.yaml ps
curl http://127.0.0.1:8081/livez
curl http://127.0.0.1:8081/readyz
```

等待 mysql、redis 和 rabbitmq healthy 后，再从压测机检查：

```bash
curl http://<被测机私网IP或公网IP>:8081/livez
```

如果云平台提供私有网络，优先使用压测机到被测机的私网 IP。

## 7. 创建商品和活动

准备脚本是 PowerShell 版本，可以在本地运行，只要 `BASE_URL` 指向云服务器：

```powershell
$env:BASE_URL = 'http://<被测机IP>:8081'
$env:ADMIN_USERNAME = '已有管理员用户名'
$env:ADMIN_PASSWORD = '管理员密码'
.\benchmark\scripts\prepare-optimized.ps1
```

脚本会创建商品和活动，并调用 publish 接口把活动库存预热到 Redis，最后输出 `PRODUCT_ID` 和 `ACTIVITY_ID`。

## 8. 从压测机运行 k6

```bash
export BASE_URL=http://<被测机私网IP或公网IP>:8081
export ACTIVITY_ID=活动ID
export K6_VUS=100
export K6_DURATION=60s
k6 run --out json=optimized-admission.json benchmark/k6/optimized-admission.js
```

建议分别运行受理、只读、重复 request_id 和售罄场景。被测机另开窗口观察：

```bash
docker stats
docker compose -f compose.optimized.yaml logs -f api worker rabbitmq
```

## 9. 普通版与优化版对比

1. 在专用压测环境清理上一版数据；
2. 部署普通版并记录 commit SHA；
3. 用相同库存、用户规模、VU、持续时间和机器规格压测普通版；
4. 保存 k6 JSON、资源数据和数据库核对结果；
5. 清理数据后部署 `optimized-code-v1`；
6. 用完全相同条件压测优化版；
7. 等待 Worker 消费完 RabbitMQ 队列；
8. 查询 request status，并在 MySQL 核对订单和库存；
9. 最后比较吞吐量、受理延迟、异步完成延迟和资源使用。

`docker compose down -v` 会删除 MySQL、Redis 和 RabbitMQ 数据卷，只能在专用压测环境执行。

## 10. 必须核对的正确性

- RabbitMQ 最终没有未处理消息；
- `request_id` 没有重复订单；
- `(user_id, activity_id)` 没有重复秒杀订单；
- 成功订单数不超过活动初始库存；
- MySQL 库存没有变成负数；
- MQ 发布失败或 Worker 失败的请求最终为 `FAILED` 并完成 Redis 补偿；
- `202` 请求最终能够归类为 `SUCCEEDED` 或 `FAILED`。

## 11. 当前边界

- 这是单机实验部署，不是生产级高可用部署；
- MySQL、Redis、RabbitMQ、API 和 Worker 都是单实例；
- 当前没有独立死信集群和 Outbox 事务消息；
- Redis 预扣与 RabbitMQ 发布之间存在进程突然宕机的极小窗口；
- 所有 QPS、p95、p99 和成功率必须来自真实云服务器压测，不能使用示例值。

最合适的实习项目配置是：1 台 4C8G 被测机 + 1 台 2C4G 压测机。预算有限时可先用一台机器完成部署验证，再增加独立压测机。
