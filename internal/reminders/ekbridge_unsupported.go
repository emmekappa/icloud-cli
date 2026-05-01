//go:build !darwin || !cgo

package reminders

import "errors"

var errUnsupported = errors.New(
	"reminder commands require a CGO-enabled macOS build: " +
		"iCloud Reminders are accessed via Apple's EventKit framework, " +
		"which exists only on macOS. Build this binary on macOS with CGO_ENABLED=1.",
)

type stubBackend struct{}

func newBackend() Backend { return stubBackend{} }

func (stubBackend) RequestAccess() error                           { return errUnsupported }
func (stubBackend) ListLists() ([]List, error)                     { return nil, errUnsupported }
func (stubBackend) ListReminders(string, bool) ([]Reminder, error) { return nil, errUnsupported }
func (stubBackend) GetReminder(string) (*Reminder, error)          { return nil, errUnsupported }
func (stubBackend) Create(CreateInput) (*Reminder, error)          { return nil, errUnsupported }
func (stubBackend) Update(string, UpdateInput) (*Reminder, error)  { return nil, errUnsupported }
func (stubBackend) SetCompleted(string, bool) (*Reminder, error)   { return nil, errUnsupported }
