package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

type LearningContentRepository struct {
	mu         sync.RWMutex
	courses    map[string]domain.LearningCourse
	concepts   map[string]domain.LearningConcept
	objectives map[string]domain.LearningObjective
	materials  map[string]domain.LearningMaterial
}

func NewLearningContentRepository(
	courses []domain.LearningCourse,
	concepts []domain.LearningConcept,
	objectives []domain.LearningObjective,
	materials []domain.LearningMaterial,
) *LearningContentRepository {
	repo := &LearningContentRepository{
		courses:    make(map[string]domain.LearningCourse, len(courses)),
		concepts:   make(map[string]domain.LearningConcept, len(concepts)),
		objectives: make(map[string]domain.LearningObjective, len(objectives)),
		materials:  make(map[string]domain.LearningMaterial, len(materials)),
	}
	for _, course := range courses {
		repo.courses[course.ID] = course
	}
	for _, concept := range concepts {
		repo.concepts[concept.ID] = concept
	}
	for _, objective := range objectives {
		repo.objectives[objective.ID] = objective
	}
	for _, material := range materials {
		repo.materials[material.ID] = material
	}
	return repo
}

func (r *LearningContentRepository) FindCourse(_ context.Context, id string) (domain.LearningCourse, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	course, ok := r.courses[id]
	return course, ok, nil
}

func (r *LearningContentRepository) ListCourses(_ context.Context) ([]domain.LearningCourse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.LearningCourse, 0, len(r.courses))
	for _, course := range r.courses {
		result = append(result, course)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Title < result[j].Title })
	return result, nil
}

func (r *LearningContentRepository) SaveCourse(_ context.Context, course domain.LearningCourse) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.courses[course.ID] = course
	return nil
}

func (r *LearningContentRepository) FindConcept(_ context.Context, id string) (domain.LearningConcept, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	concept, ok := r.concepts[id]
	return concept, ok, nil
}

func (r *LearningContentRepository) ListConcepts(_ context.Context, courseID string) ([]domain.LearningConcept, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.LearningConcept, 0, len(r.concepts))
	for _, concept := range r.concepts {
		if concept.CourseID == courseID {
			result = append(result, concept)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Order < result[j].Order })
	return result, nil
}

func (r *LearningContentRepository) SaveConcept(_ context.Context, concept domain.LearningConcept) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.concepts[concept.ID] = concept
	return nil
}

func (r *LearningContentRepository) ListObjectives(_ context.Context, courseID, conceptID string) ([]domain.LearningObjective, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.LearningObjective, 0)
	for _, objective := range r.objectives {
		if objective.CourseID != courseID {
			continue
		}
		if conceptID != "" && objective.ConceptID != conceptID {
			continue
		}
		result = append(result, objective)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r *LearningContentRepository) SaveObjective(_ context.Context, objective domain.LearningObjective) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.objectives[objective.ID] = objective
	return nil
}

func (r *LearningContentRepository) FindMaterial(_ context.Context, id string) (domain.LearningMaterial, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	material, ok := r.materials[id]
	return material, ok, nil
}

func (r *LearningContentRepository) ListMaterials(_ context.Context, courseID, conceptID string) ([]domain.LearningMaterial, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.LearningMaterial, 0)
	for _, material := range r.materials {
		if material.CourseID != courseID {
			continue
		}
		if conceptID != "" && material.ConceptID != conceptID {
			continue
		}
		result = append(result, material)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Title < result[j].Title })
	return result, nil
}

func (r *LearningContentRepository) ListAllMaterials(ctx context.Context, courseID string) ([]domain.LearningMaterial, error) {
	return r.ListMaterials(ctx, courseID, "")
}

func (r *LearningContentRepository) SaveMaterial(_ context.Context, material domain.LearningMaterial) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.materials[material.ID] = material
	return nil
}

type LearningSessionRepository struct {
	mu       sync.RWMutex
	sessions map[string]domain.LearningSession
	turns    map[string][]domain.LearningTurn
}

func NewLearningSessionRepository(seedSessions []domain.LearningSession, seedTurns []domain.LearningTurn) *LearningSessionRepository {
	repo := &LearningSessionRepository{
		sessions: make(map[string]domain.LearningSession, len(seedSessions)),
		turns:    make(map[string][]domain.LearningTurn),
	}
	for _, session := range seedSessions {
		repo.sessions[session.ID] = session
	}
	for _, turn := range seedTurns {
		repo.turns[turn.SessionID] = append(repo.turns[turn.SessionID], turn)
	}
	for sessionID := range repo.turns {
		sort.Slice(repo.turns[sessionID], func(i, j int) bool {
			return repo.turns[sessionID][i].CreatedAt.Before(repo.turns[sessionID][j].CreatedAt)
		})
	}
	return repo
}

func (r *LearningSessionRepository) FindByID(_ context.Context, id string) (domain.LearningSession, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[id]
	return session, ok, nil
}

func (r *LearningSessionRepository) FindActive(_ context.Context, studentID, courseID string) (domain.LearningSession, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var active domain.LearningSession
	found := false
	for _, session := range r.sessions {
		if session.StudentID != studentID || session.CourseID != courseID || session.Status != domain.LearningSessionActive {
			continue
		}
		if !found || session.LastActiveAt.After(active.LastActiveAt) {
			active = session
			found = true
		}
	}
	return active, found, nil
}

func (r *LearningSessionRepository) Save(_ context.Context, session domain.LearningSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = session
	return nil
}

func (r *LearningSessionRepository) AppendTurn(_ context.Context, turn domain.LearningTurn) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessions[turn.SessionID]; !ok {
		return errLearningSessionNotFound
	}
	r.turns[turn.SessionID] = append(r.turns[turn.SessionID], turn)
	return nil
}

func (r *LearningSessionRepository) ListTurns(_ context.Context, sessionID string) ([]domain.LearningTurn, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := append([]domain.LearningTurn(nil), r.turns[sessionID]...)
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.Before(result[j].CreatedAt) })
	return result, nil
}

type ConceptProgressRepository struct {
	mu    sync.RWMutex
	items map[string]domain.ConceptProgress
}

func progressKey(studentID, courseID, conceptID string) string {
	return studentID + "\x00" + courseID + "\x00" + conceptID
}

func NewConceptProgressRepository(seed []domain.ConceptProgress) *ConceptProgressRepository {
	repo := &ConceptProgressRepository{items: make(map[string]domain.ConceptProgress, len(seed))}
	for _, progress := range seed {
		repo.items[progressKey(progress.StudentID, progress.CourseID, progress.ConceptID)] = progress
	}
	return repo
}

func (r *ConceptProgressRepository) Find(_ context.Context, studentID, courseID, conceptID string) (domain.ConceptProgress, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	progress, ok := r.items[progressKey(studentID, courseID, conceptID)]
	return progress, ok, nil
}

func (r *ConceptProgressRepository) ListByStudent(_ context.Context, studentID, courseID string) ([]domain.ConceptProgress, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.ConceptProgress, 0)
	for _, progress := range r.items {
		if progress.StudentID == studentID && progress.CourseID == courseID {
			result = append(result, progress)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ConceptID < result[j].ConceptID })
	return result, nil
}

func (r *ConceptProgressRepository) ListAll(_ context.Context) ([]domain.ConceptProgress, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.ConceptProgress, 0, len(r.items))
	for _, progress := range r.items {
		result = append(result, progress)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].CourseID != result[j].CourseID {
			return result[i].CourseID < result[j].CourseID
		}
		if result[i].StudentID != result[j].StudentID {
			return result[i].StudentID < result[j].StudentID
		}
		return result[i].ConceptID < result[j].ConceptID
	})
	return result, nil
}

func (r *ConceptProgressRepository) Save(_ context.Context, progress domain.ConceptProgress) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[progressKey(progress.StudentID, progress.CourseID, progress.ConceptID)] = progress
	return nil
}

type LearningEvidenceRepository struct {
	mu    sync.RWMutex
	items map[string]domain.LearningEvidence
}

func NewLearningEvidenceRepository(seed []domain.LearningEvidence) *LearningEvidenceRepository {
	repo := &LearningEvidenceRepository{items: make(map[string]domain.LearningEvidence, len(seed))}
	for _, evidence := range seed {
		repo.items[evidence.ID] = evidence
	}
	return repo
}

func (r *LearningEvidenceRepository) Save(_ context.Context, evidence domain.LearningEvidence) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[evidence.ID] = evidence
	return nil
}

func (r *LearningEvidenceRepository) ListByCourse(_ context.Context, courseID string) ([]domain.LearningEvidence, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.LearningEvidence, 0)
	for _, evidence := range r.items {
		if evidence.CourseID == courseID {
			result = append(result, evidence)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ObservedAt.After(result[j].ObservedAt) })
	return result, nil
}

type UnderstandingCheckRepository struct {
	mu    sync.RWMutex
	items map[string]domain.UnderstandingCheck
}

func NewUnderstandingCheckRepository(seed []domain.UnderstandingCheck) *UnderstandingCheckRepository {
	repo := &UnderstandingCheckRepository{items: make(map[string]domain.UnderstandingCheck, len(seed))}
	for _, check := range seed {
		repo.items[check.ID] = check
	}
	return repo
}

func (r *UnderstandingCheckRepository) Save(_ context.Context, check domain.UnderstandingCheck) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[check.ID] = check
	return nil
}

func (r *UnderstandingCheckRepository) FindLatestBySession(_ context.Context, sessionID string) (domain.UnderstandingCheck, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var latest domain.UnderstandingCheck
	found := false
	for _, check := range r.items {
		if check.SessionID != sessionID {
			continue
		}
		if !found || check.CreatedAt.After(latest.CreatedAt) {
			latest = check
			found = true
		}
	}
	return latest, found, nil
}

func (r *UnderstandingCheckRepository) ListByCourse(_ context.Context, courseID string) ([]domain.UnderstandingCheck, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.UnderstandingCheck, 0)
	for _, check := range r.items {
		if check.CourseID == courseID {
			result = append(result, check)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

var errLearningSessionNotFound = learningSessionNotFoundError{}

type learningSessionNotFoundError struct{}

func (learningSessionNotFoundError) Error() string { return "learning session not found" }
