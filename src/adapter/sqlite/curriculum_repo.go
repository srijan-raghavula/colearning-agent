package sqlite

import (
	"context"
	"database/sql"

	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

func (r *SQLiteRepo) FindModule(ctx context.Context, id string) (domain.CourseModule, bool, error) {
	var module domain.CourseModule
	err := r.db.QueryRowContext(ctx, `SELECT id,course_id,title,description,"order" FROM course_modules WHERE id=? AND archived=0`, id).Scan(&module.ID, &module.CourseID, &module.Title, &module.Description, &module.Order)
	if err == sql.ErrNoRows {
		return module, false, nil
	}
	return module, err == nil, err
}

func (r *SQLiteRepo) ListModules(ctx context.Context, courseID string) ([]domain.CourseModule, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,course_id,title,description,"order" FROM course_modules WHERE course_id=? AND archived=0 ORDER BY "order",title`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CourseModule
	for rows.Next() {
		var module domain.CourseModule
		if err := rows.Scan(&module.ID, &module.CourseID, &module.Title, &module.Description, &module.Order); err != nil {
			return nil, err
		}
		out = append(out, module)
	}
	return out, rows.Err()
}

func (r *SQLiteRepo) SaveModule(ctx context.Context, module domain.CourseModule) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO course_modules(id,course_id,title,description,"order",archived) VALUES(?,?,?,?,?,0) ON CONFLICT(id) DO UPDATE SET course_id=excluded.course_id,title=excluded.title,description=excluded.description,"order"=excluded."order",archived=0`, module.ID, module.CourseID, module.Title, module.Description, module.Order)
	return err
}

func (r *SQLiteRepo) FindChapter(ctx context.Context, id string) (domain.CourseChapter, bool, error) {
	var chapter domain.CourseChapter
	err := r.db.QueryRowContext(ctx, `SELECT id,course_id,module_id,title,description,"order" FROM course_chapters WHERE id=? AND archived=0`, id).Scan(&chapter.ID, &chapter.CourseID, &chapter.ModuleID, &chapter.Title, &chapter.Description, &chapter.Order)
	if err == sql.ErrNoRows {
		return chapter, false, nil
	}
	return chapter, err == nil, err
}

func (r *SQLiteRepo) ListChapters(ctx context.Context, moduleID string) ([]domain.CourseChapter, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,course_id,module_id,title,description,"order" FROM course_chapters WHERE module_id=? AND archived=0 ORDER BY "order",title`, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CourseChapter
	for rows.Next() {
		var chapter domain.CourseChapter
		if err := rows.Scan(&chapter.ID, &chapter.CourseID, &chapter.ModuleID, &chapter.Title, &chapter.Description, &chapter.Order); err != nil {
			return nil, err
		}
		out = append(out, chapter)
	}
	return out, rows.Err()
}

func (r *SQLiteRepo) SaveChapter(ctx context.Context, chapter domain.CourseChapter) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO course_chapters(id,course_id,module_id,title,description,"order",archived) VALUES(?,?,?,?,?,?,0) ON CONFLICT(id) DO UPDATE SET course_id=excluded.course_id,module_id=excluded.module_id,title=excluded.title,description=excluded.description,"order"=excluded."order",archived=0`, chapter.ID, chapter.CourseID, chapter.ModuleID, chapter.Title, chapter.Description, chapter.Order)
	return err
}

func (r *SQLiteRepo) FindLesson(ctx context.Context, id string) (domain.CourseLesson, bool, error) {
	var lesson domain.CourseLesson
	err := r.db.QueryRowContext(ctx, `SELECT id,course_id,chapter_id,title,summary,"order" FROM course_lessons WHERE id=? AND archived=0`, id).Scan(&lesson.ID, &lesson.CourseID, &lesson.ChapterID, &lesson.Title, &lesson.Summary, &lesson.Order)
	if err == sql.ErrNoRows {
		return lesson, false, nil
	}
	return lesson, err == nil, err
}

func (r *SQLiteRepo) ListLessons(ctx context.Context, chapterID string) ([]domain.CourseLesson, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,course_id,chapter_id,title,summary,"order" FROM course_lessons WHERE chapter_id=? AND archived=0 ORDER BY "order",title`, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CourseLesson
	for rows.Next() {
		var lesson domain.CourseLesson
		if err := rows.Scan(&lesson.ID, &lesson.CourseID, &lesson.ChapterID, &lesson.Title, &lesson.Summary, &lesson.Order); err != nil {
			return nil, err
		}
		out = append(out, lesson)
	}
	return out, rows.Err()
}

func (r *SQLiteRepo) SaveLesson(ctx context.Context, lesson domain.CourseLesson) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO course_lessons(id,course_id,chapter_id,title,summary,"order",archived) VALUES(?,?,?,?,?,?,0) ON CONFLICT(id) DO UPDATE SET course_id=excluded.course_id,chapter_id=excluded.chapter_id,title=excluded.title,summary=excluded.summary,"order"=excluded."order",archived=0`, lesson.ID, lesson.CourseID, lesson.ChapterID, lesson.Title, lesson.Summary, lesson.Order)
	return err
}

func (r *SQLiteRepo) FindLessonContent(ctx context.Context, id string) (domain.LessonContent, bool, error) {
	var content domain.LessonContent
	var published int
	err := r.db.QueryRowContext(ctx, `SELECT id,course_id,lesson_id,kind,title,body,"order",published FROM lesson_contents WHERE id=? AND archived=0`, id).Scan(&content.ID, &content.CourseID, &content.LessonID, &content.Kind, &content.Title, &content.Body, &content.Order, &published)
	if err == sql.ErrNoRows {
		return content, false, nil
	}
	content.Published = published == 1
	return content, err == nil, err
}

func (r *SQLiteRepo) ListLessonContents(ctx context.Context, lessonID string) ([]domain.LessonContent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,course_id,lesson_id,kind,title,body,"order",published FROM lesson_contents WHERE lesson_id=? AND archived=0 ORDER BY "order",title`, lessonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LessonContent
	for rows.Next() {
		var content domain.LessonContent
		var published int
		if err := rows.Scan(&content.ID, &content.CourseID, &content.LessonID, &content.Kind, &content.Title, &content.Body, &content.Order, &published); err != nil {
			return nil, err
		}
		content.Published = published == 1
		out = append(out, content)
	}
	return out, rows.Err()
}

func (r *SQLiteRepo) SaveLessonContent(ctx context.Context, content domain.LessonContent) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO lesson_contents(id,course_id,lesson_id,kind,title,body,"order",published,archived) VALUES(?,?,?,?,?,?,?,?,0) ON CONFLICT(id) DO UPDATE SET course_id=excluded.course_id,lesson_id=excluded.lesson_id,kind=excluded.kind,title=excluded.title,body=excluded.body,"order"=excluded."order",published=excluded.published,archived=0`, content.ID, content.CourseID, content.LessonID, content.Kind, content.Title, content.Body, content.Order, boolToInt(content.Published))
	return err
}
