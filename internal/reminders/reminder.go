package reminders

import "errors"

var ErrNotFound = errors.New("reminder not found")

type List struct {
	ID                  string `json:"id"`
	Title               string `json:"title"`
	Color               string `json:"color,omitempty"`
	AllowsModifications bool   `json:"allows_modifications"`
	IsDefault           bool   `json:"is_default"`
	Source              string `json:"source,omitempty"`
}

type Reminder struct {
	UID         string  `json:"uid"`
	Title       string  `json:"title"`
	Notes       string  `json:"notes,omitempty"`
	URL         string  `json:"url,omitempty"`
	List        string  `json:"list"`
	ListID      string  `json:"list_id"`
	Completed   bool    `json:"completed"`
	Priority    int     `json:"priority"`
	Due         string  `json:"due,omitempty"`
	DueHasTime  bool    `json:"due_has_time,omitempty"`
	CompletedAt string  `json:"completed_at,omitempty"`
	CreatedAt   string  `json:"created_at,omitempty"`
	ModifiedAt  string  `json:"modified_at,omitempty"`
	Alarms      []Alarm `json:"alarms,omitempty"`
}

type Alarm struct {
	Type          string  `json:"type"`
	At            string  `json:"at,omitempty"`
	OffsetSeconds float64 `json:"offset_seconds,omitempty"`
}

type DueInput struct {
	Year    int  `json:"year"`
	Month   int  `json:"month"`
	Day     int  `json:"day"`
	Hour    *int `json:"hour,omitempty"`
	Minute  *int `json:"minute,omitempty"`
}

type CreateInput struct {
	Title    string    `json:"title"`
	ListID   string    `json:"list_id,omitempty"`
	Notes    string    `json:"notes,omitempty"`
	URL      string    `json:"url,omitempty"`
	Priority *int      `json:"priority,omitempty"`
	Due      *DueInput `json:"due,omitempty"`
}

type UpdateInput struct {
	Title    *string   `json:"title,omitempty"`
	ListID   *string   `json:"list_id,omitempty"`
	Notes    *string   `json:"notes,omitempty"`
	URL      *string   `json:"url,omitempty"`
	Priority *int      `json:"priority,omitempty"`
	Due      *DueInput `json:"due,omitempty"`
	ClearDue bool      `json:"clear_due,omitempty"`
}

type Backend interface {
	RequestAccess() error
	ListLists() ([]List, error)
	ListReminders(listID string, includeCompleted bool) ([]Reminder, error)
	GetReminder(uid string) (*Reminder, error)
	Create(in CreateInput) (*Reminder, error)
	Update(uid string, in UpdateInput) (*Reminder, error)
	SetCompleted(uid string, completed bool) (*Reminder, error)
}

func New() Backend {
	return newBackend()
}

// PriorityLabel maps an RFC 5545 priority (0-9) to the label Apple Reminders shows.
// 0 = none, 1-4 = high, 5 = medium, 6-9 = low.
func PriorityLabel(p int) string {
	switch {
	case p == 0:
		return "none"
	case p >= 1 && p <= 4:
		return "high"
	case p == 5:
		return "medium"
	case p >= 6 && p <= 9:
		return "low"
	}
	return ""
}
