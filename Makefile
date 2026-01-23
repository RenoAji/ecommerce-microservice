# Swagger documentation generation
swagger-user:
	cd user-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger-product:
	cd product-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger-cart:
	cd cart-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger: swagger-user swagger-product swagger-cart

.PHONY: swagger swagger-user swagger-product swagger-cart
# Run all services
up:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

.PHONY: up

# Run in production mode
prod-up:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build

.PHONY: prod-up
