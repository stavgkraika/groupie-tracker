# Groupie Trackers

A Go web application that consumes the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api) and presents artists, members, album info, and concert relations in a clean UI. Concert locations are geocoded and displayed on an interactive map. Built with the Go standard library only — no third-party Go dependencies.

## Requirements

- Go 1.25+
- Internet access (the upstream API and Nominatim geocoding service are fetched at runtime)

## Run

```bash
go run ./cmd
```

Open http://localhost:8080

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `ADDR` | `:8080` | Address the server listens on |
| `GROUPIE_API_BASE` | `https://groupietrackers.herokuapp.com/api` | Base URL of the upstream API |

Example with custom port:

```bash
ADDR=:9090 go run ./cmd
```

## Test

```bash
go test ./...
```

## Project structure

```
cmd/
  main.go               # Entry point: wires server, routes, templates
  main_test.go          # Tests for getenv and HTTP route registration
internal/
  api/
    client.go           # HTTP client for the upstream API
    client_test.go      # Tests for the HTTP client
    models.go           # Raw API response types
  geo/
    geocoder.go         # Nominatim geocoder with in-memory cache and rate limiting
    geocoder_test.go    # Tests for geocoding, caching, and query formatting
  handlers/
    app.go              # HTTP handlers and middleware
  service/
    repository.go       # In-memory store, filter logic, and data assembly
    format.go           # Location/date normalisation helpers
    repository_test.go  # Tests for data building and search
    format_test.go      # Tests for normalisation helpers
static/
  css/style.css
  js/app.js             # Live search, filters, and refresh (fetch API, no framework)
templates/
  home.html             # Artist grid with search bar, filter panel, and stats
  artist.html           # Artist detail with members, concert table, and interactive map
  error.html            # Shared error page
```

## Routes

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Home page — artist grid with optional filter params |
| `GET` | `/artist?id=<n>` | Artist detail page |
| `GET` | `/api/search` | JSON search/filter endpoint (used by the filter panel) |
| `GET` | `/api/suggest?q=<text>` | JSON typed autocomplete suggestions for the search bar |
| `GET` | `/api/geocode?id=<n>` | Returns geocoded concert locations for an artist as JSON |
| `POST` | `/api/refresh` | Re-fetches all data from the upstream API |
| `GET` | `/static/` | Static assets (CSS, JS) |

## Filter parameters

All parameters are optional and combinable. They are accepted by both `GET /` and `GET /api/search`.

| Parameter | Type | Description |
|---|---|---|
| `q` | string | Free-text search — matches name, members, year, location, date |
| `creation_min` | int | Creation date ≥ value |
| `creation_max` | int | Creation date ≤ value |
| `album_min` | int | First album year ≥ value |
| `album_max` | int | First album year ≤ value |
| `members_min` | int | Member count ≥ value |
| `members_max` | int | Member count ≤ value |
| `location` | string | Has a concert in a location containing this string |

## Features

- Search suggestions — typing in the main search bar calls `/api/suggest` and shows matching artists, bands, members, locations, album dates, years, and concert dates with a role badge such as `band`, `member`, `location`, or `album year`
- Filter panel — sticky sidebar with range inputs for creation date, first album year, and member count, plus a concert location field; all filters are combinable
- Checkbox shortcuts — member count checkboxes sync to the min/max range inputs for quick exact-count filtering
- Live filtering — every input change is debounced 250 ms, then calls `/api/search` via `fetch()` and re-renders the grid without a page reload
- Reset button — clears all filters and restores the full artist list
- Refresh button — sends a `POST /api/refresh` to pull fresh data from the upstream API at runtime
- Concert relations — locations and dates from the `relation` endpoint are joined and displayed in a table on each artist page
- Interactive concert map — each artist detail page includes a Leaflet.js map (OpenStreetMap tiles) with a marker for every concert location; markers show a popup with the location name and dates; the map auto-zooms to fit all venues
- Click-to-map geolocation — concert locations in the artist table are clickable; selecting one scrolls to the map, centers that exact marker, and opens its popup without the popup nudging the pin away from the middle
- Geocoding — concert location slugs (e.g. `north_carolina-usa`) are converted to geographic coordinates via the [Nominatim](https://nominatim.openstreetmap.org) API; results are cached in memory and requests are rate-limited to one per second per the Nominatim usage policy
- Stats bar — total artists, total concerts, unique locations, and average group size computed at load time
- Error handling — returns proper 400 / 404 / 405 / 500 responses rendered through the shared error template
- Middleware — request logging and panic recovery on every request
- Thread-safe in-memory store — `sync.RWMutex` guards concurrent reads and refreshes
