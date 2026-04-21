// Package api provides an HTTP client for the Groupie Trackers upstream API.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client wraps an HTTP client configured for the upstream API base URL.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Client targeting baseURL with the given request timeout.
// A trailing slash on baseURL is trimmed to avoid double-slash URLs.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetArtists fetches the full list of artists from /artists.
func (c *Client) GetArtists() ([]Artist, error) {
	var artists []Artist
	if err := c.getJSON(c.baseURL+"/artists", &artists); err != nil {
		return nil, err
	}
	return artists, nil
}

// GetLocations fetches the locations index from /locations.
func (c *Client) GetLocations() ([]Location, error) {
	var payload LocationsIndex
	if err := c.getJSON(c.baseURL+"/locations", &payload); err != nil {
		return nil, err
	}
	return payload.Index, nil
}

// GetDates fetches the dates index from /dates.
func (c *Client) GetDates() ([]Date, error) {
	var payload DatesIndex
	if err := c.getJSON(c.baseURL+"/dates", &payload); err != nil {
		return nil, err
	}
	return payload.Index, nil
}

// GetRelations fetches the relations index from /relation.
func (c *Client) GetRelations() ([]Relation, error) {
	var payload RelationsIndex
	if err := c.getJSON(c.baseURL+"/relation", &payload); err != nil {
		return nil, err
	}
	return payload.Index, nil
}

// getJSON performs a GET request to url and JSON-decodes the response body into target.
// Returns an error if the request fails, the status is not 200, or decoding fails.
func (c *Client) getJSON(url string, target any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api returned status %d for %s", resp.StatusCode, url)
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode json from %s: %w", url, err)
	}

	return nil
}
