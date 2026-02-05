# Ecommerce microservice in go

This project is for learning purpose inspired from https://roadmap.sh/projects/scalable-ecommerce-platform .

## 🚀 Architecture

- **User Service**: Handles Authentication (JWT), Profile, and Registration.
- **Product Service**: Manages Catalog, Categories, Stock adjustments, and gRPC API for internal service communication.
- **Cart Service**: Manages shopping cart operations with Redis storage and communicates with Product Service via gRPC.
- **Order Service**: Handles order creation, orchestrates cart, product, and payment services via gRPC.
- **Payment Service**: Integrates with Midtrans payment gateway, handles webhooks, and manages payment lifecycle.
- **API Gateway (Nginx)**: Routes HTTP traffic to services. Accessible via port `8080` (specified in .env).

### Service Communication

- **HTTP/REST**: External client ↔ API Gateway ↔ Services
- **gRPC**: Internal service-to-service communication
  - Order Service → Cart Service
  - Order Service → Product Service
  - Order Service → Payment Service
- **Redis Streams**: Event-driven architecture for:
  - Stock reservation events
  - Payment success/failure events
  - Stock insufficient events
- **Redis**: Cart data storage with 7-day TTL

## 🏁 Getting Started

### 1. Prerequisites

- Docker & Docker Compose installed.

### 2. Setup Environment

Create a `.env` file in the root (.env.example):

```env
DB_USER=user
DB_PASSWORD=secretpassword

USER_DB_NAME=user_db
JWT_SECRET=super_secret_key_change_me

PRODUCT_DB_NAME=product_db
ORDER_DB_NAME=order_db
PAYMENT_DB_NAME=payment_db

REDIS_PASSWORD=

MIDTRANS_SERVER_KEY=your_midtrans_server_key
MIDTRANS_CLIENT_KEY=your_midtrans_client_key

INTERNAL_SECRET=internal_secret_key_change_me

GATEWAY_PORT=8080
USER_DB_PORT=5432
PRODUCT_DB_PORT=5433
ORDER_DB_PORT=5434
PAYMENT_DB_PORT=5435
REDIS_PORT=6379
```

### 3. Run with Docker

#### Development Mode

For active development with hot-reload and database port access:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

**Development Features:**

- 🔄 **Auto-restart on code changes** (volumes mounted)
- 🔌 **Database ports mapped** to host (can be configured in `.env`):
  - User DB: `localhost:5432`
  - Product DB: `localhost:5433`
  - Order DB: `localhost:5434`
  - Payment DB: `localhost:5435`
  - Redis: `localhost:6379` (DB 0 for Cart, DB 1 for Broker)
- 📦 **Larger images** (includes dev tools)

#### Production Mode

For optimized container:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build
```

**Production Features:**

- 📦 **Smaller container images**
- 🔒 **Database ports not exposed**
- 🚀 **No auto-restart** (code changes require rebuild)

## 🔧 Development

### Generate Swagger Documentation

```bash
# Generate for all services
make swagger
```

Or generate for individual services:

```bash
make swagger-user
make swagger-product
make swagger-cart
make swagger-order
make swagger-payment
```

### Generate Protocol Buffers

```bash
# Generate for all services
make proto
```

Or generate for individual proto files:

```bash
make proto-product  # Generates product.proto for product, cart, and order services
make proto-cart     # Generates cart.proto for cart and order services
make proto-payment  # Generates payment.proto for payment and order services
```

## 🔐 Authentication

Protected routes require a JWT token in the Authorization header:

```
Authorization: Bearer <your_jwt_token>
```

### Default Admin Account

After first run, a default admin account is created:

- **Email:** `admin@admin.com`
- **Password:** `admin123`
- **Role:** `admin`

⚠️ **Change the admin password immediately in production!**

## 📚 API Documentation

Interactive API documentation is available via Swagger UI (after running the services):

- **User Service**: http://localhost:8080/user-docs/index.html
- **Product Service**: http://localhost:8080/product-docs/index.html
- **Cart Service**: http://localhost:8080/cart-docs/index.html
- **Order Service**: http://localhost:8080/order-docs/index.html
- **Payment Service**: http://localhost:8080/payment-docs/index.html

The Swagger UI provides:

- Interactive API testing
- Request/response schemas
- Authentication support (use "Authorize" button to add JWT token)
- Complete endpoint documentation with examples
