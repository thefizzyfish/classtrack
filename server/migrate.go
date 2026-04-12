package main

// migrate.go — one-time import of Obsidian markdown todo files into SQLite.
// Run with: go run . --migrate [--vault /path/to/NVCC]
//
// After a successful migration this file can be deleted along with parser.go.
// The main server binary does not call any of these functions at runtime.

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// runMigration walks vaultPath for *Todo.md files and inserts all classes,
// tasks, and historical completion events into the database.
// Uses INSERT OR IGNORE for classes so it is safe to re-run after a partial failure.
func runMigration(db *sql.DB, vaultPath string) error {
	classes, err := discoverClasses(vaultPath)
	if err != nil {
		return fmt.Errorf("discover: %w", err)
	}

	totalTasks, totalEvents := 0, 0

	for _, class := range classes {
		if _, err := db.Exec(
			`INSERT OR IGNORE INTO classes (id, name, semester, archived, created_at) VALUES (?, ?, ?, ?, ?)`,
			class.ID, class.Name, class.Semester, boolToInt(class.Archived), class.Created,
		); err != nil {
			return fmt.Errorf("insert class %s: %w", class.ID, err)
		}

		sortOrder := 0
		for _, task := range class.Tasks {
			sortOrder++
			result, err := db.Exec(
				`INSERT INTO tasks (class_id, parent_id, text, task_type, points, due_date, week_date, sort_order)
				 VALUES (?, NULL, ?, ?, ?, ?, ?, ?)`,
				class.ID, task.Text, string(task.Type), task.Points,
				task.DueDate, task.Week, sortOrder,
			)
			if err != nil {
				return fmt.Errorf("insert task %q: %w", task.Text, err)
			}
			taskID, _ := result.LastInsertId()
			totalTasks++

			if task.CompletedAt != nil {
				if _, err := db.Exec(
					`INSERT INTO task_events (task_id, event_type, occurred_at, note)
					 VALUES (?, 'completed', ?, 'migrated from markdown')`,
					taskID, *task.CompletedAt,
				); err != nil {
					return fmt.Errorf("insert event for task %d: %w", taskID, err)
				}
				totalEvents++
			}

			for _, sub := range task.Subtasks {
				sortOrder++
				result2, err := db.Exec(
					`INSERT INTO tasks (class_id, parent_id, text, task_type, points, due_date, week_date, sort_order)
					 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
					class.ID, taskID, sub.Text, string(sub.Type), sub.Points,
					sub.DueDate, sub.Week, sortOrder,
				)
				if err != nil {
					return fmt.Errorf("insert subtask %q: %w", sub.Text, err)
				}
				subID, _ := result2.LastInsertId()
				totalTasks++

				if sub.CompletedAt != nil {
					if _, err := db.Exec(
						`INSERT INTO task_events (task_id, event_type, occurred_at, note)
						 VALUES (?, 'completed', ?, 'migrated from markdown')`,
						subID, *sub.CompletedAt,
					); err != nil {
						return fmt.Errorf("insert event for subtask %d: %w", subID, err)
					}
					totalEvents++
				}
			}
		}
		log.Printf("migrated: %s — %d tasks", class.ID, len(class.Tasks))
	}

	log.Printf("migration complete: %d classes, %d tasks, %d completion events",
		len(classes), totalTasks, totalEvents)
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- Markdown parser (migration only) ---

var (
	migrH1Re       = regexp.MustCompile(`^# (.+)$`)
	migrH2DateRe   = regexp.MustCompile(`^## (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (\d{1,2})(?:,\s*(\d{4}))?`)
	migrTaskRe     = regexp.MustCompile(`^(\s*)- \[([ x])\] (.+?)(?:\s+✅\s+(\d{4}-\d{2}-\d{2}))?$`)
	migrCreatedRe  = regexp.MustCompile(`created:\s*(\d{4}-\d{2}-\d{2})`)
	migrArchivedRe = regexp.MustCompile(`archived:\s*true`)
	migrSemesterRe = regexp.MustCompile(`semester:\s*(.+)`)
	migrFMRe       = regexp.MustCompile(`^---\s*$`)
)

func discoverClasses(vaultPath string) ([]Class, error) {
	var classes []Class
	err := filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), "Todo.md") {
			class, parseErr := parseTodoFile(path)
			if parseErr != nil {
				return fmt.Errorf("parse %s: %w", path, parseErr)
			}
			classes = append(classes, class)
		}
		return nil
	})
	return classes, err
}

func parseTodoFile(path string) (Class, error) {
	f, err := os.Open(path)
	if err != nil {
		return Class{}, err
	}
	defer f.Close()

	dir := filepath.Base(filepath.Dir(path))
	id := strings.ToLower(strings.ReplaceAll(dir, " ", "-"))

	class := Class{
		ID:    id,
		Tasks: []Task{},
	}

	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Parse frontmatter
	createdYear := time.Now().Year()
	inFM := false
	contentStart := 0

	for i, line := range lines {
		if i == 0 && migrFMRe.MatchString(line) {
			inFM = true
			continue
		}
		if inFM {
			if migrFMRe.MatchString(line) {
				contentStart = i + 1
				break
			}
			if m := migrCreatedRe.FindStringSubmatch(line); m != nil {
				class.Created = m[1]
				if t, err := time.Parse("2006-01-02", m[1]); err == nil {
					createdYear = t.Year()
				}
			}
			if migrArchivedRe.MatchString(line) {
				class.Archived = true
			}
			if m := migrSemesterRe.FindStringSubmatch(line); m != nil {
				class.Semester = strings.TrimSpace(m[1])
			}
		}
	}

	// Parse content
	var currentDueDate time.Time
	var currentWeek time.Time
	nameSet := false
	currentParentIdx := -1

	for i := contentStart; i < len(lines); i++ {
		line := lines[i]

		if m := migrH1Re.FindStringSubmatch(line); m != nil {
			if !nameSet {
				class.Name = strings.TrimSuffix(strings.TrimSuffix(m[1], " Todo"), " todo")
				nameSet = true
			}
			continue
		}

		if m := migrH2DateRe.FindStringSubmatch(line); m != nil {
			month := months[m[1]]
			var day int
			fmt.Sscanf(m[2], "%d", &day)
			year := createdYear
			if m[3] != "" {
				fmt.Sscanf(m[3], "%d", &year)
			}
			currentDueDate = time.Date(year, month, day, 0, 0, 0, 0, time.Local)
			currentWeek = sundayOfWeek(currentDueDate)
			currentParentIdx = -1
			continue
		}

		if m := migrTaskRe.FindStringSubmatch(line); m != nil {
			if currentDueDate.IsZero() {
				continue
			}
			indent := len(m[1])
			text := strings.TrimSpace(m[3])
			var completedAt *string
			if m[4] != "" {
				s := m[4]
				completedAt = &s
			}
			tt := classifyTask(text)
			task := Task{
				Text:        text,
				Completed:   m[2] == "x",
				CompletedAt: completedAt,
				DueDate:     currentDueDate.Format("2006-01-02"),
				Week:        currentWeek.Format("2006-01-02"),
				Type:        tt,
				Points:      taskPoints[tt],
				Subtasks:    []Task{},
			}

			if indent == 0 {
				class.Tasks = append(class.Tasks, task)
				currentParentIdx = len(class.Tasks) - 1
			} else if currentParentIdx >= 0 {
				class.Tasks[currentParentIdx].Subtasks = append(
					class.Tasks[currentParentIdx].Subtasks, task,
				)
			}
		}
	}

	return class, nil
}
