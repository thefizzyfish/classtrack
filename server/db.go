package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps a SQLite database connection and provides all data access methods.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the SQLite database at dbPath and initialises the schema.
func NewStore(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("schema init: %w", err)
	}
	log.Printf("database: %s", dbPath)
	return &Store{db: db}, nil
}

func initSchema(db *sql.DB) error {
	// Enable WAL mode and foreign key enforcement
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS classes (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			semester   TEXT NOT NULL DEFAULT '',
			archived   INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			class_id   TEXT NOT NULL REFERENCES classes(id),
			parent_id  INTEGER REFERENCES tasks(id),
			text       TEXT NOT NULL,
			task_type  TEXT NOT NULL,
			points     INTEGER NOT NULL,
			due_date   TEXT NOT NULL,
			week_date  TEXT NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
		)`,
		`CREATE TABLE IF NOT EXISTS task_events (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id     INTEGER NOT NULL REFERENCES tasks(id),
			event_type  TEXT NOT NULL,
			occurred_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
			note        TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_class_id ON tasks(class_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date)`,
		`CREATE INDEX IF NOT EXISTS idx_events_task_id ON task_events(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON task_events(occurred_at)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("schema stmt failed: %w", err)
		}
	}
	return nil
}

// GetClasses returns all non-archived classes with full task lists and computed stats.
func (s *Store) GetClasses() []Class {
	return s.loadClasses(false)
}

// GetAllClasses returns every class including archived ones (used for heatmap history).
func (s *Store) GetAllClasses() []Class {
	return s.loadClasses(true)
}

// GetClass returns a single class by ID regardless of archived status.
func (s *Store) GetClass(id string) (Class, bool) {
	var c Class
	var archivedInt int
	err := s.db.QueryRow(
		`SELECT id, name, semester, archived, created_at FROM classes WHERE id=?`, id,
	).Scan(&c.ID, &c.Name, &c.Semester, &archivedInt, &c.Created)
	if err == sql.ErrNoRows {
		return Class{}, false
	}
	if err != nil {
		log.Printf("GetClass(%s): %v", id, err)
		return Class{}, false
	}
	c.Archived = archivedInt == 1

	tasks, err := s.loadTasksForClass(id)
	if err != nil {
		log.Printf("loadTasks(%s): %v", id, err)
		tasks = []Task{}
	}
	c.Tasks = tasks
	c.Stats = computeClassStats(c)
	return c, true
}

func (s *Store) loadClasses(includeArchived bool) []Class {
	q := `SELECT id, name, semester, archived, created_at FROM classes`
	if !includeArchived {
		q += ` WHERE archived = 0`
	}
	q += ` ORDER BY name`

	rows, err := s.db.Query(q)
	if err != nil {
		log.Printf("loadClasses: %v", err)
		return nil
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var c Class
		var archivedInt int
		if err := rows.Scan(&c.ID, &c.Name, &c.Semester, &archivedInt, &c.Created); err != nil {
			log.Printf("loadClasses scan: %v", err)
			continue
		}
		c.Archived = archivedInt == 1

		tasks, err := s.loadTasksForClass(c.ID)
		if err != nil {
			log.Printf("loadTasks(%s): %v", c.ID, err)
			tasks = []Task{}
		}
		c.Tasks = tasks
		c.Stats = computeClassStats(c)
		classes = append(classes, c)
	}
	return classes
}

func (s *Store) loadTasksForClass(classID string) ([]Task, error) {
	rows, err := s.db.Query(`
		SELECT
			t.id,
			t.parent_id,
			t.text,
			t.task_type,
			t.points,
			t.due_date,
			t.week_date,
			COALESCE(e.event_type, '') AS last_event,
			CASE WHEN e.event_type = 'completed' THEN date(e.occurred_at) ELSE NULL END AS completed_at
		FROM tasks t
		LEFT JOIN task_events e ON e.id = (
			SELECT id FROM task_events
			WHERE task_id = t.id
			ORDER BY occurred_at DESC
			LIMIT 1
		)
		WHERE t.class_id = ?
		ORDER BY t.due_date, t.sort_order, t.id
	`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Collect tasks, grouping children by parent ID.
	// Two-pass: first collect all, then attach subtasks.
	childrenOf := make(map[int64][]Task)
	var topLevel []int64
	taskByID := make(map[int64]Task)

	for rows.Next() {
		var id int64
		var parentID sql.NullInt64
		var text, taskTypeStr, dueDate, week, lastEvent string
		var points int
		var completedAt sql.NullString

		if err := rows.Scan(&id, &parentID, &text, &taskTypeStr, &points,
			&dueDate, &week, &lastEvent, &completedAt); err != nil {
			return nil, err
		}

		t := Task{
			ID:        id,
			Text:      text,
			Type:      TaskType(taskTypeStr),
			Points:    points,
			DueDate:   dueDate,
			Week:      week,
			Completed: lastEvent == "completed",
			Subtasks:  []Task{},
		}
		if completedAt.Valid {
			s := completedAt.String
			t.CompletedAt = &s
		}

		taskByID[id] = t
		if parentID.Valid {
			childrenOf[parentID.Int64] = append(childrenOf[parentID.Int64], t)
		} else {
			topLevel = append(topLevel, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attach subtasks to their parents.
	result := make([]Task, 0, len(topLevel))
	for _, id := range topLevel {
		t := taskByID[id]
		if subs, ok := childrenOf[id]; ok {
			t.Subtasks = subs
		}
		result = append(result, t)
	}
	return result, nil
}

// UpdateClass sets the semester label and archived flag for an existing class.
func (s *Store) UpdateClass(id, semester string, archived bool) error {
	archivedInt := 0
	if archived {
		archivedInt = 1
	}
	res, err := s.db.Exec(
		`UPDATE classes SET semester=?, archived=? WHERE id=?`,
		semester, archivedInt, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("class not found: %s", id)
	}
	return nil
}

// CreateClass inserts a new class and returns its derived ID.
// Returns an error containing "UNIQUE" if a class with that name already exists.
func (s *Store) CreateClass(name, semester string) (string, error) {
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	_, err := s.db.Exec(
		`INSERT INTO classes (id, name, semester, created_at) VALUES (?, ?, ?, ?)`,
		id, name, semester, time.Now().Format("2006-01-02"),
	)
	return id, err
}

// ToggleTask inserts a 'completed' or 'uncompleted' event for the given task ID.
// The most recent event wins, so history is never destroyed.
func (s *Store) ToggleTask(taskID int64, completed bool) error {
	eventType := "uncompleted"
	if completed {
		eventType = "completed"
	}
	_, err := s.db.Exec(
		`INSERT INTO task_events (task_id, event_type) VALUES (?, ?)`,
		taskID, eventType,
	)
	return err
}

// ImportCanvasTasks bulk-inserts parsed Canvas tasks, skipping exact duplicates
// (same class + due_date + text). Returns the count of newly inserted tasks.
func (s *Store) ImportCanvasTasks(classID string, tasks []CanvasTask) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var maxOrder int
	tx.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) FROM tasks WHERE class_id = ?`, classID).Scan(&maxOrder)

	imported := 0
	for _, ct := range tasks {
		dueDate := ct.Date.Format("2006-01-02")

		// Skip exact duplicates within this class
		var count int
		tx.QueryRow(
			`SELECT COUNT(*) FROM tasks WHERE class_id=? AND due_date=? AND text=?`,
			classID, dueDate, ct.Text,
		).Scan(&count)
		if count > 0 {
			continue
		}

		tt := classifyTask(ct.Text)
		maxOrder++
		_, err := tx.Exec(
			`INSERT INTO tasks (class_id, text, task_type, points, due_date, week_date, sort_order)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			classID, ct.Text, string(tt), taskPoints[tt],
			dueDate, sundayOfWeek(ct.Date).Format("2006-01-02"), maxOrder,
		)
		if err != nil {
			return 0, err
		}
		imported++
	}
	return imported, tx.Commit()
}
