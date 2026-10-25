package main

import (
	"path/filepath"
	"testing"
	"time"
)

func day(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02", s, time.Local)
	return t
}

func TestRescheduleKeepsProgress(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.CreateClass("ITN 263", "Fall 2026")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ImportCanvasTasks(id, []CanvasTask{
		{Date: day("2026-10-11"), Text: "Lab 06", HintType: TaskTypeAssignment},
		{Date: day("2026-10-11"), Text: "Exam 1 (HONORLOCK)", HintType: TaskTypeQuiz},
		{Date: day("2026-10-18"), Text: "Module 8: Quiz 8", HintType: TaskTypeQuiz},
		{Date: day("2026-10-16"), Text: "Office Hour"},
		{Date: day("2026-10-23"), Text: "Office Hour"},
		{Date: day("2026-10-25"), Text: "Dropped Assignment"},
		{Date: day("2026-10-25"), Text: "Twice"},
	}); err != nil {
		t.Fatal(err)
	}
	cls, _ := store.GetClass(id)
	byText := map[string]Task{}
	for _, tk := range cls.Tasks {
		byText[tk.Text] = tk
	}
	if err := store.ToggleTask(byText["Lab 06"].ID, true); err != nil {
		t.Fatal(err)
	}

	paste := []CanvasTask{
		{Date: day("2026-10-18"), Text: "lab  06"}, // case/whitespace differences still match
		{Date: day("2026-10-11"), Text: "Exam 1 (HONORLOCK)"},
		{Date: day("2026-10-25"), Text: "Module 8: Quiz 8"},
		{Date: day("2026-10-30"), Text: "Office Hour"}, // out of order on purpose
		{Date: day("2026-10-23"), Text: "Office Hour"},
		{Date: day("2026-11-01"), Text: "Brand New Lab"},
		{Date: day("2026-11-01"), Text: "Twice"},
		{Date: day("2026-11-08"), Text: "Twice"},
	}

	// Preview must not write anything.
	pre, err := store.RescheduleCanvasTasks(id, paste, false)
	if err != nil {
		t.Fatal(err)
	}
	if pre.Applied || len(pre.Moved) != 4 {
		t.Fatalf("preview: applied=%v moved=%d, want false/4: %+v", pre.Applied, len(pre.Moved), pre.Moved)
	}
	if after, _ := store.GetClass(id); after.Tasks[0].DueDate != "2026-10-11" {
		t.Fatalf("preview wrote to the database")
	}

	res, err := store.RescheduleCanvasTasks(id, paste, true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Applied || len(res.Moved) != 4 || res.Unchanged != 1 {
		t.Fatalf("apply: applied=%v moved=%d unchanged=%d", res.Applied, len(res.Moved), res.Unchanged)
	}
	if len(res.New) != 1 || res.New[0].Text != "Brand New Lab" {
		t.Errorf("new = %+v", res.New)
	}
	if len(res.Missing) != 1 || res.Missing[0].Text != "Dropped Assignment" {
		t.Errorf("missing = %+v", res.Missing)
	}
	if len(res.Ambiguous) != 1 {
		t.Errorf("ambiguous = %+v", res.Ambiguous)
	}

	after, _ := store.GetClass(id)
	got := map[string][]Task{}
	for _, tk := range after.Tasks {
		got[tk.Text] = append(got[tk.Text], tk)
	}
	lab := got["Lab 06"][0]
	if lab.ID != byText["Lab 06"].ID || lab.DueDate != "2026-10-18" || lab.Week != "2026-10-18" {
		t.Errorf("Lab 06 = %+v", lab)
	}
	if !lab.Completed {
		t.Errorf("Lab 06 lost its completion")
	}
	if q := got["Module 8: Quiz 8"][0]; q.DueDate != "2026-10-25" || q.Week != "2026-10-25" {
		t.Errorf("Quiz 8 = %+v", q)
	}
	if oh := got["Office Hour"]; oh[0].DueDate != "2026-10-23" || oh[1].DueDate != "2026-10-30" {
		t.Errorf("Office Hour paired wrong: %s, %s", oh[0].DueDate, oh[1].DueDate)
	}
	if tw := got["Twice"][0]; tw.DueDate != "2026-10-25" {
		t.Errorf("ambiguous task was moved: %+v", tw)
	}
	if len(after.Tasks) != 7 {
		t.Errorf("task count changed: %d", len(after.Tasks))
	}
}

func TestParseSkipsTimeContinuation(t *testing.T) {
	paste := "Fri Oct 16, 2026\tCalendar Event Office Hour\t1pm\nto 2pm\nSun Oct 18, 2026\tQuiz Module 7: Quiz 7\tdue by 11:59pm\n"
	tasks, _ := parseCanvasPaste(paste, 2026)
	for _, tk := range tasks {
		if tk.Text == "to 2pm" {
			t.Fatalf("time continuation parsed as a task: %+v", tasks)
		}
	}
	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2: %+v", len(tasks), tasks)
	}
}
