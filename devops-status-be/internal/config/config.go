package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	SessionKey  string
	Environment string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://devops:devops_local@localhost:5432/devops_status?sslmode=disable"),
		SessionKey:  getEnv("SESSION_KEY", "devops-status-dev-session-key-change-in-prod"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

func (c *Config) Addr() string {
	return fmt.Sprintf(":%s", c.Port)
}

func (c *Config) IsDev() bool {
	return c.Environment == "development"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
