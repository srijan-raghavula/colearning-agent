package evaluation

import "testing"

func TestComputeMasteryPercentage(t *testing.T) {
	svc := NewService()

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
			got := svc.ComputeMasteryPercentage(tt.score, tt.maxScore)
			if got != tt.want {
				t.Fatalf("ComputeMasteryPercentage(%v, %v) = %v, want %v", tt.score, tt.maxScore, got, tt.want)
			}
		})
	}
}
