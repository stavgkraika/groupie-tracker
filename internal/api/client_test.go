// Package api — client_test.go tests the HTTP client against a local httptest.Server
// so no real network calls are made.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newStubServer starts a test server that serves the given payload as JSON on every
// route and returns it together with a Client already pointed at that server.
func newStubServer(t *testing.T, status int, payload any) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if payload != nil {
			json.NewEncoder(w).Encode(payload)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, NewClient(srv.URL, 5*time.Second)
}

// TestNewClient_TrimsTrailingSlash verifies that a trailing slash in the base URL
// is removed so endpoint paths are never double-slashed.
func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("http://example.com/api/", 5*time.Second)
	if c.baseURL != "http://example.com/api" {
		t.Fatalf("got %q, want trailing slash trimmed", c.baseURL)
	}
}

// TestGetArtists_OK verifies that a 200 response is decoded into the correct slice.
func TestGetArtists_OK(t *testing.T) {
	want := []Artist{{ID: 1, Name: "Queen", Members: []string{"Freddie", "Brian"}}}
	_, client := newStubServer(t, http.StatusOK, want)

	got, err := client.GetArtists()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 || got[0].Name != "Queen" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

// TestGetLocations_OK verifies that the LocationsIndex wrapper is unwrapped correctly.
func TestGetLocations_OK(t *testing.T) {
	want := LocationsIndex{Index: []Location{{ID: 1, Locations: []string{"paris-france"}}}}
	_, client := newStubServer(t, http.StatusOK, want)

	got, err := client.GetLocations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("unexpected result: %+v", got)
	}
}

// TestGetDates_OK verifies that the DatesIndex wrapper is unwrapped correctly.
func TestGetDates_OK(t *testing.T) {
	want := DatesIndex{Index: []Date{{ID: 1, Dates: []string{"01-01-2020"}}}}
	_, client := newStubServer(t, http.StatusOK, want)

	got, err := client.GetDates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Dates[0] != "01-01-2020" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

// TestGetRelations_OK verifies that the RelationsIndex wrapper is unwrapped correctly.
func TestGetRelations_OK(t *testing.T) {
	want := RelationsIndex{Index: []Relation{{ID: 1, DatesLocations: map[string][]string{"paris-france": {"01-01-2020"}}}}}
	_, client := newStubServer(t, http.StatusOK, want)

	got, err := client.GetRelations()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("unexpected result: %+v", got)
	}
}

// TestGetJSON_NonOKStatus verifies that a non-200 response returns an error
// containing the status code.
func TestGetJSON_NonOKStatus(t *testing.T) {
	_, client := newStubServer(t, http.StatusInternalServerError, nil)

	if _, err := client.GetArtists(); err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
}

// TestGetJSON_InvalidJSON verifies that a malformed response body returns a
// decode error rather than silently producing a zero-value result.
func TestGetJSON_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, 5*time.Second)
	if _, err := client.GetArtists(); err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

// TestGetJSON_UnreachableServer verifies that a connection error is returned when
// the server is not reachable.
func TestGetJSON_UnreachableServer(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", 1*time.Second)
	if _, err := client.GetArtists(); err == nil {
		t.Fatal("expected connection error, got nil")
	}
}
