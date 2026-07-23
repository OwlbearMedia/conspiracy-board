package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port        int
	DatabaseURL string
	// ServeStatic enables serving the embedded SPA bundle from the API
	// container. Used by preview environments (single-container deploys);
	// production serves the frontend from S3 + CloudFront instead.
	ServeStatic bool
	Env         string // local | preview | production
}

func Load() (Config, error) {
	cfg := Config{
		Port:        8080,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		ServeStatic: os.Getenv("SERVE_STATIC") == "1",
		Env:         getEnv("APP_ENV", "local"),
	}

	if p := os.Getenv("PORT"); p != "" {
		port, err := strconv.Atoi(p)
		if err != nil {
			return Config{}, fmt.Errorf("invalid PORT %q: %w", p, err)
		}
		cfg.Port = port
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
