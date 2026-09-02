package domain

import "time"

type TopicScore struct {
	Topic       string
	Score       float64
	MaxPossible float64
	Mastery     float64
}

type Evaluation struct {
	SubmissionID      string
	Score             float64
	MaxScore          float64
	MasteryPercentage float64
	TopicScores       []TopicScore
	Feedback          []string
	EvaluatedAt       time.Time
}
