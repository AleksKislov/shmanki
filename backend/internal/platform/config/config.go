package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	Env             string
	DatabaseURL     string
	JWTSecret       string
	JWTTTLHours     int
	LLMAPIURL       string
	LLMAPIKey       string
	LLMModel        string
	LLMProvider     string
	LLMTimeoutSeconds int
	DefaultLanguage string
}

func Load() (Config, error) {
	if _, err := os.Stat(".env"); err != nil {
		return Config{}, fmt.Errorf("missing backend/.env: copy backend/.env.example to backend/.env: %w", err)
	}

	if err := godotenv.Load(".env"); err != nil {
		return Config{}, fmt.Errorf("load backend/.env: %w", err)
	}

	cfg := Config{
		Port:              getEnv("PORT", "8080"),
		Env:               getEnv("ENV", "development"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/shmanki?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "change-me-change-me-change-me-change-me"),
		LLMAPIURL:         getEnv("LLM_API_URL", "https://api.openai.com/v1/chat/completions"),
		LLMAPIKey:         os.Getenv("LLM_API_KEY"),
		LLMModel:          getEnv("LLM_MODEL", "gpt-4.1-mini"),
		LLMProvider:       getEnv("LLM_PROVIDER", "openai-compatible"),
		DefaultLanguage:   getEnv("DEFAULT_LANGUAGE", "en"),
	}

	jwtTTLHours, err := strconv.Atoi(getEnv("JWT_TTL_HOURS", "168"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JWT_TTL_HOURS: %w", err)
	}
	cfg.JWTTTLHours = jwtTTLHours

	llmTimeoutSeconds, err := strconv.Atoi(getEnv("LLM_TIMEOUT_SECONDS", "30"))
	if err != nil {
		return Config{}, fmt.Errorf("parse LLM_TIMEOUT_SECONDS: %w", err)
	}
	cfg.LLMTimeoutSeconds = llmTimeoutSeconds

	return cfg, nil
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
