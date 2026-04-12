package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	migrateFlag := flag.Bool("migrate", false, "import Obsidian markdown vault into SQLite and exit")
	dbPath := flag.String("db", filepath.Join(os.Getenv("HOME"), ".classtrack.db"), "SQLite database path")
	vaultPath := flag.String("vault",
		filepath.Join(os.Getenv("HOME"), "Obsidian", "Obsidian_Vault", "NVCC"),
		"Obsidian vault path (only used with --migrate)",
	)
	flag.Parse()

	store, err := NewStore(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	if *migrateFlag {
		if err := runMigration(store.db, *vaultPath); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		return
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "PATCH"},
		AllowHeaders: []string{"Content-Type"},
	}))

	setupRoutes(r, store)

	// Serve built SvelteKit app if present (production single-binary mode)
	webBuild := filepath.Join("..", "web", "build")
	if _, err := os.Stat(webBuild); err == nil {
		r.Static("/assets", filepath.Join(webBuild, "assets"))
		r.StaticFile("/", filepath.Join(webBuild, "index.html"))
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(webBuild, "index.html"))
		})
	}

	log.Println("classtrack → http://localhost:4000")
	if err := r.Run(":4000"); err != nil {
		log.Fatalf("server: %v", err)
	}
}
