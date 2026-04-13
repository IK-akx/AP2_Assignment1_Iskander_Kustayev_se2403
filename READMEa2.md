# AP2 Assignment 2 — gRPC Migration & Contract-First Development

## 📌 Overview

This project demonstrates migration from REST to gRPC communication between microservices using a Contract-First approach.

* **Order Service**

    * REST API for external clients
    * gRPC client for Payment Service
    * gRPC server for real-time order tracking (streaming)

* **Payment Service**

    * gRPC server for processing payments
    * REST API kept for testing/debugging

---

## Architecture

* External Client → REST → Order Service
* Order Service → gRPC → Payment Service
* Client → gRPC Stream → Order Service (order updates)

---

## Repositories

* Proto (contracts):
  https://github.com/IK-akx/ap2-protos

* Generated code:
  https://github.com/IK-akx/ap2-generated

---

## Technologies

* Go (Golang)
* gRPC & Protocol Buffers
* Gin (REST)
* GORM + PostgreSQL

---

## How to Run

### 1. Start Payment Service

```bash
cd payment
go run cmd/main.go
```

### 2. Start Order Service

```bash
cd order
go run cmd/main.go
```

---

## REST Endpoints (Order Service)

* `POST /orders`
* `GET /orders/:id`
* `PATCH /orders/:id/cancel`

---

## 🔌 gRPC Endpoints

### Payment Service

* `ProcessPayment`

### Order Service (Streaming)

* `SubscribeToOrderUpdates`

---

## Streaming Demo

Run:

```bash
grpcurl -plaintext -d '{"order_id":"YOUR_ID"}' localhost:50052 order.OrderTrackingService/SubscribeToOrderUpdates
```

Then create/update an order → updates will stream in real-time.

---

## Features

* Contract-First (separate proto + generated repos)
* gRPC client/server implementation
* Server-side streaming (real-time updates from DB)
* Clean Architecture preserved
* Environment-based configuration
* gRPC interceptor (logging)

---

## Notes

* Business logic remains in UseCase layer
* No hardcoded service addresses
* Streaming tied to real database updates (no fake loops)

---
