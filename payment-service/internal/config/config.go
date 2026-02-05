package config

import (
	"fmt"
	"os"
)

type config struct {
	DBHost         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBPort         string
	ServerPort     string
	MidtransServerKey string
	MidtransClientKey string
		RedisBroker    struct{
		Host     string
		Port     string
		Password string
		DB       int
	}
}

func LoadConfig() *config {
	return &config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "password"),
		DBName:         getEnv("DB_NAME", "payment_db"),
		DBPort:         getEnv("DB_PORT", "5432"),
		ServerPort:     getEnv("SERVER_PORT", "8081"),
		MidtransServerKey: getEnv("MIDTRANS_SERVER_KEY", ""),
		MidtransClientKey: getEnv("MIDTRANS_CLIENT_KEY", ""),
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

func (c *config) GetDSN() string {
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

func(c *config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisBroker.Host, c.RedisBroker.Port)
}