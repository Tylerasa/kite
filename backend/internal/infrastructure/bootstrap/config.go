package bootstrap

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                 string
	Port                   string
	LogFile                string
	DatabaseURL            string
	JWTSecret              string
	AllowedOrigin          string
	FXCacheTTLMinutes      int
	FXSpreadPct            float64
	ComplianceNGNThreshold int64
	PayoutMaxConcurrency   int
}

func NewConfig() (cfg *Config, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("config: %v", r)
		}
	}()

	// Load .env if present (dev only — Docker injects env vars directly)
	_ = godotenv.Load()

	requireEnv("DATABASE_URL")
	requireEnv("JWT_SECRET")
	// HS256 security requires at least 256 bits (32 bytes) of key material.
	if len(os.Getenv("JWT_SECRET")) < 32 {
		panic("JWT_SECRET must be at least 32 characters to meet HS256 security requirements")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = "log/app.log"
	}

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}

	return &Config{
		AppEnv:                 os.Getenv("APP_ENV"),
		Port:                   port,
		LogFile:                logFile,
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		AllowedOrigin:          allowedOrigin,
		FXCacheTTLMinutes:      mustParseInt("FX_CACHE_TTL_MINUTES", 5),
		FXSpreadPct:            mustParseFloat("FX_SPREAD_PCT", 0.0075),
		ComplianceNGNThreshold: mustParseInt64("COMPLIANCE_NGN_THRESHOLD", 50_000_000),
		PayoutMaxConcurrency:   mustParseInt("PAYOUT_MAX_CONCURRENCY", 5),
	}, nil
}

func requireEnv(key string) {
	if os.Getenv(key) == "" {
		panic(fmt.Sprintf("%s is required", key))
	}
}

func mustParseInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		panic(fmt.Sprintf("config: %s must be an integer, got %q", key, v))
	}
	return n
}

func mustParseFloat(key string, defaultVal float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		panic(fmt.Sprintf("config: %s must be a float, got %q", key, v))
	}
	return f
}

func mustParseInt64(key string, defaultVal int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("config: %s must be an integer, got %q", key, v))
	}
	return n
}
