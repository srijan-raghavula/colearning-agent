package repository

import "github.com/srijan-raghavula/colearning-agent/src/domain"

// LearningSnapshotVersion is the schema version of LearningSnapshot files.
// Restore logic must stay backward compatible within a version and refuse
// snapshots from newer, unsupported versions instead of corrupting state.
const LearningSnapshotVersion = 1

// LearningSnapshot is a provider-neutral durable image of the learning
// repositories. It is the contract between the runtime adapters (memory
// stores) and any persistence adapter, so a future database swap only
// replaces the adapter behind this type.
type LearningSnapshot struct {
	Version int `json:"version"`

	Courses    []domain.LearningCourse     `json:"courses,omitempty"`
	Concepts   []domain.LearningConcept    `json:"concepts,omitempty"`
	Objectives []domain.LearningObjective  `json:"objectives,omitempty"`
	Materials  []domain.LearningMaterial   `json:"materials,omitempty"`
	Sessions   []domain.LearningSession    `json:"sessions,omitempty"`
	Turns      []domain.LearningTurn       `json:"turns,omitempty"`
	Progress   []domain.ConceptProgress    `json:"progress,omitempty"`
	Evidence   []domain.LearningEvidence   `json:"evidence,omitempty"`
	Checks     []domain.UnderstandingCheck `json:"checks,omitempty"`
}
