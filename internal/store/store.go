package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Problem struct {
	FrontendID      string
	QuestionID      string
	Slug            string
	Title           string
	Difficulty      string
	Topics          []string
	StatementHTML   string
	ExampleTests    string
	CodeStub        string
	Status          string
	TimeSpentSec    int
	LastSubmit      string
	Runtime         string
	Memory          string
	LastFetchedUnix int64
}

type ProblemRow struct {
	Problem
	UpdatedAt string
}

type Stats struct {
	TotalProblems   int
	SolvedEasy      int
	SolvedMedium    int
	SolvedHard      int
	AvgSolveSec     float64
	TopicCoverage   int
	RecentActivity  []Activity
	SolvedLast7Days int
	SolvedPrev7Days int
}

type Activity struct {
	Slug      string `json:"slug"`
	Kind      string `json:"kind"`
	Payload   string `json:"payload"`
	CreatedAt string `json:"created_at"`
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS problems (
  slug TEXT PRIMARY KEY,
  frontend_id TEXT,
  question_id TEXT,
  title TEXT NOT NULL,
  difficulty TEXT NOT NULL,
  topics_json TEXT NOT NULL DEFAULT '[]',
  statement_html TEXT NOT NULL DEFAULT '',
  example_tests TEXT NOT NULL DEFAULT '',
  code_stub TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'todo',
  time_spent_sec INTEGER NOT NULL DEFAULT 0,
  last_submit TEXT NOT NULL DEFAULT '',
  runtime TEXT NOT NULL DEFAULT '',
  memory TEXT NOT NULL DEFAULT '',
  last_fetched_unix INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL,
  note TEXT NOT NULL,
  tags_json TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS timer_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL,
  start_unix INTEGER NOT NULL,
  end_unix INTEGER,
  target_minutes INTEGER NOT NULL DEFAULT 30,
  manual INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS custom_tests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL,
  input_json TEXT NOT NULL,
  expected_json TEXT,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS test_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL,
  passed INTEGER NOT NULL,
  failed_count INTEGER NOT NULL DEFAULT 0,
  output TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS activity (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT NOT NULL,
  kind TEXT NOT NULL,
  payload TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
`
	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	return nil
}

func (s *Store) UpsertProblem(ctx context.Context, p Problem) error {
	topics, _ := json.Marshal(p.Topics)
	if p.Status == "" {
		p.Status = "todo"
	}
	if p.LastFetchedUnix == 0 {
		p.LastFetchedUnix = time.Now().Unix()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO problems (slug, frontend_id, question_id, title, difficulty, topics_json, statement_html, example_tests, code_stub, status, time_spent_sec, last_submit, runtime, memory, last_fetched_unix, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(NULLIF(?, ''), 'todo'), ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(slug) DO UPDATE SET
  frontend_id=excluded.frontend_id,
  question_id=excluded.question_id,
  title=excluded.title,
  difficulty=excluded.difficulty,
  topics_json=excluded.topics_json,
  statement_html=excluded.statement_html,
  example_tests=excluded.example_tests,
  code_stub=excluded.code_stub,
  last_fetched_unix=excluded.last_fetched_unix,
  updated_at=CURRENT_TIMESTAMP
`, p.Slug, p.FrontendID, p.QuestionID, p.Title, p.Difficulty, string(topics), p.StatementHTML, p.ExampleTests, p.CodeStub, p.Status, p.TimeSpentSec, p.LastSubmit, p.Runtime, p.Memory, p.LastFetchedUnix)
	if err != nil {
		return fmt.Errorf("upsert problem: %w", err)
	}
	return nil
}

func (s *Store) GetProblem(ctx context.Context, slug string) (ProblemRow, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT slug, frontend_id, question_id, title, difficulty, topics_json, statement_html, example_tests, code_stub, status, time_spent_sec, last_submit, runtime, memory, last_fetched_unix, updated_at
FROM problems WHERE slug = ?`, slug)
	var pr ProblemRow
	var topicsJSON string
	if err := row.Scan(&pr.Slug, &pr.FrontendID, &pr.QuestionID, &pr.Title, &pr.Difficulty, &topicsJSON, &pr.StatementHTML, &pr.ExampleTests, &pr.CodeStub, &pr.Status, &pr.TimeSpentSec, &pr.LastSubmit, &pr.Runtime, &pr.Memory, &pr.LastFetchedUnix, &pr.UpdatedAt); err != nil {
		return ProblemRow{}, err
	}
	_ = json.Unmarshal([]byte(topicsJSON), &pr.Topics)
	return pr, nil
}

func (s *Store) ListProblems(ctx context.Context, difficulty, status, query string) ([]ProblemRow, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT slug, frontend_id, question_id, title, difficulty, topics_json, statement_html, example_tests, code_stub, status, time_spent_sec, last_submit, runtime, memory, last_fetched_unix, updated_at
FROM problems
WHERE (? = '' OR difficulty = ?)
  AND (? = '' OR status = ?)
  AND (? = '' OR slug LIKE '%' || ? || '%' OR title LIKE '%' || ? || '%')
ORDER BY CAST(frontend_id AS INTEGER) ASC, slug ASC
`, difficulty, difficulty, status, status, query, query, query)
	if err != nil {
		return nil, fmt.Errorf("list problems: %w", err)
	}
	defer rows.Close()

	out := make([]ProblemRow, 0)
	for rows.Next() {
		var pr ProblemRow
		var topicsJSON string
		if err := rows.Scan(&pr.Slug, &pr.FrontendID, &pr.QuestionID, &pr.Title, &pr.Difficulty, &topicsJSON, &pr.StatementHTML, &pr.ExampleTests, &pr.CodeStub, &pr.Status, &pr.TimeSpentSec, &pr.LastSubmit, &pr.Runtime, &pr.Memory, &pr.LastFetchedUnix, &pr.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(topicsJSON), &pr.Topics)
		out = append(out, pr)
	}
	return out, rows.Err()
}

func (s *Store) SetProblemStatus(ctx context.Context, slug, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE problems SET status=?, updated_at=CURRENT_TIMESTAMP WHERE slug=?`, status, slug)
	if err != nil {
		return fmt.Errorf("set status: %w", err)
	}
	if status == "solved" {
		_, _ = s.db.ExecContext(ctx, `INSERT INTO activity(slug, kind, payload) VALUES(?, 'solved', '')`, slug)
	}
	return nil
}

func (s *Store) AddNote(ctx context.Context, slug, note string, tags []string) error {
	t, _ := json.Marshal(tags)
	_, err := s.db.ExecContext(ctx, `INSERT INTO notes(slug, note, tags_json) VALUES(?, ?, ?)`, slug, note, string(t))
	if err != nil {
		return fmt.Errorf("add note: %w", err)
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO activity(slug, kind, payload) VALUES(?, 'note', ?)`, slug, note)
	return nil
}

func (s *Store) StartTimer(ctx context.Context, slug string, targetMinutes int, manual bool) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO timer_sessions(slug, start_unix, target_minutes, manual) VALUES(?, ?, ?, ?)`, slug, time.Now().Unix(), targetMinutes, boolToInt(manual))
	if err != nil {
		return fmt.Errorf("start timer: %w", err)
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO activity(slug, kind, payload) VALUES(?, 'timer_start', ?)`, slug, fmt.Sprintf("%d", targetMinutes))
	return nil
}

func (s *Store) StopTimer(ctx context.Context, slug string) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, start_unix FROM timer_sessions WHERE slug=? AND end_unix IS NULL ORDER BY id DESC LIMIT 1`, slug)
	var id int64
	var start int64
	if err := row.Scan(&id, &start); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("find active timer: %w", err)
	}
	now := time.Now().Unix()
	dur := int(now - start)
	_, err := s.db.ExecContext(ctx, `UPDATE timer_sessions SET end_unix=? WHERE id=?`, now, id)
	if err != nil {
		return 0, fmt.Errorf("stop timer: %w", err)
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE problems SET time_spent_sec=time_spent_sec+?, updated_at=CURRENT_TIMESTAMP WHERE slug=?`, dur, slug)
	_, _ = s.db.ExecContext(ctx, `INSERT INTO activity(slug, kind, payload) VALUES(?, 'timer_stop', ?)`, slug, fmt.Sprintf("%d", dur))
	return dur, nil
}

func (s *Store) AddManualTime(ctx context.Context, slug string, minutes int) error {
	sec := minutes * 60
	_, err := s.db.ExecContext(ctx, `UPDATE problems SET time_spent_sec=time_spent_sec+?, updated_at=CURRENT_TIMESTAMP WHERE slug=?`, sec, slug)
	if err != nil {
		return fmt.Errorf("add manual time: %w", err)
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO activity(slug, kind, payload) VALUES(?, 'manual_time', ?)`, slug, fmt.Sprintf("%d", minutes))
	return nil
}

func (s *Store) SetCurrentProblem(ctx context.Context, slug string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES('current_slug', ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, slug)
	return err
}

func (s *Store) CurrentProblem(ctx context.Context) (string, error) {
	row := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key='current_slug'`)
	var slug string
	if err := row.Scan(&slug); err != nil {
		return "", err
	}
	return slug, nil
}

func (s *Store) SaveTestRun(ctx context.Context, slug string, passed bool, failed int, output string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO test_runs(slug, passed, failed_count, output) VALUES(?, ?, ?, ?)`, slug, boolToInt(passed), failed, output)
	if err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO activity(slug, kind, payload) VALUES(?, 'test', ?)`, slug, fmt.Sprintf("passed=%t failed=%d", passed, failed))
	return nil
}

func (s *Store) SaveSubmissionResult(ctx context.Context, slug, status, runtime, memory string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE problems SET last_submit=?, runtime=?, memory=?, updated_at=CURRENT_TIMESTAMP WHERE slug=?`, status, runtime, memory, slug)
	if err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO activity(slug, kind, payload) VALUES(?, 'submit', ?)`, slug, status)
	if status == "Accepted" {
		_, _ = s.db.ExecContext(ctx, `UPDATE problems SET status='solved' WHERE slug=?`, slug)
	}
	return nil
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM problems`).Scan(&st.TotalProblems)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM problems WHERE status='solved' AND difficulty='Easy'`).Scan(&st.SolvedEasy)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM problems WHERE status='solved' AND difficulty='Medium'`).Scan(&st.SolvedMedium)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM problems WHERE status='solved' AND difficulty='Hard'`).Scan(&st.SolvedHard)
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(AVG(time_spent_sec), 0) FROM problems WHERE status='solved' AND time_spent_sec > 0`).Scan(&st.AvgSolveSec)
	_ = s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM (
  SELECT DISTINCT value
  FROM problems, json_each(problems.topics_json)
  WHERE status='solved'
)`).Scan(&st.TopicCoverage)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM activity WHERE kind='solved' AND datetime(created_at) >= datetime('now','-7 day')`).Scan(&st.SolvedLast7Days)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM activity WHERE kind='solved' AND datetime(created_at) < datetime('now','-7 day') AND datetime(created_at) >= datetime('now','-14 day')`).Scan(&st.SolvedPrev7Days)

	rows, err := s.db.QueryContext(ctx, `SELECT slug, kind, payload, created_at FROM activity ORDER BY id DESC LIMIT 10`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a Activity
			if scanErr := rows.Scan(&a.Slug, &a.Kind, &a.Payload, &a.CreatedAt); scanErr == nil {
				st.RecentActivity = append(st.RecentActivity, a)
			}
		}
	}
	return st, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
