// Package geo provides geocoding via the Nominatim OpenStreetMap API.
// No API key is required. Requests are rate-limited to one per second as
// required by the Nominatim usage policy.
package geo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Result holds the geographic coordinates for a single location string.
type Result struct {
	Location string  `json:"location"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
}

// Geocoder geocodes location strings using the Nominatim REST API.
// Results are cached in memory so each unique location is only fetched once.
// Concurrent callers share a single rate-limit token bucket (1 req/sec).
type Geocoder struct {
	httpClient *http.Client
	mu         sync.Mutex
	cache      map[string]*Result // nil value means "not found"
	ticker     *time.Ticker       // one tick per second — the Nominatim rate limit
}

// NewGeocoder creates a Geocoder with a 10-second HTTP timeout.
func NewGeocoder() *Geocoder {
	return &Geocoder{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cache:      make(map[string]*Result),
		ticker:     time.NewTicker(time.Second),
	}
}

// GeocodeAll geocodes every location in the slice concurrently, respecting the
// Nominatim 1 req/sec limit via a shared ticker. Results are returned in the
// same order as the input; locations that cannot be resolved are returned as nil.
func (g *Geocoder) GeocodeAll(locations []string) []*Result {
	results := make([]*Result, len(locations))
	var wg sync.WaitGroup

	for i, loc := range locations {
		wg.Add(1)
		go func(idx int, location string) {
			defer wg.Done()
			// Block until a rate-limit token is available.
			<-g.ticker.C
			r, err := g.geocodeOne(location)
			if err == nil {
				results[idx] = r
			}
		}(i, loc)
	}

	wg.Wait()
	return results
}

// Geocode converts a single location string into coordinates.
// Returns nil, nil when Nominatim finds no match.
func (g *Geocoder) Geocode(location string) (*Result, error) {
	<-g.ticker.C
	return g.geocodeOne(location)
}

// geocodeOne performs the actual lookup, consulting the cache first.
func (g *Geocoder) geocodeOne(location string) (*Result, error) {
	g.mu.Lock()
	if cached, ok := g.cache[location]; ok {
		g.mu.Unlock()
		return cached, nil
	}
	g.mu.Unlock()

	query := nominatimQuery(location)
	apiURL := "https://nominatim.openstreetmap.org/search?format=json&limit=1&q=" +
		url.QueryEscape(query)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("geocode build request: %w", err)
	}
	req.Header.Set("User-Agent", "groupie-tracker/1.0")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocode request: %w", err)
	}
	defer resp.Body.Close()

	var hits []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hits); err != nil {
		return nil, fmt.Errorf("geocode decode: %w", err)
	}

	var result *Result
	if len(hits) > 0 {
		lat, _ := strconv.ParseFloat(hits[0].Lat, 64)
		lon, _ := strconv.ParseFloat(hits[0].Lon, 64)
		result = &Result{Location: location, Lat: lat, Lon: lon}
	}

	g.mu.Lock()
	g.cache[location] = result
	g.mu.Unlock()

	return result, nil
}

// nominatimQuery converts the raw API location slug (e.g. "north_carolina-usa")
// into a human-readable search string ("north carolina usa").
func nominatimQuery(location string) string {
	out := make([]byte, len(location))
	for i := range location {
		switch location[i] {
		case '-', '_':
			out[i] = ' '
		default:
			out[i] = location[i]
		}
	}
	return string(out)
}
