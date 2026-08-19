package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	JWTExpiration time.Duration

	InstagramAccessToken        string
	InstagramAppSecret          string
	InstagramWebhookVerifyToken string
}

// Load lee las variables de entorno esperadas (ver CLAUDE.md) y falla si
// falta alguna de las que ya son necesarias en esta etapa del proyecto.
func Load() (Config, error) {
	cfg := Config{
		Port:                        getEnv("PORT", "8080"),
		DatabaseURL:                 os.Getenv("DATABASE_URL"),
		JWTSecret:                   os.Getenv("JWT_SECRET"),
		JWTExpiration:               24 * time.Hour,
		InstagramAccessToken:        os.Getenv("INSTAGRAM_ACCESS_TOKEN"),
		InstagramAppSecret:          os.Getenv("INSTAGRAM_APP_SECRET"),
		InstagramWebhookVerifyToken: os.Getenv("INSTAGRAM_WEBHOOK_VERIFY_TOKEN"),
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("variables de entorno faltantes: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
