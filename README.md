# Ecommerce microservice in go

This project is for learning purpose inspired from https://roadmap.sh/projects/scalable-ecommerce-platform .

## 🚀 Architecture

- **User Service**: Handles Authentication (JWT), Profile, and Registration.
- **Product Service**: Manages Catalog, Categories, Stock adjustments, and gRPC API for internal service communication.
- **Cart Service**: Manages shopping cart operations with Redis storage and communicates with Product Service via gRPC.
- **Order Service**: Handles order creation, orchestrates cart, product, and payment services via gRPC.
- **Payment Service**: Integrates with Midtrans payment gateway, handles webhooks, and manages payment lifecycle.
- **Delivery Service**: Manages order delivery.
- **API Gateway (Nginx)**: Routes HTTP traffic to services. Accessible via port `8080` (specified in .env).

### Service Communication

- **HTTP/REST**: External client ↔ API Gateway ↔ Services
- **gRPC**: Orchestration between services:
  - Order Service:
    - Get cart details from Cart Service
    - Get product details from Product Service
    - Update product stock in Product Service
  - Cart Service:
    - Get product details from Product Service
      ![alt text](<readme_img/microservice_ecomm_grpc%20(1).png>)

- **Redis Streams**: Messaging between services:
  - OrderCreated event from Order Service consumed by:
    - Payment Service to initiate payment
  - StockReserved event from Product Service consumed by:
    - Order Service to confirm order
  - StockInsufficient event from Product Service consumed by:
    - Order Service to mark order as failed
  - PaymentSuccess event from Payment Service consumed by:
    - Order Service to update order status
    - Cart Service to clear purchased items
    - Delivery Service to start delivery process
  - PaymentFailed event from Payment Service consumed by:
    - Order Service to mark order as failed
  - DeliverySuccess event from Delivery Service consumed by:
    - Order Service to mark order as delivered
  - DeliveryFailed event from Delivery Service consumed by:
    - Order Service to mark delivery as failed
    - Payment Service to initiate refund (not implemented yet)
      ![alt text](<readme_img/microservice_ecomm_messaging%20(1).png>)
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
DELIVERY_DB_NAME=delivery_db

REDIS_PASSWORD=

MIDTRANS_SERVER_KEY=your_midtrans_server_key
MIDTRANS_CLIENT_KEY=your_midtrans_client_key

INTERNAL_SECRET=internal_secret_key_change_me

GATEWAY_PORT=8080
USER_DB_PORT=5432
PRODUCT_DB_PORT=5433
ORDER_DB_PORT=5434
PAYMENT_DB_PORT=5435
DELIVERY_DB_PORT=5436
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
  - Delivery DB: `localhost:5436`
  - Cart Redis (DB 0): `localhost:6379`
  - Broker Redis (DB 1): `localhost:6379`
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
make proto-delivery # Generates delivery.proto for delivery and order services
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

## 💳 Payment Service Setup (Midtrans)

The Payment Service integrates with **Midtrans** for payment processing. To enable webhooks in your local development environment:

### 1. Create Midtrans Account

1. Sign up at [Midtrans Dashboard](https://dashboard.midtrans.com)
2. Create a **Sandbox** account for testing
3. Get your credentials:
   - **Server Key** (from Settings → Access Keys)
   - **Client Key** (from Settings → Access Keys)

### 2. Configure Environment Variables

Add to your `.env` file:

```env
MIDTRANS_SERVER_KEY=SB-Mid-server-xxxxxxxxxxxx
MIDTRANS_CLIENT_KEY=SB-Mid-client-xxxxxxxxxxxx
MIDTRANS_ENVIRONMENT=sandbox
```

### 3. Setup Webhook for Local Development

Since Midtrans needs to send webhooks to your local machine, use **ngrok** to expose your local server:

```bash
# Install ngrok (if not installed)
# macOS: brew install ngrok
# Linux: download from https://ngrok.com/download

# Start ngrok tunnel (forwards localhost:8080 to public URL)
ngrok http 8080
```

This gives you a public URL like: `https://xxxx-xx-xxx-xxx.ngrok-free.app`

### 4. Configure Midtrans Webhook URL

1. Go to [Midtrans Dashboard](https://dashboard.midtrans.com)
2. Navigate to **Settings → Notifications**
3. Set **HTTP Notification URL** to:
   ```
   https://xxxx-xx-xxx-xxx.ngrok-free.app/api/v1/payment/webhook
   ```
4. Enable notifications for: `Payment Success`, `Payment Failure`, `Payment Pending`

### 5. Test Payment Flow

1. Create an order via the Order Service API
2. Get the payment URL from the response
3. Open the payment URL in a browser
4. Use [Midtrans Sandbox Testing Credentials](https://docs.midtrans.com/reference/sandbox-test-payment-page):
   - **Card Number**: `4011111111111111`
   - **Expiry**: Any future date (e.g., `12/25`)
   - **CVV**: Any 3 digits (e.g., `123`)
5. Complete the payment
6. Midtrans will send a webhook to your ngrok URL
7. Check that the order status updated to `PAID` and delivery was created

### 6. Monitor Webhook Delivery

- Check Midtrans Dashboard → **Logs** → **HTTP Notifications** to see webhook history
- View application logs to confirm webhook was processed:
  ```bash
  docker compose logs payment-service | grep -i webhook
  ```

## 📚 API Documentation

Interactive API documentation is available via Swagger UI (after running the services):

- **User Service**: http://localhost:8080/user-docs/index.html
- **Product Service**: http://localhost:8080/product-docs/index.html
- **Cart Service**: http://localhost:8080/cart-docs/index.html
- **Order Service**: http://localhost:8080/order-docs/index.html
- **Payment Service**: http://localhost:8080/payment-docs/index.html
- **Delivery Service**: http://localhost:8080/delivery-docs/index.html

The Swagger UI provides:

- Interactive API testing
- Request/response schemas
- Authentication support (use "Authorize" button to add JWT token)
- Complete endpoint documentation with examples

## 🛠️ Future Improvements

- Implementing outbox pattern for reliable messaging
- Adding a notification service for email/SMS updates
- Adding OrderPaid event so delivery service and cart service can consume product Ids and quantity from OrderPaid event instead of consuming payment success event which only contains order Id.
