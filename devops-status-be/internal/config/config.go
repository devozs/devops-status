package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                 string
	DatabaseURL          string
	SessionKey           string
	SecretEncryptionKey  string
	Environment          string
	ExternalURL          string
	FrontendURL          string
	K8SInsecureSkipTLS   bool
	CLIRunnerImage        string
	CLIRunnerImageAlpine  string
	CLIRunnerImageUbuntu  string
	CLIRunnerImageToolkit string
	CLIBackendExecutor   string // docker | k8s
	CLIDockerNetwork     string
	CLIBackendK8sCluster string // UUID of cluster used when CLIBackendExecutor=k8s
	CLILogMaxBytes       int
}

func Load() *Config {
	env := getEnv("ENVIRONMENT", "development")
	secKey := getEnv("SECRET_ENCRYPTION_KEY", "")
	if secKey == "" && env != "production" {
		// Deterministic 32-byte key for local dev only (not for production).
		sum := sha256.Sum256([]byte(getEnv("SESSION_KEY", "devops-status-dev-session-key-change-in-prod")))
		secKey = hex.EncodeToString(sum[:])
	}
	legacy := getEnv("CLI_RUNNER_IMAGE", "alpine:3.20")
	alpineImg := getEnv("CLI_RUNNER_IMAGE_ALPINE", legacy)
	ubuntuImg := getEnv("CLI_RUNNER_IMAGE_UBUNTU_24", "ubuntu:24.04")
	toolkitImg := strings.TrimSpace(getEnv("CLI_RUNNER_IMAGE_TOOLKIT", ""))
	return &Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://devops:devops_local@localhost:5432/devops_status?sslmode=disable"),
		SessionKey:           getEnv("SESSION_KEY", "devops-status-dev-session-key-change-in-prod"),
		SecretEncryptionKey:  secKey,
		Environment:          env,
		ExternalURL:          getEnv("EXTERNAL_URL", "http://localhost:8080"),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:3000"),
		K8SInsecureSkipTLS:   envBool("K8S_INSECURE_SKIP_TLS_VERIFY", false),
		CLIRunnerImage:        legacy,
		CLIRunnerImageAlpine:  alpineImg,
		CLIRunnerImageUbuntu:  ubuntuImg,
		CLIRunnerImageToolkit: toolkitImg,
		CLIBackendExecutor:   strings.ToLower(strings.TrimSpace(getEnv("CLI_BACKEND_EXECUTOR", "docker"))),
		CLIDockerNetwork:     getEnv("CLI_DOCKER_NETWORK", "bridge"),
		CLIBackendK8sCluster: strings.TrimSpace(getEnv("CLI_BACKEND_K8S_CLUSTER_ID", "")),
		CLILogMaxBytes:       envInt("CLI_LOG_MAX_BYTES", 32768),
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

func envInt(key string, defaultVal int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}
