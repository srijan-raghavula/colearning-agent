package agentic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

type AuthoringContentDraft struct {
	Kind  string `json:"kind"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type AuthoringLessonDraft struct {
	Title    string                  `json:"title"`
	Summary  string                  `json:"summary"`
	Contents []AuthoringContentDraft `json:"contents"`
}

type AuthoringChapterDraft struct {
	Title   string                 `json:"title"`
	Summary string                 `json:"summary"`
	Lessons []AuthoringLessonDraft `json:"lessons"`
}

type AuthoringModuleDraft struct {
	Title    string                  `json:"title"`
	Summary  string                  `json:"summary"`
	Chapters []AuthoringChapterDraft `json:"chapters"`
}

type CurriculumDraft struct {
	Provider string                 `json:"provider"`
	Summary  string                 `json:"summary"`
	Modules  []AuthoringModuleDraft `json:"modules"`
}

type CurriculumAuthoringGateway interface {
	GenerateCurriculum(context.Context, domain.LearningCourse) (CurriculumDraft, error)
}

type CurriculumAuthoringAgent struct {
	Gateway CurriculumAuthoringGateway
	Logger  ExecutionLogger
}

func (a CurriculumAuthoringAgent) Generate(ctx context.Context, course domain.LearningCourse) (CurriculumDraft, error) {
	if a.Gateway == nil {
		return CurriculumDraft{}, errors.New("curriculum authoring provider is not configured")
	}
	if strings.TrimSpace(course.ID) == "" || strings.TrimSpace(course.Title) == "" {
		return CurriculumDraft{}, errors.New("course identity and title are required")
	}
	draft, err := a.Gateway.GenerateCurriculum(ctx, course)
	if err != nil {
		return CurriculumDraft{}, fmt.Errorf("generate curriculum draft: %w", err)
	}
	if err := validateCurriculumDraft(draft); err != nil {
		return CurriculumDraft{}, err
	}
	if a.Logger != nil {
		_ = a.Logger.AgentLogForSession(ctx, "", "CurriculumAuthoringAgent", "generate_curriculum_draft", map[string]any{"course_id": course.ID, "provider": draft.Provider}, map[string]any{"module_count": len(draft.Modules), "summary": draft.Summary}, nil)
	}
	return draft, nil
}

func validateCurriculumDraft(draft CurriculumDraft) error {
	if len(draft.Modules) == 0 || len(draft.Modules) > 8 {
		return errors.New("authoring provider returned an invalid module count")
	}
	if strings.TrimSpace(draft.Provider) == "" || strings.TrimSpace(draft.Summary) == "" {
		return errors.New("authoring provider returned an incomplete draft")
	}
	for _, module := range draft.Modules {
		if !boundedDraftText(module.Title, 160) || !boundedDraftText(module.Summary, 3000) || len(module.Chapters) == 0 || len(module.Chapters) > 8 {
			return errors.New("authoring provider returned an invalid module")
		}
		for _, chapter := range module.Chapters {
			if !boundedDraftText(chapter.Title, 160) || !boundedDraftText(chapter.Summary, 3000) || len(chapter.Lessons) == 0 || len(chapter.Lessons) > 8 {
				return errors.New("authoring provider returned an invalid chapter")
			}
			for _, lesson := range chapter.Lessons {
				if !boundedDraftText(lesson.Title, 160) || !boundedDraftText(lesson.Summary, 3000) || len(lesson.Contents) == 0 || len(lesson.Contents) > 8 {
					return errors.New("authoring provider returned an invalid lesson")
				}
				for _, content := range lesson.Contents {
					if !boundedDraftText(content.Title, 200) || !boundedDraftText(content.Body, 12000) {
						return errors.New("authoring provider returned invalid lesson content")
					}
				}
			}
		}
	}
	// Ensure the bounded draft fits the execution log and storage contracts.
	if _, err := json.Marshal(draft); err != nil {
		return fmt.Errorf("encode curriculum draft: %w", err)
	}
	return nil
}

func boundedDraftText(value string, max int) bool {
	return strings.TrimSpace(value) != "" && len([]rune(value)) <= max
}
