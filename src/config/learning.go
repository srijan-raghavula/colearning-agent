package config

import "os"

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
