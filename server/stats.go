package main

import (
	"sort"
	"time"
)

// localToday returns today's date as a UTC-midnight time so it compares
// correctly against dates from time.Parse("2006-01-02", ...) which are also UTC.
func localToday() time.Time {
	now := time.Now().Local()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func computeClassStats(class Class) ClassStats {
	s := ClassStats{}
	today := localToday()

	var dueDates []time.Time

	for _, task := range class.Tasks {
		if task.DueDate == "" {
			continue
		}
		s.Total++
		s.TotalPoints += task.Points
		if task.Completed {
			s.Completed++
			s.EarnedPoints += task.Points
		}
		t, _ := time.Parse("2006-01-02", task.DueDate)
		dueDates = append(dueDates, t)
	}

	if s.Total > 0 {
		s.Percentage = float64(s.Completed) / float64(s.Total) * 100
	}

	// On-pace: compare task completion % vs time elapsed through the semester
	if len(dueDates) >= 2 {
		sort.Slice(dueDates, func(i, j int) bool { return dueDates[i].Before(dueDates[j]) })
		start := dueDates[0]
		end := dueDates[len(dueDates)-1]
		totalDays := end.Sub(start).Hours() / 24

		if totalDays > 0 {
			elapsed := today.Sub(start).Hours() / 24
			if elapsed < 0 {
				elapsed = 0
			}
			if elapsed > totalDays {
				elapsed = totalDays
			}
			timeProgress := elapsed / totalDays
			taskProgress := float64(s.Completed) / float64(s.Total)
			// positive = ahead of pace, negative = behind
			s.OnPaceScore = (taskProgress - timeProgress) * 100
		}
	}

	s.Streak = computeStreak(class)
	return s
}

// computeStreak counts consecutive days ending on today where at least one
// task was completed.  Uses string comparison to avoid timezone ambiguity.
func computeStreak(class Class) int {
	dateSet := make(map[string]bool)
	for _, task := range class.Tasks {
		if task.CompletedAt != nil {
			dateSet[*task.CompletedAt] = true
		}
		for _, sub := range task.Subtasks {
			if sub.CompletedAt != nil {
				dateSet[*sub.CompletedAt] = true
			}
		}
	}
	if len(dateSet) == 0 {
		return 0
	}

	var sorted []string
	for d := range dateSet {
		sorted = append(sorted, d)
	}
	sort.Strings(sorted)

	now := time.Now().Local()
	streak := 0
	for i := len(sorted) - 1; i >= 0; i-- {
		expected := now.AddDate(0, 0, -streak).Format("2006-01-02")
		if sorted[i] == expected {
			streak++
		} else if sorted[i] < expected {
			break
		}
	}
	return streak
}

// computeBurndown returns one point per day from the first completion to today.
// Uses a task-centric view: remaining(D) = total tasks - tasks completed by D.
// This gives a meaningful burndown even when work is done ahead of due dates.
func computeBurndown(classes []Class) []BurndownPoint {
	type entry struct {
		completedAt time.Time // zero if not completed
	}

	var entries []entry
	var start time.Time

	for _, class := range classes {
		for _, task := range class.Tasks {
			e := entry{}
			if task.Completed && task.CompletedAt != nil {
				t, _ := time.Parse("2006-01-02", *task.CompletedAt)
				e.completedAt = t
				if start.IsZero() || t.Before(start) {
					start = t
				}
			}
			entries = append(entries, e)
		}
	}

	if len(entries) == 0 || start.IsZero() {
		return []BurndownPoint{}
	}

	total := len(entries)
	today := localToday()

	points := make([]BurndownPoint, 0)
	for d := start; !d.After(today); d = d.AddDate(0, 0, 1) {
		completedByD := 0
		for _, e := range entries {
			if !e.completedAt.IsZero() && !e.completedAt.After(d) {
				completedByD++
			}
		}
		points = append(points, BurndownPoint{
			Date:      d.Format("2006-01-02"),
			Remaining: total - completedByD,
			Completed: completedByD,
		})
	}
	return points
}

// computeHeatmap aggregates completions per calendar day.
func computeHeatmap(classes []Class) []HeatmapPoint {
	countMap := make(map[string]int)
	ptMap := make(map[string]int)

	record := func(task Task) {
		if task.CompletedAt != nil {
			countMap[*task.CompletedAt]++
			ptMap[*task.CompletedAt] += task.Points
		}
	}

	for _, class := range classes {
		for _, task := range class.Tasks {
			record(task)
			for _, sub := range task.Subtasks {
				record(sub)
			}
		}
	}

	points := make([]HeatmapPoint, 0)
	for date, count := range countMap {
		points = append(points, HeatmapPoint{
			Date:   date,
			Count:  count,
			Points: ptMap[date],
		})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Date < points[j].Date })
	return points
}

// computeSummary returns quick counts for nav badges.
func computeSummary(classes []Class) Summary {
	now := time.Now().Local()
	today := now.Format("2006-01-02")
	weekStart := now.AddDate(0, 0, -int(now.Weekday())).Format("2006-01-02")
	weekEnd := now.AddDate(0, 0, 6-int(now.Weekday())).Format("2006-01-02")

	s := Summary{}
	for _, class := range classes {
		for _, task := range class.Tasks {
			if task.Completed || task.DueDate == "" {
				continue
			}
			if task.DueDate < today {
				s.Overdue++
			}
			if task.DueDate == today {
				s.DueToday++
			}
			if task.DueDate >= weekStart && task.DueDate <= weekEnd {
				s.DueThisWeek++
			}
		}
	}
	return s
}

// getWeekSummary returns all tasks due in the Sun–Sat week containing weekAnchor.
func getWeekSummary(classes []Class, weekAnchor time.Time) WeekSummary {
	offset := int(weekAnchor.Weekday())
	weekStart := weekAnchor.Truncate(24 * time.Hour).AddDate(0, 0, -offset)
	weekEnd := weekStart.AddDate(0, 0, 6)

	summary := WeekSummary{
		WeekStart: weekStart.Format("2006-01-02"),
		WeekEnd:   weekEnd.Format("2006-01-02"),
		ByDay:     make(map[string][]TaskSummary),
	}
	// pre-populate all 7 days
	for i := 0; i < 7; i++ {
		d := weekStart.AddDate(0, 0, i).Format("2006-01-02")
		summary.ByDay[d] = []TaskSummary{}
	}

	for _, class := range classes {
		for _, task := range class.Tasks {
			due, _ := time.Parse("2006-01-02", task.DueDate)
			if due.IsZero() {
				continue
			}
			if !due.Before(weekStart) && !due.After(weekEnd) {
				ds := due.Format("2006-01-02")
				summary.ByDay[ds] = append(summary.ByDay[ds], TaskSummary{
					ID:        task.ID,
					ClassID:   class.ID,
					ClassName: class.Name,
					Text:      task.Text,
					Completed: task.Completed,
					DueDate:   task.DueDate,
					Type:      task.Type,
					Points:    task.Points,
				})
			}
		}
	}
	return summary
}
