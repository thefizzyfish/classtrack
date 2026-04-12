package main

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

type Task struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Completed   bool     `json:"completed"`
	CompletedAt *string  `json:"completedAt"` // YYYY-MM-DD or nil
	DueDate     string   `json:"dueDate"`     // YYYY-MM-DD
	Week        string   `json:"week"`        // Sunday of the week, YYYY-MM-DD
	Type        TaskType `json:"type"`
	Points      int      `json:"points"`
	Indent      int      `json:"indent"`
	Subtasks    []Task   `json:"subtasks,omitempty"`
	// private — not serialised
	lineNum  int
	filePath string
	classID  string
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
	File     string     `json:"file"`
	Tasks    []Task     `json:"tasks"`
	Created  string     `json:"created"`
	Semester string     `json:"semester"` // e.g. "Spring 2026"
	Archived bool       `json:"archived"`
	Stats    ClassStats `json:"stats"`
}

type TaskSummary struct {
	ID        string   `json:"id"`
	ClassID   string   `json:"classID"`
	ClassName string   `json:"className"`
	Text      string   `json:"text"`
	Completed bool     `json:"completed"`
	DueDate   string   `json:"dueDate"`
	Type      TaskType `json:"type"`
	Points    int      `json:"points"`
}

type WeekSummary struct {
	WeekStart string                 `json:"weekStart"`
	WeekEnd   string                 `json:"weekEnd"`
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
