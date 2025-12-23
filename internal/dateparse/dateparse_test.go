package dateparse

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	// Fixed reference time for consistent tests
	ref := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, got time.Time)
	}{
		{
			name:  "ISO 8601 date",
			input: "2025-01-20",
			check: func(t *testing.T, got time.Time) {
				if got.Year() != 2025 || got.Month() != 1 || got.Day() != 20 {
					t.Errorf("expected 2025-01-20, got %v", got)
				}
			},
		},
		{
			name:  "ISO 8601 datetime",
			input: "2025-01-20T14:30:00",
			check: func(t *testing.T, got time.Time) {
				if got.Hour() != 14 || got.Minute() != 30 {
					t.Errorf("expected 14:30, got %v", got)
				}
			},
		},
		{
			name:  "ISO 8601 with timezone",
			input: "2025-01-20T14:30:00Z",
			check: func(t *testing.T, got time.Time) {
				if got.Hour() != 14 || got.Minute() != 30 {
					t.Errorf("expected 14:30, got %v", got)
				}
			},
		},
		{
			name:  "today",
			input: "today",
			check: func(t *testing.T, got time.Time) {
				if got.Day() != ref.Day() {
					t.Errorf("expected day %d, got %d", ref.Day(), got.Day())
				}
			},
		},
		{
			name:  "tomorrow",
			input: "tomorrow",
			check: func(t *testing.T, got time.Time) {
				expected := ref.AddDate(0, 0, 1)
				if got.Day() != expected.Day() {
					t.Errorf("expected day %d, got %d", expected.Day(), got.Day())
				}
			},
		},
		{
			name:  "yesterday",
			input: "yesterday",
			check: func(t *testing.T, got time.Time) {
				expected := ref.AddDate(0, 0, -1)
				if got.Day() != expected.Day() {
					t.Errorf("expected day %d, got %d", expected.Day(), got.Day())
				}
			},
		},
		{
			name:  "next week",
			input: "next week",
			check: func(t *testing.T, got time.Time) {
				// Should be about 7 days from ref
				diff := got.Sub(ref)
				if diff < 6*24*time.Hour || diff > 8*24*time.Hour {
					t.Errorf("expected about 7 days from ref, got %v", diff)
				}
			},
		},
		{
			name:  "in 3 days",
			input: "in 3 days",
			check: func(t *testing.T, got time.Time) {
				expected := ref.AddDate(0, 0, 3)
				if got.Day() != expected.Day() {
					t.Errorf("expected day %d, got %d", expected.Day(), got.Day())
				}
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		// Note: go-naturaldate is quite permissive and may parse partial matches.
		// We don't test truly invalid strings as the library behavior varies.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestParseWithPast(t *testing.T) {
	ref := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)

	tests := []struct {
		name  string
		input string
		check func(t *testing.T, got time.Time)
	}{
		{
			name:  "last week",
			input: "last week",
			check: func(t *testing.T, got time.Time) {
				// Should be about 7 days before ref
				diff := ref.Sub(got)
				if diff < 6*24*time.Hour || diff > 8*24*time.Hour {
					t.Errorf("expected about 7 days before ref, got %v", diff)
				}
			},
		},
		{
			name:  "3 days ago",
			input: "3 days ago",
			check: func(t *testing.T, got time.Time) {
				expected := ref.AddDate(0, 0, -3)
				if got.Day() != expected.Day() {
					t.Errorf("expected day %d, got %d", expected.Day(), got.Day())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseWithPast(tt.input, ref)
			if err != nil {
				t.Errorf("ParseWithPast() error = %v", err)
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestStartOfDay(t *testing.T) {
	input := time.Date(2025, 1, 15, 14, 30, 45, 123, time.Local)
	got := StartOfDay(input)

	if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
		t.Errorf("expected midnight, got %v", got)
	}
	if got.Day() != 15 {
		t.Errorf("expected day 15, got %d", got.Day())
	}
}

func TestEndOfDay(t *testing.T) {
	input := time.Date(2025, 1, 15, 14, 30, 45, 123, time.Local)
	got := EndOfDay(input)

	if got.Hour() != 23 || got.Minute() != 59 || got.Second() != 59 {
		t.Errorf("expected 23:59:59, got %v", got)
	}
	if got.Day() != 15 {
		t.Errorf("expected day 15, got %d", got.Day())
	}
}

func TestAddDays(t *testing.T) {
	input := time.Date(2025, 1, 15, 10, 0, 0, 0, time.Local)

	got := AddDays(input, 5)
	if got.Day() != 20 {
		t.Errorf("expected day 20, got %d", got.Day())
	}

	got = AddDays(input, -3)
	if got.Day() != 12 {
		t.Errorf("expected day 12, got %d", got.Day())
	}
}

func TestFormatISO8601(t *testing.T) {
	input := time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC)
	got := FormatISO8601(input)

	expected := "2025-01-15T14:30:00Z"
	if got != expected {
		t.Errorf("expected %s, got %s", expected, got)
	}
}
