package controller

import (
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

// The API DTOs are intentionally separate from the domain read models. The
// dashboard can evolve internally without silently changing the v1 JSON
// contract.

type apiTutorPolicy struct {
	DefaultMode          domain.TutorMode   `json:"default_mode"`
	AllowedModes         []domain.TutorMode `json:"allowed_modes"`
	AllowExternalSources bool               `json:"allow_external_sources"`
	SourcePolicy         string             `json:"source_policy"`
}

type apiSourceRef struct {
	ID        string                    `json:"id"`
	Title     string                    `json:"title"`
	Kind      domain.LearningSourceKind `json:"kind"`
	Reference string                    `json:"reference,omitempty"`
}

type apiCourse struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	Subject       string         `json:"subject"`
	Description   string         `json:"description"`
	TeacherID     string         `json:"teacher_id"`
	TeacherName   string         `json:"teacher_name"`
	StudentCount  int            `json:"student_count"`
	ConceptCount  int            `json:"concept_count"`
	TutorPolicy   apiTutorPolicy `json:"tutor_policy"`
	NextConceptID string         `json:"next_concept_id,omitempty"`
}

type apiConcept struct {
	ID               string   `json:"id"`
	CourseID         string   `json:"course_id"`
	Title            string   `json:"title"`
	Summary          string   `json:"summary"`
	Level            string   `json:"level"`
	Order            int      `json:"order"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	PrerequisiteIDs  []string `json:"prerequisite_ids,omitempty"`
}

type apiObjective struct {
	ID          string   `json:"id"`
	CourseID    string   `json:"course_id"`
	ConceptID   string   `json:"concept_id"`
	Description string   `json:"description"`
	Outcomes    []string `json:"outcomes"`
}

type apiMaterial struct {
	ID        string       `json:"id"`
	CourseID  string       `json:"course_id"`
	ConceptID string       `json:"concept_id"`
	Title     string       `json:"title"`
	Body      string       `json:"body"`
	Source    apiSourceRef `json:"source"`
	Published bool         `json:"published"`
	Version   int          `json:"version"`
}

type apiSession struct {
	ID           string                       `json:"id"`
	StudentID    string                       `json:"student_id"`
	CourseID     string                       `json:"course_id"`
	ConceptID    string                       `json:"concept_id"`
	Mode         domain.TutorMode             `json:"mode"`
	Status       domain.LearningSessionStatus `json:"status"`
	StartedAt    time.Time                    `json:"started_at"`
	LastActiveAt time.Time                    `json:"last_active_at"`
}

type apiTurn struct {
	ID              string                  `json:"id"`
	SessionID       string                  `json:"session_id"`
	Role            domain.LearningTurnRole `json:"role"`
	Kind            string                  `json:"kind"`
	Mode            domain.TutorMode        `json:"mode"`
	Body            string                  `json:"body"`
	Sources         []apiSourceRef          `json:"sources,omitempty"`
	ConfidenceDelta int                     `json:"confidence_delta,omitempty"`
	CreatedAt       time.Time               `json:"created_at"`
}

type apiProgress struct {
	StudentID      string                       `json:"student_id"`
	CourseID       string                       `json:"course_id"`
	ConceptID      string                       `json:"concept_id"`
	Status         domain.ConceptProgressStatus `json:"status"`
	Confidence     int                          `json:"confidence"`
	EvidenceCount  int                          `json:"evidence_count"`
	NextAction     string                       `json:"next_action"`
	LastActivityAt time.Time                    `json:"last_activity_at"`
}

type apiConceptCard struct {
	Concept  apiConcept  `json:"concept"`
	Progress apiProgress `json:"progress"`
	State    string      `json:"state"`
}

type apiStats struct {
	ConceptsStarted int `json:"concepts_started"`
	ConceptsSecure  int `json:"concepts_secure"`
	LearningMinutes int `json:"learning_minutes"`
	StreakDays      int `json:"streak_days"`
}

type apiStudentDashboard struct {
	StudentID        string           `json:"student_id"`
	StudentName      string           `json:"student_name"`
	Course           apiCourse        `json:"course"`
	Concepts         []apiConceptCard `json:"concepts"`
	FeaturedConcept  apiConcept       `json:"featured_concept"`
	FeaturedProgress apiProgress      `json:"featured_progress"`
	ActiveSession    *apiSession      `json:"active_session,omitempty"`
	Turns            []apiTurn        `json:"turns"`
	Materials        []apiMaterial    `json:"materials"`
	Objective        apiObjective     `json:"objective"`
	Policy           apiTutorPolicy   `json:"policy"`
	Stats            apiStats         `json:"stats"`
}

type apiTeacherStats struct {
	ActiveLearners    int `json:"active_learners"`
	ConceptsPublished int `json:"concepts_published"`
	AverageConfidence int `json:"average_confidence"`
	NeedsSupport      int `json:"needs_support"`
}

type apiSupport struct {
	StudentID    string `json:"student_id"`
	StudentName  string `json:"student_name"`
	ConceptTitle string `json:"concept_title"`
	Status       string `json:"status"`
	Confidence   int    `json:"confidence"`
	NextAction   string `json:"next_action"`
}

type apiEvidence struct {
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

type apiTeacherDashboard struct {
	Course         apiCourse        `json:"course"`
	Concepts       []apiConceptCard `json:"concepts"`
	Materials      []apiMaterial    `json:"materials"`
	Policy         apiTutorPolicy   `json:"policy"`
	Stats          apiTeacherStats  `json:"stats"`
	SupportNeeded  []apiSupport     `json:"support_needed"`
	RecentEvidence []apiEvidence    `json:"recent_evidence"`
}

type apiTurnResult struct {
	Session    apiSession  `json:"session"`
	Turn       apiTurn     `json:"turn"`
	Progress   apiProgress `json:"progress"`
	NextAction string      `json:"next_action"`
}

type apiUnderstandingCheck struct {
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

func toAPIPolicy(policy domain.TutorPolicy) apiTutorPolicy {
	return apiTutorPolicy{
		DefaultMode:          policy.DefaultMode,
		AllowedModes:         append([]domain.TutorMode(nil), policy.AllowedModes...),
		AllowExternalSources: policy.AllowExternalSources,
		SourcePolicy:         policy.SourcePolicy,
	}
}

func toAPISource(source domain.LearningSourceRef) apiSourceRef {
	return apiSourceRef{ID: source.ID, Title: source.Title, Kind: source.Kind, Reference: source.Reference}
}

func toAPIConcept(concept domain.LearningConcept) apiConcept {
	return apiConcept{ID: concept.ID, CourseID: concept.CourseID, Title: concept.Title, Summary: concept.Summary, Level: concept.Level, Order: concept.Order, EstimatedMinutes: concept.EstimatedMinutes, PrerequisiteIDs: append([]string(nil), concept.PrerequisiteIDs...)}
}

func toAPIObjective(objective domain.LearningObjective) apiObjective {
	return apiObjective{ID: objective.ID, CourseID: objective.CourseID, ConceptID: objective.ConceptID, Description: objective.Description, Outcomes: append([]string(nil), objective.Outcomes...)}
}

func toAPIMaterial(material domain.LearningMaterial) apiMaterial {
	return apiMaterial{ID: material.ID, CourseID: material.CourseID, ConceptID: material.ConceptID, Title: material.Title, Body: material.Body, Source: toAPISource(material.Source), Published: material.Published, Version: material.Version}
}

func toAPICourse(course domain.LearningCourse) apiCourse {
	return apiCourse{ID: course.ID, Title: course.Title, Subject: course.Subject, Description: course.Description, TeacherID: course.TeacherID, TeacherName: course.TeacherName, StudentCount: course.StudentCount, ConceptCount: course.ConceptCount, TutorPolicy: toAPIPolicy(course.TutorPolicy), NextConceptID: course.NextConceptID}
}

func toAPISession(session domain.LearningSession) apiSession {
	return apiSession{ID: session.ID, StudentID: session.StudentID, CourseID: session.CourseID, ConceptID: session.ConceptID, Mode: session.Mode, Status: session.Status, StartedAt: session.StartedAt, LastActiveAt: session.LastActiveAt}
}

func toAPITurn(turn domain.LearningTurn) apiTurn {
	sources := make([]apiSourceRef, 0, len(turn.Sources))
	for _, source := range turn.Sources {
		sources = append(sources, toAPISource(source))
	}
	return apiTurn{ID: turn.ID, SessionID: turn.SessionID, Role: turn.Role, Kind: turn.Kind, Mode: turn.Mode, Body: turn.Body, Sources: sources, ConfidenceDelta: turn.ConfidenceDelta, CreatedAt: turn.CreatedAt}
}

func toAPIProgress(progress domain.ConceptProgress) apiProgress {
	return apiProgress{StudentID: progress.StudentID, CourseID: progress.CourseID, ConceptID: progress.ConceptID, Status: progress.Status, Confidence: progress.Confidence, EvidenceCount: progress.EvidenceCount, NextAction: progress.NextAction, LastActivityAt: progress.LastActivityAt}
}

func toAPICard(card domain.ConceptProgressCard) apiConceptCard {
	return apiConceptCard{Concept: toAPIConcept(card.Concept), Progress: toAPIProgress(card.Progress), State: card.State}
}

func toAPIStudentDashboard(dashboard domain.StudentLearningDashboard) apiStudentDashboard {
	cards := make([]apiConceptCard, 0, len(dashboard.Concepts))
	for _, card := range dashboard.Concepts {
		cards = append(cards, toAPICard(card))
	}
	turns := make([]apiTurn, 0, len(dashboard.Turns))
	for _, turn := range dashboard.Turns {
		turns = append(turns, toAPITurn(turn))
	}
	materials := make([]apiMaterial, 0, len(dashboard.Materials))
	for _, material := range dashboard.Materials {
		materials = append(materials, toAPIMaterial(material))
	}
	var active *apiSession
	if dashboard.ActiveSession != nil {
		converted := toAPISession(*dashboard.ActiveSession)
		active = &converted
	}
	return apiStudentDashboard{StudentID: dashboard.StudentID, StudentName: dashboard.StudentName, Course: toAPICourse(dashboard.Course), Concepts: cards, FeaturedConcept: toAPIConcept(dashboard.FeaturedConcept), FeaturedProgress: toAPIProgress(dashboard.FeaturedProgress), ActiveSession: active, Turns: turns, Materials: materials, Objective: toAPIObjective(dashboard.Objective), Policy: toAPIPolicy(dashboard.Policy), Stats: apiStats(dashboard.Stats)}
}

func toAPITeacherDashboard(dashboard domain.TeacherLearningDashboard) apiTeacherDashboard {
	cards := make([]apiConceptCard, 0, len(dashboard.Concepts))
	for _, card := range dashboard.Concepts {
		cards = append(cards, toAPICard(card))
	}
	materials := make([]apiMaterial, 0, len(dashboard.Materials))
	for _, material := range dashboard.Materials {
		materials = append(materials, toAPIMaterial(material))
	}
	support := make([]apiSupport, 0, len(dashboard.SupportNeeded))
	for _, item := range dashboard.SupportNeeded {
		support = append(support, apiSupport{StudentID: item.StudentID, StudentName: item.StudentName, ConceptTitle: item.ConceptTitle, Status: item.Status, Confidence: item.Confidence, NextAction: item.NextAction})
	}
	evidence := make([]apiEvidence, 0, len(dashboard.RecentEvidence))
	for _, item := range dashboard.RecentEvidence {
		evidence = append(evidence, apiEvidence{ID: item.ID, SessionID: item.SessionID, TurnID: item.TurnID, StudentID: item.StudentID, CourseID: item.CourseID, ConceptID: item.ConceptID, Kind: item.Kind, Note: item.Note, ObservedAt: item.ObservedAt})
	}
	return apiTeacherDashboard{Course: toAPICourse(dashboard.Course), Concepts: cards, Materials: materials, Policy: toAPIPolicy(dashboard.Policy), Stats: apiTeacherStats(dashboard.Stats), SupportNeeded: support, RecentEvidence: evidence}
}

func toAPITurnResult(result domain.LearningTurnResult) apiTurnResult {
	return apiTurnResult{Session: toAPISession(result.Session), Turn: toAPITurn(result.Turn), Progress: toAPIProgress(result.Progress), NextAction: result.NextAction}
}

func toAPICheck(check domain.UnderstandingCheck) apiUnderstandingCheck {
	return apiUnderstandingCheck{ID: check.ID, SessionID: check.SessionID, TurnID: check.TurnID, StudentID: check.StudentID, CourseID: check.CourseID, ConceptID: check.ConceptID, Summary: check.Summary, Feedback: append([]string(nil), check.Feedback...), Evidence: append([]string(nil), check.Evidence...), NextAction: check.NextAction, CreatedAt: check.CreatedAt}
}
