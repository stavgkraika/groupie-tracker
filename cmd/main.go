// Package main is the entry point for the Groupie Tracker server.
// It wires together the API client, service layer, templates, and HTTP routes.
package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/service"
)

func main() {
	addr := getenv("ADDR", ":8080")
	baseURL := getenv("GROUPIE_API_BASE", "https://groupietrackers.herokuapp.com/api")

	// Build the upstream API client with a 15-second timeout.
	client := api.NewClient(baseURL, 15*time.Second)
	repository := service.NewRepository(client)

	// Fetch and cache all data from the upstream API before accepting requests.
	if err := repository.Refresh(); err != nil {
		log.Fatalf("failed to fetch API data at startup: %v", err)
	}

	// Parse all HTML templates from the templates/ directory.
	templates, err := template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	app := handlers.NewApp(repository, templates)

	mux := http.NewServeMux()
	// Serve static assets (CSS, JS) from the static/ directory.
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/artist", app.Artist)
	mux.HandleFunc("/api/search", app.Search)
	mux.HandleFunc("/api/refresh", app.Refresh)

	server := &http.Server{
		Addr:              addr,
		Handler:           app.RecoverMiddleware(app.LoggingMiddleware(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Groupie Tracker running on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

// getenv returns the value of the environment variable named by key,
// or fallback if the variable is not set or empty.
func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
