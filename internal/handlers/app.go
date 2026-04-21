// Package handlers contains the HTTP handlers and middleware for the web application.
package handlers

import (
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"groupie-tracker/internal/service"
)

// App holds the shared dependencies for all HTTP handlers.
type App struct {
	repo      *service.Repository
	templates *template.Template
}

// HomePageData is the template data passed to home.html.
type HomePageData struct {
	Title        string
	Artists      []service.ArtistView
	Stats        service.Stats
	Filter       service.Filter
	MemberCounts []int // distinct member counts for the checkbox group
}

// ArtistPageData is the template data passed to artist.html.
type ArtistPageData struct {
	Title  string
	Artist service.ArtistView
}

// ErrorPageData is the template data passed to error.html.
type ErrorPageData struct {
	Title   string
	Code    int
	Message string
}

// NewApp creates an App with the given repository and parsed templates.
func NewApp(repo *service.Repository, templates *template.Template) *App {
	return &App{repo: repo, templates: templates}
}

// Home handles GET / and renders the artist grid.
// Accepts ?q=, ?creation_min=, ?creation_max=, ?album_min=, ?album_max=,
// ?members_min=, ?members_max=, and ?location= query parameters.
// Any path other than "/" returns a 404.
func (a *App) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		a.renderError(w, http.StatusNotFound, "The page you requested does not exist.")
		return
	}
	if r.Method != http.MethodGet {
		a.renderError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	f := parseFilter(r)
	artists := a.repo.Filter(f)

	data := HomePageData{
		Title:        "Groupie Trackers",
		Artists:      artists,
		Stats:        a.repo.Stats(),
		Filter:       f,
		MemberCounts: a.repo.DistinctMemberCounts(),
	}

	a.render(w, http.StatusOK, "home.html", data)
}

// Artist handles GET /artist?id=<n> and renders the detail page for a single artist.
// Returns 400 for a missing or non-positive id, 404 if the artist does not exist.
func (a *App) Artist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.renderError(w, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id <= 0 {
		a.renderError(w, http.StatusBadRequest, "Invalid artist id.")
		return
	}

	artist, err := a.repo.ByID(id)
	if err != nil {
		if errors.Is(err, service.ErrArtistNotFound) {
			a.renderError(w, http.StatusNotFound, "Artist not found.")
			return
		}
		a.renderError(w, http.StatusInternalServerError, "Failed to load artist.")
		return
	}

	data := ArtistPageData{
		Title:  artist.Name,
		Artist: artist,
	}
	a.render(w, http.StatusOK, "artist.html", data)
}

// Search handles GET /api/search and returns a JSON array of matching artists.
// Accepts the same filter parameters as the home page.
// This endpoint is called by the client-side filter panel in app.js.
func (a *App) Search(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	artists := a.repo.Filter(parseFilter(r))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(artists); err != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
		return
	}
}

// Refresh handles POST /api/refresh and re-fetches all data from the upstream API.
// Returns 502 if the upstream API call fails, leaving the existing cached data intact.
func (a *App) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if err := a.repo.Refresh(); err != nil {
		http.Error(w, `{"error":"failed to refresh data"}`, http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// render writes the named template to the response with the given status code.
func (a *App) render(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := a.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template render error for %s: %v", name, err)
	}
}

// parseFilter reads all filter query parameters from r and returns a Filter.
func parseFilter(r *http.Request) service.Filter {
	q := r.URL.Query()
	return service.Filter{
		Query:         strings.TrimSpace(q.Get("q")),
		CreationMin:   queryInt(q.Get("creation_min")),
		CreationMax:   queryInt(q.Get("creation_max")),
		FirstAlbumMin: queryInt(q.Get("album_min")),
		FirstAlbumMax: queryInt(q.Get("album_max")),
		MembersMin:    queryInt(q.Get("members_min")),
		MembersMax:    queryInt(q.Get("members_max")),
		Location:      strings.TrimSpace(q.Get("location")),
	}
}

// queryInt parses s as a positive integer, returning 0 on any error.
func queryInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// renderError writes the error.html template with the given HTTP status and message.
// Falls back to a plain-text response if template execution fails.
func (a *App) renderError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	data := ErrorPageData{
		Title:   http.StatusText(status),
		Code:    status,
		Message: message,
	}

	if err := a.templates.ExecuteTemplate(w, "error.html", data); err != nil {
		http.Error(w, message, status)
	}
}

// LoggingMiddleware logs the HTTP method, path, and elapsed time for every request.
func (a *App) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started))
	})
}

// RecoverMiddleware catches any panic in a handler, logs it, and returns a 500 response
// so the server stays online after an unexpected runtime error.
func (a *App) RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered: %v", rec)
				a.renderError(w, http.StatusInternalServerError, "Something went wrong on the server.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
