package repository

import (
	"context"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

type CurriculumRepository interface {
	FindModule(ctx context.Context, id string) (domain.CourseModule, bool, error)
	ListModules(ctx context.Context, courseID string) ([]domain.CourseModule, error)
	SaveModule(ctx context.Context, module domain.CourseModule) error
	FindChapter(ctx context.Context, id string) (domain.CourseChapter, bool, error)
	ListChapters(ctx context.Context, moduleID string) ([]domain.CourseChapter, error)
	SaveChapter(ctx context.Context, chapter domain.CourseChapter) error
	FindLesson(ctx context.Context, id string) (domain.CourseLesson, bool, error)
	ListLessons(ctx context.Context, chapterID string) ([]domain.CourseLesson, error)
	SaveLesson(ctx context.Context, lesson domain.CourseLesson) error
	FindLessonContent(ctx context.Context, id string) (domain.LessonContent, bool, error)
	ListLessonContents(ctx context.Context, lessonID string) ([]domain.LessonContent, error)
	SaveLessonContent(ctx context.Context, content domain.LessonContent) error
}

type CurriculumDraft struct {
	Course   domain.LearningCourse
	Modules  []domain.CourseModule
	Chapters []domain.CourseChapter
	Lessons  []domain.CourseLesson
	Contents []domain.LessonContent
}

type CurriculumDraftRepository interface {
	SaveCurriculumDraft(context.Context, CurriculumDraft) error
}
