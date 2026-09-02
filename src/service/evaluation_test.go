package service

import (
	"testing"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

func TestComputeMasteryPercentage(t *testing.T) {
	engine := NewDefaultEvaluationEngine()

	tests := []struct {
		name     string
		score    float64
		maxScore float64
		want     float64
	}{
		{name: "basic ratio", score: 18, maxScore: 20, want: 90},
		{name: "max score zero", score: 10, maxScore: 0, want: 0},
		{name: "clamp below zero", score: -4, maxScore: 20, want: 0},
		{name: "clamp above max", score: 30, maxScore: 20, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.ComputeMasteryPercentage(tt.score, tt.maxScore)
			if got != tt.want {
				t.Fatalf("ComputeMasteryPercentage(%v, %v) = %v, want %v", tt.score, tt.maxScore, got, tt.want)
			}
		})
	}
}

func TestEvaluateIncludesTopicFeedback(t *testing.T) {
	engine := NewDefaultEvaluationEngine()

	evaluation := engine.Evaluate(domain.Submission{ID: "SUB-1"}, 15, 20, []domain.TopicScore{{Topic: "algebra", Score: 3, MaxPossible: 10}})
	if evaluation.MasteryPercentage != 75 {
		t.Fatalf("expected mastery 75, got %v", evaluation.MasteryPercentage)
	}
	if len(evaluation.Feedback) != 1 {
		t.Fatalf("expected one feedback item, got %d", len(evaluation.Feedback))
	}
}
