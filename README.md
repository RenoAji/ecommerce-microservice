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

```
docker compose up --build
```
