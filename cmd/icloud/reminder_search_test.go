package icloud

import (
	"testing"

	"github.com/jingkaihe/icloud-cli/internal/reminders"
)

func TestMatchesTerms(t *testing.T) {
	r := reminders.Reminder{
		Title: "Buy organic milk at Whole Foods",
		Notes: "Check expiration date",
	}

	// matchesTerms contract: callers pass already-lowercased terms.
	cases := []struct {
		name  string
		terms []string
		want  bool
	}{
		{"no terms matches everything", nil, true},
		{"empty slice matches everything", []string{}, true},
		{"single term in title", []string{"milk"}, true},
		{"match is case insensitive on haystack", []string{"whole foods"}, true},
		{"single term in notes", []string{"expiration"}, true},
		{"AND across title+notes", []string{"milk", "expiration"}, true},
		{"AND with one miss fails", []string{"milk", "unicorn"}, false},
		{"substring match", []string{"organ"}, true},
		{"non-matching term", []string{"bread"}, false},
	}
	for _, c := range cases {
		got := matchesTerms(r, c.terms)
		if got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
