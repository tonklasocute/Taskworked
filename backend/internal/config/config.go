package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	DatabaseURL      string
	RedisAddr        string
	RedisPassword    string
	JWTAccessSecret  string
	JWTRefreshSecret string
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	AnthropicAPIKey  string
	MinioEndpoint    string
	MinioAccessKey   string
	MinioSecretKey   string
	MinioBucket      string
	SMTPHost         string
	SMTPPort         string
	SMTPUser         string
	SMTPPassword     string
	LineNotifyToken  string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:             env("PORT", "8080"),
		DatabaseURL:      mustEnv("DATABASE_URL"),
		RedisAddr:        env("REDIS_ADDR", "localhost:6379"),
		RedisPassword:    env("REDIS_PASSWORD", ""),
		JWTAccessSecret:  mustEnv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret: mustEnv("JWT_REFRESH_SECRET"),
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  7 * 24 * time.Hour,
		AnthropicAPIKey:  env("ANTHROPIC_API_KEY", ""),
		MinioEndpoint:    env("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:   env("MINIO_ACCESS_KEY", ""),
		MinioSecretKey:   env("MINIO_SECRET_KEY", ""),
		MinioBucket:      env("MINIO_BUCKET", "taskworked"),
		SMTPHost:         env("SMTP_HOST", ""),
		SMTPPort:         env("SMTP_PORT", "587"),
		SMTPUser:         env("SMTP_USER", ""),
		SMTPPassword:     env("SMTP_PASSWORD", ""),
		LineNotifyToken:  env("LINE_NOTIFY_TOKEN", ""),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// mustEnv panics on startup (not at request time) if a required secret is
// missing, so misconfiguration fails fast instead of surfacing as a
// confusing runtime error later.
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required environment variable: " + key)
	}
	return v
}
