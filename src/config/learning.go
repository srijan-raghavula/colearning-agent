package config

import (
	"os"
	"strconv"
	"time"
)

// LearningStore controls the durable snapshot for progress, evidence, and
// formative checks. Accounts, content, sessions, and turns use SQLite.
// Set LEARNING_STORE_PATH=off to disable snapshot persistence.
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
	return AI{
		Provider: envOr("COLEARN_AI_PROVIDER", "stub"),
		BaseURL:  envOr("COLEARN_AI_BASE_URL", "https://api.openai.com/v1"),
		APIKey:   os.Getenv("COLEARN_AI_API_KEY"),
		Model:    envOr("COLEARN_AI_MODEL", "gpt-4o-mini"),
		Timeout:  timeout,
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
