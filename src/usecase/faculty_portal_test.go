package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/adapter/memory"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

func TestFacultyDashboardAggregatesMissingAndMastery(t *testing.T) {
	now := time.Now().UTC()
	submissions := []domain.Submission{
		{ID: "1", StudentID: "S1", SubjectID: "math", TopicID: "algebra", DueAt: now.Add(-time.Hour), SubmittedAt: now.Add(-2 * time.Hour), Status: domain.SubmissionStatusLate},
		{ID: "2", StudentID: "S2", SubjectID: "math", TopicID: "geometry", DueAt: now.Add(-time.Hour), Status: domain.SubmissionStatusMissing},
	}
	evaluations := []domain.Evaluation{
		{SubmissionID: "1", MasteryPercentage: 50, TopicScores: []domain.TopicScore{{Topic: "algebra", Mastery: 45}}},
	}

	portal := NewFacultyPortal(memory.NewSubmissionRepository(submissions), memory.NewEvaluationRepository(evaluations))
	dashboard, err := portal.Dashboard(context.Background())
	if err != nil {
		t.Fatalf("Dashboard() error = %v", err)
	}

	if len(dashboard.Summaries) != 1 {
		t.Fatalf("expected 1 subject summary, got %d", len(dashboard.Summaries))
	}
	summary := dashboard.Summaries[0]
	if summary.MissingCount != 1 {
		t.Fatalf("expected 1 missing, got %d", summary.MissingCount)
	}
	if len(dashboard.MissingSubmissions) != 1 {
		t.Fatalf("expected one missing submission in alerts")
	}
}
