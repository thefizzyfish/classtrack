package main

import (
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// maxBodyBytes is the largest request body the server will accept.
// A full-semester Canvas paste is ~20 KB; 512 KB is a generous ceiling.
const maxBodyBytes = 512 * 1024

func main() {
	migrateFlag := flag.Bool("migrate", false, "import markdown todo files into SQLite and exit")
	dbPath := flag.String("db", filepath.Join(os.Getenv("HOME"), ".classtrack.db"), "SQLite database path")
	vaultPath := flag.String("vault", "", "directory to walk for *Todo.md files (required with --migrate)")
	flag.Parse()

	store, err := NewStore(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}

	if *migrateFlag {
		if *vaultPath == "" {
			log.Fatal("--vault is required with --migrate: specify the directory containing your todo files")
		}
		if err := runMigration(store.db, *vaultPath); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		return
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "PATCH", "DELETE"},
		AllowHeaders: []string{"Content-Type"},
	}))

	// Enforce a request body size limit on all routes.
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		c.Next()
	})

	setupRoutes(r, store)

	// Serve embedded SvelteKit app (production single-binary mode).
	// static/ is populated by build.sh before compilation — empty in dev,
	// which is fine since start.sh runs the SvelteKit dev server separately.
	if _, err := staticFiles.Open("static/index.html"); err == nil {
		sub, _ := fs.Sub(staticFiles, "static")
		httpFS := http.FS(sub)
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path[1:] // strip leading /
			// Serve the file if it exists in the embedded FS (JS, CSS, images, etc.)
			if path != "" {
				if f, err := sub.Open(path); err == nil {
					f.Close()
					c.FileFromFS(path, httpFS)
					return
				}
			}
			// Fall back to index.html for all SPA routes
			data, err := fs.ReadFile(sub, "index.html")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		})
	}

	log.Println("classtrack → http://localhost:4000")
	if err := r.Run(":4000"); err != nil {
		log.Fatalf("server: %v", err)
	}
}
