package bootstrap

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                 string
	Port                   string
	DatabaseURL            string
	JWTSecret              string
	FXCacheTTLMinutes      int
	FXSpreadPct            float64
	ComplianceNGNThreshold int64
}

func NewConfig() (*Config, error) {
	// Load .env if present (dev only — Docker injects env vars directly)
	_ = godotenv.Load()

	cacheTTL := 5
	if v := os.Getenv("FX_CACHE_TTL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cacheTTL = n
		}
	}

	spread := 0.0075
	if v := os.Getenv("FX_SPREAD_PCT"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			spread = f
		}
	}

	threshold := int64(50_000_000) // ₦500,000 in kobo
	if v := os.Getenv("COMPLIANCE_NGN_THRESHOLD"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			threshold = n
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		AppEnv:                 os.Getenv("APP_ENV"),
		Port:                   port,
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		FXCacheTTLMinutes:      cacheTTL,
		FXSpreadPct:            spread,
		ComplianceNGNThreshold: threshold,
	}, nil
}
