package reminders

import "testing"

func TestPriorityLabel(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "none"},
		{1, "high"},
		{2, "high"},
		{3, "high"},
		{4, "high"},
		{5, "medium"},
		{6, "low"},
		{7, "low"},
		{8, "low"},
		{9, "low"},
		{-1, ""},
		{10, ""},
		{100, ""},
	}
	for _, c := range cases {
		got := PriorityLabel(c.in)
		if got != c.want {
			t.Errorf("PriorityLabel(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
