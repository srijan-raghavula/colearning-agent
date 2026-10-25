package persist

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/adapter/memory"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

func newTestRepositories() (*memory.LearningContentRepository, *memory.LearningSessionRepository, *memory.ConceptProgressRepository, *memory.LearningEvidenceRepository, *memory.UnderstandingCheckRepository) {
	content := memory.NewLearningContentRepository(
		[]domain.LearningCourse{{ID: "course-1", Title: "Algebra foundations", TutorPolicy: domain.TutorPolicy{DefaultMode: domain.TutorModeGuided, AllowedModes: []domain.TutorMode{domain.TutorModeGuided}}}},
		[]domain.LearningConcept{{ID: "linear-equations", CourseID: "course-1", Title: "Linear equations", Order: 1}},
		[]domain.LearningObjective{{ID: "objective-1", CourseID: "course-1", ConceptID: "linear-equations", Description: "Solve a linear equation"}},
		[]domain.LearningMaterial{{ID: "material-1", CourseID: "course-1", ConceptID: "linear-equations", Title: "Teacher note", Published: true, Version: 1}},
	)
	now := time.Now().UTC().Truncate(time.Second)
	sessions := memory.NewLearningSessionRepository(
		[]domain.LearningSession{{ID: "session-1", StudentID: "S-001", CourseID: "course-1", ConceptID: "linear-equations", Mode: domain.TutorModeGuided, Status: domain.LearningSessionActive, StartedAt: now, LastActiveAt: now}},
		[]domain.LearningTurn{
			{ID: "turn-1", SessionID: "session-1", Role: domain.LearningTurnTutor, Kind: "explanation", Mode: domain.TutorModeGuided, Body: "Balance both sides.", CreatedAt: now.Add(-time.Minute)},
			{ID: "turn-2", SessionID: "session-1", Role: domain.LearningTurnStudent, Kind: "student_message", Mode: domain.TutorModeGuided, Body: "I'll try.", CreatedAt: now},
		},
	)
	progress := memory.NewConceptProgressRepository([]domain.ConceptProgress{{StudentID: "S-001", CourseID: "course-1", ConceptID: "linear-equations", Status: domain.ConceptProgressLearning, Confidence: 62, EvidenceCount: 3}})
	evidence := memory.NewLearningEvidenceRepository([]domain.LearningEvidence{{ID: "evidence-1", SessionID: "session-1", StudentID: "S-001", CourseID: "course-1", ConceptID: "linear-equations", Kind: "explanation", Note: "Balance acknowledged", ObservedAt: now}})
	checks := memory.NewUnderstandingCheckRepository([]domain.UnderstandingCheck{{ID: "check-1", SessionID: "session-1", TurnID: "turn-2", StudentID: "S-001", CourseID: "course-1", ConceptID: "linear-equations", Summary: "Explain the check", CreatedAt: now}})
	return content, sessions, progress, evidence, checks
}

func TestLearningStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "learning-state.json")

	sourceContent, sourceSessions, sourceProgress, sourceEvidence, sourceChecks := newTestRepositories()
	store := NewLearningStore(path, sourceContent, sourceSessions, sourceProgress, sourceEvidence, sourceChecks)

	loaded, err := store.Load(context.Background())
	if loaded {
		t.Fatalf("first load should report no snapshot, got loaded=true")
	}
	if err != nil {
		t.Fatalf("first load returned error: %v", err)
	}

	if err := store.Save(context.Background()); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	freshContent, freshSessions, freshProgress, freshEvidence, freshChecks := newTestRepositories()
	restored := NewLearningStore(path, freshContent, freshSessions, freshProgress, freshEvidence, freshChecks)

	loaded, err = restored.Load(context.Background())
	if err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	if !loaded {
		t.Fatalf("restore should report a snapshot was loaded")
	}

	ctx := context.Background()
	course, ok, err := freshContent.FindCourse(ctx, "course-1")
	if err != nil || !ok || course.Title != "Algebra foundations" {
		t.Fatalf("course not restored (ok=%v err=%v): %+v", ok, err, course)
	}
	if _, ok, err := freshContent.FindMaterial(ctx, "material-1"); err != nil || !ok {
		t.Fatalf("material not restored: ok=%v err=%v", ok, err)
	}

	turns, err := freshSessions.ListTurns(ctx, "session-1")
	if err != nil {
		t.Fatalf("list restored turns: %v", err)
	}
	if len(turns) != 2 || turns[0].ID != "turn-1" || turns[1].ID != "turn-2" {
		t.Fatalf("turns not restored in order: %+v", turns)
	}
	session, ok, err := freshSessions.FindByID(ctx, "session-1")
	if err != nil || !ok || session.ConceptID != "linear-equations" {
		t.Fatalf("session not restored (ok=%v err=%v): %+v", ok, err, session)
	}

	progress, ok, err := freshProgress.Find(ctx, "S-001", "course-1", "linear-equations")
	if err != nil || !ok || progress.Confidence != 62 {
		t.Fatalf("progress not restored (ok=%v err=%v): %+v", ok, err, progress)
	}
	evidence, err := freshEvidence.ListByCourse(ctx, "course-1")
	if err != nil || len(evidence) != 1 || evidence[0].ID != "evidence-1" {
		t.Fatalf("evidence not restored: err=%v evidence=%+v", err, evidence)
	}
	check, ok, err := freshChecks.FindLatestBySession(ctx, "session-1")
	if err != nil || !ok || check.Summary != "Explain the check" {
		t.Fatalf("understanding check not restored (ok=%v err=%v): %+v", ok, err, check)
	}
}

func TestLearningStorePreservesUnsupportedSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning-state.json")
	original := []byte(`{"version":999,"courses":[]}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("write unsupported snapshot: %v", err)
	}

	content, sessions, progress, evidence, checks := newTestRepositories()
	store := NewLearningStore(path, content, sessions, progress, evidence, checks)
	loaded, err := store.Load(context.Background())
	var unsupported ErrUnsupportedSnapshot
	if loaded || !errors.As(err, &unsupported) || unsupported.Version != 999 {
		t.Fatalf("expected unsupported-version error, loaded=%v err=%v", loaded, err)
	}
	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read unsupported snapshot: %v", err)
	}
	if !bytes.Equal(persisted, original) {
		t.Fatalf("unsupported snapshot was modified: %s", persisted)
	}
}

func TestLearningStoreQuarantinesCorruptSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning-state.json")
	original := []byte(fmt.Sprintf(`{"version":%d,"courses":[]}`, repository.LearningSnapshotVersion))
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("write corrupt snapshot: %v", err)
	}

	content, sessions, progress, evidence, checks := newTestRepositories()
	store := NewLearningStore(path, content, sessions, progress, evidence, checks)
	loaded, err := store.Load(context.Background())
	var corrupt *ErrCorruptSnapshot
	if loaded || !errors.As(err, &corrupt) {
		t.Fatalf("expected corrupt-snapshot error, loaded=%v err=%v", loaded, err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("corrupt snapshot should be moved from the active path, stat err=%v", err)
	}
	quarantined, err := os.ReadFile(corrupt.QuarantinePath)
	if err != nil {
		t.Fatalf("read quarantined snapshot: %v", err)
	}
	if !bytes.Equal(quarantined, original) {
		t.Fatalf("quarantine did not preserve original bytes: %s", quarantined)
	}
}

func TestLearningStoreRestoreReplacesSeedData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning-state.json")

	content, sessions, progress, evidence, checks := newTestRepositories()
	store := NewLearningStore(path, content, sessions, progress, evidence, checks)
	if err := store.Save(context.Background()); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Restore into repositories that already hold the same seed records: the
	// restore must replace state wholesale instead of duplicating turns.
	if _, err := store.Load(context.Background()); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	turns, err := sessions.ListTurns(context.Background(), "session-1")
	if err != nil {
		t.Fatalf("list turns: %v", err)
	}
	if len(turns) != 2 {
		t.Fatalf("expected 2 turns after re-restore, got %d", len(turns))
	}
}

func TestLearningStoreSaveAfterMutationKeepsChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "learning-state.json")

	content, sessions, progress, evidence, checks := newTestRepositories()
	store := NewLearningStore(path, content, sessions, progress, evidence, checks)

	ctx := context.Background()
	now := time.Now().UTC()
	if err := sessions.AppendTurn(ctx, domain.LearningTurn{ID: "turn-3", SessionID: "session-1", Role: domain.LearningTurnStudent, Kind: "student_message", Mode: domain.TutorModeGuided, Body: "Checked by substitution.", CreatedAt: now}); err != nil {
		t.Fatalf("append turn: %v", err)
	}
	if err := store.Save(ctx); err != nil {
		t.Fatalf("save: %v", err)
	}

	freshCheckpoint := memory.NewLearningSessionRepository(nil, nil)
	freshStore := NewLearningStore(path, content, freshCheckpoint, progress, evidence, checks)
	if _, err := freshStore.Load(ctx); err != nil {
		t.Fatalf("restore: %v", err)
	}
	turns, err := freshCheckpoint.ListTurns(ctx, "session-1")
	if err != nil {
		t.Fatalf("list turns: %v", err)
	}
	if len(turns) != 3 || turns[2].Body != "Checked by substitution." {
		t.Fatalf("mutation not persisted: %+v", turns)
	}
	if _, ok, _ := freshCheckpoint.FindByID(ctx, "session-1"); !ok {
		// Sessions saved separately through the same snapshot must also be present.
		t.Fatalf("sessions missing on live-store reload")
	}
}
