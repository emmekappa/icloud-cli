package icloud

import (
	"strings"
	"testing"
)

func TestParsePriority(t *testing.T) {
	cases := []struct {
		in        string
		want      int
		wantError bool
	}{
		{"", 0, false},
		{"none", 0, false},
		{"NONE", 0, false},
		{"  none  ", 0, false},
		{"high", 1, false},
		{"HIGH", 1, false},
		{"medium", 5, false},
		{"med", 5, false},
		{"low", 9, false},
		{"0", 0, false},
		{"1", 1, false},
		{"5", 5, false},
		{"9", 9, false},
		{"10", 0, true},
		{"-1", 0, true},
		{"foo", 0, true},
	}
	for _, c := range cases {
		got, err := parsePriority(c.in)
		if (err != nil) != c.wantError {
			t.Errorf("parsePriority(%q): wantError=%v got err=%v", c.in, c.wantError, err)
			continue
		}
		if err == nil && got != c.want {
			t.Errorf("parsePriority(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseDue(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		got, err := parseDue("")
		if err != nil || got != nil {
			t.Fatalf("parseDue(\"\") = (%v, %v), want (nil, nil)", got, err)
		}
	})

	t.Run("date-only has no hour", func(t *testing.T) {
		got, err := parseDue("2026-12-31")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || got.Year != 2026 || got.Month != 12 || got.Day != 31 {
			t.Fatalf("unexpected Y/M/D: %+v", got)
		}
		if got.Hour != nil || got.Minute != nil {
			t.Fatalf("date-only should not set hour/minute, got %+v", got)
		}
	})

	t.Run("datetime space sets hour and minute", func(t *testing.T) {
		got, err := parseDue("2026-12-31 18:00")
		if err != nil || got == nil {
			t.Fatalf("parseDue err=%v got=%v", err, got)
		}
		if got.Hour == nil || *got.Hour != 18 {
			t.Fatalf("want hour=18, got %+v", got)
		}
		if got.Minute == nil || *got.Minute != 0 {
			t.Fatalf("want minute=0, got %+v", got)
		}
	})

	t.Run("datetime T separator accepted", func(t *testing.T) {
		got, err := parseDue("2026-12-31T09:15")
		if err != nil || got == nil || got.Hour == nil || *got.Hour != 9 || *got.Minute != 15 {
			t.Fatalf("unexpected: got=%+v err=%v", got, err)
		}
	})

	t.Run("invalid input errors", func(t *testing.T) {
		for _, s := range []string{"not a date", "2026-13-01", "2026/12/31"} {
			if _, err := parseDue(s); err == nil || !strings.Contains(err.Error(), "invalid due format") {
				t.Errorf("parseDue(%q) should error, got err=%v", s, err)
			}
		}
	})
}
