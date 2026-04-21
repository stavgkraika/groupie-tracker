# Groupie Trackers

A Go web application that consumes the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api) and presents artists, members, album info, and concert relations in a clean UI. Built with the Go standard library only — no third-party dependencies.

## Requirements

- Go 1.25+
- Internet access (the API is fetched on startup)

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
  artist.html           # Artist detail with members and concert table
  error.html            # Shared error page
```

## Routes

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Home page — artist grid with optional filter params |
| `GET` | `/artist?id=<n>` | Artist detail page |
| `GET` | `/api/search` | JSON search/filter endpoint (used by the filter panel) |
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

- Filter panel — sticky sidebar with range inputs for creation date, first album year, and member count, plus a concert location field; all filters are combinable
- Checkbox shortcuts — member count checkboxes sync to the min/max range inputs for quick exact-count filtering
- Live filtering — every input change is debounced 250 ms, then calls `/api/search` via `fetch()` and re-renders the grid without a page reload
- Reset button — clears all filters and restores the full artist list
- Refresh button — sends a `POST /api/refresh` to pull fresh data from the upstream API at runtime
- Concert relations — locations and dates from the `relation` endpoint are joined and displayed in a table on each artist page
- Stats bar — total artists, total concerts, unique locations, and average group size computed at load time
- Error handling — returns proper 400 / 404 / 405 / 500 responses rendered through the shared error template
- Middleware — request logging and panic recovery on every request
- Thread-safe in-memory store — `sync.RWMutex` guards concurrent reads and refreshes
