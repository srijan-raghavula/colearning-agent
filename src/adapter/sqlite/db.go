package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func OpenDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create sqlite data directory: %w", err)
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("secure sqlite data directory: %w", err)
	}
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(0)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("secure sqlite database file: %w", err)
	}
	return db, nil
}

func Migrate(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('student','teacher')),
    name TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS courses (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    subject TEXT,
    description TEXT,
    teacher_id TEXT NOT NULL REFERENCES users(id),
    tutor_policy_json TEXT DEFAULT '{}',
    next_concept_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS concepts (
    id TEXT PRIMARY KEY,
    course_id TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    summary TEXT,
    level TEXT,
    "order" INTEGER DEFAULT 1,
    estimated_minutes INTEGER,
    prerequisites_json TEXT DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_concepts_course ON concepts(course_id);
CREATE TABLE IF NOT EXISTS objectives (
    id TEXT PRIMARY KEY,
    course_id TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    concept_id TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    description TEXT,
    outcomes_json TEXT DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_objectives_course ON objectives(course_id);
CREATE TABLE IF NOT EXISTS materials (
    id TEXT PRIMARY KEY,
    course_id TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    concept_id TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT,
    source_kind TEXT CHECK(source_kind IN ('teacher','external')) DEFAULT 'teacher',
    source_ref TEXT,
    published INTEGER DEFAULT 1,
    version INTEGER DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_materials_course ON materials(course_id);
CREATE TABLE IF NOT EXISTS learning_sessions (
    id TEXT PRIMARY KEY,
    student_id TEXT NOT NULL REFERENCES users(id),
    course_id TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    concept_id TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    mode TEXT DEFAULT 'guided',
    status TEXT DEFAULT 'active',
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_active_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_sessions_student ON learning_sessions(student_id);
CREATE INDEX IF NOT EXISTS idx_sessions_course ON learning_sessions(course_id);
CREATE TABLE IF NOT EXISTS learning_turns (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES learning_sessions(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK(role IN ('student','tutor')),
    kind TEXT,
    mode TEXT,
    body TEXT,
    sources_json TEXT DEFAULT '[]',
    confidence_delta INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_turns_session ON learning_turns(session_id);
CREATE TABLE IF NOT EXISTS concept_progress (
    student_id TEXT NOT NULL REFERENCES users(id),
    course_id TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    concept_id TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    status TEXT DEFAULT 'not_started',
    confidence INTEGER DEFAULT 0,
    evidence_count INTEGER DEFAULT 0,
    next_action TEXT,
    last_activity_at DATETIME,
    PRIMARY KEY (student_id, course_id, concept_id)
);
CREATE INDEX IF NOT EXISTS idx_progress_student ON concept_progress(student_id);
CREATE TABLE IF NOT EXISTS learning_evidence (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES learning_sessions(id) ON DELETE CASCADE,
    turn_id TEXT NOT NULL REFERENCES learning_turns(id),
    student_id TEXT NOT NULL REFERENCES users(id),
    course_id TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    concept_id TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    kind TEXT,
    note TEXT,
    observed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_evidence_session ON learning_evidence(session_id);
CREATE INDEX IF NOT EXISTS idx_evidence_course ON learning_evidence(course_id);
CREATE TABLE IF NOT EXISTS understanding_checks (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES learning_sessions(id) ON DELETE CASCADE,
    turn_id TEXT NOT NULL REFERENCES learning_turns(id),
    student_id TEXT NOT NULL REFERENCES users(id),
    course_id TEXT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    concept_id TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    summary TEXT,
    feedback_json TEXT DEFAULT '[]',
    evidence_json TEXT DEFAULT '[]',
    next_action TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_checks_session ON understanding_checks(session_id);
CREATE TABLE IF NOT EXISTS agent_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT,
    agent_name TEXT NOT NULL,
    action TEXT NOT NULL,
    reasoning_json TEXT,
    tool_calls_json TEXT DEFAULT '[]',
    result_json TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_agent_logs_session ON agent_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_agent_logs_agent ON agent_logs(agent_name);
`
	_, err := db.Exec(schema)
	return err
}
