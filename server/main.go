package main

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Store holds the parsed vault state with an RW lock.
type Store struct {
	mu      sync.RWMutex
	classes []Class
	vault   string
}

func NewStore(vaultPath string) *Store {
	s := &Store{vault: vaultPath}
	if err := s.Reload(); err != nil {
		log.Printf("initial load error: %v", err)
	}
	return s
}

func (s *Store) Reload() error {
	classes, err := discoverClasses(s.vault)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.classes = classes
	s.mu.Unlock()
	log.Printf("vault reloaded: %d class(es)", len(classes))
	return nil
}

// GetClasses returns only non-archived classes.
func (s *Store) GetClasses() []Class {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var active []Class
	for _, c := range s.classes {
		if !c.Archived {
			active = append(active, c)
		}
	}
	return active
}

// GetAllClasses returns all classes including archived ones.
func (s *Store) GetAllClasses() []Class {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.classes
}

// VaultPath returns the watched vault directory.
func (s *Store) VaultPath() string { return s.vault }

func main() {
	vaultPath := filepath.Join(os.Getenv("HOME"), "Obsidian", "Obsidian_Vault", "NVCC")

	store := NewStore(vaultPath)

	// File watcher — debounced reload on any change under the vault
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("watcher: %v", err)
	}
	defer watcher.Close()

	var debounce *time.Timer
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// If a new directory appears, watch it so files created inside are tracked
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						watcher.Add(event.Name)
						log.Printf("watching new dir: %s", event.Name)
					}
				}
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					if debounce != nil {
						debounce.Stop()
					}
					debounce = time.AfterFunc(500*time.Millisecond, func() {
						if err := store.Reload(); err != nil {
							log.Printf("reload error: %v", err)
						}
					})
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %v", err)
			}
		}
	}()

	// Watch all directories under vault recursively
	_ = filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "PATCH"},
		AllowHeaders: []string{"Content-Type"},
	}))

	setupRoutes(r, store)

	// Serve built SvelteKit app (production)
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
