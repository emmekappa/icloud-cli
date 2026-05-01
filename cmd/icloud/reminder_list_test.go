package icloud

import "testing"

func TestParseReminderDue(t *testing.T) {
	cases := []struct {
		in     string
		wantOK bool
	}{
		{"", false},
		{"2026-12-31", true},
		{"2026-12-31T18:00:00+01:00", true},
		{"2026-12-31T18:00:00Z", true},
		{"not a date", false},
		{"31/12/2026", false},
	}
	for _, c := range cases {
		_, ok := parseReminderDue(c.in)
		if ok != c.wantOK {
			t.Errorf("parseReminderDue(%q) ok=%v, want %v", c.in, ok, c.wantOK)
		}
	}
}
