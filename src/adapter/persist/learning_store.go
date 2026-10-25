// Package persist holds durable adapters for the learning repositories. The
// first adapter is a file-backed snapshot store: each save writes a full,
// versioned image of the in-memory learning state using an atomic
// write-and-rename so a crash never leaves a half-written snapshot behind.
//
// This is a deliberately small first step toward real persistence: the
// repository ports do not change, and a future database adapter only has to
// satisfy the same ports (or produce/consume repository.LearningSnapshot).
package persist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/srijan-raghavula/colearning-agent/src/adapter/memory"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

// ErrUnsupportedSnapshot is returned when a stored snapshot carries a schema
// version this build cannot restore safely. The file is left untouched.
type ErrUnsupportedSnapshot struct{ Version int }

func (e ErrUnsupportedSnapshot) Error() string {
	return fmt.Sprintf("unsupported learning snapshot version %d (want %d)", e.Version, repository.LearningSnapshotVersion)
}

// ErrCorruptSnapshot reports a snapshot that could not be decoded or
// validated and was moved aside before seed fallback. QuarantinePath retains
// the original bytes for inspection or recovery.
type ErrCorruptSnapshot struct {
	Path           string
	QuarantinePath string
	Cause          error
}

func (e *ErrCorruptSnapshot) Error() string {
	return fmt.Sprintf("corrupt learning snapshot %s quarantined at %s: %v", e.Path, e.QuarantinePath, e.Cause)
}

func (e *ErrCorruptSnapshot) Unwrap() error { return e.Cause }

// LearningStore persists a rounded image of the in-memory learning
// repositories to a single JSON file.
type LearningStore struct {
	path string

	content  *memory.LearningContentRepository
	sessions *memory.LearningSessionRepository
	progress *memory.ConceptProgressRepository
	evidence *memory.LearningEvidenceRepository
	checks   *memory.UnderstandingCheckRepository

	mu sync.Mutex
}

func NewLearningStore(
	path string,
	content *memory.LearningContentRepository,
	sessions *memory.LearningSessionRepository,
	progress *memory.ConceptProgressRepository,
	evidence *memory.LearningEvidenceRepository,
	checks *memory.UnderstandingCheckRepository,
) *LearningStore {
	return &LearningStore{
		path:     path,
		content:  content,
		sessions: sessions,
		progress: progress,
		evidence: evidence,
		checks:   checks,
	}
}

func (s *LearningStore) Path() string { return s.path }

// Load restores the stored snapshot, replacing everything currently held by
// the repositories. It returns false, nil when no snapshot exists yet (first
// boot must keep seed behavior). Corrupt snapshots are quarantined before an
// ErrCorruptSnapshot is returned; unsupported versions are never moved.
func (s *LearningStore) Load(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return false, fmt.Errorf("prepare snapshot directory: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read learning snapshot: %w", err)
	}

	var header struct {
		Version *int `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return false, s.quarantineCorrupt(fmt.Errorf("decode snapshot header: %w", err))
	}
	if header.Version == nil || *header.Version <= 0 {
		return false, s.quarantineCorrupt(errors.New("snapshot version is missing or invalid"))
	}
	if *header.Version != repository.LearningSnapshotVersion {
		return false, ErrUnsupportedSnapshot{Version: *header.Version}
	}

	var snap repository.LearningSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return false, s.quarantineCorrupt(fmt.Errorf("decode version %d snapshot: %w", *header.Version, err))
	}
	if err := validateSnapshot(snap); err != nil {
		return false, s.quarantineCorrupt(fmt.Errorf("validate version %d snapshot: %w", snap.Version, err))
	}

	s.content.RestoreSnapshot(snap)
	s.sessions.RestoreSnapshot(snap)
	s.progress.RestoreSnapshot(snap)
	s.evidence.RestoreSnapshot(snap)
	s.checks.RestoreSnapshot(snap)
	return true, nil
}

// Save writes the full learning state to disk. Writers serialize on an
// internal mutex, so callers may invoke Save from concurrent workflows.
func (s *LearningStore) Save(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	snap := s.buildSnapshot()
	if err := validateSnapshot(snap); err != nil {
		return fmt.Errorf("validate learning snapshot: %w", err)
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("encode learning snapshot: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("prepare snapshot directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".learning-snapshot-*")
	if err != nil {
		return fmt.Errorf("stage learning snapshot: %w", err)
	}
	tmpName := tmp.Name()
	remove := true
	defer func() {
		if remove {
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write learning snapshot: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync learning snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close learning snapshot: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("replace learning snapshot: %w", err)
	}
	remove = false
	return nil
}

func (s *LearningStore) quarantineCorrupt(cause error) error {
	quarantinePath, err := quarantineSnapshot(s.path)
	if err != nil {
		return fmt.Errorf("learning snapshot is corrupt (%v) and could not be quarantined: %w", cause, err)
	}
	return &ErrCorruptSnapshot{
		Path:           s.path,
		QuarantinePath: quarantinePath,
		Cause:          cause,
	}
}

func quarantineSnapshot(path string) (string, error) {
	dir := filepath.Dir(path)
	placeholder, err := os.CreateTemp(dir, "."+filepath.Base(path)+".corrupt-*")
	if err != nil {
		return "", fmt.Errorf("reserve quarantine path: %w", err)
	}
	quarantinePath := placeholder.Name()
	if err := placeholder.Close(); err != nil {
		os.Remove(quarantinePath)
		return "", fmt.Errorf("close quarantine placeholder: %w", err)
	}
	if err := os.Remove(quarantinePath); err != nil {
		return "", fmt.Errorf("release quarantine placeholder: %w", err)
	}
	if err := os.Rename(path, quarantinePath); err != nil {
		return "", fmt.Errorf("quarantine learning snapshot: %w", err)
	}
	return quarantinePath, nil
}

func validateSnapshot(snap repository.LearningSnapshot) error {
	if snap.Version != repository.LearningSnapshotVersion {
		return fmt.Errorf("snapshot version %d is unsupported (want %d)", snap.Version, repository.LearningSnapshotVersion)
	}
	if len(snap.Courses) == 0 {
		return errors.New("snapshot contains no courses")
	}
	if len(snap.Concepts) == 0 {
		return errors.New("snapshot contains no concepts")
	}

	courseIDs := make(map[string]struct{}, len(snap.Courses))
	for i, course := range snap.Courses {
		if err := addUniqueSnapshotID("course", i, course.ID, courseIDs); err != nil {
			return err
		}
	}

	conceptCourses := make(map[string]string, len(snap.Concepts))
	conceptIDs := make(map[string]struct{}, len(snap.Concepts))
	for i, concept := range snap.Concepts {
		if err := addUniqueSnapshotID("concept", i, concept.ID, conceptIDs); err != nil {
			return err
		}
		if strings.TrimSpace(concept.CourseID) == "" {
			return fmt.Errorf("concept %q has empty course id", concept.ID)
		}
		if _, ok := courseIDs[concept.CourseID]; !ok {
			return fmt.Errorf("concept %q references missing course %q", concept.ID, concept.CourseID)
		}
		conceptCourses[concept.ID] = concept.CourseID
	}

	for _, course := range snap.Courses {
		if course.NextConceptID == "" {
			continue
		}
		conceptCourse, ok := conceptCourses[course.NextConceptID]
		if !ok {
			return fmt.Errorf("course %q references missing next concept %q", course.ID, course.NextConceptID)
		}
		if conceptCourse != course.ID {
			return fmt.Errorf("course %q references next concept %q from course %q", course.ID, course.NextConceptID, conceptCourse)
		}
	}
	for _, concept := range snap.Concepts {
		prerequisites := make(map[string]struct{}, len(concept.PrerequisiteIDs))
		for _, prerequisiteID := range concept.PrerequisiteIDs {
			if strings.TrimSpace(prerequisiteID) == "" {
				return fmt.Errorf("concept %q has an empty prerequisite id", concept.ID)
			}
			if prerequisiteID == concept.ID {
				return fmt.Errorf("concept %q cannot require itself", concept.ID)
			}
			if _, duplicate := prerequisites[prerequisiteID]; duplicate {
				return fmt.Errorf("concept %q repeats prerequisite %q", concept.ID, prerequisiteID)
			}
			prerequisites[prerequisiteID] = struct{}{}
			prerequisiteCourse, ok := conceptCourses[prerequisiteID]
			if !ok {
				return fmt.Errorf("concept %q references missing prerequisite %q", concept.ID, prerequisiteID)
			}
			if prerequisiteCourse != concept.CourseID {
				return fmt.Errorf("concept %q references prerequisite %q from course %q", concept.ID, prerequisiteID, prerequisiteCourse)
			}
		}
	}

	objectiveIDs := make(map[string]struct{}, len(snap.Objectives))
	for i, objective := range snap.Objectives {
		if err := addUniqueSnapshotID("objective", i, objective.ID, objectiveIDs); err != nil {
			return err
		}
		if err := validateCourseConceptReference("objective", objective.ID, objective.CourseID, objective.ConceptID, courseIDs, conceptCourses); err != nil {
			return err
		}
	}

	materialIDs := make(map[string]struct{}, len(snap.Materials))
	for i, material := range snap.Materials {
		if err := addUniqueSnapshotID("material", i, material.ID, materialIDs); err != nil {
			return err
		}
		if err := validateCourseConceptReference("material", material.ID, material.CourseID, material.ConceptID, courseIDs, conceptCourses); err != nil {
			return err
		}
	}

	type sessionReference = struct {
		studentID string
		courseID  string
		conceptID string
	}
	sessions := make(map[string]sessionReference, len(snap.Sessions))
	sessionIDs := make(map[string]struct{}, len(snap.Sessions))
	for i, session := range snap.Sessions {
		if err := addUniqueSnapshotID("session", i, session.ID, sessionIDs); err != nil {
			return err
		}
		if strings.TrimSpace(session.StudentID) == "" {
			return fmt.Errorf("session %q has empty student id", session.ID)
		}
		if err := validateCourseConceptReference("session", session.ID, session.CourseID, session.ConceptID, courseIDs, conceptCourses); err != nil {
			return err
		}
		sessions[session.ID] = sessionReference{studentID: session.StudentID, courseID: session.CourseID, conceptID: session.ConceptID}
	}

	turnSessions := make(map[string]string, len(snap.Turns))
	turnIDs := make(map[string]struct{}, len(snap.Turns))
	for i, turn := range snap.Turns {
		if err := addUniqueSnapshotID("turn", i, turn.ID, turnIDs); err != nil {
			return err
		}
		if _, ok := sessions[turn.SessionID]; !ok {
			return fmt.Errorf("turn %q references missing session %q", turn.ID, turn.SessionID)
		}
		turnSessions[turn.ID] = turn.SessionID
	}

	type progressIdentity struct {
		studentID string
		courseID  string
		conceptID string
	}
	progressKeys := make(map[progressIdentity]struct{}, len(snap.Progress))
	for i, progress := range snap.Progress {
		if strings.TrimSpace(progress.StudentID) == "" {
			return fmt.Errorf("progress[%d] has empty student id", i)
		}
		if err := validateCourseConceptReference("progress", fmt.Sprintf("%d", i), progress.CourseID, progress.ConceptID, courseIDs, conceptCourses); err != nil {
			return err
		}
		key := progressIdentity{studentID: progress.StudentID, courseID: progress.CourseID, conceptID: progress.ConceptID}
		if _, duplicate := progressKeys[key]; duplicate {
			return fmt.Errorf("duplicate progress for student %q, course %q, concept %q", progress.StudentID, progress.CourseID, progress.ConceptID)
		}
		progressKeys[key] = struct{}{}
	}

	evidenceIDs := make(map[string]struct{}, len(snap.Evidence))
	for i, evidence := range snap.Evidence {
		if err := addUniqueSnapshotID("evidence", i, evidence.ID, evidenceIDs); err != nil {
			return err
		}
		if err := validateSessionBoundRecord("evidence", evidence.ID, evidence.SessionID, evidence.TurnID, evidence.StudentID, evidence.CourseID, evidence.ConceptID, false, courseIDs, conceptCourses, sessions, turnSessions); err != nil {
			return err
		}
	}

	checkIDs := make(map[string]struct{}, len(snap.Checks))
	for i, check := range snap.Checks {
		if err := addUniqueSnapshotID("understanding check", i, check.ID, checkIDs); err != nil {
			return err
		}
		if err := validateSessionBoundRecord("understanding check", check.ID, check.SessionID, check.TurnID, check.StudentID, check.CourseID, check.ConceptID, true, courseIDs, conceptCourses, sessions, turnSessions); err != nil {
			return err
		}
	}

	return nil
}

func addUniqueSnapshotID(kind string, index int, id string, seen map[string]struct{}) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s[%d] has empty id", kind, index)
	}
	if _, duplicate := seen[id]; duplicate {
		return fmt.Errorf("duplicate %s id %q", kind, id)
	}
	seen[id] = struct{}{}
	return nil
}

func validateCourseConceptReference(kind, id, courseID, conceptID string, courseIDs map[string]struct{}, conceptCourses map[string]string) error {
	if strings.TrimSpace(courseID) == "" {
		return fmt.Errorf("%s %q has empty course id", kind, id)
	}
	if _, ok := courseIDs[courseID]; !ok {
		return fmt.Errorf("%s %q references missing course %q", kind, id, courseID)
	}
	if strings.TrimSpace(conceptID) == "" {
		return fmt.Errorf("%s %q has empty concept id", kind, id)
	}
	conceptCourse, ok := conceptCourses[conceptID]
	if !ok {
		return fmt.Errorf("%s %q references missing concept %q", kind, id, conceptID)
	}
	if conceptCourse != courseID {
		return fmt.Errorf("%s %q references concept %q from course %q, not %q", kind, id, conceptID, conceptCourse, courseID)
	}
	return nil
}

func validateSessionBoundRecord(
	kind, id, sessionID, turnID, studentID, courseID, conceptID string,
	requireTurn bool,
	courseIDs map[string]struct{},
	conceptCourses map[string]string,
	sessions map[string]struct {
		studentID string
		courseID  string
		conceptID string
	},
	turnSessions map[string]string,
) error {
	if strings.TrimSpace(studentID) == "" {
		return fmt.Errorf("%s %q has empty student id", kind, id)
	}
	if err := validateCourseConceptReference(kind, id, courseID, conceptID, courseIDs, conceptCourses); err != nil {
		return err
	}
	session, ok := sessions[sessionID]
	if !ok {
		return fmt.Errorf("%s %q references missing session %q", kind, id, sessionID)
	}
	if session.studentID != studentID || session.courseID != courseID || session.conceptID != conceptID {
		return fmt.Errorf("%s %q does not match student/course/concept of session %q", kind, id, sessionID)
	}

	// Origin/main emitted version-1 evidence whose optional turn reference could
	// be empty or point at a turn omitted from that snapshot. Preserve those
	// records after validating their session identity. Checks, by contrast,
	// always need a concrete turn because their idempotency is turn-based.
	if strings.TrimSpace(turnID) == "" {
		if requireTurn {
			return fmt.Errorf("%s %q has empty turn id", kind, id)
		}
		return nil
	}
	turnSessionID, ok := turnSessions[turnID]
	if !ok {
		if requireTurn {
			return fmt.Errorf("%s %q references missing turn %q", kind, id, turnID)
		}
		return nil
	}
	if turnSessionID != sessionID {
		return fmt.Errorf("%s %q references turn %q from session %q", kind, id, turnID, turnSessionID)
	}
	return nil
}

func (s *LearningStore) buildSnapshot() repository.LearningSnapshot {
	snap := repository.LearningSnapshot{Version: repository.LearningSnapshotVersion}
	content := s.content.ExportSnapshot()
	snap.Courses, snap.Concepts = content.Courses, content.Concepts
	snap.Objectives, snap.Materials = content.Objectives, content.Materials
	sessions := s.sessions.ExportSnapshot()
	snap.Sessions, snap.Turns = sessions.Sessions, sessions.Turns
	snap.Progress = s.progress.ExportSnapshot().Progress
	snap.Evidence = s.evidence.ExportSnapshot().Evidence
	snap.Checks = s.checks.ExportSnapshot().Checks
	return snap
}
