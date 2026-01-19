# Ecommerce microservice in go

This project is for learning purpose inspired from https://roadmap.sh/projects/scalable-ecommerce-platform .

## 🚀 Architecture

- **User Service**: Handles Authentication (JWT), Profile, and Registration.
- **Product Service**: Manages Catalog, Categories, and Stock adjustments.
- **API Gateway (Nginx)**: Routes traffic to services. Accessible via port `8080` (specified in .env).

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

GATEWAY_PORT=8080
USER_DB_PORT=5432
PRODUCT_DB_PORT=5433
```

### 3. Run with docker

```bash
docker compose up --build
```

## 📡 API Endpoints

All endpoints are accessible through the API Gateway at `http://localhost:8080`

### User Service (`/api/v1`)

#### Public Routes

| Method | Endpoint    | Description             |
| ------ | ----------- | ----------------------- |
| `POST` | `/register` | Register a new user     |
| `POST` | `/login`    | Login and get JWT token |

#### Protected Routes (Require JWT Token)

| Method | Endpoint                | Description              |
| ------ | ----------------------- | ------------------------ |
| `GET`  | `/auth/profile`         | Get current user profile |
| `POST` | `/auth/change-password` | Change user password     |
| `POST` | `/auth/logout`          | Logout user              |
| `POST` | `/auth/refresh`         | Refresh JWT token        |

### Product Service (`/api/v1`)

#### Public Routes

| Method | Endpoint        | Description                         | Query Parameters                                                                       |
| ------ | --------------- | ----------------------------------- | -------------------------------------------------------------------------------------- |
| `GET`  | `/products`     | Get paginated products with filters | `page`, `limit`, `search`, `category_id`, `min_price`, `max_price`, `sort_by`, `order` |
| `GET`  | `/products/:id` | Get product by ID                   | -                                                                                      |

#### Admin Routes (Require Admin JWT Token)

| Method   | Endpoint              | Description           |
| -------- | --------------------- | --------------------- |
| `POST`   | `/products`           | Create a new product  |
| `PUT`    | `/products/:id`       | Update product        |
| `PATCH`  | `/products/:id/stock` | Add/remove stock      |
| `DELETE` | `/products/:id`       | Delete product        |
| `POST`   | `/categories`         | Create a new category |

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

Interactive API documentation is available via Swagger UI:

- **User Service**: http://localhost:8080/user-docs/index.html
- **Product Service**: http://localhost:8080/product-docs/index.html

The Swagger UI provides:

- Interactive API testing
- Request/response schemas
- Authentication support (use "Authorize" button to add JWT token)
- Complete endpoint documentation with examples
