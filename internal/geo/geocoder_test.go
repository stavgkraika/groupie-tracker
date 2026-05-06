package geo

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newTestGeocoder(fn roundTripFunc) *Geocoder {
	return &Geocoder{
		httpClient: &http.Client{Transport: fn},
		cache:      make(map[string]*Result),
		ticker:     time.NewTicker(time.Millisecond),
	}
}

func response(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNominatimQuery(t *testing.T) {
	cases := map[string]string{
		"north_carolina-usa": "north carolina usa",
		"london-uk":          "london uk",
		"rio_de_janeiro":     "rio de janeiro",
		"athens":             "athens",
	}

	for input, want := range cases {
		if got := nominatimQuery(input); got != want {
			t.Errorf("nominatimQuery(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNewGeocoderInitializesDependencies(t *testing.T) {
	g := NewGeocoder()
	defer g.ticker.Stop()

	if g.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	if g.cache == nil {
		t.Fatal("cache is nil")
	}
	if g.ticker == nil {
		t.Fatal("ticker is nil")
	}
}

func TestGeocodeOneBuildsRequestParsesResultAndCaches(t *testing.T) {
	var calls int
	g := newTestGeocoder(func(r *http.Request) (*http.Response, error) {
		calls++

		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("User-Agent"); got != "groupie-tracker/1.0" {
			t.Fatalf("User-Agent = %q, want groupie-tracker/1.0", got)
		}
		if got := r.URL.Query().Get("q"); got != "north carolina usa" {
			t.Fatalf("q = %q, want north carolina usa", got)
		}

		return response(`[{"lat":"35.7596","lon":"-79.0193"}]`), nil
	})
	defer g.ticker.Stop()

	got, err := g.geocodeOne("north_carolina-usa")
	if err != nil {
		t.Fatalf("geocodeOne returned error: %v", err)
	}
	if got == nil {
		t.Fatal("geocodeOne returned nil result")
	}
	if got.Location != "north_carolina-usa" || got.Lat != 35.7596 || got.Lon != -79.0193 {
		t.Fatalf("unexpected result: %+v", got)
	}

	cached, err := g.geocodeOne("north_carolina-usa")
	if err != nil {
		t.Fatalf("cached geocodeOne returned error: %v", err)
	}
	if cached != got {
		t.Fatal("cached result was not reused")
	}
	if calls != 1 {
		t.Fatalf("HTTP calls = %d, want 1", calls)
	}
}

func TestGeocodeOneCachesNotFound(t *testing.T) {
	var calls int
	g := newTestGeocoder(func(*http.Request) (*http.Response, error) {
		calls++
		return response(`[]`), nil
	})
	defer g.ticker.Stop()

	for i := 0; i < 2; i++ {
		got, err := g.geocodeOne("unknown-place")
		if err != nil {
			t.Fatalf("geocodeOne returned error: %v", err)
		}
		if got != nil {
			t.Fatalf("geocodeOne returned %+v, want nil", got)
		}
	}
	if calls != 1 {
		t.Fatalf("HTTP calls = %d, want 1", calls)
	}
}

func TestGeocodeOneReturnsRequestError(t *testing.T) {
	wantErr := errors.New("network down")
	g := newTestGeocoder(func(*http.Request) (*http.Response, error) {
		return nil, wantErr
	})
	defer g.ticker.Stop()

	got, err := g.geocodeOne("paris-france")
	if got != nil {
		t.Fatalf("result = %+v, want nil", got)
	}
	if err == nil || !strings.Contains(err.Error(), "geocode request") {
		t.Fatalf("error = %v, want geocode request error", err)
	}
}

func TestGeocodeOneReturnsDecodeError(t *testing.T) {
	g := newTestGeocoder(func(*http.Request) (*http.Response, error) {
		return response(`not-json`), nil
	})
	defer g.ticker.Stop()

	got, err := g.geocodeOne("paris-france")
	if got != nil {
		t.Fatalf("result = %+v, want nil", got)
	}
	if err == nil || !strings.Contains(err.Error(), "geocode decode") {
		t.Fatalf("error = %v, want geocode decode error", err)
	}
}

func TestGeocodeWaitsForTickerAndUsesCache(t *testing.T) {
	g := newTestGeocoder(func(*http.Request) (*http.Response, error) {
		t.Fatal("cached geocode should not make an HTTP request")
		return nil, nil
	})
	defer g.ticker.Stop()
	want := &Result{Location: "athens-greece", Lat: 37.9838, Lon: 23.7275}
	g.cache["athens-greece"] = want

	got, err := g.Geocode("athens-greece")
	if err != nil {
		t.Fatalf("Geocode returned error: %v", err)
	}
	if got != want {
		t.Fatalf("Geocode returned %+v, want cached result %+v", got, want)
	}
}

func TestGeocodeAllPreservesOrderAndSkipsFailedLookups(t *testing.T) {
	var mu sync.Mutex
	seen := make(map[string]int)
	g := newTestGeocoder(func(r *http.Request) (*http.Response, error) {
		query := r.URL.Query().Get("q")
		mu.Lock()
		seen[query]++
		mu.Unlock()

		switch query {
		case "paris france":
			return response(`[{"lat":"48.8566","lon":"2.3522"}]`), nil
		case "broken place":
			return nil, errors.New("lookup failed")
		case "berlin germany":
			return response(`[{"lat":"52.5200","lon":"13.4050"}]`), nil
		default:
			t.Fatalf("unexpected query %q", query)
			return nil, nil
		}
	})
	defer g.ticker.Stop()

	got := g.GeocodeAll([]string{"paris-france", "broken-place", "berlin-germany"})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0] == nil || got[0].Location != "paris-france" || got[0].Lat != 48.8566 || got[0].Lon != 2.3522 {
		t.Fatalf("first result = %+v", got[0])
	}
	if got[1] != nil {
		t.Fatalf("second result = %+v, want nil", got[1])
	}
	if got[2] == nil || got[2].Location != "berlin-germany" || got[2].Lat != 52.52 || got[2].Lon != 13.405 {
		t.Fatalf("third result = %+v", got[2])
	}

	for query, calls := range seen {
		if calls != 1 {
			t.Fatalf("query %q calls = %d, want 1", query, calls)
		}
	}
}
