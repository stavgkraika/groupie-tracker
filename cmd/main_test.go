// Package main — main_test.go tests the getenv helper and verifies that all
// HTTP routes are registered and return the expected status codes.
package main

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"groupie-tracker/internal/handlers"
	"groupie-tracker/internal/service"
)

// TestGetenv verifies that getenv returns the environment variable value when set
// and falls back to the default when the variable is absent or empty.
func TestGetenv(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		os.Setenv("TEST_KEY", "hello")
		defer os.Unsetenv("TEST_KEY")
		if got := getenv("TEST_KEY", "default"); got != "hello" {
			t.Fatalf("got %q, want %q", got, "hello")
		}
	})

	t.Run("returns fallback when unset", func(t *testing.T) {
		os.Unsetenv("TEST_KEY")
		if got := getenv("TEST_KEY", "default"); got != "default" {
			t.Fatalf("got %q, want %q", got, "default")
		}
	})

	t.Run("returns fallback when empty", func(t *testing.T) {
		os.Setenv("TEST_KEY", "")
		defer os.Unsetenv("TEST_KEY")
		if got := getenv("TEST_KEY", "default"); got != "default" {
			t.Fatalf("got %q, want %q", got, "default")
		}
	})
}

// newTestServer builds a minimal httptest.Server with an empty repository and
// stub templates so route-level tests do not require the upstream API or disk files.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	// Stub templates with the three names the handlers reference.
	const tmpl = `{{define "home.html"}}home{{end}}` +
		`{{define "artist.html"}}artist{{end}}` +
		`{{define "error.html"}}{{.Code}}{{end}}`
	templates := template.Must(template.New("").Parse(tmpl))

	// NewRepository with a nil client is safe as long as Refresh is never called.
	repo := service.NewRepository(nil)
	app := handlers.NewApp(repo, templates)

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/artist", app.Artist)
	mux.HandleFunc("/api/search", app.Search)
	mux.HandleFunc("/api/refresh", app.Refresh)

	return httptest.NewServer(app.RecoverMiddleware(app.LoggingMiddleware(mux)))
}

// TestRoutes checks that every registered route responds with the expected
// HTTP status code for a basic valid request.
func TestRoutes(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	cases := []struct {
		method string
		path   string
		want   int
	}{
		// Home page
		{http.MethodGet, "/", http.StatusOK},
		// Unknown path under / returns 404
		{http.MethodGet, "/notfound", http.StatusNotFound},
		// Artist without id returns 400
		{http.MethodGet, "/artist", http.StatusBadRequest},
		// Artist with non-existent id returns 404
		{http.MethodGet, "/artist?id=99999", http.StatusNotFound},
		// Search with no query returns 200 (empty result set)
		{http.MethodGet, "/api/search", http.StatusOK},
		// Refresh requires POST
		{http.MethodGet, "/api/refresh", http.StatusMethodNotAllowed},
	}

	for _, tc := range cases {
		req, err := http.NewRequest(tc.method, srv.URL+tc.path, nil)
		if err != nil {
			t.Fatalf("build request %s %s: %v", tc.method, tc.path, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("do request %s %s: %v", tc.method, tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != tc.want {
			t.Errorf("%s %s: got %d, want %d", tc.method, tc.path, resp.StatusCode, tc.want)
		}
	}
}
