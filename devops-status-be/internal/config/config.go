package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port               string
	DatabaseURL        string
	SessionKey         string
	Environment        string
	ExternalURL        string
	FrontendURL        string
	K8SInsecureSkipTLS bool
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://devops:devops_local@localhost:5432/devops_status?sslmode=disable"),
		SessionKey:  getEnv("SESSION_KEY", "devops-status-dev-session-key-change-in-prod"),
		Environment: getEnv("ENVIRONMENT", "development"),
		ExternalURL: getEnv("EXTERNAL_URL", "http://localhost:8080"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		K8SInsecureSkipTLS: envBool("K8S_INSECURE_SKIP_TLS_VERIFY", false),
	}
}

func (c *Config) IsProd() bool {
	return c.Environment == "production"
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

func envBool(key string, defaultVal bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return defaultVal
	}
	return v == "1" || v == "true" || v == "yes"
}
