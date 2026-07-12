package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type config struct {

	// Database, Broker Related

	DatabaseName       string
	DbConnectionString string
	RedisURI           string

	// Addresses

	ServerAddr string

	// JWT-Related

	JwtSecretKey  string
	JwtExpiryTime time.Duration

	// Development

	Environment string
}

func LoadConfig() (*config, error) {
	log.Println(os.Getwd())
	for _, path := range []string{".env", ".env.local"} {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := godotenv.Load(path); err != nil {
			return nil, fmt.Errorf("error loading %s: %w", path, err)
		}
	}

	return &config{
		DatabaseName:       getStringFromEnv("DATABASE_NAME", "online_exam_db"),
		DbConnectionString: getStringFromEnv("DB_CONNECTION_STRING", ""),
		RedisURI:           getStringFromEnv("REDIS_URI", "localhost:6379"),

		ServerAddr: getStringFromEnv("SERVER_ADDR", ":3000"),

		JwtSecretKey:  getStringFromEnv("JWT_SECRET_KEY", "secret-key"),
		JwtExpiryTime: getDurationFromEnv("JWT_EXPIRY_TIME", 1*time.Hour),

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
