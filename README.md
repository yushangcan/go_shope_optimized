# Go Shop: basic CRUD version

This branch implements a MySQL-only monolith. It deliberately contains no Redis, RabbitMQ, caching, rate limiting, sharding, or distributed components.

## Included business flow

1. Register and log in with bcrypt password hashes and a JWT.
2. Admin CRUD for products and seckill activities.
3. Public product and activity queries.
4. Authenticated synchronous seckill ordering.
5. Pay or cancel an order. Cancelling a pending order restores MySQL inventory in the same transaction.

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
| GET | `/api/seckill/activities` | List activities |
| POST | `/api/seckill/activities/:id/orders` | Create one synchronous seckill order |
| GET | `/api/orders` | List the current user's orders |
| POST | `/api/orders/:id/pay` | Simulate payment |
| POST | `/api/orders/:id/cancel` | Cancel a pending order and restore inventory |

Product and activity write endpoints are under `/api/admin`. A newly registered user has the `USER` role. To test admin endpoints, change the required user's `role` to `ADMIN` directly in MySQL during this basic development stage.

## Important baseline guarantees

- Product prices are stored as integer cents, never `float`.
- Passwords are stored only as bcrypt hashes.
- `orders.request_id` prevents the same request from creating more than one order.
- `(activity_id, user_id)` prevents one user from buying the same activity twice.
- Creating an order and deducting both MySQL stock counters occurs in one transaction.
- Cancelling a pending order and restoring both MySQL stock counters occurs in one transaction.
