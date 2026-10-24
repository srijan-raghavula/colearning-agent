package memory

import (
	"sort"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

// The learning snapshot seam adds export and wholesale-restore support to the
// in-memory learning repositories. Read/write method signatures stay defined
// by the repository ports; these helpers only feed the durable snapshot
// adapter so the same repositories can later be fronted by a real database.

func (r *LearningContentRepository) ExportSnapshot() repository.LearningSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return repository.LearningSnapshot{
		Courses:    snapshotList(r.courses),
		Concepts:   snapshotList(r.concepts),
		Objectives: snapshotList(r.objectives),
		Materials:  snapshotList(r.materials),
	}
}

func (r *LearningContentRepository) RestoreSnapshot(snap repository.LearningSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.courses = make(map[string]domain.LearningCourse, len(snap.Courses))
	r.concepts = make(map[string]domain.LearningConcept, len(snap.Concepts))
	r.objectives = make(map[string]domain.LearningObjective, len(snap.Objectives))
	r.materials = make(map[string]domain.LearningMaterial, len(snap.Materials))
	for _, course := range snap.Courses {
		r.courses[course.ID] = course
	}
	for _, concept := range snap.Concepts {
		r.concepts[concept.ID] = concept
	}
	for _, objective := range snap.Objectives {
		r.objectives[objective.ID] = objective
	}
	for _, material := range snap.Materials {
		r.materials[material.ID] = material
	}
}

func (r *LearningSessionRepository) ExportSnapshot() repository.LearningSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	turns := make([]domain.LearningTurn, 0)
	for sessionID := range r.turns {
		turns = append(turns, r.turns[sessionID]...)
	}
	sort.Slice(turns, func(i, j int) bool { return turns[i].CreatedAt.Before(turns[j].CreatedAt) })
	return repository.LearningSnapshot{Sessions: snapshotList(r.sessions), Turns: turns}
}

func (r *LearningSessionRepository) RestoreSnapshot(snap repository.LearningSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions = make(map[string]domain.LearningSession, len(snap.Sessions))
	r.turns = make(map[string][]domain.LearningTurn, len(snap.Sessions))
	for _, session := range snap.Sessions {
		r.sessions[session.ID] = session
	}
	for _, turn := range snap.Turns {
		if _, ok := r.sessions[turn.SessionID]; !ok {
			continue
		}
		r.turns[turn.SessionID] = append(r.turns[turn.SessionID], turn)
	}
	for sessionID := range r.turns {
		sort.Slice(r.turns[sessionID], func(i, j int) bool {
			return r.turns[sessionID][i].CreatedAt.Before(r.turns[sessionID][j].CreatedAt)
		})
	}
}

func (r *ConceptProgressRepository) ExportSnapshot() repository.LearningSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return repository.LearningSnapshot{Progress: snapshotList(r.items)}
}

func (r *ConceptProgressRepository) RestoreSnapshot(snap repository.LearningSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]domain.ConceptProgress, len(snap.Progress))
	for _, progress := range snap.Progress {
		r.items[progressKey(progress.StudentID, progress.CourseID, progress.ConceptID)] = progress
	}
}

func (r *LearningEvidenceRepository) ExportSnapshot() repository.LearningSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return repository.LearningSnapshot{Evidence: snapshotList(r.items)}
}

func (r *LearningEvidenceRepository) RestoreSnapshot(snap repository.LearningSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]domain.LearningEvidence, len(snap.Evidence))
	for _, evidence := range snap.Evidence {
		r.items[evidence.ID] = evidence
	}
}

func (r *UnderstandingCheckRepository) ExportSnapshot() repository.LearningSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return repository.LearningSnapshot{Checks: snapshotList(r.items)}
}

func (r *UnderstandingCheckRepository) RestoreSnapshot(snap repository.LearningSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]domain.UnderstandingCheck, len(snap.Checks))
	for _, check := range snap.Checks {
		r.items[check.ID] = check
	}
}

func snapshotList[T any](source map[string]T) []T {
	result := make([]T, 0, len(source))
	for _, item := range source {
		result = append(result, item)
	}
	return result
}
