package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
	"github.com/srijan-raghavula/colearning-agent/src/service"
)

const defaultLearningStudentID = "S-001"

var learningIDSequence atomic.Uint64

type StartLearningRequest struct {
	StudentID string           `json:"student_id,omitempty"`
	CourseID  string           `json:"course_id,omitempty"`
	ConceptID string           `json:"concept_id,omitempty"`
	SessionID string           `json:"session_id,omitempty"`
	Mode      domain.TutorMode `json:"mode,omitempty"`
}

type SendLearningTurnRequest struct {
	StudentID string           `json:"student_id,omitempty"`
	SessionID string           `json:"session_id,omitempty"`
	Mode      domain.TutorMode `json:"mode,omitempty"`
	Message   string           `json:"message"`
}

type PublishMaterialRequest struct {
	CourseID   string                    `json:"course_id"`
	ConceptID  string                    `json:"concept_id"`
	Title      string                    `json:"title"`
	Body       string                    `json:"body"`
	SourceKind domain.LearningSourceKind `json:"source_kind"`
	SourceRef  string                    `json:"source_ref,omitempty"`
}

type UpdateTutorPolicyRequest struct {
	CourseID             string             `json:"course_id,omitempty"`
	DefaultMode          domain.TutorMode   `json:"default_mode"`
	AllowedModes         []domain.TutorMode `json:"allowed_modes"`
	AllowExternalSources bool               `json:"allow_external_sources"`
	SourcePolicy         string             `json:"source_policy,omitempty"`
}

// StudentLearningPortal is the application workflow for the student learning
// loop. The current slice deliberately keeps orchestration thin and delegates
// the tutor provider contract to service.TutorAgent.
type StudentLearningPortal struct {
	content  repository.LearningContentRepository
	sessions repository.LearningSessionRepository
	progress repository.ConceptProgressRepository
	evidence repository.LearningEvidenceRepository
	checks   repository.UnderstandingCheckRepository
	agent    *service.TutorAgent
	checker  *service.UnderstandingCheckService
}

func NewStudentLearningPortal(
	content repository.LearningContentRepository,
	sessions repository.LearningSessionRepository,
	progress repository.ConceptProgressRepository,
	evidence repository.LearningEvidenceRepository,
	checks repository.UnderstandingCheckRepository,
	agent *service.TutorAgent,
	checker *service.UnderstandingCheckService,
) StudentLearningPortal {
	return StudentLearningPortal{
		content:  content,
		sessions: sessions,
		progress: progress,
		evidence: evidence,
		checks:   checks,
		agent:    agent,
		checker:  checker,
	}
}

func (p StudentLearningPortal) Dashboard(ctx context.Context, studentID string) (domain.StudentLearningDashboard, error) {
	studentID = defaultStudentIDValue(studentID)
	course, concepts, err := p.primaryCourse(ctx)
	if err != nil {
		return domain.StudentLearningDashboard{}, err
	}

	progressRecords, err := p.progress.ListByStudent(ctx, studentID, course.ID)
	if err != nil {
		return domain.StudentLearningDashboard{}, err
	}
	progressByConcept := make(map[string]domain.ConceptProgress, len(progressRecords))
	for _, record := range progressRecords {
		progressByConcept[record.ConceptID] = record
	}

	cards := make([]domain.ConceptProgressCard, 0, len(concepts))
	started := 0
	secure := 0
	for _, concept := range concepts {
		progress, ok := progressByConcept[concept.ID]
		if !ok {
			progress = newProgress(studentID, course.ID, concept.ID)
		}
		if progress.Status != domain.ConceptProgressNotStarted {
			started++
		}
		if progress.Status == domain.ConceptProgressSecure {
			secure++
		}
		cards = append(cards, domain.ConceptProgressCard{
			Concept:  concept,
			Progress: progress,
			State:    progress.Status.Label(),
		})
	}

	featuredIndex := featuredConceptIndex(cards)
	featured := concepts[featuredIndex]
	featuredProgress := progressByConcept[featured.ID]
	if featuredProgress.ConceptID == "" {
		featuredProgress = newProgress(studentID, course.ID, featured.ID)
	}

	activeSession, hasSession, err := p.sessions.FindActive(ctx, studentID, course.ID)
	if err != nil {
		return domain.StudentLearningDashboard{}, err
	}
	var turns []domain.LearningTurn
	if hasSession {
		turns, err = p.sessions.ListTurns(ctx, activeSession.ID)
		if err != nil {
			return domain.StudentLearningDashboard{}, err
		}
	}
	if hasSession {
		featuredIndex = conceptCardIndex(cards, activeSession.ConceptID)
		featured = cards[featuredIndex].Concept
		featuredProgress = cards[featuredIndex].Progress
	}

	objective, err := p.objectiveFor(ctx, course.ID, featured.ID)
	if err != nil {
		return domain.StudentLearningDashboard{}, err
	}
	materials, err := p.publishedMaterials(ctx, course.ID, featured.ID)
	if err != nil {
		return domain.StudentLearningDashboard{}, err
	}
	if len(materials) == 0 {
		materials, err = p.publishedMaterials(ctx, course.ID, "")
		if err != nil {
			return domain.StudentLearningDashboard{}, err
		}
	}

	var active *domain.LearningSession
	if hasSession {
		active = &activeSession
	}
	return domain.StudentLearningDashboard{
		StudentID:        studentID,
		StudentName:      learningStudentName(studentID),
		Course:           course,
		Concepts:         cards,
		FeaturedConcept:  featured,
		FeaturedProgress: featuredProgress,
		ActiveSession:    active,
		Turns:            turns,
		Materials:        materials,
		Objective:        objective,
		Policy:           course.TutorPolicy,
		Stats: domain.StudentLearningStats{
			ConceptsStarted: started,
			ConceptsSecure:  secure,
			LearningMinutes: 86 + len(turns)*7,
			StreakDays:      6,
		},
	}, nil
}

func (p StudentLearningPortal) primaryCourse(ctx context.Context) (domain.LearningCourse, []domain.LearningConcept, error) {
	courses, err := p.content.ListCourses(ctx)
	if err != nil {
		return domain.LearningCourse{}, nil, err
	}
	if len(courses) == 0 {
		return domain.LearningCourse{}, nil, errors.New("no learning course is configured")
	}
	concepts, err := p.content.ListConcepts(ctx, courses[0].ID)
	if err != nil {
		return domain.LearningCourse{}, nil, err
	}
	if len(concepts) == 0 {
		return domain.LearningCourse{}, nil, errors.New("course has no learning concepts")
	}
	return courses[0], concepts, nil
}

func (p StudentLearningPortal) objectiveFor(ctx context.Context, courseID, conceptID string) (domain.LearningObjective, error) {
	objectives, err := p.content.ListObjectives(ctx, courseID, conceptID)
	if err != nil {
		return domain.LearningObjective{}, err
	}
	if len(objectives) == 0 {
		return domain.LearningObjective{}, nil
	}
	return objectives[0], nil
}

func (p StudentLearningPortal) publishedMaterials(ctx context.Context, courseID, conceptID string) ([]domain.LearningMaterial, error) {
	materials, err := p.content.ListMaterials(ctx, courseID, conceptID)
	if err != nil {
		return nil, err
	}
	published := make([]domain.LearningMaterial, 0, len(materials))
	for _, material := range materials {
		if material.Published {
			published = append(published, material)
		}
	}
	return published, nil
}

func (p StudentLearningPortal) StartOrResume(ctx context.Context, req StartLearningRequest) (domain.LearningSession, error) {
	studentID := defaultLearningStudentIDValue(req.StudentID)
	course, concepts, err := p.primaryCourse(ctx)
	if err != nil {
		return domain.LearningSession{}, err
	}
	if req.CourseID != "" && req.CourseID != course.ID {
		selected, found, findErr := p.content.FindCourse(ctx, req.CourseID)
		if findErr != nil {
			return domain.LearningSession{}, findErr
		}
		if !found {
			return domain.LearningSession{}, fmt.Errorf("course %q was not found", req.CourseID)
		}
		course = selected
		concepts, err = p.content.ListConcepts(ctx, course.ID)
		if err != nil {
			return domain.LearningSession{}, err
		}
	}

	if req.SessionID != "" {
		session, found, findErr := p.sessions.FindByID(ctx, req.SessionID)
		if findErr != nil {
			return domain.LearningSession{}, findErr
		}
		if !found || session.StudentID != studentID {
			return domain.LearningSession{}, errors.New("learning session was not found for this student")
		}
		return session, nil
	}

	active, found, err := p.sessions.FindActive(ctx, studentID, course.ID)
	if err != nil {
		return domain.LearningSession{}, err
	}
	if found && (req.ConceptID == "" || active.ConceptID == req.ConceptID) {
		return active, nil
	}

	conceptID := req.ConceptID
	if conceptID == "" {
		if found {
			conceptID = active.ConceptID
		} else {
			conceptID = concepts[0].ID
		}
	}
	concept, found, err := p.content.FindConcept(ctx, conceptID)
	if err != nil {
		return domain.LearningSession{}, err
	}
	if !found || concept.CourseID != course.ID {
		return domain.LearningSession{}, fmt.Errorf("concept %q is not part of course %q", conceptID, course.ID)
	}

	mode := req.Mode
	if mode == "" {
		mode = course.TutorPolicy.DefaultMode
	}
	if !mode.IsValid() {
		return domain.LearningSession{}, fmt.Errorf("unsupported tutor mode %q", mode)
	}
	if !course.TutorPolicy.AllowsMode(mode) {
		return domain.LearningSession{}, fmt.Errorf("tutor mode %q is not enabled for this course", mode)
	}

	now := time.Now().UTC()
	session := domain.LearningSession{
		ID:           newLearningID("session"),
		StudentID:    studentID,
		CourseID:     course.ID,
		ConceptID:    concept.ID,
		Mode:         mode,
		Status:       domain.LearningSessionActive,
		StartedAt:    now,
		LastActiveAt: now,
	}
	if err := p.sessions.Save(ctx, session); err != nil {
		return domain.LearningSession{}, err
	}
	return session, nil
}

func (p StudentLearningPortal) Session(ctx context.Context, sessionID string) (domain.LearningSession, []domain.LearningTurn, error) {
	session, found, err := p.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return domain.LearningSession{}, nil, err
	}
	if !found {
		return domain.LearningSession{}, nil, errors.New("learning session was not found")
	}
	turns, err := p.sessions.ListTurns(ctx, session.ID)
	if err != nil {
		return domain.LearningSession{}, nil, err
	}
	return session, turns, nil
}

func (p StudentLearningPortal) LatestCheck(ctx context.Context, sessionID string) (*domain.UnderstandingCheck, error) {
	if p.checks == nil {
		return nil, nil
	}
	check, found, err := p.checks.FindLatestBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &check, nil
}

func (p StudentLearningPortal) CheckUnderstanding(ctx context.Context, sessionID, studentID string) (domain.UnderstandingCheck, error) {
	if p.checker == nil {
		return domain.UnderstandingCheck{}, errors.New("understanding check service is not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return domain.UnderstandingCheck{}, errors.New("learning session is required")
	}
	studentID = defaultLearningStudentIDValue(studentID)
	session, turns, err := p.Session(ctx, sessionID)
	if err != nil {
		return domain.UnderstandingCheck{}, err
	}
	if session.StudentID != studentID {
		return domain.UnderstandingCheck{}, errors.New("learning session does not belong to this student")
	}
	var latest domain.LearningTurn
	foundStudentTurn := false
	for _, turn := range turns {
		if turn.Role == domain.LearningTurnStudent {
			latest = turn
			foundStudentTurn = true
		}
	}
	if !foundStudentTurn {
		return domain.UnderstandingCheck{}, errors.New("send a learning explanation before requesting an understanding check")
	}
	if p.checks != nil {
		if existing, found, findErr := p.checks.FindLatestBySession(ctx, session.ID); findErr != nil {
			return domain.UnderstandingCheck{}, findErr
		} else if found && existing.TurnID == latest.ID {
			return existing, nil
		}
	}
	course, found, err := p.content.FindCourse(ctx, session.CourseID)
	if err != nil {
		return domain.UnderstandingCheck{}, err
	}
	if !found {
		return domain.UnderstandingCheck{}, errors.New("course for learning session was not found")
	}
	concept, found, err := p.content.FindConcept(ctx, session.ConceptID)
	if err != nil {
		return domain.UnderstandingCheck{}, err
	}
	if !found {
		return domain.UnderstandingCheck{}, errors.New("concept for learning session was not found")
	}
	objective, err := p.objectiveFor(ctx, course.ID, concept.ID)
	if err != nil {
		return domain.UnderstandingCheck{}, err
	}
	materials, err := p.publishedMaterials(ctx, course.ID, concept.ID)
	if err != nil {
		return domain.UnderstandingCheck{}, err
	}
	progressRecord, found, err := p.progress.Find(ctx, session.StudentID, course.ID, concept.ID)
	if err != nil {
		return domain.UnderstandingCheck{}, err
	}
	if !found {
		progressRecord = newProgress(session.StudentID, course.ID, concept.ID)
	}
	response, err := p.checker.Check(ctx, repository.FormativeCheckRequest{
		SessionID:   session.ID,
		StudentID:   session.StudentID,
		CourseID:    course.ID,
		ConceptID:   concept.ID,
		TurnID:      latest.ID,
		Answer:      latest.Body,
		Objective:   objective,
		Materials:   materials,
		RecentTurns: turns,
		Progress:    progressRecord,
	})
	if err != nil {
		return domain.UnderstandingCheck{}, err
	}
	now := time.Now().UTC()
	check := domain.UnderstandingCheck{
		ID:         newLearningID("check"),
		SessionID:  session.ID,
		TurnID:     latest.ID,
		StudentID:  session.StudentID,
		CourseID:   course.ID,
		ConceptID:  concept.ID,
		Summary:    response.Summary,
		Feedback:   response.Feedback,
		Evidence:   response.Evidence,
		NextAction: response.NextAction,
		CreatedAt:  now,
	}
	if p.checks == nil {
		return domain.UnderstandingCheck{}, errors.New("understanding check repository is not configured")
	}
	if err := p.checks.Save(ctx, check); err != nil {
		return domain.UnderstandingCheck{}, err
	}
	drafts := make([]repository.LearningEvidenceDraft, 0, len(response.Evidence))
	for _, evidence := range response.Evidence {
		drafts = append(drafts, repository.LearningEvidenceDraft{Kind: "understanding_check", Note: evidence})
	}
	if len(drafts) == 0 {
		drafts = []repository.LearningEvidenceDraft{{Kind: "understanding_check", Note: response.Summary}}
	}
	for _, draft := range drafts {
		if err := p.evidence.Save(ctx, domain.LearningEvidence{
			ID:         newLearningID("evidence"),
			SessionID:  session.ID,
			TurnID:     latest.ID,
			StudentID:  session.StudentID,
			CourseID:   course.ID,
			ConceptID:  concept.ID,
			Kind:       draft.Kind,
			Note:       draft.Note,
			ObservedAt: now,
		}); err != nil {
			return domain.UnderstandingCheck{}, err
		}
	}
	progressRecord = updateProgress(progressRecord, domain.LearningTurn{ConfidenceDelta: response.ConfidenceDelta, CreatedAt: now}, drafts, response.NextAction)
	if err := p.progress.Save(ctx, progressRecord); err != nil {
		return domain.UnderstandingCheck{}, err
	}
	return check, nil
}

func (p StudentLearningPortal) Continue(ctx context.Context, req SendLearningTurnRequest) (domain.LearningTurnResult, error) {
	if p.agent == nil {
		return domain.LearningTurnResult{}, errors.New("tutor agent is not configured")
	}
	if strings.TrimSpace(req.Message) == "" {
		return domain.LearningTurnResult{}, errors.New("a learning message is required")
	}

	start := StartLearningRequest{
		StudentID: req.StudentID,
		SessionID: req.SessionID,
		Mode:      req.Mode,
	}
	session, err := p.StartOrResume(ctx, start)
	if err != nil {
		return domain.LearningTurnResult{}, err
	}
	if session.StudentID != defaultLearningStudentIDValue(req.StudentID) {
		return domain.LearningTurnResult{}, errors.New("learning session does not belong to this student")
	}
	if req.Mode == "" {
		req.Mode = session.Mode
	}

	now := time.Now().UTC()
	studentTurn := domain.LearningTurn{
		ID:        newLearningID("turn"),
		SessionID: session.ID,
		Role:      domain.LearningTurnStudent,
		Kind:      "student_message",
		Mode:      req.Mode,
		Body:      strings.TrimSpace(req.Message),
		CreatedAt: now,
	}
	if err := p.sessions.AppendTurn(ctx, studentTurn); err != nil {
		return domain.LearningTurnResult{}, err
	}

	course, found, err := p.content.FindCourse(ctx, session.CourseID)
	if err != nil {
		return domain.LearningTurnResult{}, err
	}
	if !found {
		return domain.LearningTurnResult{}, errors.New("course for learning session was not found")
	}
	concept, found, err := p.content.FindConcept(ctx, session.ConceptID)
	if err != nil {
		return domain.LearningTurnResult{}, err
	}
	if !found {
		return domain.LearningTurnResult{}, errors.New("concept for learning session was not found")
	}
	objective, err := p.objectiveFor(ctx, course.ID, concept.ID)
	if err != nil {
		return domain.LearningTurnResult{}, err
	}
	materials, err := p.publishedMaterials(ctx, course.ID, concept.ID)
	if err != nil {
		return domain.LearningTurnResult{}, err
	}
	if len(materials) == 0 {
		materials, err = p.publishedMaterials(ctx, course.ID, "")
		if err != nil {
			return domain.LearningTurnResult{}, err
		}
	}
	progressRecord, found, err := p.progress.Find(ctx, session.StudentID, course.ID, concept.ID)
	if err != nil {
		return domain.LearningTurnResult{}, err
	}
	if !found {
		progressRecord = newProgress(session.StudentID, course.ID, concept.ID)
	}
	turns, err := p.sessions.ListTurns(ctx, session.ID)
	if err != nil {
		return domain.LearningTurnResult{}, err
	}
	if len(turns) > 8 {
		turns = turns[len(turns)-8:]
	}

	response, err := p.agent.Respond(ctx, repository.LearningRequest{
		StudentID:   session.StudentID,
		CourseID:    course.ID,
		ConceptID:   concept.ID,
		SessionID:   session.ID,
		Mode:        req.Mode,
		Message:     req.Message,
		CourseTitle: course.Title,
		Concept:     concept,
		Objective:   objective,
		Materials:   materials,
		RecentTurns: turns,
		Progress:    progressRecord,
		Policy:      course.TutorPolicy,
	})
	if err != nil {
		return domain.LearningTurnResult{}, err
	}

	tutorTurn := domain.LearningTurn{
		ID:              newLearningID("turn"),
		SessionID:       session.ID,
		Role:            domain.LearningTurnTutor,
		Kind:            response.Kind,
		Mode:            req.Mode,
		Body:            response.Body,
		Sources:         response.Sources,
		ConfidenceDelta: response.ConfidenceDelta,
		CreatedAt:       time.Now().UTC(),
	}
	if err := p.sessions.AppendTurn(ctx, tutorTurn); err != nil {
		return domain.LearningTurnResult{}, err
	}
	session.LastActiveAt = tutorTurn.CreatedAt
	if err := p.sessions.Save(ctx, session); err != nil {
		return domain.LearningTurnResult{}, err
	}

	evidenceDrafts := response.Evidence
	if len(evidenceDrafts) == 0 {
		evidenceDrafts = []repository.LearningEvidenceDraft{{Kind: "conversation", Note: response.Body}}
	}
	for _, draft := range evidenceDrafts {
		if err := p.evidence.Save(ctx, domain.LearningEvidence{
			ID:         newLearningID("evidence"),
			SessionID:  session.ID,
			TurnID:     tutorTurn.ID,
			StudentID:  session.StudentID,
			CourseID:   course.ID,
			ConceptID:  concept.ID,
			Kind:       draft.Kind,
			Note:       draft.Note,
			ObservedAt: tutorTurn.CreatedAt,
		}); err != nil {
			return domain.LearningTurnResult{}, err
		}
	}
	progressRecord = updateProgress(progressRecord, tutorTurn, evidenceDrafts, response.NextAction)
	if err := p.progress.Save(ctx, progressRecord); err != nil {
		return domain.LearningTurnResult{}, err
	}

	return domain.LearningTurnResult{
		Session:    session,
		Turn:       tutorTurn,
		Progress:   progressRecord,
		NextAction: response.NextAction,
	}, nil
}

func updateProgress(progress domain.ConceptProgress, turn domain.LearningTurn, drafts []repository.LearningEvidenceDraft, nextAction string) domain.ConceptProgress {
	progress.EvidenceCount += len(drafts)
	progress.Confidence += turn.ConfidenceDelta
	if progress.Confidence < 0 {
		progress.Confidence = 0
	}
	if progress.Confidence > 100 {
		progress.Confidence = 100
	}
	progress.LastActivityAt = turn.CreatedAt
	if strings.TrimSpace(nextAction) != "" {
		progress.NextAction = nextAction
	}
	switch {
	case progress.Confidence >= 80:
		progress.Status = domain.ConceptProgressSecure
	case progress.Confidence <= 35:
		progress.Status = domain.ConceptProgressNeedsSupport
	default:
		progress.Status = domain.ConceptProgressLearning
	}
	return progress
}

func newProgress(studentID, courseID, conceptID string) domain.ConceptProgress {
	return domain.ConceptProgress{
		StudentID:      studentID,
		CourseID:       courseID,
		ConceptID:      conceptID,
		Status:         domain.ConceptProgressNotStarted,
		Confidence:     0,
		NextAction:     "Start with a guided explanation",
		LastActivityAt: time.Time{},
	}
}

func featuredConceptIndex(cards []domain.ConceptProgressCard) int {
	for i, card := range cards {
		if card.Progress.Status == domain.ConceptProgressNeedsSupport {
			return i
		}
	}
	for i, card := range cards {
		if card.Progress.Status == domain.ConceptProgressLearning {
			return i
		}
	}
	return 0
}

func conceptCardIndex(cards []domain.ConceptProgressCard, conceptID string) int {
	for i, card := range cards {
		if card.Concept.ID == conceptID {
			return i
		}
	}
	return 0
}

func defaultLearningStudentIDValue(studentID string) string {
	if strings.TrimSpace(studentID) == "" {
		return defaultLearningStudentID
	}
	return strings.TrimSpace(studentID)
}

func defaultStudentIDValue(studentID string) string {
	return defaultLearningStudentIDValue(studentID)
}

func learningStudentName(studentID string) string {
	switch studentID {
	case "S-001":
		return "Alex Morgan"
	case "S-002":
		return "Jordan Lee"
	default:
		return "Learner"
	}
}

func newLearningID(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), learningIDSequence.Add(1))
}

// TeacherLearningPortal owns the small teacher-side slice needed to demonstrate
// the learning architecture: published context, policy controls, and class
// progress read models.
type TeacherLearningPortal struct {
	content  repository.LearningContentRepository
	progress repository.ConceptProgressRepository
	evidence repository.LearningEvidenceRepository
}

func NewTeacherLearningPortal(content repository.LearningContentRepository, progress repository.ConceptProgressRepository, evidence repository.LearningEvidenceRepository) TeacherLearningPortal {
	return TeacherLearningPortal{content: content, progress: progress, evidence: evidence}
}

func (p TeacherLearningPortal) Dashboard(ctx context.Context) (domain.TeacherLearningDashboard, error) {
	courses, err := p.content.ListCourses(ctx)
	if err != nil {
		return domain.TeacherLearningDashboard{}, err
	}
	if len(courses) == 0 {
		return domain.TeacherLearningDashboard{}, errors.New("no learning course is configured")
	}
	course := courses[0]
	concepts, err := p.content.ListConcepts(ctx, course.ID)
	if err != nil {
		return domain.TeacherLearningDashboard{}, err
	}
	allProgress, err := p.progress.ListAll(ctx)
	if err != nil {
		return domain.TeacherLearningDashboard{}, err
	}
	allMaterials, err := p.content.ListAllMaterials(ctx, course.ID)
	if err != nil {
		return domain.TeacherLearningDashboard{}, err
	}
	recentEvidence, err := p.evidence.ListByCourse(ctx, course.ID)
	if err != nil {
		return domain.TeacherLearningDashboard{}, err
	}
	if len(recentEvidence) > 8 {
		recentEvidence = recentEvidence[:8]
	}

	progressByConcept := make(map[string][]domain.ConceptProgress)
	students := map[string]bool{}
	for _, item := range allProgress {
		if item.CourseID != course.ID {
			continue
		}
		students[item.StudentID] = true
		progressByConcept[item.ConceptID] = append(progressByConcept[item.ConceptID], item)
	}

	cards := make([]domain.ConceptProgressCard, 0, len(concepts))
	totalConfidence := 0
	totalProgress := 0
	needsSupport := 0
	for _, concept := range concepts {
		items := progressByConcept[concept.ID]
		aggregate := newProgress("class", course.ID, concept.ID)
		if len(items) > 0 {
			confidence := 0
			evidenceCount := 0
			for _, item := range items {
				confidence += item.Confidence
				evidenceCount += item.EvidenceCount
				if item.Status == domain.ConceptProgressNeedsSupport {
					needsSupport++
				}
			}
			aggregate.Confidence = confidence / len(items)
			aggregate.EvidenceCount = evidenceCount / len(items)
			aggregate.Status = aggregateStatus(aggregate.Confidence)
			aggregate.NextAction = "Review class pattern"
		}
		totalConfidence += aggregate.Confidence
		totalProgress++
		cards = append(cards, domain.ConceptProgressCard{Concept: concept, Progress: aggregate, State: aggregate.Status.Label()})
	}

	support := make([]domain.LearnerSupportCard, 0)
	for _, item := range allProgress {
		if item.CourseID != course.ID || item.Status != domain.ConceptProgressNeedsSupport {
			continue
		}
		conceptTitle := item.ConceptID
		for _, concept := range concepts {
			if concept.ID == item.ConceptID {
				conceptTitle = concept.Title
				break
			}
		}
		support = append(support, domain.LearnerSupportCard{
			StudentID:    item.StudentID,
			StudentName:  learningStudentName(item.StudentID),
			ConceptTitle: conceptTitle,
			Status:       item.Status.Label(),
			Confidence:   item.Confidence,
			NextAction:   item.NextAction,
		})
	}
	sort.Slice(support, func(i, j int) bool { return support[i].Confidence < support[j].Confidence })
	if len(support) > 4 {
		support = support[:4]
	}

	averageConfidence := 0
	if totalProgress > 0 {
		averageConfidence = totalConfidence / totalProgress
	}
	return domain.TeacherLearningDashboard{
		Course:    course,
		Concepts:  cards,
		Materials: allMaterials,
		Policy:    course.TutorPolicy,
		Stats: domain.TeacherLearningStats{
			ActiveLearners:    len(students),
			ConceptsPublished: len(concepts),
			AverageConfidence: averageConfidence,
			NeedsSupport:      needsSupport,
		},
		SupportNeeded:  support,
		RecentEvidence: recentEvidence,
	}, nil
}

func modeAllowed(modes []domain.TutorMode, mode domain.TutorMode) bool {
	for _, allowed := range modes {
		if allowed == mode {
			return true
		}
	}
	return false
}

func aggregateStatus(confidence int) domain.ConceptProgressStatus {
	switch {
	case confidence >= 80:
		return domain.ConceptProgressSecure
	case confidence <= 35:
		return domain.ConceptProgressNeedsSupport
	default:
		return domain.ConceptProgressLearning
	}
}

func (p TeacherLearningPortal) UpdateTutorPolicy(ctx context.Context, req UpdateTutorPolicyRequest) (domain.TutorPolicy, error) {
	if strings.TrimSpace(req.CourseID) == "" {
		return domain.TutorPolicy{}, errors.New("course_id is required")
	}
	course, found, err := p.content.FindCourse(ctx, req.CourseID)
	if err != nil {
		return domain.TutorPolicy{}, err
	}
	if !found {
		return domain.TutorPolicy{}, fmt.Errorf("course %q was not found", req.CourseID)
	}
	if req.DefaultMode == "" {
		req.DefaultMode = domain.TutorModeGuided
	}
	if len(req.AllowedModes) == 0 {
		req.AllowedModes = []domain.TutorMode{domain.TutorModeGuided, domain.TutorModeSocratic, domain.TutorModeDirect, domain.TutorModePractice, domain.TutorModeHint, domain.TutorModeDiagnostic}
	}
	if !req.DefaultMode.IsValid() {
		return domain.TutorPolicy{}, fmt.Errorf("unsupported default tutor mode %q", req.DefaultMode)
	}
	if !modeAllowed(req.AllowedModes, req.DefaultMode) {
		return domain.TutorPolicy{}, fmt.Errorf("default mode %q is not in the allowed mode list", req.DefaultMode)
	}
	for _, mode := range req.AllowedModes {
		if !mode.IsValid() {
			return domain.TutorPolicy{}, fmt.Errorf("unsupported tutor mode %q", mode)
		}
	}
	if req.SourcePolicy == "" {
		req.SourcePolicy = "Teacher material is authoritative. External sources are labeled."
	}
	course.TutorPolicy = domain.TutorPolicy{
		DefaultMode:          req.DefaultMode,
		AllowedModes:         req.AllowedModes,
		AllowExternalSources: req.AllowExternalSources,
		SourcePolicy:         req.SourcePolicy,
	}
	if err := p.content.SaveCourse(ctx, course); err != nil {
		return domain.TutorPolicy{}, err
	}
	return course.TutorPolicy, nil
}

func (p TeacherLearningPortal) PublishMaterial(ctx context.Context, req PublishMaterialRequest) (domain.LearningMaterial, error) {
	required := []string{req.CourseID, req.ConceptID, req.Title, req.Body}
	for _, value := range required {
		if strings.TrimSpace(value) == "" {
			return domain.LearningMaterial{}, errors.New("course_id, concept_id, title, and body are required")
		}
	}
	if req.SourceKind == "" {
		req.SourceKind = domain.LearningSourceTeacher
	}
	if req.SourceKind != domain.LearningSourceTeacher && req.SourceKind != domain.LearningSourceExternal {
		return domain.LearningMaterial{}, errors.New("source_kind must be teacher or external")
	}
	if req.SourceKind == domain.LearningSourceExternal {
		sourceRef := strings.TrimSpace(req.SourceRef)
		if sourceRef == "" {
			return domain.LearningMaterial{}, errors.New("source_ref is required for external material")
		}
		parsed, parseErr := url.Parse(sourceRef)
		if parseErr != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return domain.LearningMaterial{}, errors.New("source_ref must be a valid http or https URL")
		}
	}
	course, found, err := p.content.FindCourse(ctx, req.CourseID)
	if err != nil {
		return domain.LearningMaterial{}, err
	}
	if !found {
		return domain.LearningMaterial{}, fmt.Errorf("course %q was not found", req.CourseID)
	}
	concept, found, err := p.content.FindConcept(ctx, req.ConceptID)
	if err != nil {
		return domain.LearningMaterial{}, err
	}
	if !found || concept.CourseID != course.ID {
		return domain.LearningMaterial{}, errors.New("concept is not part of the course")
	}
	materialID := newLearningID("material")
	material := domain.LearningMaterial{
		ID:        materialID,
		CourseID:  course.ID,
		ConceptID: concept.ID,
		Title:     strings.TrimSpace(req.Title),
		Body:      strings.TrimSpace(req.Body),
		Published: true,
		Version:   1,
		Source: domain.LearningSourceRef{
			ID:        materialID,
			Title:     strings.TrimSpace(req.Title),
			Kind:      req.SourceKind,
			Reference: strings.TrimSpace(req.SourceRef),
		},
	}
	if err := p.content.SaveMaterial(ctx, material); err != nil {
		return domain.LearningMaterial{}, err
	}
	return material, nil
}
