package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	h1Re        = regexp.MustCompile(`^# (.+)$`)
	h2DateRe    = regexp.MustCompile(`^## (Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (\d{1,2})(?:,\s*(\d{4}))?`)
	taskRe      = regexp.MustCompile(`^(\s*)- \[([ x])\] (.+?)(?:\s+✅\s+(\d{4}-\d{2}-\d{2}))?$`)
	createdRe    = regexp.MustCompile(`created:\s*(\d{4}-\d{2}-\d{2})`)
	archivedRe   = regexp.MustCompile(`archived:\s*true`)
	semesterRe   = regexp.MustCompile(`semester:\s*(.+)`)
	frontmatterRe = regexp.MustCompile(`^---\s*$`)
)

var months = map[string]time.Month{
	"Jan": time.January, "Feb": time.February, "Mar": time.March,
	"Apr": time.April, "May": time.May, "Jun": time.June,
	"Jul": time.July, "Aug": time.August, "Sep": time.September,
	"Oct": time.October, "Nov": time.November, "Dec": time.December,
}

func discoverClasses(vaultPath string) ([]Class, error) {
	var classes []Class
	err := filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), "Todo.md") {
			class, err := parseTodoFile(path)
			if err != nil {
				return fmt.Errorf("parse %s: %w", path, err)
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
		File:  path,
		Tasks: []Task{}, // ensure JSON serializes as [] not null
	}

	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// --- parse frontmatter ---
	createdYear := time.Now().Year()
	inFM := false
	contentStart := 0

	for i, line := range lines {
		if i == 0 && frontmatterRe.MatchString(line) {
			inFM = true
			continue
		}
		if inFM {
			if frontmatterRe.MatchString(line) {
				contentStart = i + 1
				break
			}
			if m := createdRe.FindStringSubmatch(line); m != nil {
				class.Created = m[1]
				t, _ := time.Parse("2006-01-02", m[1])
				createdYear = t.Year()
			}
			if archivedRe.MatchString(line) {
				class.Archived = true
			}
			if m := semesterRe.FindStringSubmatch(line); m != nil {
				class.Semester = strings.TrimSpace(m[1])
			}
		}
	}

	// --- parse content ---
	var currentDueDate time.Time
	var currentWeek time.Time
	nameSet := false
	currentParentIdx := -1

	for i := contentStart; i < len(lines); i++ {
		line := lines[i]

		// H1 — class name (first one only)
		if m := h1Re.FindStringSubmatch(line); m != nil {
			if !nameSet {
				class.Name = strings.TrimSuffix(strings.TrimSuffix(m[1], " Todo"), " todo")
				nameSet = true
			}
			continue
		}

		// H2 date header
		if m := h2DateRe.FindStringSubmatch(line); m != nil {
			month := months[m[1]]
			var day int
			fmt.Sscanf(m[2], "%d", &day)
			year := createdYear
			if m[3] != "" {
				fmt.Sscanf(m[3], "%d", &year)
			}
			currentDueDate = time.Date(year, month, day, 0, 0, 0, 0, time.Local)
			// Sunday of this week
			offset := int(currentDueDate.Weekday())
			currentWeek = currentDueDate.AddDate(0, 0, -offset)
			currentParentIdx = -1
			continue
		}

		// task line
		if m := taskRe.FindStringSubmatch(line); m != nil {
			if currentDueDate.IsZero() {
				continue
			}
			indent := len(m[1])
			completed := m[2] == "x"
			text := strings.TrimSpace(m[3])
			var completedAt *string
			if m[4] != "" {
				s := m[4]
				completedAt = &s
			}

			task := Task{
				ID:          fmt.Sprintf("%s:%d", id, i),
				Text:        text,
				Completed:   completed,
				CompletedAt: completedAt,
				DueDate:     currentDueDate.Format("2006-01-02"),
				Week:        currentWeek.Format("2006-01-02"),
				Type:        classifyTask(text),
				Points:      taskPoints[classifyTask(text)],
				Indent:      indent,
				lineNum:     i,
				filePath:    path,
				classID:     id,
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

	class.Stats = computeClassStats(class)
	return class, nil
}

func classifyTask(text string) TaskType {
	lower := strings.ToLower(text)
	if containsAny(lower, "exam", "honorlock", "midterm", "final exam") {
		return TaskTypeExam
	}
	if containsAny(lower, "quiz") {
		return TaskTypeQuiz
	}
	if containsAny(lower, "discussion") {
		return TaskTypeDiscussion
	}
	if containsAny(lower, "assignment", "lab", "simulation", "capstone", "project", "portfolio") {
		return TaskTypeAssignment
	}
	return TaskTypeOther
}

func containsAny(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
