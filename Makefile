# Swagger documentation generation
swagger-user:
	cd user-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger-product:
	cd product-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger-cart:
	cd cart-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger-order:
	cd order-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger-payment:
	cd payment-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger-delivery:
	cd delivery-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger: swagger-user swagger-product swagger-cart swagger-order swagger-payment swagger-delivery

.PHONY: swagger swagger-user swagger-product swagger-cart swagger-order swagger-payment swagger-delivery

# Protocol Buffer generation
proto-product:
	protoc --go_out=product-service --go-grpc_out=product-service proto/product.proto
	protoc --go_out=cart-service --go-grpc_out=cart-service proto/product.proto
	protoc --go_out=order-service --go-grpc_out=order-service proto/product.proto

proto-cart:
	protoc --go_out=cart-service --go-grpc_out=cart-service proto/cart.proto
	protoc --go_out=order-service --go-grpc_out=order-service proto/cart.proto

proto-payment:
	protoc --go_out=payment-service --go-grpc_out=payment-service proto/payment.proto
	protoc --go_out=order-service --go-grpc_out=order-service proto/payment.proto

proto-delivery:
	protoc --go_out=delivery-service --go-grpc_out=delivery-service proto/delivery.proto
	protoc --go_out=order-service --go-grpc_out=order-service proto/delivery.proto

proto: proto-product proto-cart proto-payment proto-delivery

.PHONY: proto proto-product proto-cart proto-payment proto-delivery
# Run all services
up:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

.PHONY: up

# Run in production mode
prod-up:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up --build

.PHONY: prod-up
