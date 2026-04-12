package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func setupRoutes(r *gin.Engine, store *Store) {
	api := r.Group("/api")

	// GET /api/classes        — active classes only
	// GET /api/classes?all=1  — include archived
	api.GET("/classes", func(c *gin.Context) {
		if c.Query("all") == "true" {
			c.JSON(http.StatusOK, store.GetAllClasses())
		} else {
			c.JSON(http.StatusOK, store.GetClasses())
		}
	})

	// POST /api/classes — create a new class (dir + Todo.md)
	// body: {"name": "CS 101", "semester": "Fall 2026"}
	api.POST("/classes", func(c *gin.Context) {
		var body struct {
			Name     string `json:"name"`
			Semester string `json:"semester"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		body.Name = strings.TrimSpace(body.Name)
		if body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		dir := filepath.Join(store.VaultPath(), body.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		filePath := filepath.Join(dir, body.Name+" Todo.md")
		if _, err := os.Stat(filePath); err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "class already exists"})
			return
		}

		frontmatter := "---\ntags: [nvcc, todo]\ncreated: " + time.Now().Format("2006-01-02")
		if body.Semester != "" {
			frontmatter += "\nsemester: " + body.Semester
		}
		frontmatter += "\n---\n# " + body.Name + " Todo\n"

		if err := os.WriteFile(filePath, []byte(frontmatter), 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		store.Reload()
		c.JSON(http.StatusCreated, gin.H{"id": strings.ToLower(strings.ReplaceAll(body.Name, " ", "-"))})
	})

	// GET /api/classes/:id — single class with all tasks
	api.GET("/classes/:id", func(c *gin.Context) {
		id := c.Param("id")
		for _, cl := range store.GetClasses() {
			if cl.ID == id {
				c.JSON(http.StatusOK, cl)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
	})

	// PATCH /api/tasks/:id/toggle  body: {"completed": true|false}
	api.PATCH("/tasks/:id/toggle", func(c *gin.Context) {
		taskID := c.Param("id")

		var body struct {
			Completed bool `json:"completed"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// parse classID:lineNum from task ID
		idx := strings.LastIndex(taskID, ":")
		if idx < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}
		classID := taskID[:idx]
		var lineNum int
		if _, err := fmt.Sscanf(taskID[idx+1:], "%d", &lineNum); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid line number"})
			return
		}

		var filePath string
		for _, cl := range store.GetClasses() {
			if cl.ID == classID {
				filePath = cl.File
				break
			}
		}
		if filePath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
			return
		}

		if err := toggleTask(filePath, lineNum, body.Completed); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		store.Reload()
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// GET /api/week?date=YYYY-MM-DD  (defaults to current week)
	api.GET("/week", func(c *gin.Context) {
		anchor := time.Now()
		if ds := c.Query("date"); ds != "" {
			if t, err := time.Parse("2006-01-02", ds); err == nil {
				anchor = t
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date, use YYYY-MM-DD"})
				return
			}
		}
		c.JSON(http.StatusOK, getWeekSummary(store.GetClasses(), anchor))
	})

	// GET /api/stats/burndown
	api.GET("/stats/burndown", func(c *gin.Context) {
		c.JSON(http.StatusOK, computeBurndown(store.GetClasses()))
	})

	// GET /api/stats/summary — lightweight counts for nav badges
	api.GET("/stats/summary", func(c *gin.Context) {
		c.JSON(http.StatusOK, computeSummary(store.GetClasses()))
	})

	// GET /api/stats/heatmap — always includes archived classes for full history
	api.GET("/stats/heatmap", func(c *gin.Context) {
		c.JSON(http.StatusOK, computeHeatmap(store.GetAllClasses()))
	})

	// POST /api/import  body: {"classId": "itn-106", "text": "<paste>"}
	api.POST("/import", func(c *gin.Context) {
		var body struct {
			ClassID string `json:"classId"`
			Text    string `json:"text"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var filePath string
		var baseYear = time.Now().Year()
		for _, cl := range store.GetClasses() {
			if cl.ID == body.ClassID {
				filePath = cl.File
				if cl.Created != "" {
					if t, err := time.Parse("2006-01-02", cl.Created); err == nil {
						baseYear = t.Year()
					}
				}
				break
			}
		}
		if filePath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
			return
		}

		tasks, err := parseCanvasPaste(body.Text, baseYear)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := appendCanvasTasks(filePath, tasks); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		store.Reload()
		c.JSON(http.StatusOK, gin.H{"imported": len(tasks)})
	})

	// POST /api/import/preview  — parse without writing
	api.POST("/import/preview", func(c *gin.Context) {
		var body struct {
			Text     string `json:"text"`
			BaseYear int    `json:"baseYear"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.BaseYear == 0 {
			body.BaseYear = time.Now().Year()
		}
		tasks, err := parseCanvasPaste(body.Text, body.BaseYear)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		type preview struct {
			Date string `json:"date"`
			Text string `json:"text"`
		}
		var result []preview
		for _, t := range tasks {
			result = append(result, preview{
				Date: t.Date.Format("2006-01-02"),
				Text: t.Text,
			})
		}
		c.JSON(http.StatusOK, result)
	})
}
