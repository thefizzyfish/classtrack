package main

import (
	"strings"
	"time"
)

type TaskType string

const (
	TaskTypeExam       TaskType = "exam"
	TaskTypeQuiz       TaskType = "quiz"
	TaskTypeDiscussion TaskType = "discussion"
	TaskTypeAssignment TaskType = "assignment"
	TaskTypeOther      TaskType = "other"
)

var taskPoints = map[TaskType]int{
	TaskTypeExam:       100,
	TaskTypeQuiz:       25,
	TaskTypeDiscussion: 20,
	TaskTypeAssignment: 40,
	TaskTypeOther:      10,
}

// months maps month abbreviations to time.Month (shared by canvas parser and migration).
var months = map[string]time.Month{
	"Jan": time.January, "Feb": time.February, "Mar": time.March,
	"Apr": time.April, "May": time.May, "Jun": time.June,
	"Jul": time.July, "Aug": time.August, "Sep": time.September,
	"Oct": time.October, "Nov": time.November, "Dec": time.December,
}

// classifyTask returns the TaskType inferred from keywords in text.
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

// sundayOfWeek returns the Sunday of the week containing t (local time).
func sundayOfWeek(t time.Time) time.Time {
	return t.AddDate(0, 0, -int(t.Weekday()))
}

type Task struct {
	ID          int64    `json:"id"`
	Text        string   `json:"text"`
	Completed   bool     `json:"completed"`
	CompletedAt *string  `json:"completedAt"` // YYYY-MM-DD or nil
	DueDate     string   `json:"dueDate"`     // YYYY-MM-DD
	Week        string   `json:"week"`        // Sunday of the week, YYYY-MM-DD
	Type        TaskType `json:"type"`
	Points      int      `json:"points"`
	Subtasks    []Task   `json:"subtasks,omitempty"`
}

type ClassStats struct {
	Total        int     `json:"total"`
	Completed    int     `json:"completed"`
	Percentage   float64 `json:"percentage"`
	TotalPoints  int     `json:"totalPoints"`
	EarnedPoints int     `json:"earnedPoints"`
	OnPaceScore  float64 `json:"onPaceScore"` // % ahead (positive) or behind (negative)
	Streak       int     `json:"streak"`
}

type Class struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Tasks    []Task     `json:"tasks"`
	Created  string     `json:"created"`
	Semester string     `json:"semester"`
	Archived bool       `json:"archived"`
	Stats    ClassStats `json:"stats"`
}

type TaskSummary struct {
	ID        int64    `json:"id"`
	ClassID   string   `json:"classID"`
	ClassName string   `json:"className"`
	Text      string   `json:"text"`
	Completed bool     `json:"completed"`
	DueDate   string   `json:"dueDate"`
	Type      TaskType `json:"type"`
	Points    int      `json:"points"`
}

type WeekSummary struct {
	WeekStart string                   `json:"weekStart"`
	WeekEnd   string                   `json:"weekEnd"`
	ByDay     map[string][]TaskSummary `json:"byDay"`
}

type BurndownPoint struct {
	Date      string `json:"date"`
	Remaining int    `json:"remaining"`
	Completed int    `json:"completed"`
}

type HeatmapPoint struct {
	Date   string `json:"date"`
	Count  int    `json:"count"`
	Points int    `json:"points"`
}

type Summary struct {
	Overdue     int `json:"overdue"`
	DueToday    int `json:"dueToday"`
	DueThisWeek int `json:"dueThisWeek"`
}
