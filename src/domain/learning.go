package domain

import "time"

// TutorMode is a configurable teaching behavior for the learning agent.
// Modes are intentionally data-driven so a teacher can change the policy
// without changing the transport or persistence layers.
type TutorMode string

const (
	TutorModeGuided     TutorMode = "guided"
	TutorModeSocratic   TutorMode = "socratic"
	TutorModeDirect     TutorMode = "direct"
	TutorModePractice   TutorMode = "practice"
	TutorModeHint       TutorMode = "hint"
	TutorModeDiagnostic TutorMode = "diagnostic"
)

func (m TutorMode) IsValid() bool {
	switch m {
	case TutorModeGuided, TutorModeSocratic, TutorModeDirect, TutorModePractice, TutorModeHint, TutorModeDiagnostic:
		return true
	default:
		return false
	}
}

func (m TutorMode) Label() string {
	switch m {
	case TutorModeGuided:
		return "Guided path"
	case TutorModeSocratic:
		return "Socratic"
	case TutorModeDirect:
		return "Direct explanation"
	case TutorModePractice:
		return "Practice first"
	case TutorModeHint:
		return "Hints only"
	case TutorModeDiagnostic:
		return "Diagnose first"
	default:
		return "Guided path"
	}
}

type LearningSourceKind string

const (
	LearningSourceTeacher  LearningSourceKind = "teacher"
	LearningSourceExternal LearningSourceKind = "external"
)

type LearningSourceRef struct {
	ID        string             `json:"id"`
	Title     string             `json:"title"`
	Kind      LearningSourceKind `json:"kind"`
	Reference string             `json:"reference,omitempty"`
}

func (s LearningSourceRef) IsExternal() bool {
	return s.Kind == LearningSourceExternal
}

type TutorPolicy struct {
	DefaultMode          TutorMode   `json:"default_mode"`
	AllowedModes         []TutorMode `json:"allowed_modes"`
	AllowExternalSources bool        `json:"allow_external_sources"`
	SourcePolicy         string      `json:"source_policy"`
}

func (p TutorPolicy) AllowsMode(mode TutorMode) bool {
	if mode == "" {
		mode = p.DefaultMode
	}
	if len(p.AllowedModes) == 0 {
		return true
	}
	for _, allowed := range p.AllowedModes {
		if allowed == mode {
			return true
		}
	}
	return false
}

type LearningCourse struct {
	ID            string      `json:"id"`
	Title         string      `json:"title"`
	Subject       string      `json:"subject"`
	Description   string      `json:"description"`
	TeacherID     string      `json:"teacher_id"`
	TeacherName   string      `json:"teacher_name"`
	StudentCount  int         `json:"student_count"`
	ConceptCount  int         `json:"concept_count"`
	TutorPolicy   TutorPolicy `json:"tutor_policy"`
	NextConceptID string      `json:"next_concept_id,omitempty"`
}

type LearningConcept struct {
	ID               string   `json:"id"`
	CourseID         string   `json:"course_id"`
	Title            string   `json:"title"`
	Summary          string   `json:"summary"`
	Level            string   `json:"level"`
	Order            int      `json:"order"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	PrerequisiteIDs  []string `json:"prerequisite_ids,omitempty"`
}

type LearningObjective struct {
	ID          string   `json:"id"`
	CourseID    string   `json:"course_id"`
	ConceptID   string   `json:"concept_id"`
	Description string   `json:"description"`
	Outcomes    []string `json:"outcomes"`
}

type LearningMaterial struct {
	ID        string            `json:"id"`
	CourseID  string            `json:"course_id"`
	ConceptID string            `json:"concept_id"`
	Title     string            `json:"title"`
	Body      string            `json:"body"`
	Source    LearningSourceRef `json:"source"`
	Published bool              `json:"published"`
	Version   int               `json:"version"`
}

type LearningSessionStatus string

const (
	LearningSessionActive   LearningSessionStatus = "active"
	LearningSessionComplete LearningSessionStatus = "complete"
)

type LearningSession struct {
	ID           string                `json:"id"`
	StudentID    string                `json:"student_id"`
	CourseID     string                `json:"course_id"`
	ConceptID    string                `json:"concept_id"`
	Mode         TutorMode             `json:"mode"`
	Status       LearningSessionStatus `json:"status"`
	StartedAt    time.Time             `json:"started_at"`
	LastActiveAt time.Time             `json:"last_active_at"`
}

type LearningTurnRole string

const (
	LearningTurnStudent LearningTurnRole = "student"
	LearningTurnTutor   LearningTurnRole = "tutor"
)

type LearningTurn struct {
	ID              string              `json:"id"`
	SessionID       string              `json:"session_id"`
	Role            LearningTurnRole    `json:"role"`
	Kind            string              `json:"kind"`
	Mode            TutorMode           `json:"mode"`
	Body            string              `json:"body"`
	Sources         []LearningSourceRef `json:"sources,omitempty"`
	ConfidenceDelta int                 `json:"confidence_delta,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
}

type LearningEvidence struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	TurnID     string    `json:"turn_id"`
	StudentID  string    `json:"student_id"`
	CourseID   string    `json:"course_id"`
	ConceptID  string    `json:"concept_id"`
	Kind       string    `json:"kind"`
	Note       string    `json:"note"`
	ObservedAt time.Time `json:"observed_at"`
}

type ConceptProgressStatus string

const (
	ConceptProgressNotStarted   ConceptProgressStatus = "not_started"
	ConceptProgressLearning     ConceptProgressStatus = "learning"
	ConceptProgressNeedsSupport ConceptProgressStatus = "needs_support"
	ConceptProgressSecure       ConceptProgressStatus = "secure"
)

func (s ConceptProgressStatus) Label() string {
	switch s {
	case ConceptProgressLearning:
		return "In progress"
	case ConceptProgressNeedsSupport:
		return "Needs support"
	case ConceptProgressSecure:
		return "Secure"
	default:
		return "Not started"
	}
}

type ConceptProgress struct {
	StudentID      string                `json:"student_id"`
	CourseID       string                `json:"course_id"`
	ConceptID      string                `json:"concept_id"`
	Status         ConceptProgressStatus `json:"status"`
	Confidence     int                   `json:"confidence"`
	EvidenceCount  int                   `json:"evidence_count"`
	NextAction     string                `json:"next_action"`
	LastActivityAt time.Time             `json:"last_activity_at"`
}

func (p ConceptProgress) Percent() int {
	if p.Confidence < 0 {
		return 0
	}
	if p.Confidence > 100 {
		return 100
	}
	return p.Confidence
}

type ConceptProgressCard struct {
	Concept  LearningConcept `json:"concept"`
	Progress ConceptProgress `json:"progress"`
	State    string          `json:"state"`
}

type StudentLearningStats struct {
	ConceptsStarted int `json:"concepts_started"`
	ConceptsSecure  int `json:"concepts_secure"`
	LearningMinutes int `json:"learning_minutes"`
	StreakDays      int `json:"streak_days"`
}

type StudentLearningDashboard struct {
	StudentID        string                `json:"student_id"`
	StudentName      string                `json:"student_name"`
	Course           LearningCourse        `json:"course"`
	Concepts         []ConceptProgressCard `json:"concepts"`
	FeaturedConcept  LearningConcept       `json:"featured_concept"`
	FeaturedProgress ConceptProgress       `json:"featured_progress"`
	ActiveSession    *LearningSession      `json:"active_session,omitempty"`
	Turns            []LearningTurn        `json:"turns"`
	Materials        []LearningMaterial    `json:"materials"`
	Objective        LearningObjective     `json:"objective"`
	Policy           TutorPolicy           `json:"policy"`
	Stats            StudentLearningStats  `json:"stats"`
}

type TeacherLearningStats struct {
	ActiveLearners    int `json:"active_learners"`
	ConceptsPublished int `json:"concepts_published"`
	AverageConfidence int `json:"average_confidence"`
	NeedsSupport      int `json:"needs_support"`
}

type LearnerSupportCard struct {
	StudentID    string `json:"student_id"`
	StudentName  string `json:"student_name"`
	ConceptTitle string `json:"concept_title"`
	Status       string `json:"status"`
	Confidence   int    `json:"confidence"`
	NextAction   string `json:"next_action"`
}

type TeacherLearningDashboard struct {
	Course         LearningCourse        `json:"course"`
	Concepts       []ConceptProgressCard `json:"concepts"`
	Materials      []LearningMaterial    `json:"materials"`
	Policy         TutorPolicy           `json:"policy"`
	Stats          TeacherLearningStats  `json:"stats"`
	SupportNeeded  []LearnerSupportCard  `json:"support_needed"`
	RecentEvidence []LearningEvidence    `json:"recent_evidence"`
}

type LearningTurnResult struct {
	Session    LearningSession `json:"session"`
	Turn       LearningTurn    `json:"turn"`
	Progress   ConceptProgress `json:"progress"`
	NextAction string          `json:"next_action"`
}

type UnderstandingCheck struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	TurnID     string    `json:"turn_id"`
	StudentID  string    `json:"student_id"`
	CourseID   string    `json:"course_id"`
	ConceptID  string    `json:"concept_id"`
	Summary    string    `json:"summary"`
	Feedback   []string  `json:"feedback"`
	Evidence   []string  `json:"evidence"`
	NextAction string    `json:"next_action"`
	CreatedAt  time.Time `json:"created_at"`
}
