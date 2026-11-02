package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// LoadDotEnv reads a local .env file without overriding values already supplied
// by the process environment. Missing files are normal in containers.
func LoadDotEnv() error {
	file, err := os.Open(".env")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open .env: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		raw = strings.TrimPrefix(raw, "export ")
		key, value, ok := strings.Cut(raw, "=")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if !ok || key == "" {
			return fmt.Errorf("invalid .env entry on line %d", line)
		}
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		} else if comment := strings.Index(value, " #"); comment >= 0 {
			value = strings.TrimSpace(value[:comment])
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("load .env key %q: %w", key, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read .env: %w", err)
	}
	return nil
}

// LearningStore controls optional import of the legacy learning snapshot.
// Runtime learning records use SQLite.
// Set LEARNING_STORE_PATH=off to skip legacy snapshot import.
type LearningStore struct {
	Enabled bool
	Path    string
}

func LoadLearningStore() LearningStore {
	path := os.Getenv("LEARNING_STORE_PATH")
	switch path {
	case "":
		return LearningStore{Enabled: true, Path: "data/learning-state.json"}
	case "off":
		return LearningStore{Enabled: false}
	default:
		return LearningStore{Enabled: true, Path: path}
	}
}

type AI struct {
	Provider string
	BaseURL  string
	APIKey   string
	Model    string
	Timeout  time.Duration
}

func LoadAI() AI {
	timeout := 45 * time.Second
	if raw := os.Getenv("COLEARN_AI_TIMEOUT_SECONDS"); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 && seconds <= 300 {
			timeout = time.Duration(seconds) * time.Second
		}
	}
	provider := os.Getenv("COLEARN_AI_PROVIDER")
	apiKey := envOr("COLEARN_AI_API_KEY", os.Getenv("API_KEY"))
	baseURL := os.Getenv("COLEARN_AI_BASE_URL")
	model := os.Getenv("COLEARN_AI_MODEL")
	if provider == "" {
		provider = "stub"
		if apiKey != "" {
			provider = "openai-compatible"
		}
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
		if strings.HasPrefix(strings.TrimSpace(apiKey), "gsk_") {
			baseURL = "https://api.groq.com/openai/v1"
		}
	}
	if model == "" {
		model = "gpt-4o-mini"
		if strings.HasPrefix(strings.TrimSpace(apiKey), "gsk_") {
			model = "openai/gpt-oss-120b"
		}
	}
	return AI{
		Provider: provider,
		BaseURL:  baseURL,
		APIKey:   apiKey,
		Model:    model,
		Timeout:  timeout,
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
