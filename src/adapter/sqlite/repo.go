package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/srijan-raghavula/colearning-agent/src/auth"
	"github.com/srijan-raghavula/colearning-agent/src/domain"
)

type SQLiteRepo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *SQLiteRepo {
	return &SQLiteRepo{db: db}
}

func (r *SQLiteRepo) FindCourse(ctx context.Context, id string) (domain.LearningCourse, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,title,subject,description,teacher_id,tutor_policy_json,next_concept_id FROM courses WHERE id=?`, id)
	var c domain.LearningCourse
	var tpJSON, next string
	if err := row.Scan(&c.ID, &c.Title, &c.Subject, &c.Description, &c.TeacherID, &tpJSON, &next); err == sql.ErrNoRows {
		return c, false, nil
	} else if err != nil {
		return c, false, err
	}
	c.NextConceptID = next
	_ = json.Unmarshal([]byte(tpJSON), &c.TutorPolicy)
	return c, true, nil
}

func (r *SQLiteRepo) ListCourses(ctx context.Context) ([]domain.LearningCourse, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,title,subject,description,teacher_id,tutor_policy_json,next_concept_id FROM courses ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LearningCourse
	for rows.Next() {
		var c domain.LearningCourse
		var tpJSON, next string
		if err := rows.Scan(&c.ID, &c.Title, &c.Subject, &c.Description, &c.TeacherID, &tpJSON, &next); err != nil {
			return nil, err
		}
		c.NextConceptID = next
		_ = json.Unmarshal([]byte(tpJSON), &c.TutorPolicy)
		out = append(out, c)
	}
	return out, nil
}

func (r *SQLiteRepo) SaveCourse(ctx context.Context, course domain.LearningCourse) error {
	tpJSON, _ := json.Marshal(course.TutorPolicy)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO courses(id,title,subject,description,teacher_id,tutor_policy_json,next_concept_id) VALUES(?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET title=excluded.title,subject=excluded.subject,description=excluded.description,teacher_id=excluded.teacher_id,tutor_policy_json=excluded.tutor_policy_json,next_concept_id=excluded.next_concept_id`,
		course.ID, course.Title, course.Subject, course.Description, course.TeacherID, string(tpJSON), course.NextConceptID)
	return err
}

func (r *SQLiteRepo) FindConcept(ctx context.Context, id string) (domain.LearningConcept, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,course_id,title,summary,level,"order",estimated_minutes,prerequisites_json FROM concepts WHERE id=?`, id)
	var c domain.LearningConcept
	var prereqJSON string
	if err := row.Scan(&c.ID, &c.CourseID, &c.Title, &c.Summary, &c.Level, &c.Order, &c.EstimatedMinutes, &prereqJSON); err == sql.ErrNoRows {
		return c, false, nil
	} else if err != nil {
		return c, false, err
	}
	_ = json.Unmarshal([]byte(prereqJSON), &c.PrerequisiteIDs)
	return c, true, nil
}

func (r *SQLiteRepo) ListConcepts(ctx context.Context, courseID string) ([]domain.LearningConcept, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,course_id,title,summary,level,"order",estimated_minutes,prerequisites_json FROM concepts WHERE course_id=? ORDER BY "order"`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LearningConcept
	for rows.Next() {
		var c domain.LearningConcept
		var prereqJSON string
		if err := rows.Scan(&c.ID, &c.CourseID, &c.Title, &c.Summary, &c.Level, &c.Order, &c.EstimatedMinutes, &prereqJSON); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(prereqJSON), &c.PrerequisiteIDs)
		out = append(out, c)
	}
	return out, nil
}

func (r *SQLiteRepo) SaveConcept(ctx context.Context, concept domain.LearningConcept) error {
	prereqJSON, _ := json.Marshal(concept.PrerequisiteIDs)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO concepts(id,course_id,title,summary,level,"order",estimated_minutes,prerequisites_json) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET course_id=excluded.course_id,title=excluded.title,summary=excluded.summary,level=excluded.level,"order"=excluded."order",estimated_minutes=excluded.estimated_minutes,prerequisites_json=excluded.prerequisites_json`,
		concept.ID, concept.CourseID, concept.Title, concept.Summary, concept.Level, concept.Order, concept.EstimatedMinutes, string(prereqJSON))
	return err
}

func (r *SQLiteRepo) ListObjectives(ctx context.Context, courseID, conceptID string) ([]domain.LearningObjective, error) {
	var rows *sql.Rows
	var err error
	if conceptID != "" {
		rows, err = r.db.QueryContext(ctx, `SELECT id,course_id,concept_id,description,outcomes_json FROM objectives WHERE course_id=? AND concept_id=?`, courseID, conceptID)
	} else {
		rows, err = r.db.QueryContext(ctx, `SELECT id,course_id,concept_id,description,outcomes_json FROM objectives WHERE course_id=?`, courseID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LearningObjective
	for rows.Next() {
		var o domain.LearningObjective
		var outcomesJSON string
		if err := rows.Scan(&o.ID, &o.CourseID, &o.ConceptID, &o.Description, &outcomesJSON); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(outcomesJSON), &o.Outcomes)
		out = append(out, o)
	}
	return out, nil
}

func (r *SQLiteRepo) SaveObjective(ctx context.Context, objective domain.LearningObjective) error {
	outcomesJSON, _ := json.Marshal(objective.Outcomes)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO objectives(id,course_id,concept_id,description,outcomes_json) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET course_id=excluded.course_id,concept_id=excluded.concept_id,description=excluded.description,outcomes_json=excluded.outcomes_json`,
		objective.ID, objective.CourseID, objective.ConceptID, objective.Description, string(outcomesJSON))
	return err
}

func (r *SQLiteRepo) FindMaterial(ctx context.Context, id string) (domain.LearningMaterial, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,course_id,concept_id,title,body,source_kind,source_ref,published,version FROM materials WHERE id=?`, id)
	var m domain.LearningMaterial
	var skind, sref string
	var published int
	if err := row.Scan(&m.ID, &m.CourseID, &m.ConceptID, &m.Title, &m.Body, &skind, &sref, &published, &m.Version); err == sql.ErrNoRows {
		return m, false, nil
	} else if err != nil {
		return m, false, err
	}
	m.Published = published == 1
	m.Source = domain.LearningSourceRef{ID: m.ID, Title: m.Title, Kind: domain.LearningSourceKind(skind), Reference: sref}
	return m, true, nil
}

func (r *SQLiteRepo) ListMaterials(ctx context.Context, courseID, conceptID string) ([]domain.LearningMaterial, error) {
	query := `SELECT id,course_id,concept_id,title,body,source_kind,source_ref,published,version FROM materials WHERE course_id=?`
	args := []any{courseID}
	if conceptID != "" {
		query += ` AND concept_id=?`
		args = append(args, conceptID)
	}
	query += ` ORDER BY title`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LearningMaterial
	for rows.Next() {
		var m domain.LearningMaterial
		var skind, sref string
		var published int
		if err := rows.Scan(&m.ID, &m.CourseID, &m.ConceptID, &m.Title, &m.Body, &skind, &sref, &published, &m.Version); err != nil {
			return nil, err
		}
		m.Published = published == 1
		m.Source = domain.LearningSourceRef{ID: m.ID, Title: m.Title, Kind: domain.LearningSourceKind(skind), Reference: sref}
		out = append(out, m)
	}
	return out, nil
}

func (r *SQLiteRepo) ListAllMaterials(ctx context.Context, courseID string) ([]domain.LearningMaterial, error) {
	return r.ListMaterials(ctx, courseID, "")
}

func (r *SQLiteRepo) SaveMaterial(ctx context.Context, material domain.LearningMaterial) error {
	skind := string(material.Source.Kind)
	if skind == "" {
		skind = "teacher"
	}
	sref := material.Source.Reference
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO materials(id,course_id,concept_id,title,body,source_kind,source_ref,published,version) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET course_id=excluded.course_id,concept_id=excluded.concept_id,title=excluded.title,body=excluded.body,source_kind=excluded.source_kind,source_ref=excluded.source_ref,published=excluded.published,version=excluded.version`,
		material.ID, material.CourseID, material.ConceptID, material.Title, material.Body, skind, sref, boolToInt(material.Published), material.Version)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Session repo
func (r *SQLiteRepo) FindByID(ctx context.Context, id string) (domain.LearningSession, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,student_id,course_id,concept_id,mode,status,started_at,last_active_at FROM learning_sessions WHERE id=?`, id)
	var s domain.LearningSession
	var mode, status string
	var started, last time.Time
	if err := row.Scan(&s.ID, &s.StudentID, &s.CourseID, &s.ConceptID, &mode, &status, &started, &last); err == sql.ErrNoRows {
		return s, false, nil
	} else if err != nil {
		return s, false, err
	}
	s.Mode = domain.TutorMode(mode)
	s.Status = domain.LearningSessionStatus(status)
	s.StartedAt = started
	s.LastActiveAt = last
	return s, true, nil
}

func (r *SQLiteRepo) FindActive(ctx context.Context, studentID, courseID, conceptID string) (domain.LearningSession, bool, error) {
	query := `SELECT id,student_id,course_id,concept_id,mode,status,started_at,last_active_at FROM learning_sessions WHERE student_id=? AND status='active'`
	args := []any{studentID}
	if courseID != "" {
		query += ` AND course_id=?`
		args = append(args, courseID)
	}
	if conceptID != "" {
		query += ` AND concept_id=?`
		args = append(args, conceptID)
	}
	query += ` ORDER BY last_active_at DESC LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, args...)
	var s domain.LearningSession
	var mode, status string
	var started, last time.Time
	if err := row.Scan(&s.ID, &s.StudentID, &s.CourseID, &s.ConceptID, &mode, &status, &started, &last); err == sql.ErrNoRows {
		return s, false, nil
	} else if err != nil {
		return s, false, err
	}
	s.Mode = domain.TutorMode(mode)
	s.Status = domain.LearningSessionStatus(status)
	s.StartedAt = started
	s.LastActiveAt = last
	return s, true, nil
}

func (r *SQLiteRepo) Save(ctx context.Context, session domain.LearningSession) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO learning_sessions(id,student_id,course_id,concept_id,mode,status,started_at,last_active_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET student_id=excluded.student_id,course_id=excluded.course_id,concept_id=excluded.concept_id,mode=excluded.mode,status=excluded.status,last_active_at=excluded.last_active_at`,
		session.ID, session.StudentID, session.CourseID, session.ConceptID, string(session.Mode), string(session.Status), session.StartedAt, session.LastActiveAt)
	return err
}

func (r *SQLiteRepo) AppendTurn(ctx context.Context, turn domain.LearningTurn) error {
	sourcesJSON, _ := json.Marshal(turn.Sources)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO learning_turns(id,session_id,role,kind,mode,body,sources_json,confidence_delta,created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		turn.ID, turn.SessionID, string(turn.Role), turn.Kind, string(turn.Mode), turn.Body, string(sourcesJSON), turn.ConfidenceDelta, turn.CreatedAt)
	return err
}

func (r *SQLiteRepo) ListTurns(ctx context.Context, sessionID string) ([]domain.LearningTurn, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,session_id,role,kind,mode,body,sources_json,confidence_delta,created_at FROM learning_turns WHERE session_id=? ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LearningTurn
	for rows.Next() {
		var t domain.LearningTurn
		var roleStr, modeStr, sourcesJSON string
		var created time.Time
		if err := rows.Scan(&t.ID, &t.SessionID, &roleStr, &t.Kind, &modeStr, &t.Body, &sourcesJSON, &t.ConfidenceDelta, &created); err != nil {
			return nil, err
		}
		t.Role = domain.LearningTurnRole(roleStr)
		t.Mode = domain.TutorMode(modeStr)
		t.CreatedAt = created
		_ = json.Unmarshal([]byte(sourcesJSON), &t.Sources)
		out = append(out, t)
	}
	return out, nil
}

// Progress repo
func (r *SQLiteRepo) Find(ctx context.Context, studentID, courseID, conceptID string) (domain.ConceptProgress, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT student_id,course_id,concept_id,status,confidence,evidence_count,next_action,last_activity_at FROM concept_progress WHERE student_id=? AND course_id=? AND concept_id=?`, studentID, courseID, conceptID)
	var p domain.ConceptProgress
	var status string
	var last time.Time
	if err := row.Scan(&p.StudentID, &p.CourseID, &p.ConceptID, &status, &p.Confidence, &p.EvidenceCount, &p.NextAction, &last); err == sql.ErrNoRows {
		return p, false, nil
	} else if err != nil {
		return p, false, err
	}
	p.Status = domain.ConceptProgressStatus(status)
	p.LastActivityAt = last
	return p, true, nil
}

func (r *SQLiteRepo) ListByStudent(ctx context.Context, studentID, courseID string) ([]domain.ConceptProgress, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT student_id,course_id,concept_id,status,confidence,evidence_count,next_action,last_activity_at FROM concept_progress WHERE student_id=? AND course_id=?`, studentID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ConceptProgress
	for rows.Next() {
		var p domain.ConceptProgress
		var status string
		var last time.Time
		if err := rows.Scan(&p.StudentID, &p.CourseID, &p.ConceptID, &status, &p.Confidence, &p.EvidenceCount, &p.NextAction, &last); err != nil {
			return nil, err
		}
		p.Status = domain.ConceptProgressStatus(status)
		p.LastActivityAt = last
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ConceptID < out[j].ConceptID })
	return out, nil
}

func (r *SQLiteRepo) ListAll(ctx context.Context) ([]domain.ConceptProgress, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT student_id,course_id,concept_id,status,confidence,evidence_count,next_action,last_activity_at FROM concept_progress`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ConceptProgress
	for rows.Next() {
		var p domain.ConceptProgress
		var status string
		var last time.Time
		if err := rows.Scan(&p.StudentID, &p.CourseID, &p.ConceptID, &status, &p.Confidence, &p.EvidenceCount, &p.NextAction, &last); err != nil {
			return nil, err
		}
		p.Status = domain.ConceptProgressStatus(status)
		p.LastActivityAt = last
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CourseID != out[j].CourseID {
			return out[i].CourseID < out[j].CourseID
		}
		if out[i].StudentID != out[j].StudentID {
			return out[i].StudentID < out[j].StudentID
		}
		return out[i].ConceptID < out[j].ConceptID
	})
	return out, nil
}

// SaveProgress writes progress (interface-specific)
func (r *SQLiteRepo) SaveProgress(ctx context.Context, progress domain.ConceptProgress) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO concept_progress(student_id,course_id,concept_id,status,confidence,evidence_count,next_action,last_activity_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(student_id,course_id,concept_id) DO UPDATE SET status=excluded.status,confidence=excluded.confidence,evidence_count=excluded.evidence_count,next_action=excluded.next_action,last_activity_at=excluded.last_activity_at`,
		progress.StudentID, progress.CourseID, progress.ConceptID, string(progress.Status), progress.Confidence, progress.EvidenceCount, progress.NextAction, progress.LastActivityAt)
	return err
}

// Evidence repo
// Evidence repo
func (r *SQLiteRepo) SaveEvidence(ctx context.Context, evidence domain.LearningEvidence) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO learning_evidence(id,session_id,turn_id,student_id,course_id,concept_id,kind,note,observed_at) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`,
		evidence.ID, evidence.SessionID, evidence.TurnID, evidence.StudentID, evidence.CourseID, evidence.ConceptID, evidence.Kind, evidence.Note, evidence.ObservedAt)
	return err
}

func (r *SQLiteRepo) ListByCourse(ctx context.Context, courseID string) ([]domain.LearningEvidence, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,session_id,turn_id,student_id,course_id,concept_id,kind,note,observed_at FROM learning_evidence WHERE course_id=? ORDER BY observed_at DESC`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.LearningEvidence
	for rows.Next() {
		var e domain.LearningEvidence
		var observed time.Time
		if err := rows.Scan(&e.ID, &e.SessionID, &e.TurnID, &e.StudentID, &e.CourseID, &e.ConceptID, &e.Kind, &e.Note, &observed); err != nil {
			return nil, err
		}
		e.ObservedAt = observed
		out = append(out, e)
	}
	return out, nil
}

// Check repo
func (r *SQLiteRepo) SaveCheck(ctx context.Context, check domain.UnderstandingCheck) error {
	feedbackJSON, _ := json.Marshal(check.Feedback)
	evidenceJSON, _ := json.Marshal(check.Evidence)
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO understanding_checks(id,session_id,turn_id,student_id,course_id,concept_id,summary,feedback_json,evidence_json,next_action,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`,
		check.ID, check.SessionID, check.TurnID, check.StudentID, check.CourseID, check.ConceptID, check.Summary, string(feedbackJSON), string(evidenceJSON), check.NextAction, check.CreatedAt)
	return err
}

func (r *SQLiteRepo) FindLatestBySession(ctx context.Context, sessionID string) (domain.UnderstandingCheck, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,session_id,turn_id,student_id,course_id,concept_id,summary,feedback_json,evidence_json,next_action,created_at FROM understanding_checks WHERE session_id=? ORDER BY created_at DESC LIMIT 1`, sessionID)
	var c domain.UnderstandingCheck
	var feedbackJSON, evidenceJSON string
	var created time.Time
	if err := row.Scan(&c.ID, &c.SessionID, &c.TurnID, &c.StudentID, &c.CourseID, &c.ConceptID, &c.Summary, &feedbackJSON, &evidenceJSON, &c.NextAction, &created); err == sql.ErrNoRows {
		return c, false, nil
	} else if err != nil {
		return c, false, err
	}
	_ = json.Unmarshal([]byte(feedbackJSON), &c.Feedback)
	_ = json.Unmarshal([]byte(evidenceJSON), &c.Evidence)
	c.CreatedAt = created
	return c, true, nil
}

func (r *SQLiteRepo) ListCheckByCourse(ctx context.Context, courseID string) ([]domain.UnderstandingCheck, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,session_id,turn_id,student_id,course_id,concept_id,summary,feedback_json,evidence_json,next_action,created_at FROM understanding_checks WHERE course_id=? ORDER BY created_at DESC`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.UnderstandingCheck
	for rows.Next() {
		var c domain.UnderstandingCheck
		var feedbackJSON, evidenceJSON string
		var created time.Time
		if err := rows.Scan(&c.ID, &c.SessionID, &c.TurnID, &c.StudentID, &c.CourseID, &c.ConceptID, &c.Summary, &feedbackJSON, &evidenceJSON, &c.NextAction, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(feedbackJSON), &c.Feedback)
		_ = json.Unmarshal([]byte(evidenceJSON), &c.Evidence)
		c.CreatedAt = created
		out = append(out, c)
	}
	return out, nil
}

func (r *SQLiteRepo) AgentLog(ctx context.Context, agentName, action string, reasoning, result map[string]any, toolCalls []map[string]any) error {
	return r.AgentLogForSession(ctx, "", agentName, action, reasoning, result, toolCalls)
}

func (r *SQLiteRepo) AgentLogForSession(ctx context.Context, sessionID, agentName, action string, reasoning, result map[string]any, toolCalls []map[string]any) error {
	reasonJSON, _ := json.Marshal(reasoning)
	toolJSON, _ := json.Marshal(toolCalls)
	resultJSON, _ := json.Marshal(result)
	_, err := r.db.ExecContext(ctx, `INSERT INTO agent_logs(session_id,agent_name,action,reasoning_json,tool_calls_json,result_json) VALUES(?,?,?,?,?,?)`, nullableString(sessionID), agentName, action, string(reasonJSON), string(toolJSON), string(resultJSON))
	return err
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *SQLiteRepo) AgentLogsForSession(ctx context.Context, sessionID string) ([]map[string]any, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT agent_name,action,reasoning_json,tool_calls_json,result_json,created_at FROM agent_logs WHERE session_id=? ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var agentName, action string
		var reasoningJSON, toolJSON, resultJSON string
		var created time.Time
		rows.Scan(&agentName, &action, &reasoningJSON, &toolJSON, &resultJSON, &created)
		var reasoning, result map[string]any
		_ = json.Unmarshal([]byte(reasoningJSON), &reasoning)
		_ = json.Unmarshal([]byte(resultJSON), &result)
		var toolCalls []map[string]any
		_ = json.Unmarshal([]byte(toolJSON), &toolCalls)
		out = append(out, map[string]any{
			"agent":     agentName,
			"action":    action,
			"reasoning": reasoning,
			"tools":     toolCalls,
			"result":    result,
			"time":      created,
		})
	}
	return out, nil
}

func (r *SQLiteRepo) UserFind(ctx context.Context, id string) (map[string]any, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,email,role,name FROM users WHERE id=?`, id)
	var idStr, email, role, name string
	if err := row.Scan(&idStr, &email, &role, &name); err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return map[string]any{"id": idStr, "email": email, "role": role, "name": name}, true, nil
}

func (r *SQLiteRepo) UserFindByEmail(ctx context.Context, email string) (map[string]any, bool, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,email,role,name FROM users WHERE email=?`, email)
	var idStr, emailStr, role, name string
	if err := row.Scan(&idStr, &emailStr, &role, &name); err == sql.ErrNoRows {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return map[string]any{"id": idStr, "email": emailStr, "role": role, "name": name}, true, nil
}

func (r *SQLiteRepo) UserCreate(ctx context.Context, id, email, hash, role, name string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO users(id,email,password_hash,role,name) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET email=excluded.email,password_hash=excluded.password_hash,role=excluded.role,name=excluded.name`, id, email, hash, role, name)
	return err
}

func (r *SQLiteRepo) CreateAccount(ctx context.Context, account auth.Account) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO users(id,email,password_hash,role,name) VALUES(?,?,?,?,?)`, account.ID, account.Email, account.PasswordHash, account.Role, account.Name)
	return err
}

func (r *SQLiteRepo) FindAccountByEmail(ctx context.Context, email string) (auth.Account, bool, error) {
	var account auth.Account
	err := r.db.QueryRowContext(ctx, `SELECT id,email,password_hash,role,name FROM users WHERE email=?`, email).Scan(&account.ID, &account.Email, &account.PasswordHash, &account.Role, &account.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Account{}, false, nil
	}
	if err != nil {
		return auth.Account{}, false, err
	}
	return account, true, nil
}

func (r *SQLiteRepo) FindAccountByID(ctx context.Context, id string) (auth.Account, bool, error) {
	var account auth.Account
	err := r.db.QueryRowContext(ctx, `SELECT id,email,password_hash,role,name FROM users WHERE id=?`, id).Scan(&account.ID, &account.Email, &account.PasswordHash, &account.Role, &account.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Account{}, false, nil
	}
	if err != nil {
		return auth.Account{}, false, err
	}
	return account, true, nil
}

// Implement repository interfaces with SQLiteRepo
