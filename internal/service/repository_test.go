package service

import (
	"testing"

	"groupie-tracker/internal/api"
)

// TestBuildArtistViews verifies that buildArtistViews correctly assembles an
// ArtistView from raw API data and computes accurate Stats.
func TestBuildArtistViews(t *testing.T) {
	artists := []api.Artist{{
		ID:           1,
		Name:         "Queen",
		Image:        "queen.jpeg",
		Members:      []string{"Freddie Mercury", "Brian May"},
		CreationDate: 1970,
		FirstAlbum:   "14-12-1973",
	}}

	locations := []api.Location{{ID: 1, Locations: []string{"osaka-japan"}}}
	dates := []api.Date{{ID: 1, Dates: []string{"*28-01-2020"}}}
	relations := []api.Relation{{
		ID: 1,
		DatesLocations: map[string][]string{
			"osaka-japan": {"28-01-2020"},
		},
	}}

	got, stats := buildArtistViews(artists, locations, dates, relations)
	if len(got) != 1 {
		t.Fatalf("expected one artist, got %d", len(got))
	}
	if got[0].ConcertCount != 1 {
		t.Fatalf("expected one concert date, got %d", got[0].ConcertCount)
	}
	if got[0].LocationCount != 1 {
		t.Fatalf("expected one location, got %d", got[0].LocationCount)
	}
	if got[0].Concerts[0].Location != "Osaka, Japan" {
		t.Fatalf("unexpected location: %q", got[0].Concerts[0].Location)
	}
	if stats.ArtistCount != 1 || stats.TotalConcerts != 1 || stats.UniqueLocations != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

// TestContainsArtist verifies that containsArtist matches on all searchable fields:
// name, member name, creation year, first album year, location, and concert date.
func TestContainsArtist(t *testing.T) {
	artist := ArtistView{
		Name:         "Queen",
		Members:      []string{"Freddie Mercury"},
		CreationDate: 1970,
		FirstAlbum:   "14-12-1973",
		Concerts: []Concert{{
			Location: "Osaka, Japan",
			Dates:    []string{"28-01-2020"},
		}},
	}

	cases := []string{"queen", "freddie", "1970", "1973", "osaka", "2020"}
	for _, tc := range cases {
		if !containsArtist(artist, tc) {
			t.Fatalf("expected query %q to match", tc)
		}
	}
}
