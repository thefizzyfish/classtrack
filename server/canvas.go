package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var canvasDateRe = regexp.MustCompile(
	`^(Sun|Mon|Tue|Wed|Thu|Fri|Sat)\s+(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2}),\s+(\d{4})`,
)

// canvasTimeContRe matches the wrapped second line of a calendar event's time
// range ("1pm\nto 2pm"), which would otherwise be parsed as its own task.
var canvasTimeContRe = regexp.MustCompile(`(?i)^to\s+\d{1,2}(:\d{2})?\s*(am|pm)$`)

var junkPhrases = []string{
	"refund date", "withdrawal", "initial post due", "last day",
	"assignment start", "assignment end", "nvrk",
}

// CanvasTask is a single task parsed from a Canvas calendar paste.
// HintType carries the task type inferred from the Canvas prefix (e.g. "Discussion Topic ")
// so it survives prefix-stripping. Empty string means fall back to keyword classification.
type CanvasTask struct {
	Date     time.Time
	Text     string
	HintType TaskType // set from Canvas prefix; empty = use classifyTask
}

// parseCanvasPaste parses the tab-separated Canvas calendar export.
// The date column carries forward when blank (Canvas's sparse format).
// baseYear is accepted for API compatibility but Canvas dates include the year.
func parseCanvasPaste(raw string, baseYear int) ([]CanvasTask, error) {
	var tasks []CanvasTask
	var currentDate time.Time
	first := true

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || canvasTimeContRe.MatchString(trimmed) {
			continue
		}

		// Skip the header row
		if first {
			first = false
			if strings.HasPrefix(strings.ToLower(trimmed), "date\t") ||
				strings.ToLower(trimmed) == "date" {
				continue
			}
		}

		parts := strings.Split(line, "\t")
		var taskText string

		switch {
		case canvasDateRe.MatchString(parts[0]):
			// "Sun Mar 29, 2026\tTask name\t..."
			m := canvasDateRe.FindStringSubmatch(parts[0])
			month := months[m[2]]
			var day, year int
			fmt.Sscanf(m[3], "%d", &day)
			fmt.Sscanf(m[4], "%d", &year)
			currentDate = time.Date(year, month, day, 0, 0, 0, 0, time.Local)
			if len(parts) > 1 {
				taskText = strings.TrimSpace(parts[1])
			}
		case strings.TrimSpace(parts[0]) == "" && len(parts) > 1:
			// Blank first column — date carries forward
			taskText = strings.TrimSpace(parts[1])
		default:
			// No date column at all
			taskText = strings.TrimSpace(parts[0])
		}

		if taskText == "" || currentDate.IsZero() {
			continue
		}

		// Strip embedded tab suffix ("due by ...")
		if idx := strings.Index(taskText, "\t"); idx != -1 {
			taskText = taskText[:idx]
		}

		hint, taskText := stripCanvasPrefix(taskText)
		taskText = strings.TrimSpace(taskText)
		if isJunk(taskText) {
			continue
		}

		tasks = append(tasks, CanvasTask{Date: currentDate, Text: taskText, HintType: hint})
	}
	return tasks, nil
}

// stripCanvasPrefix removes the Canvas type prefix from a task name and returns
// both the inferred TaskType hint and the stripped text.
// If no known prefix is found, hint is empty and the text is returned unchanged.
func stripCanvasPrefix(s string) (hint TaskType, text string) {
	prefixes := []struct {
		prefix string
		hint   TaskType
	}{
		{"Discussion Topic ", TaskTypeDiscussion},
		{"Assignment ", TaskTypeAssignment},
		{"Quiz ", TaskTypeQuiz},
	}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p.prefix) {
			return p.hint, strings.TrimPrefix(s, p.prefix)
		}
	}
	return "", s
}

func isJunk(s string) bool {
	lower := strings.ToLower(s)
	for _, j := range junkPhrases {
		if strings.Contains(lower, j) {
			return true
		}
	}
	return false
}
