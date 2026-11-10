package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	OpenAIKey   string
	OpenAIModel string
	AgentToken  string
	ListenAddr  string
}

// LoadConfig loads configuration from environment variables and a .env file if present.
func LoadConfig() (Config, error) {
	// load .env file if it exists
	if err := loadDotEnv(); err != nil {
		return Config{}, err
	}

	// load configuration from environment variables
	cfg := Config{
		OpenAIKey:   strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		OpenAIModel: strings.TrimSpace(os.Getenv("OPENAI_MODEL")),
		AgentToken:  strings.TrimSpace(os.Getenv("AGENT_API_TOKEN")),
		ListenAddr:  strings.TrimSpace(os.Getenv("LISTEN_ADDR")),
	}

	if cfg.OpenAIKey == "" {
		return Config{}, errors.New("OPENAI_API_KEY is required")
	}
	if cfg.AgentToken == "" || len(cfg.AgentToken) < 24 {
		return Config{}, errors.New("AGENT_API_TOKEN is required and must have at least 24 characters")
	}
	if cfg.OpenAIModel == "" {
		cfg.OpenAIModel = "gpt-5.4-mini"
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = "127.0.0.1:8080"
	}
	return cfg, nil
}

// loadDotEnv loads environment variables from a .env file if it exists, without overwriting existing environment variables.
func loadDotEnv() error {
	content, err := os.ReadFile(".env")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read .env: %w", err)
	}

	content = bytes.TrimPrefix(content, []byte{0xEF, 0xBB, 0xBF})
	values, err := godotenv.UnmarshalBytes(content)
	if err != nil {
		return fmt.Errorf("load .env: %w", err)
	}

	for key, value := range values {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from .env: %w", key, err)
		}
	}
	return nil
}
