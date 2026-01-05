package main

import (
	"log"
	"net/http"
	"os"

    "github.com/magnusohle/openanki-backend/internal/api"
    "github.com/magnusohle/openanki-backend/internal/database"
    "github.com/magnusohle/openanki-backend/internal/media"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
    // Load .env file (if present) to populate R2/AppStore keys
    if err := godotenv.Load(); err != nil {
        log.Println("ℹ️ No .env file loaded via godotenv (systemd vars will be used if set)")
    }

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize Database with absolute path for persistence
	homeDir, _ := os.UserHomeDir()
	dbPath := homeDir + "/.checkst/openanki.db"
	// Ensure directory exists
	os.MkdirAll(homeDir+"/.checkst", 0755)
    repo, err := database.InitDB(dbPath)
    if err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
	log.Printf("Database path: %s", dbPath)

    // Initialize sync schema (additional tables)
    if err := repo.InitSyncSchema(); err != nil {
        log.Fatalf("Failed to initialize sync schema: %v", err)
    }

    // Initialize S3/R2 for media uploads
    s3Service, err := media.InitS3()
    if err != nil {
        log.Printf("Warning: S3/R2 not configured: %v", err)
    }

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check endpoint
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        if err := repo.DB.Ping(); err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            w.Write([]byte("Database disconnected"))
            return
        }
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    // API Routes
    r.Route("/api/v1", func(r chi.Router) {
        r.Route("/auth", api.RegisterAuthRoutes)
        r.Route("/users", api.RegisterProfileRoutes)
        r.Route("/groups", api.RegisterGroupsRoutes)
        r.Route("/decks", api.RegisterDecksRoutes)
        api.RegisterSyncRoutes(r, repo, s3Service)
        r.Route("/leaderboard", api.RegisterLeaderboardRoutes)
        r.Route("/iap", api.RegisterIAPRoutes)
    })

    // Serve static web files (Landing Page, Login, Account)
    webDir := http.Dir("./web/public")
    fileServer := http.FileServer(webDir)
    r.Handle("/*", fileServer)

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
