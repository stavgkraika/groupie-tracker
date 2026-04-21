// Package handlers — app_test.go tests every HTTP handler, the filter parameter
// parser, the queryInt helper, and both middleware wrappers.
package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/service"
)

// ── Test helpers ──────────────────────────────────────────────────────────────

const stubTemplates = `{{define "home.html"}}home{{end}}` +
	`{{define "artist.html"}}artist:{{.Artist.Name}}{{end}}` +
	`{{define "error.html"}}{{.Code}}{{end}}`

// newApp builds an App pre-loaded with the given artists and stub templates.
func newApp(t *testing.T, artists ...api.Artist) *App {
	t.Helper()
	tmpl := template.Must(template.New("").Parse(stubTemplates))

	relations := make([]api.Relation, len(artists))
	for i, a := range artists {
		relations[i] = api.Relation{ID: a.ID, DatesLocations: map[string][]string{}}
	}

	repo := service.NewRepository(nil)
	repo.LoadForTest(artists, relations)
	return NewApp(repo, tmpl)
}

// get fires a GET request against handler and returns the recorder.
func get(handler http.HandlerFunc, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, path, nil)
	handler(w, r)
	return w
}

// ── Home ──────────────────────────────────────────────────────────────────────

func TestHome_OK(t *testing.T) {
	app := newApp(t)
	w := get(app.Home, "/")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}

func TestHome_NotFound(t *testing.T) {
	app := newApp(t)
	w := get(app.Home, "/unknown")
	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", w.Code)
	}
}

func TestHome_MethodNotAllowed(t *testing.T) {
	app := newApp(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	app.Home(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405", w.Code)
	}
}

// ── Artist ────────────────────────────────────────────────────────────────────

func TestArtist_OK(t *testing.T) {
	app := newApp(t, api.Artist{ID: 1, Name: "Queen", Members: []string{"Freddie"}})
	w := get(app.Artist, "/artist?id=1")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	if body := w.Body.String(); body != "artist:Queen" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestArtist_BadID(t *testing.T) {
	app := newApp(t)
	for _, path := range []string{"/artist", "/artist?id=0", "/artist?id=abc"} {
		w := get(app.Artist, path)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: got %d, want 400", path, w.Code)
		}
	}
}

func TestArtist_NotFound(t *testing.T) {
	app := newApp(t)
	w := get(app.Artist, "/artist?id=99")
	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", w.Code)
	}
}

func TestArtist_MethodNotAllowed(t *testing.T) {
	app := newApp(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/artist?id=1", nil)
	app.Artist(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405", w.Code)
	}
}

// ── Search ────────────────────────────────────────────────────────────────────

func TestSearch_ReturnsJSON(t *testing.T) {
	app := newApp(t, api.Artist{ID: 1, Name: "Queen", Members: []string{"Freddie"}})
	w := get(app.Search, "/api/search")
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var artists []service.ArtistView
	if err := json.NewDecoder(w.Body).Decode(&artists); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(artists) != 1 {
		t.Fatalf("got %d artists, want 1", len(artists))
	}
}

func TestSearch_FilterByCreationRange(t *testing.T) {
	app := newApp(t,
		api.Artist{ID: 1, Name: "Old", CreationDate: 1970, Members: []string{"a"}},
		api.Artist{ID: 2, Name: "New", CreationDate: 2010, Members: []string{"b"}},
	)
	w := get(app.Search, "/api/search?creation_min=2000&creation_max=2024")
	var artists []service.ArtistView
	json.NewDecoder(w.Body).Decode(&artists)
	if len(artists) != 1 || artists[0].Name != "New" {
		t.Fatalf("unexpected result: %+v", artists)
	}
}

func TestSearch_FilterByMembers(t *testing.T) {
	app := newApp(t,
		api.Artist{ID: 1, Name: "Solo", Members: []string{"a"}},
		api.Artist{ID: 2, Name: "Duo", Members: []string{"a", "b"}},
		api.Artist{ID: 3, Name: "Trio", Members: []string{"a", "b", "c"}},
	)
	w := get(app.Search, "/api/search?members_min=2&members_max=2")
	var artists []service.ArtistView
	json.NewDecoder(w.Body).Decode(&artists)
	if len(artists) != 1 || artists[0].Name != "Duo" {
		t.Fatalf("unexpected result: %+v", artists)
	}
}

func TestSearch_FilterByLocation(t *testing.T) {
	repo := service.NewRepository(nil)
	repo.LoadForTest(
		[]api.Artist{
			{ID: 1, Name: "Band A", Members: []string{"x"}},
			{ID: 2, Name: "Band B", Members: []string{"y"}},
		},
		[]api.Relation{
			{ID: 1, DatesLocations: map[string][]string{"texas-usa": {"01-01-2020"}}},
			{ID: 2, DatesLocations: map[string][]string{"paris-france": {"01-01-2020"}}},
		},
	)
	tmpl := template.Must(template.New("").Parse(stubTemplates))
	app := NewApp(repo, tmpl)

	w := get(app.Search, "/api/search?location=texas")
	var artists []service.ArtistView
	json.NewDecoder(w.Body).Decode(&artists)
	if len(artists) != 1 || artists[0].Name != "Band A" {
		t.Fatalf("unexpected result: %+v", artists)
	}
}

func TestSearch_MethodNotAllowed(t *testing.T) {
	app := newApp(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/search", nil)
	app.Search(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405", w.Code)
	}
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func TestRefresh_MethodNotAllowed(t *testing.T) {
	app := newApp(t)
	w := get(app.Refresh, "/api/refresh")
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("got %d, want 405", w.Code)
	}
}

// ── parseFilter / queryInt ────────────────────────────────────────────────────

func TestParseFilter(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet,
		"/?q=queen&creation_min=1970&creation_max=2000&album_min=1973&album_max=1980&members_min=2&members_max=5&location=texas",
		nil)
	f := parseFilter(r)
	if f.Query != "queen"   { t.Errorf("Query: got %q", f.Query) }
	if f.CreationMin != 1970 { t.Errorf("CreationMin: got %d", f.CreationMin) }
	if f.CreationMax != 2000 { t.Errorf("CreationMax: got %d", f.CreationMax) }
	if f.FirstAlbumMin != 1973 { t.Errorf("FirstAlbumMin: got %d", f.FirstAlbumMin) }
	if f.FirstAlbumMax != 1980 { t.Errorf("FirstAlbumMax: got %d", f.FirstAlbumMax) }
	if f.MembersMin != 2    { t.Errorf("MembersMin: got %d", f.MembersMin) }
	if f.MembersMax != 5    { t.Errorf("MembersMax: got %d", f.MembersMax) }
	if f.Location != "texas" { t.Errorf("Location: got %q", f.Location) }
}

func TestQueryInt(t *testing.T) {
	cases := []struct{ in string; want int }{
		{"42", 42}, {"0", 0}, {"-1", 0}, {"abc", 0}, {"", 0}, {" 7 ", 7},
	}
	for _, tc := range cases {
		if got := queryInt(tc.in); got != tc.want {
			t.Errorf("queryInt(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// ── Middleware ────────────────────────────────────────────────────────────────

func TestRecoverMiddleware(t *testing.T) {
	app := newApp(t)
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	app.RecoverMiddleware(panicking).ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", w.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	app := newApp(t)
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	app.LoggingMiddleware(next).ServeHTTP(w, r)
	if !called {
		t.Fatal("next handler was not called")
	}
}
