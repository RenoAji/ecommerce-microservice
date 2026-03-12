package config

import (
	"fmt"
	"os"
)

type Config struct {
	ServerPort    string
	GRPCPort      string
	ConsulAddr    string
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	RedisBroker   struct {
		Host     string
		Port     string
		Password string
		DB       int
	}
}

func LoadConfig() *Config {
	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8081"),
		GRPCPort:      getEnv("GRPC_PORT", "50051"),
		ConsulAddr:    getEnv("CONSUL_ADDR", "consul:8500"),
		RedisHost:     getEnv("REDIS_HOST", "redis"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
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
