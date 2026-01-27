# Ecommerce microservice in go

This project is for learning purpose inspired from https://roadmap.sh/projects/scalable-ecommerce-platform .

## 🚀 Architecture

- **User Service**: Handles Authentication (JWT), Profile, and Registration.
- **Product Service**: Manages Catalog, Categories, Stock adjustments, and gRPC API for internal service communication.
- **Cart Service**: Manages shopping cart operations with Redis storage and communicates with Product Service via gRPC.
- **API Gateway (Nginx)**: Routes HTTP traffic to services. Accessible via port `8080` (specified in .env).

### Service Communication

- **HTTP/REST**: External client ↔ API Gateway ↔ Services
- **gRPC**: Internal service-to-service (Cart Service → Product Service)
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

CART_REDIS_PASSWORD=

INTERNAL_SECRET=internal_secret_key_change_me

GATEWAY_PORT=8080
USER_DB_PORT=5432
PRODUCT_DB_PORT=5433
CART_REDIS_PORT=6379
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
  - Cart Redis: `localhost:6379`
- 📦 **Larger images** (includes dev tools)

#### Production Mode

For optimized container:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build
```

**Production Features:**

- 📦 **Smaller container images** (multi-stage builds)
- 🔒 **Database ports not exposed** (internal network only)
- 🚀 **No auto-restart** (code changes require rebuild)

## 🔧 Development

### Generate Swagger Documentation

```bash
make swagger
```

Or generate for individual services:

```bash
make swagger-user
make swagger-product
make swagger-cart
```

### Generate Protocol Buffers

```bash
# Generate for all services
protoc --go_out=product-service --go-grpc_out=product-service proto/product.proto
protoc --go_out=cart-service --go-grpc_out=cart-service proto/product.proto
protoc --go_out=order-service --go-grpc_out=order-service proto/product.proto

protoc --go_out=cart-service --go-grpc_out=cart-service proto/cart.proto
protoc --go_out=order-service --go-grpc_out=order-service proto/cart.proto
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

The Swagger UI provides:

- Interactive API testing
- Request/response schemas
- Authentication support (use "Authorize" button to add JWT token)
- Complete endpoint documentation with examples
