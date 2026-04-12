package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func setupRoutes(r *gin.Engine, store *Store) {
	api := r.Group("/api")

	// GET /api/classes        — active (non-archived) classes
	// GET /api/classes?all=true — all classes including archived
	api.GET("/classes", func(c *gin.Context) {
		if c.Query("all") == "true" {
			c.JSON(http.StatusOK, store.GetAllClasses())
		} else {
			c.JSON(http.StatusOK, store.GetClasses())
		}
	})

	// POST /api/classes  body: {"name": "CS 101", "semester": "Fall 2026"}
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

		id, err := store.CreateClass(body.Name, body.Semester)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				c.JSON(http.StatusConflict, gin.H{"error": "class already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	})

	// PATCH /api/classes/:id  body: {"semester": "Spring 2026", "archived": false}
	api.PATCH("/classes/:id", func(c *gin.Context) {
		var body struct {
			Semester string `json:"semester"`
			Archived bool   `json:"archived"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := store.UpdateClass(c.Param("id"), body.Semester, body.Archived); err != nil {
			if strings.Contains(err.Error(), "not found") {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// GET /api/classes/:id
	api.GET("/classes/:id", func(c *gin.Context) {
		cls, ok := store.GetClass(c.Param("id"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
			return
		}
		c.JSON(http.StatusOK, cls)
	})

	// PATCH /api/tasks/:id/toggle  body: {"completed": true|false}
	api.PATCH("/tasks/:id/toggle", func(c *gin.Context) {
		taskID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
			return
		}
		var body struct {
			Completed bool `json:"completed"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := store.ToggleTask(taskID, body.Completed); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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

	// GET /api/stats/summary — lightweight badge counts for nav
	api.GET("/stats/summary", func(c *gin.Context) {
		c.JSON(http.StatusOK, computeSummary(store.GetClasses()))
	})

	// GET /api/stats/heatmap — includes archived classes for full history
	api.GET("/stats/heatmap", func(c *gin.Context) {
		c.JSON(http.StatusOK, computeHeatmap(store.GetAllClasses()))
	})

	// POST /api/import  body: {"classId": "itn-106", "text": "<canvas paste>"}
	api.POST("/import", func(c *gin.Context) {
		var body struct {
			ClassID string `json:"classId"`
			Text    string `json:"text"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cls, ok := store.GetClass(body.ClassID)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "class not found"})
			return
		}

		baseYear := time.Now().Year()
		if cls.Created != "" {
			if t, err := time.Parse("2006-01-02", cls.Created); err == nil {
				baseYear = t.Year()
			}
		}

		tasks, err := parseCanvasPaste(body.Text, baseYear)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		n, err := store.ImportCanvasTasks(body.ClassID, tasks)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"imported": n})
	})

	// POST /api/import/preview — parse without writing
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
		type previewItem struct {
			Date string `json:"date"`
			Text string `json:"text"`
		}
		result := make([]previewItem, 0, len(tasks))
		for _, t := range tasks {
			result = append(result, previewItem{
				Date: t.Date.Format("2006-01-02"),
				Text: t.Text,
			})
		}
		c.JSON(http.StatusOK, result)
	})
}
