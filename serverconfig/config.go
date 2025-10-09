package serverconfig

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {

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
	LogLevel    slog.Level
}

func LoadConfig(envPath string) (*Config, error) {

	if err := godotenv.Load(envPath); err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	return &Config{
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
		LogLevel:    getLogLevelFromEnv("LOG_LEVEL", slog.LevelInfo),
	}, nil
}

// Helper Functions for Getting .env Variables

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

var stringToLogLevel = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func getLogLevelFromEnv(key string, defaultValue slog.Level) slog.Level {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultValue
	}

	logLevel, ok := stringToLogLevel[value]
	if !ok {
		return defaultValue
	}

	return logLevel
}
