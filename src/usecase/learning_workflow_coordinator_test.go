package usecase

import (
	"sync"
	"testing"
	"time"
)

func TestLearningWorkflowCoordinatorSerializesSameKeyAndCleansUp(t *testing.T) {
	coordinator := newLearningWorkflowCoordinator()
	unlockFirst := coordinator.lock("same-key")
	var releaseFirst sync.Once
	release := func() {
		releaseFirst.Do(unlockFirst)
	}
	defer release()

	acquired := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		unlock := coordinator.lock("same-key")
		close(acquired)
		unlock()
		close(finished)
	}()

	deadline := time.Now().Add(time.Second)
	for {
		coordinator.mu.Lock()
		users := coordinator.locks["same-key"].users
		coordinator.mu.Unlock()
		if users == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("second user did not register for the same key")
		}
		time.Sleep(time.Millisecond)
	}

	select {
	case <-acquired:
		t.Fatal("same-key workflow acquired its lock concurrently")
	default:
	}

	release()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second same-key workflow did not acquire the lock after release")
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("second same-key workflow did not finish")
	}

	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()
	if len(coordinator.locks) != 0 {
		t.Fatalf("expected lock registry to be empty after release, got %d entries", len(coordinator.locks))
	}
}

func TestLearningWorkflowCoordinatorDoesNotSerializeDifferentKeys(t *testing.T) {
	coordinator := newLearningWorkflowCoordinator()
	unlockFirst := coordinator.lock("first")
	defer unlockFirst()

	acquired := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		unlock := coordinator.lock("second")
		close(acquired)
		unlock()
		close(finished)
	}()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("different-key workflow was blocked by an unrelated lock")
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("different-key workflow did not finish")
	}
}
