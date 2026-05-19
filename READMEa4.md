# Assignment 4 - Performance Optimization & External Integrations

## Project Overview

This project implements a production-ready microservices system with:
- **Redis Caching** for Order Service (Cache-Aside Pattern)
- **Rate Limiting** for API protection
- **Background Jobs** with Exponential Backoff retries
- **Idempotency** using Redis
- **Adapter Pattern** for email providers

## Architecture Diagram

![Architecture Diagram](./architecture.png)

*See Mermaid diagram above or view on [GitHub]*

## Services

| Service | Ports | Description |
|---------|-------|-------------|
| Order Service | HTTP:8080, gRPC:50052 | Order management with Redis cache |
| Payment Service | HTTP:8081, gRPC:50051 | Payment processing, NATS publisher |
| Notification Service | - | Background worker for email notifications |
| PostgreSQL | 5433, 5434 | Persistent storage |
| Redis | 6379 | Cache, Idempotency, Rate Limiting |
| NATS | 4222 | Message queue |

---

## 1. Cache Invalidation Strategy

### Cache-Aside Pattern Implementation

The Order Service implements the **Cache-Aside Pattern** (also known as Lazy Loading):


GET /orders/:id
1. Check Redis cache (key: order:{id})
2. If MISS → Query PostgreSQL
3. Store result in Redis with TTL
4. Return order data

UPDATE /orders/:id/cancel
1. Update database
2. DELETE from Redis cache
3. Cache is invalidated


### Invalidation Rules

| Operation | Cache Action | Why |
|-----------|--------------|-----|
| `GET /orders/:id` (miss) | SET with TTL=5min | Populate cache |
| `PATCH /orders/:id/cancel` | DELETE | Status changed → stale data risk |
| `POST /orders` (payment success) | DELETE | Order status updated to Paid/Failed |
| TTL Expiry (5 min) | Auto DELETE | Automatic cleanup |

### Atomic Invalidation

Cache invalidation happens **immediately after** database update:
```go
// 1. Update database
uc.OrderRepo.UpdateStatus(orderID, domain.OrderStatusCancelled)

// 2. Invalidate cache (synchronous to ensure consistency)
uc.OrderCache.Delete(orderID)
```

### Why Not Update Cache?
We DELETE instead of UPDATE because:
- Order status can change in complex ways
- DELETE is atomic and simple
- Next read will refresh with fresh data
- Prevents partial/incorrect updates
- 
### Retry & Exponential Backoff Strategy
   Retry Flow Diagram

Job Created
↓

Worker Receives Job
↓

Send Email ── SUCCESS ──→ Mark Processed → Done
↓

FAILURE
↓

Retry Count < Max (5)
↓

Calculate Backoff: 2^(retry) seconds
↓

Wait (2s, 4s, 8s, 16s, 32s)
↓

Re-queue Job
↓
(loop)

If Retry Count >= Max (5) → Dead Letter Queue → Logged


### Exponential Backoff Formula

backoff = baseDelay * (2 ^ retryCount)

## Rate Limiter (Bonus)
   Fixed Window Algorithm
```env
   RATE_LIMIT_REQUESTS=10           # 10 requests
   RATE_LIMIT_WINDOW_SECONDS=60     # per 60 seconds

```
Rate Limit Headers
Every response includes:
```text
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 7
X-RateLimit-Reset: Wed, 19 May 2026 14:30:00 GMT
Retry-After: 45 (on 429)
Response Codes
Status	Meaning
200 OK	Request allowed
429 Too Many Requests	Rate limit exceeded```
