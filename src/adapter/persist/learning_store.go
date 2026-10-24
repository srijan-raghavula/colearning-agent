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
	"sync"

	"github.com/srijan-raghavula/colearning-agent/src/adapter/memory"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

// ErrUnsupportedSnapshot is returned when a stored snapshot carries a schema
// version this build cannot restore safely.
type ErrUnsupportedSnapshot struct{ Version int }

func (e ErrUnsupportedSnapshot) Error() string {
	return fmt.Sprintf("unsupported learning snapshot version %d (want %d)", e.Version, repository.LearningSnapshotVersion)
}

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
// boot must keep seed behavior).
func (s *LearningStore) Load(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return false, fmt.Errorf("prepare snapshot directory: %w", err)
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read learning snapshot: %w", err)
	}
	var snap repository.LearningSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return false, fmt.Errorf("decode learning snapshot: %w", err)
	}
	if snap.Version != repository.LearningSnapshotVersion {
		return false, ErrUnsupportedSnapshot{Version: snap.Version}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
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
