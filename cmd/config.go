package main

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type config struct {

	// Database, Broker Related

	DatabaseURI  string
	DatabaseName string
	RedisURI     string

	// Addresses

	ServerAddr string

	// JWT-Related

	JwtSecretKey  string
	JwtExpiryTime time.Duration

	// Development

	Environment string
}

func LoadConfig(envPath string) (*config, error) {

	if err := godotenv.Load(envPath); err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	return &config{
		// Database / Broker
		DatabaseURI:  getStringFromEnv("DATABASE_URI", "mongodb://localhost:27017"),
		DatabaseName: getStringFromEnv("DATABASE_NAME", "online_exam_db"),
		RedisURI:     getStringFromEnv("REDIS_URI", "localhost:6379"),
		// Server
		ServerAddr: getStringFromEnv("SERVER_ADDR", "localhost:3000"),
		// JWT
		JwtSecretKey:  getStringFromEnv("JWT_SECRET_KEY", "secret-key"),
		JwtExpiryTime: getDurationFromEnv("JWT_EXPIRY_TIME", 1*time.Minute),
		// DEVELOPMENT
		Environment: getStringFromEnv("ENVIRONMENT", "development"),
	}, nil
}

func getStringFromEnv(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	return value
}

func getDurationFromEnv(key string, defaultValue time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}

	return duration
}
