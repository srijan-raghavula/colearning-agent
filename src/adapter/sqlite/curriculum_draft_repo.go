package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/srijan-raghavula/colearning-agent/src/repository"
)

func (r *SQLiteRepo) SaveCurriculumDraft(ctx context.Context, draft repository.CurriculumDraft) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range draft.Modules {
		if err := execCurriculumDraftRow(ctx, tx, `INSERT INTO course_modules(id,course_id,title,description,"order",archived) VALUES(?,?,?,?,?,0) ON CONFLICT(id) DO UPDATE SET title=excluded.title,description=excluded.description,"order"=excluded."order",archived=0 WHERE course_modules.course_id=excluded.course_id`, item.ID, item.CourseID, item.Title, item.Description, item.Order); err != nil {
			return err
		}
	}
	for _, item := range draft.Chapters {
		if err := execCurriculumDraftRow(ctx, tx, `INSERT INTO course_chapters(id,course_id,module_id,title,description,"order",archived) VALUES(?,?,?,?,?,?,0) ON CONFLICT(id) DO UPDATE SET module_id=excluded.module_id,title=excluded.title,description=excluded.description,"order"=excluded."order",archived=0 WHERE course_chapters.course_id=excluded.course_id`, item.ID, item.CourseID, item.ModuleID, item.Title, item.Description, item.Order); err != nil {
			return err
		}
	}
	for order, item := range draft.Lessons {
		if err := execCurriculumDraftRow(ctx, tx, `INSERT INTO course_lessons(id,course_id,chapter_id,title,summary,"order",archived) VALUES(?,?,?,?,?,?,0) ON CONFLICT(id) DO UPDATE SET chapter_id=excluded.chapter_id,title=excluded.title,summary=excluded.summary,"order"=excluded."order",archived=0 WHERE course_lessons.course_id=excluded.course_id`, item.ID, item.CourseID, item.ChapterID, item.Title, item.Summary, item.Order); err != nil {
			return err
		}
		if err := execCurriculumDraftRow(ctx, tx, `INSERT INTO concepts(id,course_id,title,summary,level,"order",estimated_minutes,prerequisites_json,archived) VALUES(?,?,?,?,?,?,?,?,0) ON CONFLICT(id) DO UPDATE SET title=excluded.title,summary=excluded.summary,level=excluded.level,"order"=excluded."order",estimated_minutes=excluded.estimated_minutes,archived=0 WHERE concepts.course_id=excluded.course_id`, item.ID, item.CourseID, item.Title, item.Summary, "Lesson", order+1, 15, "[]"); err != nil {
			return err
		}
		if err := execCurriculumDraftRow(ctx, tx, `INSERT INTO objectives(id,course_id,concept_id,description,outcomes_json) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET description=excluded.description WHERE objectives.course_id=excluded.course_id`, "objective-"+item.ID, item.CourseID, item.ID, item.Summary, "[]"); err != nil {
			return err
		}
	}
	for _, item := range draft.Contents {
		published := 0
		if item.Published {
			published = 1
		}
		if err := execCurriculumDraftRow(ctx, tx, `INSERT INTO lesson_contents(id,course_id,lesson_id,kind,title,body,"order",published,archived) VALUES(?,?,?,?,?,?,?,?,0) ON CONFLICT(id) DO UPDATE SET lesson_id=excluded.lesson_id,kind=excluded.kind,title=excluded.title,body=excluded.body,"order"=excluded."order",published=excluded.published,archived=0 WHERE lesson_contents.course_id=excluded.course_id`, item.ID, item.CourseID, item.LessonID, item.Kind, item.Title, item.Body, item.Order, published); err != nil {
			return err
		}
		var version, oldPublished int
		var oldConcept, oldTitle, oldBody, oldSourceKind, oldSourceRef string
		err := tx.QueryRowContext(ctx, `SELECT version,concept_id,title,body,source_kind,source_ref,published FROM materials WHERE id=? AND course_id=?`, item.ID, item.CourseID).Scan(&version, &oldConcept, &oldTitle, &oldBody, &oldSourceKind, &oldSourceRef, &oldPublished)
		if errors.Is(err, sql.ErrNoRows) {
			version = 1
		} else if err != nil {
			return err
		} else if oldConcept != item.LessonID || oldTitle != item.Title || oldBody != item.Body || oldSourceKind != "teacher" || oldSourceRef != "" || oldPublished != published {
			version++
		}
		if err := execCurriculumDraftRow(ctx, tx, `INSERT INTO materials(id,course_id,concept_id,title,body,source_kind,source_ref,published,version) VALUES(?,?,?,?,?,'teacher','',?,?) ON CONFLICT(id) DO UPDATE SET concept_id=excluded.concept_id,title=excluded.title,body=excluded.body,published=excluded.published,version=excluded.version WHERE materials.course_id=excluded.course_id`, item.ID, item.CourseID, item.LessonID, item.Title, item.Body, published, version); err != nil {
			return err
		}
	}
	if err := archiveCurriculumMissing(ctx, tx, draft); err != nil {
		return err
	}
	policyJSON, _ := json.Marshal(draft.Course.TutorPolicy)
	if err := execCurriculumDraftRow(ctx, tx, `UPDATE courses SET next_concept_id=?,tutor_policy_json=? WHERE id=? AND teacher_id=?`, draft.Course.NextConceptID, string(policyJSON), draft.Course.ID, draft.Course.TeacherID); err != nil {
		return err
	}
	return tx.Commit()
}

func archiveCurriculumMissing(ctx context.Context, tx *sql.Tx, draft repository.CurriculumDraft) error {
	for _, entry := range []struct {
		table string
		ids   []string
	}{
		{"course_modules", curriculumModuleIDs(draft)},
		{"course_chapters", curriculumChapterIDs(draft)},
		{"course_lessons", curriculumLessonIDs(draft)},
		{"lesson_contents", curriculumContentIDs(draft)},
	} {
		query := `UPDATE ` + entry.table + ` SET archived=1 WHERE course_id=? AND archived=0`
		args := []any{draft.Course.ID}
		if len(entry.ids) > 0 {
			query += ` AND id NOT IN (` + placeholders(len(entry.ids)) + `)`
			for _, id := range entry.ids {
				args = append(args, id)
			}
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	for _, query := range []string{
		`UPDATE course_chapters SET archived=1 WHERE course_id=? AND module_id IN (SELECT id FROM course_modules WHERE course_id=? AND archived=1)`,
		`UPDATE course_lessons SET archived=1 WHERE course_id=? AND chapter_id IN (SELECT id FROM course_chapters WHERE course_id=? AND archived=1)`,
		`UPDATE lesson_contents SET archived=1,published=0 WHERE course_id=? AND lesson_id IN (SELECT id FROM course_lessons WHERE course_id=? AND archived=1)`,
		`UPDATE concepts SET archived=1 WHERE course_id=? AND id IN (SELECT id FROM course_lessons WHERE course_id=? AND archived=1)`,
		`UPDATE materials SET published=0 WHERE course_id=? AND (concept_id IN (SELECT id FROM concepts WHERE course_id=? AND archived=1) OR id IN (SELECT id FROM lesson_contents WHERE course_id=? AND archived=1))`,
	} {
		args := []any{draft.Course.ID, draft.Course.ID}
		if strings.Contains(query, "SELECT id FROM lesson_contents") {
			args = append(args, draft.Course.ID)
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func placeholders(count int) string {
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}

func curriculumModuleIDs(d repository.CurriculumDraft) []string {
	out := make([]string, 0, len(d.Modules))
	for _, v := range d.Modules {
		out = append(out, v.ID)
	}
	return out
}
func curriculumChapterIDs(d repository.CurriculumDraft) []string {
	out := make([]string, 0, len(d.Chapters))
	for _, v := range d.Chapters {
		out = append(out, v.ID)
	}
	return out
}
func curriculumLessonIDs(d repository.CurriculumDraft) []string {
	out := make([]string, 0, len(d.Lessons))
	for _, v := range d.Lessons {
		out = append(out, v.ID)
	}
	return out
}
func curriculumContentIDs(d repository.CurriculumDraft) []string {
	out := make([]string, 0, len(d.Contents))
	for _, v := range d.Contents {
		out = append(out, v.ID)
	}
	return out
}

func execCurriculumDraftRow(ctx context.Context, tx *sql.Tx, query string, args ...any) error {
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return errors.New("curriculum item conflicts with data outside the selected course")
	}
	return nil
}
