package usecase

import (
	"context"
	"sort"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

type FacultyPortal struct {
	submissions repository.SubmissionReader
	evaluations repository.EvaluationReader
}

func NewFacultyPortal(submissions repository.SubmissionReader, evaluations repository.EvaluationReader) FacultyPortal {
	return FacultyPortal{submissions: submissions, evaluations: evaluations}
}

func (p FacultyPortal) Dashboard(ctx context.Context) (domain.FacultyDashboard, error) {
	submissions, err := p.submissions.ListAll(ctx)
	if err != nil {
		return domain.FacultyDashboard{}, err
	}
	evaluations, err := p.evaluations.ListAll(ctx)
	if err != nil {
		return domain.FacultyDashboard{}, err
	}

	evalBySubmission := map[string]domain.Evaluation{}
	for _, evaluation := range evaluations {
		evalBySubmission[evaluation.SubmissionID] = evaluation
	}

	type aggregate struct {
		total      int
		submitted  int
		late       int
		missing    int
		masterySum float64
		masteryN   float64
		topicLow   map[string]bool
	}

	subjects := map[string]*aggregate{}
	missing := []domain.Submission{}
	now := time.Now().UTC()

	for _, submission := range submissions {
		agg, ok := subjects[submission.SubjectID]
		if !ok {
			agg = &aggregate{topicLow: map[string]bool{}}
			subjects[submission.SubjectID] = agg
		}
		agg.total++

		effectiveStatus := submission.Status
		if submission.SubmittedAt.IsZero() && now.After(submission.DueAt) {
			effectiveStatus = domain.SubmissionStatusMissing
		}
		switch effectiveStatus {
		case domain.SubmissionStatusSubmitted:
			agg.submitted++
		case domain.SubmissionStatusLate:
			agg.late++
		case domain.SubmissionStatusMissing:
			agg.missing++
			missing = append(missing, submission)
		}

		if evaluation, hasEvaluation := evalBySubmission[submission.ID]; hasEvaluation {
			agg.masterySum += evaluation.MasteryPercentage
			agg.masteryN++
			for _, topic := range evaluation.TopicScores {
				if topic.Mastery < 60 {
					agg.topicLow[topic.Topic] = true
				}
			}
		}
	}

	summaries := make([]domain.SubjectSummary, 0, len(subjects))
	for subjectID, agg := range subjects {
		averageMastery := 0.0
		if agg.masteryN > 0 {
			averageMastery = agg.masterySum / agg.masteryN
		}
		lowTopics := make([]string, 0, len(agg.topicLow))
		for topic := range agg.topicLow {
			lowTopics = append(lowTopics, topic)
		}
		sort.Strings(lowTopics)

		summaries = append(summaries, domain.SubjectSummary{
			SubjectID:        subjectID,
			TotalSubmissions: agg.total,
			SubmittedCount:   agg.submitted,
			LateCount:        agg.late,
			MissingCount:     agg.missing,
			AverageMastery:   averageMastery,
			LowMasteryTopics: lowTopics,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].SubjectID < summaries[j].SubjectID
	})

	return domain.FacultyDashboard{Summaries: summaries, MissingSubmissions: missing}, nil
}
