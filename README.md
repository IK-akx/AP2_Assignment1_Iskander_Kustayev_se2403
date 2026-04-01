# Order & Payment Microservices System

Name: Iskander Kustayev

Group: SE-2403

Assignment: AP2 Assignment 1

## Project Overview

This project implements a two-service microservices platform for order management and payment processing. The system demonstrates Clean Architecture principles, service decomposition, bounded contexts, separate data ownership, and resilient synchronous communication using REST.

**Technologies:**
- Go 1.21+
- Gin Web Framework
- GORM ORM
- PostgreSQL
- REST API

## Architecture

### System Architecture Diagram

--------

### Bounded Contexts

**Order Context:**
- Manages order lifecycle (creation, status updates, cancellation)
- Owns order data and business rules
- Knows about customers and items
- Does NOT know payment details, only payment result

**Payment Context:**
- Manages payment processing
- Owns transaction data
- Enforces payment limits
- Does NOT know about orders beyond order_id

### Clean Architecture Layers

Each service follows Clean Architecture with dependency inversion:

1. **Domain Layer**: Business entities and interface contracts
    - Independent of frameworks and external concerns
    - Defines ports (interfaces) for repositories and external services

2. **Use Case Layer**: Business logic implementation
    - Orchestrates domain entities
    - Implements business rules
    - Depends only on domain interfaces

3. **Repository Layer**: Data persistence
    - Implements domain repository interfaces
    - Handles database operations

4. **Delivery Layer**: HTTP handling
    - Parses requests and returns responses
    - Calls use cases
    - No business logic

5. **Client Layer**: External service communication
    - HTTP client with timeout
    - Implements domain gateway interfaces



## Database Schema

### Order Database (order_db)

**Table: orders**
```sql
CREATE TABLE orders (
    id VARCHAR(50) PRIMARY KEY,
    customer_id VARCHAR(100) NOT NULL,
    item_name VARCHAR(200) NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    status VARCHAR(20) NOT NULL DEFAULT 'Pending',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (status IN ('Pending', 'Paid', 'Failed', 'Cancelled'))
);
```

### Payment Database (payment_db)

**Table: payments**

```sql
CREATE TABLE payments (
    id VARCHAR(50) PRIMARY KEY,
    order_id VARCHAR(50) NOT NULL UNIQUE,
    transaction_id VARCHAR(100) NOT NULL UNIQUE,
    amount BIGINT NOT NULL CHECK (amount > 0),
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (status IN ('Authorized', 'Declined'))
);
```


## Environment Configuration

### order-service/.env:

```env
ORDER_SERVICE_PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=order_db
PAYMENT_SERVICE_URL=http://localhost:8081
HTTP_TIMEOUT=2
```

### payment-service/.env:

```env
PAYMENT_SERVICE_PORT=8081
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=payment_db
```


## Running the Services
### Start Order Service:

```bash
cd order-service
go mod tidy
go run cmd/main.go
```

### Start Payment Service (in new terminal):

```bash
cd payment-service
go mod tidy
go run cmd/main.go
```


## Business Rules Implementation
1. Financial Accuracy
- All monetary values stored as int64 in cents

- No floating-point operations for money

- Example: $10.00 = 1000 cents

2. Order Invariants
- Amount must be > 0 (validated in use case)

- Only "Pending" orders can be cancelled

- "Paid" orders cannot be cancelled

3. Payment Limits
- Maximum payment amount: 100,000 cents ($1,000.00)

- Amounts > limit return "Declined" status

4. Service Interaction
- HTTP client timeout: 2 seconds

- 503 Service Unavailable when payment service is down

- Order remains "Pending" for retry attempts

## Failure Handling Strategy
### Scenario: Payment Service Unavailable
Behavior:

1. Order is created with status "Pending"

2. Payment service call fails with timeout/connection error

3. Order service returns HTTP 503 Service Unavailable

4. Order remains in database for retry

Rationale:

- Following "retry later" pattern

- Prevents data loss

- Allows manual or automatic retry

- Alternative (marking as "Failed") would require manual intervention

## Architecture Decisions
1. Separate Databases per Service
   Decision: Each service has its own PostgreSQL database
   Reason: Enforces loose coupling, independent scaling, and clear ownership

2. Clean Architecture with Dependency Inversion
   Decision: Use case layer depends on interfaces, not implementations
   Reason: Testability, maintainability, and framework independence

3. REST with Timeouts
   Decision: HTTP client with 2-second timeout
   Reason: Prevents cascading failures, maintains system responsiveness

4. Amount as int64
   Decision: Store monetary values as int64 cents
   Reason: Avoids floating-point precision issues in financial calculations

5. Manual Dependency Injection
   Decision: Wire dependencies in main.go
   Reason: Clear composition root, no magic, easy to understand



## Common Errors
"payment service unavailable"

- Payment service not running

- Wrong URL in .env

- Firewall blocking port 8081

"failed to connect to database"

- PostgreSQL not running

- Wrong credentials in .env

- Database not created