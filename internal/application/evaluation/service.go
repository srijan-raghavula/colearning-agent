package evaluation

type Service struct{}

func NewService() Service {
	return Service{}
}

func (Service) ComputeMasteryPercentage(score, maxScore float64) float64 {
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
