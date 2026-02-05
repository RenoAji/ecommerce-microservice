package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBHost         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBPort         string
	ServerPort     string
	CartSvcAddr    string
	ProductSvcAddr string
	PaymentSvcAddr string
	RedisBroker    struct{
		Host     string
		Port     string
		Password string
		DB       int
	}
}

func LoadConfig() *Config {
	return &Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "password"),
		DBName:         getEnv("DB_NAME", "orders_db"),
		DBPort:         getEnv("DB_PORT", "5432"),
		ServerPort:     getEnv("SERVER_PORT", "8081"),
		CartSvcAddr:    getEnv("CART_SVC_ADDR", "cart-service:50051"),
		ProductSvcAddr: getEnv("PRODUCT_SVC_ADDR", "product-service:50051"),
		PaymentSvcAddr: getEnv("PAYMENT_SVC_ADDR", "payment-service:50051"),
		RedisBroker: struct{
			Host     string
			Port     string
			Password string
			DB       int
		}{
			Host:     getEnv("REDIS_HOST", "redis"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       1,
		},
	}
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort)
}

func(c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisBroker.Host, c.RedisBroker.Port)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}