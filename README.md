# Go Shop: basic CRUD version

This branch implements a MySQL-only monolith. It deliberately contains no Redis, RabbitMQ, caching, rate limiting, sharding, or distributed components.

## Included business flow

1. Register and log in with bcrypt password hashes and a JWT.
2. Admin CRUD for products and seckill activities.
3. Public product and activity queries.
4. Authenticated synchronous seckill ordering.
5. Pay or cancel an order. Cancelling a pending order restores MySQL inventory in the same transaction.

## Configure and run

1. Create an empty MySQL database named `go_shope`.
2. In PowerShell, set connection values for the current terminal only:

```powershell
$env:MYSQL_DSN='root:your_password@tcp(127.0.0.1:3306)/go_shope?charset=utf8mb4&parseTime=True&loc=Local'
$env:JWT_SECRET='replace-with-a-long-random-secret'
go run .
```

The application automatically creates the four basic tables: `users`, `products`, `seckill_activities`, and `orders`.

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
