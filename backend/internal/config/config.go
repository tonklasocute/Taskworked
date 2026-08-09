package config

import (
	"os"
	"strconv"
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
	// MinioEndpoint is what the API process connects to (e.g. the "minio"
	// hostname on the Docker Compose network). MinioPublicEndpoint is what
	// a browser can reach, used only when signing attachment download
	// URLs — falls back to MinioEndpoint when unset (local dev, where
	// both are "localhost:9000").
	MinioEndpoint       string
	MinioPublicEndpoint string
	MinioAccessKey      string
	MinioSecretKey      string
	MinioBucket         string
	MinioUseSSL         bool
	SMTPHost            string
	SMTPPort            string
	SMTPUser            string
	SMTPPassword        string
	SMTPFrom            string
	LineNotifyToken     string
	// Rate limiting (in-memory, per-process — see bootstrap.go). Global
	// applies to every request; Auth is a stricter limit scoped to the
	// unauthenticated auth endpoints (register/login/refresh) to blunt
	// credential-stuffing/brute-force attempts.
	RateLimitGlobalMax    int
	RateLimitGlobalWindow time.Duration
	RateLimitAuthMax      int
	RateLimitAuthWindow   time.Duration
}

func Load() *Config {
	_ = godotenv.Load()

	minioEndpoint := env("MINIO_ENDPOINT", "localhost:9000")

	return &Config{
		Port:                env("PORT", "8080"),
		DatabaseURL:         mustEnv("DATABASE_URL"),
		RedisAddr:           env("REDIS_ADDR", "localhost:6379"),
		RedisPassword:       env("REDIS_PASSWORD", ""),
		JWTAccessSecret:     mustEnv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:    mustEnv("JWT_REFRESH_SECRET"),
		AccessTokenTTL:      15 * time.Minute,
		RefreshTokenTTL:     7 * 24 * time.Hour,
		AnthropicAPIKey:     env("ANTHROPIC_API_KEY", ""),
		MinioEndpoint:       minioEndpoint,
		MinioPublicEndpoint: env("MINIO_PUBLIC_ENDPOINT", minioEndpoint),
		MinioAccessKey:      env("MINIO_ACCESS_KEY", ""),
		MinioSecretKey:      env("MINIO_SECRET_KEY", ""),
		MinioBucket:         env("MINIO_BUCKET", "taskworked"),
		MinioUseSSL:         env("MINIO_USE_SSL", "false") == "true",
		SMTPHost:            env("SMTP_HOST", ""),
		SMTPPort:            env("SMTP_PORT", "587"),
		SMTPUser:            env("SMTP_USER", ""),
		SMTPPassword:        env("SMTP_PASSWORD", ""),
		SMTPFrom:            env("SMTP_FROM", "noreply@taskworked.local"),
		LineNotifyToken:     env("LINE_NOTIFY_TOKEN", ""),

		RateLimitGlobalMax:    envInt("RATE_LIMIT_GLOBAL_MAX", 300),
		RateLimitGlobalWindow: envDuration("RATE_LIMIT_GLOBAL_WINDOW", time.Minute),
		RateLimitAuthMax:      envInt("RATE_LIMIT_AUTH_MAX", 10),
		RateLimitAuthWindow:   envDuration("RATE_LIMIT_AUTH_WINDOW", time.Minute),
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

// envInt parses an integer env var, falling back (rather than failing
// startup) on missing/invalid values — rate-limit tuning shouldn't be able
// to take the whole API down via a typo.
func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// envDuration parses a Go duration string (e.g. "90s", "2m") env var, with
// the same fallback-on-invalid behavior as envInt.
func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
