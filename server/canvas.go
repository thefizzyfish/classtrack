package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var canvasDateRe = regexp.MustCompile(
	`^(Sun|Mon|Tue|Wed|Thu|Fri|Sat)\s+(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+(\d{1,2}),\s+(\d{4})`,
)

var junkPhrases = []string{
	"refund date", "withdrawal", "initial post due", "last day",
	"assignment start", "assignment end", "nvrk",
}

type CanvasTask struct {
	Date time.Time
	Text string
}

func parseCanvasPaste(raw string, baseYear int) ([]CanvasTask, error) {
	var tasks []CanvasTask
	var currentDate time.Time
	first := true

	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// skip the header row
		if first {
			first = false
			if strings.HasPrefix(strings.ToLower(trimmed), "date\t") ||
				strings.ToLower(trimmed) == "date" {
				continue
			}
		}

		// split on tab to get columns
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
			// empty first column, task in second
			taskText = strings.TrimSpace(parts[1])
		default:
			// no date column at all — whole thing is a task name
			taskText = strings.TrimSpace(parts[0])
		}

		if taskText == "" || currentDate.IsZero() {
			continue
		}

		// strip trailing "due by ..." that may be embedded
		if idx := strings.Index(taskText, "\t"); idx != -1 {
			taskText = taskText[:idx]
		}

		// strip type prefix
		taskText = stripCanvasPrefix(taskText)
		taskText = strings.TrimSpace(taskText)

		if isJunk(taskText) {
			continue
		}

		tasks = append(tasks, CanvasTask{Date: currentDate, Text: taskText})
	}
	return tasks, nil
}

func stripCanvasPrefix(s string) string {
	for _, p := range []string{"Discussion Topic ", "Assignment ", "Quiz "} {
		if strings.HasPrefix(s, p) {
			return strings.TrimPrefix(s, p)
		}
	}
	return s
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

// appendCanvasTasks writes new tasks into the todo file, grouped under ## Mon DD headers.
// Tasks whose date header already exists in the file are skipped (to avoid duplicates).
func appendCanvasTasks(filePath string, tasks []CanvasTask) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	// find which "Mon DD" headers already exist
	existing := make(map[string]bool)
	for _, l := range lines {
		if m := h2DateRe.FindStringSubmatch(l); m != nil {
			existing[m[1]+"-"+m[2]] = true
		}
	}

	// group tasks by date, preserving order
	type group struct {
		key   string // "Mon-D" for dedup check
		label string // "## Mon D"
		tasks []string
	}
	var groups []group
	groupIdx := map[string]int{}

	monthAbbrev := map[time.Month]string{
		time.January: "Jan", time.February: "Feb", time.March: "Mar",
		time.April: "Apr", time.May: "May", time.June: "Jun",
		time.July: "Jul", time.August: "Aug", time.September: "Sep",
		time.October: "Oct", time.November: "Nov", time.December: "Dec",
	}

	for _, t := range tasks {
		abbr := monthAbbrev[t.Date.Month()]
		key := fmt.Sprintf("%s-%d", abbr, t.Date.Day())
		if existing[key] {
			continue
		}
		if _, ok := groupIdx[key]; !ok {
			groupIdx[key] = len(groups)
			groups = append(groups, group{
				key:   key,
				label: fmt.Sprintf("## %s %d", abbr, t.Date.Day()),
			})
		}
		idx := groupIdx[key]
		groups[idx].tasks = append(groups[idx].tasks, "- [ ] "+t.Text)
	}

	if len(groups) == 0 {
		return nil // nothing to add
	}

	var appended []string
	for _, g := range groups {
		appended = append(appended, "", g.label)
		appended = append(appended, g.tasks...)
	}

	lines = append(lines, appended...)
	return writeLines(filePath, lines)
}
