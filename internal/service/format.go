// Package service — format.go contains helpers that normalise raw API strings
// into human-readable location names, clean date strings, and parsed years.
package service

import (
	"strconv"
	"strings"
)

// NormalizeLocation converts a raw API location slug into a display string.
// Underscores become spaces, hyphens become ", " separators, and each word is
// title-cased. Example: "los_angeles-usa" → "Los Angeles, Usa".
func NormalizeLocation(raw string) string {
	parts := strings.Split(raw, "-")
	for i, part := range parts {
		words := strings.Split(part, "_")
		for j, word := range words {
			words[j] = capitalize(word)
		}
		parts[i] = strings.Join(words, " ")
	}
	return strings.Join(parts, ", ")
}

// NormalizeDates strips the leading "*" that the API uses to mark past concert
// dates, returning clean "DD-MM-YYYY" strings.
func NormalizeDates(rawDates []string) []string {
	dates := make([]string, 0, len(rawDates))
	for _, raw := range rawDates {
		dates = append(dates, strings.TrimPrefix(raw, "*"))
	}
	return dates
}

// ExtractYear parses the four-digit year from a "DD-MM-YYYY" date string.
// Returns 0 if the string is malformed or the year cannot be parsed.
func ExtractYear(date string) int {
	parts := strings.Split(strings.TrimSpace(date), "-")
	if len(parts) == 0 {
		return 0
	}
	year, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return year
}

// capitalize returns s with its first character uppercased and the rest lowercased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	lower := strings.ToLower(s)
	return strings.ToUpper(lower[:1]) + lower[1:]
}
