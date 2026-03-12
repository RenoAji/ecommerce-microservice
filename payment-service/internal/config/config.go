package config

import (
	"fmt"
	"os"
)

type Config struct {
	DBHost            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBPort            string
	ServerPort        string
	GRPCPort          string
	ConsulAddr        string
	MidtransServerKey string
	MidtransClientKey string
	RedisBroker       struct {
		Host     string
		Port     string
		Password string
		DB       int
	}
}

func LoadConfig() *Config {
	return &Config{
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "password"),
		DBName:            getEnv("DB_NAME", "payment_db"),
		DBPort:            getEnv("DB_PORT", "5432"),
		ServerPort:        getEnv("SERVER_PORT", "8081"),
		GRPCPort:          getEnv("GRPC_PORT", "50051"),
		ConsulAddr:        getEnv("CONSUL_ADDR", "consul:8500"),
		MidtransServerKey: getEnv("MIDTRANS_SERVER_KEY", ""),
		MidtransClientKey: getEnv("MIDTRANS_CLIENT_KEY", ""),
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
		DBName:     getEnv("TEST_DB_NAME", "testdb"),
		DBPort:     getEnv("TEST_DB_PORT", "5432"),
		RedisBroker: struct {
			Host     string
			Port     string
			Password string
			DB       int
		}{
			Host:     getEnv("TEST_REDIS_HOST", "localhost"),
			Port:     getEnv("TEST_REDIS_PORT", "6379"),
			Password: getEnv("TEST_REDIS_PASSWORD", ""),
			DB:       1,
		},
	}
}

func (c *Config) GetDSN() string {
	return "host=" + c.DBHost +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" port=" + c.DBPort +
		" sslmode=disable TimeZone=UTC"
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisBroker.Host, c.RedisBroker.Port)
}
