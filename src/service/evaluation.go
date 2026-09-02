package service

import (
	"fmt"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

type MasteryCalculator interface {
	ComputeMasteryPercentage(score, maxScore float64) float64
}

type FeedbackGenerator interface {
	GenerateFeedback(topicScores []domain.TopicScore) []string
}

type EvaluationEngine interface {
	Evaluate(submission domain.Submission, score, maxScore float64, topicScores []domain.TopicScore) domain.Evaluation
}

type DefaultEvaluationEngine struct{}

func NewDefaultEvaluationEngine() DefaultEvaluationEngine {
	return DefaultEvaluationEngine{}
}

func (DefaultEvaluationEngine) ComputeMasteryPercentage(score, maxScore float64) float64 {
	if maxScore <= 0 {
		return 0
	}
	if score < 0 {
		score = 0
	}
	if score > maxScore {
		score = maxScore
	}
	return (score / maxScore) * 100
}

func (e DefaultEvaluationEngine) GenerateFeedback(topicScores []domain.TopicScore) []string {
	feedback := make([]string, 0, len(topicScores))
	for _, score := range topicScores {
		switch {
		case score.Mastery < 50:
			feedback = append(feedback, fmt.Sprintf("Focus on topic %s with guided revision.", score.Topic))
		case score.Mastery < 75:
			feedback = append(feedback, fmt.Sprintf("Practice more problems in topic %s.", score.Topic))
		default:
			feedback = append(feedback, fmt.Sprintf("Strong understanding in topic %s.", score.Topic))
		}
	}
	return feedback
}

func (e DefaultEvaluationEngine) Evaluate(submission domain.Submission, score, maxScore float64, topicScores []domain.TopicScore) domain.Evaluation {
	normalized := make([]domain.TopicScore, 0, len(topicScores))
	for _, topicScore := range topicScores {
		topicScore.Mastery = e.ComputeMasteryPercentage(topicScore.Score, topicScore.MaxPossible)
		normalized = append(normalized, topicScore)
	}

	mastery := e.ComputeMasteryPercentage(score, maxScore)
	return domain.Evaluation{
		SubmissionID:      submission.ID,
		Score:             score,
		MaxScore:          maxScore,
		MasteryPercentage: mastery,
		TopicScores:       normalized,
		Feedback:          e.GenerateFeedback(normalized),
		EvaluatedAt:       time.Now().UTC(),
	}
}
