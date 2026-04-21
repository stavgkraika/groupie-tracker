package service

import "testing"

// TestNormalizeLocation verifies that underscores become spaces, hyphens become
// ", " separators, and each word is title-cased.
func TestNormalizeLocation(t *testing.T) {
	got := NormalizeLocation("los_angeles-usa")
	want := "Los Angeles, Usa"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

// TestNormalizeDates verifies that the leading "*" marker is stripped from each date.
func TestNormalizeDates(t *testing.T) {
	got := NormalizeDates([]string{"*23-08-2019", "*22-08-2019"})
	if got[0] != "23-08-2019" || got[1] != "22-08-2019" {
		t.Fatalf("unexpected dates: %#v", got)
	}
}

// TestExtractYear verifies that the four-digit year is correctly parsed from a
// "DD-MM-YYYY" date string.
func TestExtractYear(t *testing.T) {
	got := ExtractYear("14-12-1973")
	if got != 1973 {
		t.Fatalf("got %d want %d", got, 1973)
	}
}
