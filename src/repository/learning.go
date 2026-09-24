package repository

import (
	"context"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

// LearningContentRepository is the persistence port for the teacher-owned
// learning context. Course, concept, objective, and material records are
// deliberately grouped behind one port until a production database is chosen.
type LearningContentRepository interface {
	FindCourse(ctx context.Context, id string) (domain.LearningCourse, bool, error)
	ListCourses(ctx context.Context) ([]domain.LearningCourse, error)
	SaveCourse(ctx context.Context, course domain.LearningCourse) error

	FindConcept(ctx context.Context, id string) (domain.LearningConcept, bool, error)
	ListConcepts(ctx context.Context, courseID string) ([]domain.LearningConcept, error)
	SaveConcept(ctx context.Context, concept domain.LearningConcept) error

	ListObjectives(ctx context.Context, courseID, conceptID string) ([]domain.LearningObjective, error)
	SaveObjective(ctx context.Context, objective domain.LearningObjective) error

	FindMaterial(ctx context.Context, id string) (domain.LearningMaterial, bool, error)
	ListMaterials(ctx context.Context, courseID, conceptID string) ([]domain.LearningMaterial, error)
	ListAllMaterials(ctx context.Context, courseID string) ([]domain.LearningMaterial, error)
	SaveMaterial(ctx context.Context, material domain.LearningMaterial) error
}

type LearningSessionRepository interface {
	FindByID(ctx context.Context, id string) (domain.LearningSession, bool, error)
	FindActive(ctx context.Context, studentID, courseID string) (domain.LearningSession, bool, error)
	Save(ctx context.Context, session domain.LearningSession) error
	AppendTurn(ctx context.Context, turn domain.LearningTurn) error
	ListTurns(ctx context.Context, sessionID string) ([]domain.LearningTurn, error)
}

type ConceptProgressRepository interface {
	Find(ctx context.Context, studentID, courseID, conceptID string) (domain.ConceptProgress, bool, error)
	ListByStudent(ctx context.Context, studentID, courseID string) ([]domain.ConceptProgress, error)
	ListAll(ctx context.Context) ([]domain.ConceptProgress, error)
	Save(ctx context.Context, progress domain.ConceptProgress) error
}

type LearningEvidenceRepository interface {
	Save(ctx context.Context, evidence domain.LearningEvidence) error
	ListByCourse(ctx context.Context, courseID string) ([]domain.LearningEvidence, error)
}

type UnderstandingCheckRepository interface {
	Save(ctx context.Context, check domain.UnderstandingCheck) error
	FindLatestBySession(ctx context.Context, sessionID string) (domain.UnderstandingCheck, bool, error)
	ListByCourse(ctx context.Context, courseID string) ([]domain.UnderstandingCheck, error)
}

type LearningEvidenceDraft struct {
	Kind            string `json:"kind"`
	Note            string `json:"note"`
	ConfidenceDelta int    `json:"confidence_delta"`
}

// LearningRequest is the provider-neutral contract for a tutor turn.
type LearningRequest struct {
	StudentID   string                    `json:"student_id"`
	CourseID    string                    `json:"course_id"`
	ConceptID   string                    `json:"concept_id"`
	SessionID   string                    `json:"session_id"`
	Mode        domain.TutorMode          `json:"mode"`
	Message     string                    `json:"message"`
	CourseTitle string                    `json:"course_title"`
	Concept     domain.LearningConcept    `json:"concept"`
	Objective   domain.LearningObjective  `json:"objective"`
	Materials   []domain.LearningMaterial `json:"materials"`
	RecentTurns []domain.LearningTurn     `json:"recent_turns"`
	Progress    domain.ConceptProgress    `json:"progress"`
	Policy      domain.TutorPolicy        `json:"policy"`
}

type LearningResponse struct {
	Kind            string                     `json:"kind"`
	Body            string                     `json:"body"`
	Sources         []domain.LearningSourceRef `json:"sources"`
	Evidence        []LearningEvidenceDraft    `json:"evidence"`
	NextAction      string                     `json:"next_action"`
	ConfidenceDelta int                        `json:"confidence_delta"`
	Escalate        bool                       `json:"escalate"`
}

// LearningAgentGateway is the port for a configurable tutor agent. Provider
// SDKs stay outside the application and service layers.
type LearningAgentGateway interface {
	Respond(ctx context.Context, req LearningRequest) (LearningResponse, error)
}

type FormativeCheckRequest struct {
	SessionID   string                    `json:"session_id"`
	StudentID   string                    `json:"student_id"`
	CourseID    string                    `json:"course_id"`
	ConceptID   string                    `json:"concept_id"`
	TurnID      string                    `json:"turn_id"`
	Answer      string                    `json:"answer"`
	Objective   domain.LearningObjective  `json:"objective"`
	Materials   []domain.LearningMaterial `json:"materials"`
	RecentTurns []domain.LearningTurn     `json:"recent_turns"`
	Progress    domain.ConceptProgress    `json:"progress"`
}

type FormativeCheckResponse struct {
	Summary         string   `json:"summary"`
	Feedback        []string `json:"feedback"`
	Evidence        []string `json:"evidence"`
	NextAction      string   `json:"next_action"`
	ConfidenceDelta int      `json:"confidence_delta"`
}

type FormativeCheckGateway interface {
	Check(ctx context.Context, req FormativeCheckRequest) (FormativeCheckResponse, error)
}
