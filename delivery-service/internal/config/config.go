package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	ServerPort  string
	GRPCPort    string
	ConsulAddr  string
	RedisBroker struct {
		Host     string
		Port     string
		Password string
		DB       int
	}
}

func LoadConfig() *Config {
	return &Config{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "delivery_db"),
		DBPort:     getEnv("DB_PORT", "5432"),
		ServerPort: getEnv("SERVER_PORT", "8081"),
		GRPCPort:   getEnv("GRPC_PORT", "50051"),
		ConsulAddr: getEnv("CONSUL_ADDR", "consul:8500"),
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

func LoadTestConfig() *Config {
	return &Config{
		DBHost:     getEnv("TEST_DB_HOST", "localhost"),
		DBUser:     getEnv("TEST_DB_USER", "postgres"),
		DBPassword: getEnv("TEST_DB_PASSWORD", "password"),
		DBName:     getEnv("TEST_DB_NAME", "products_test_db"),
		DBPort:     getEnv("TEST_DB_PORT", "5432"),
		RedisBroker: struct {
			Host     string
			Port     string
			Password string
			DB       int
		}{
			Host:     getEnv("TEST_REDIS_HOST", "redis"),
			Port:     getEnv("TEST_REDIS_PORT", "6379"),
			Password: getEnv("TEST_REDIS_PASSWORD", ""),
			DB:       1,
		},
	}
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisBroker.Host, c.RedisBroker.Port)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
