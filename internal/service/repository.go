// Package service implements the in-memory data store, search logic, and data assembly
// from the raw upstream API types into view models used by the handlers.
package service

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"

	"groupie-tracker/internal/api"
)

// ErrArtistNotFound is returned by ByID when no artist matches the given id.
var ErrArtistNotFound = errors.New("artist not found")

// Repository is a thread-safe in-memory store of assembled artist view models.
// A sync.RWMutex allows many concurrent reads while a refresh holds a write lock.
type Repository struct {
	client *api.Client

	mu      sync.RWMutex
	artists []ArtistView
	stats   Stats
}

// ArtistView is the assembled view model served to templates and the JSON search API.
// It combines fields from the Artist, Location, Date, and Relation API responses.
type ArtistView struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Image          string    `json:"image"`
	Members        []string  `json:"members"`
	MemberCount    int       `json:"memberCount"`
	CreationDate   int       `json:"creationDate"`
	FirstAlbum     string    `json:"firstAlbum"`
	FirstAlbumYear int       `json:"firstAlbumYear"`
	Concerts       []Concert `json:"concerts"`
	LocationCount  int       `json:"locationCount"`
	ConcertCount   int       `json:"concertCount"`
}

// Concert pairs a normalised location string with its list of concert dates.
type Concert struct {
	Location string   `json:"location"`
	Dates    []string `json:"dates"`
}

// Stats holds aggregate figures computed across all artists during a refresh.
type Stats struct {
	ArtistCount      int
	TotalConcerts    int
	UniqueLocations  int
	AverageGroupSize float64
}

// NewRepository creates a Repository backed by the given API client.
func NewRepository(client *api.Client) *Repository {
	return &Repository{client: client}
}

// LoadForTest seeds the repository directly from raw API slices without making
// any network calls. Intended for use in tests only.
func (r *Repository) LoadForTest(artists []api.Artist, relations []api.Relation) {
	views, stats := buildArtistViews(artists, nil, nil, relations)
	r.mu.Lock()
	r.artists = views
	r.stats = stats
	r.mu.Unlock()
}

// Refresh fetches all four API endpoints, assembles the view models, and atomically
// replaces the cached data. The existing cache is untouched if any fetch fails.
func (r *Repository) Refresh() error {
	artists, err := r.client.GetArtists()
	if err != nil {
		return err
	}

	locations, err := r.client.GetLocations()
	if err != nil {
		return err
	}

	dates, err := r.client.GetDates()
	if err != nil {
		return err
	}

	relations, err := r.client.GetRelations()
	if err != nil {
		return err
	}

	views, stats := buildArtistViews(artists, locations, dates, relations)

	r.mu.Lock()
	r.artists = views
	r.stats = stats
	r.mu.Unlock()

	return nil
}

// All returns a deep copy of every cached artist view.
func (r *Repository) All() []ArtistView {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneArtists(r.artists)
}

// Stats returns the aggregate statistics computed during the last refresh.
func (r *Repository) Stats() Stats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats
}

// ByID returns a deep copy of the artist with the given id, or ErrArtistNotFound.
func (r *Repository) ByID(id int) (ArtistView, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, artist := range r.artists {
		if artist.ID == id {
			return cloneArtist(artist), nil
		}
	}

	return ArtistView{}, ErrArtistNotFound
}

// Filter holds the optional range and checkbox constraints for the filter panel.
// Zero values mean "no constraint" for that field.
type Filter struct {
	Query           string // free-text search
	CreationMin     int    // creation date >= CreationMin (0 = no lower bound)
	CreationMax     int    // creation date <= CreationMax (0 = no upper bound)
	FirstAlbumMin   int    // first album year >= FirstAlbumMin
	FirstAlbumMax   int    // first album year <= FirstAlbumMax
	MembersMin      int    // member count >= MembersMin
	MembersMax      int    // member count <= MembersMax
	Location        string // concert location contains this string (case-insensitive)
}

// Search returns all artists whose name, members, creation year, first album,
// concert locations, or concert dates contain the query string (case-insensitive).
// An empty query returns all artists.
func (r *Repository) Search(query string) []ArtistView {
	return r.Filter(Filter{Query: query})
}

// Filter applies all non-zero constraints in f and returns the matching artists.
func (r *Repository) Filter(f Filter) []ArtistView {
	artists := r.All()
	var matches []ArtistView
	for _, artist := range artists {
		if matchesFilter(artist, f) {
			matches = append(matches, artist)
		}
	}
	return matches
}

// matchesFilter reports whether artist satisfies every constraint in f.
func matchesFilter(artist ArtistView, f Filter) bool {
	// Free-text search.
	if q := strings.TrimSpace(strings.ToLower(f.Query)); q != "" {
		if !containsArtist(artist, q) {
			return false
		}
	}
	// Creation date range.
	if f.CreationMin > 0 && artist.CreationDate < f.CreationMin {
		return false
	}
	if f.CreationMax > 0 && artist.CreationDate > f.CreationMax {
		return false
	}
	// First album year range.
	if f.FirstAlbumMin > 0 && artist.FirstAlbumYear < f.FirstAlbumMin {
		return false
	}
	if f.FirstAlbumMax > 0 && artist.FirstAlbumYear > f.FirstAlbumMax {
		return false
	}
	// Member count range.
	if f.MembersMin > 0 && artist.MemberCount < f.MembersMin {
		return false
	}
	if f.MembersMax > 0 && artist.MemberCount > f.MembersMax {
		return false
	}
	// Concert location substring.
	if loc := strings.TrimSpace(strings.ToLower(f.Location)); loc != "" {
		found := false
		for _, concert := range artist.Concerts {
			if strings.Contains(strings.ToLower(concert.Location), loc) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// containsArtist reports whether any searchable field of artist contains q.
func containsArtist(artist ArtistView, q string) bool {
	if strings.Contains(strings.ToLower(artist.Name), q) {
		return true
	}
	if strings.Contains(strings.ToLower(artist.FirstAlbum), q) {
		return true
	}
	if strings.Contains(strings.ToLower(strconv.Itoa(artist.CreationDate)), q) {
		return true
	}
	if strings.Contains(strings.ToLower(strconv.Itoa(artist.FirstAlbumYear)), q) {
		return true
	}

	for _, member := range artist.Members {
		if strings.Contains(strings.ToLower(member), q) {
			return true
		}
	}
	for _, concert := range artist.Concerts {
		if strings.Contains(strings.ToLower(concert.Location), q) {
			return true
		}
		for _, date := range concert.Dates {
			if strings.Contains(strings.ToLower(date), q) {
				return true
			}
		}
	}
	return false
}

// buildArtistViews assembles ArtistView values from the four raw API slices and
// computes aggregate Stats. Relations are the primary source for concert data;
// the locations and dates slices are accepted for API symmetry but not used directly.
func buildArtistViews(artists []api.Artist, locations []api.Location, dates []api.Date, relations []api.Relation) ([]ArtistView, Stats) {
	// Index relations by artist ID for O(1) lookup.
	relationByID := make(map[int]api.Relation, len(relations))
	for _, relation := range relations {
		relationByID[relation.ID] = relation
	}

	_ = locations
	_ = dates

	result := make([]ArtistView, 0, len(artists))
	uniqueLocations := make(map[string]struct{})
	totalConcerts := 0
	totalMembers := 0

	for _, artist := range artists {
		relation := relationByID[artist.ID]
		concerts := make([]Concert, 0, len(relation.DatesLocations))

		// Sort location keys so the concert table order is deterministic.
		keys := make([]string, 0, len(relation.DatesLocations))
		for rawLocation := range relation.DatesLocations {
			keys = append(keys, rawLocation)
		}
		sort.Strings(keys)

		concertCount := 0
		for _, rawLocation := range keys {
			normalizedLocation := NormalizeLocation(rawLocation)
			cleanDates := NormalizeDates(relation.DatesLocations[rawLocation])
			concertCount += len(cleanDates)
			uniqueLocations[normalizedLocation] = struct{}{}
			concerts = append(concerts, Concert{
				Location: normalizedLocation,
				Dates:    cleanDates,
			})
		}

		firstAlbumYear := ExtractYear(artist.FirstAlbum)
		totalConcerts += concertCount
		totalMembers += len(artist.Members)

		result = append(result, ArtistView{
			ID:             artist.ID,
			Name:           artist.Name,
			Image:          artist.Image,
			Members:        append([]string(nil), artist.Members...),
			MemberCount:    len(artist.Members),
			CreationDate:   artist.CreationDate,
			FirstAlbum:     artist.FirstAlbum,
			FirstAlbumYear: firstAlbumYear,
			Concerts:       concerts,
			LocationCount:  len(concerts),
			ConcertCount:   concertCount,
		})
	}

	// Keep artists sorted by ID for a stable display order.
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	stats := Stats{
		ArtistCount:     len(result),
		TotalConcerts:   totalConcerts,
		UniqueLocations: len(uniqueLocations),
	}
	if len(result) > 0 {
		stats.AverageGroupSize = float64(totalMembers) / float64(len(result))
	}

	return result, stats
}

// DistinctMemberCounts returns a sorted slice of every unique member count
// present in the data, used to populate the checkbox group in the filter panel.
func (r *Repository) DistinctMemberCounts() []int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[int]struct{})
	for _, a := range r.artists {
		seen[a.MemberCount] = struct{}{}
	}
	counts := make([]int, 0, len(seen))
	for n := range seen {
		counts = append(counts, n)
	}
	sort.Ints(counts)
	return counts
}

// cloneArtists returns a deep copy of a slice of ArtistView values.
func cloneArtists(in []ArtistView) []ArtistView {
	out := make([]ArtistView, 0, len(in))
	for _, artist := range in {
		out = append(out, cloneArtist(artist))
	}
	return out
}

// cloneArtist returns a deep copy of a single ArtistView,
// ensuring callers cannot mutate the cached slices.
func cloneArtist(in ArtistView) ArtistView {
	out := in
	out.Members = append([]string(nil), in.Members...)
	out.Concerts = make([]Concert, 0, len(in.Concerts))
	for _, concert := range in.Concerts {
		out.Concerts = append(out.Concerts, Concert{
			Location: concert.Location,
			Dates:    append([]string(nil), concert.Dates...),
		})
	}
	return out
}
