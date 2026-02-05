package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerPort      string
	ProductSvcAddr  string
	RedisHost       string
	RedisPort       string
	RedisPassword   string
	RedisDB         int
	RedisBroker     struct {
		Host     string
		Port     string
		Password string
		DB       int
	}
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "8081"),
		ProductSvcAddr: getEnv("PRODUCT_SERVICE_ADDR", "product-service:50051"),
		RedisHost:      getEnv("REDIS_HOST", "redis"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        0,
		RedisBroker: struct {
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

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

func (c *Config) GetRedisBrokerAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisBroker.Host, c.RedisBroker.Port)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}