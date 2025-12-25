// Package dateparse provides natural language date parsing for CLI flags.
package dateparse

import (
	"fmt"
	"time"

	"github.com/tj/go-naturaldate"
)

// Parse parses a date string which can be:
// - Natural language: "today", "tomorrow", "next week", "last month", etc.
// - ISO 8601 date: "2025-01-15"
// - ISO 8601 datetime: "2025-01-15T09:00:00"
//
// The reference time is used for relative expressions (e.g., "tomorrow" is relative to ref).
// If ref is zero, time.Now() is used.
func Parse(s string, ref time.Time) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	if ref.IsZero() {
		ref = time.Now()
	}

	// Try ISO 8601 datetime first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try ISO 8601 datetime without timezone
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, ref.Location()); err == nil {
		return t, nil
	}

	// Try ISO 8601 date only (midnight local time)
	if t, err := time.ParseInLocation("2006-01-02", s, ref.Location()); err == nil {
		return t, nil
	}

	// Try natural language parsing with future direction (for "next week", etc.)
	t, err := naturaldate.Parse(s, ref, naturaldate.WithDirection(naturaldate.Future))
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse date %q: %w", s, err)
	}

	return t, nil
}

// ParseWithPast parses a date string with past direction for expressions like "last week".
func ParseWithPast(s string, ref time.Time) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	if ref.IsZero() {
		ref = time.Now()
	}

	// Try ISO formats first (same as Parse)
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, ref.Location()); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, ref.Location()); err == nil {
		return t, nil
	}

	// Try natural language parsing with past direction
	t, err := naturaldate.Parse(s, ref, naturaldate.WithDirection(naturaldate.Past))
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse date %q: %w", s, err)
	}

	return t, nil
}

// StartOfDay returns the start of day (midnight) for the given time.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of day (23:59:59.999999999) for the given time.
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// AddDays adds the specified number of days to a time.
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// FormatISO8601 formats a time as ISO 8601 string for Graph API.
func FormatISO8601(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseDuration parses a duration string like "30m", "1h", "90m"
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
