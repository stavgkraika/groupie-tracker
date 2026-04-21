// Package api contains the raw response types that mirror the upstream API's JSON schema.
package api

// Artist is the raw API representation of a music artist.
type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Locations    string   `json:"locations"`    // URL to the artist's locations endpoint
	ConcertDates string   `json:"concertDates"` // URL to the artist's dates endpoint
	Relations    string   `json:"relations"`    // URL to the artist's relations endpoint
}

// LocationsIndex is the top-level wrapper returned by /locations.
type LocationsIndex struct {
	Index []Location `json:"index"`
}

// DatesIndex is the top-level wrapper returned by /dates.
type DatesIndex struct {
	Index []Date `json:"index"`
}

// RelationsIndex is the top-level wrapper returned by /relation.
type RelationsIndex struct {
	Index []Relation `json:"index"`
}

// Location holds the concert locations for a single artist.
type Location struct {
	ID        int      `json:"id"`
	Locations []string `json:"locations"`
	Dates     string   `json:"dates"` // URL to the artist's dates endpoint
}

// Date holds the concert dates for a single artist.
type Date struct {
	ID    int      `json:"id"`
	Dates []string `json:"dates"`
}

// Relation maps each concert location to its list of dates for a single artist.
type Relation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}
