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
	GRPCPort       string
	ServerPort     string
	ConsulAddr     string
}

func LoadConfig() *Config {
	return &Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "password"),
		DBName:         getEnv("DB_NAME", "orders_db"),
		DBPort:         getEnv("DB_PORT", "5432"),
		ServerPort:     getEnv("SERVER_PORT", "8081"),
		GRPCPort:       getEnv("GRPC_PORT", "50051"),
		ConsulAddr:    getEnv("CONSUL_ADDR", "consul:8500"),
	}
}

func LoadTestConfig() *Config {
	return &Config{
		DBHost:     getEnv("TEST_DB_HOST", "localhost"),
		DBUser:     getEnv("TEST_DB_USER", "postgres"),
		DBPassword: getEnv("TEST_DB_PASSWORD", "password"),
		DBName:     getEnv("TEST_DB_NAME", "products_test_db"),
		DBPort:     getEnv("TEST_DB_PORT", "5432"),
	}
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
