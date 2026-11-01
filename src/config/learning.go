package config

import "os"

// LearningStore controls the durable snapshot of the learning repositories.
// The default keeps persistence on so learning state survives restarts; set
// LEARNING_STORE_PATH=off to run purely in memory (useful for demos and CI).
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
