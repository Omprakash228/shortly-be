package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL           string
	BaseURL               string // Backend base URL
	FrontendURL           string // Frontend base URL (for QR codes and short URLs)
	RedisURL              string
	JWTSecret             string  // Secret key for JWT token signing
	JWTTTL                int     // JWT token expiration time in hours
	RateLimitRPS          float64 // Rate limit for general API endpoints (requests per second)
	RateLimitBurst        int     // Burst size for rate limiting
	RateLimitAuthRPS      float64 // Rate limit for auth endpoints (stricter)
	RateLimitAuthBurst    int     // Burst size for auth endpoints
	RateLimitShortenRPS   float64 // Rate limit for URL shortening (stricter)
	RateLimitShortenBurst int     // Burst size for URL shortening
}

func Load() *Config {
	// Try to load .env file (ignore error if file doesn't exist)
	if err := godotenv.Load(); err != nil {
		log.Println(err)
		log.Println("No .env file found, using environment variables or defaults")
	}

	return &Config{
		DatabaseURL:           getEnv("DATABASE_URL", ""),
		BaseURL:               getEnv("BASE_URL", ""),
		FrontendURL:           getEnv("FRONTEND_URL", ""),
		RedisURL:              getEnv("REDIS_URL", ""),
		JWTSecret:             getEnv("JWT_SECRET", ""),
		JWTTTL:                getEnvInt("JWT_TTL_HOURS", 0),            // 24 hours default
		RateLimitRPS:          getEnvFloat("RATE_LIMIT_RPS", 0),         // 10 requests per second for general API
		RateLimitBurst:        getEnvInt("RATE_LIMIT_BURST", 0),         // Allow bursts of 20
		RateLimitAuthRPS:      getEnvFloat("RATE_LIMIT_AUTH_RPS", 0),    // 5 requests per second for auth (stricter)
		RateLimitAuthBurst:    getEnvInt("RATE_LIMIT_AUTH_BURST", 0),    // Allow bursts of 10
		RateLimitShortenRPS:   getEnvFloat("RATE_LIMIT_SHORTEN_RPS", 0), // 2 requests per second for URL shortening (stricter)
		RateLimitShortenBurst: getEnvInt("RATE_LIMIT_SHORTEN_BURST", 0), // Allow bursts of 5
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
