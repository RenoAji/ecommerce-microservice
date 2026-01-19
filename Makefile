# Swagger documentation generation
swagger-user:
	cd user-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger-product:
	cd product-service && swag init -g cmd/api/main.go --parseDependency --parseInternal

swagger: swagger-user swagger-product

.PHONY: swagger swagger-user swagger-product