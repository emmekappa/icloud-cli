//go:build acceptance

package acceptance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	binaryPath       string
	testCalendarName = "icalendar-integration-test"
	calendarID       string
)

func TestMain(m *testing.M) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to find project root: %v\n", err)
		os.Exit(1)
	}

	binaryPath = filepath.Join(projectRoot, "icloud-test")

	fmt.Println("Building icloud binary...")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Binary built successfully")

	code := m.Run()

	os.Remove(binaryPath)
	os.Exit(code)
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (no go.mod found)")
		}
		dir = parent
	}
}

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()
	return output, err
}

func mustRunCLI(t *testing.T, args ...string) string {
	t.Helper()
	output, err := runCLI(t, args...)
	if err != nil {
		t.Fatalf("Command failed: %s %v\nOutput: %s\nError: %v", binaryPath, args, output, err)
	}
	return output
}

func assertContains(t *testing.T, output, expected string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain %q, got: %s", expected, output)
	}
}

func assertNotContains(t *testing.T, output, unexpected string) {
	t.Helper()
	if strings.Contains(output, unexpected) {
		t.Errorf("Expected output NOT to contain %q, got: %s", unexpected, output)
	}
}

func tomorrow() string {
	return time.Now().AddDate(0, 0, 1).Format("2006-01-02")
}

func nextWeek() string {
	return time.Now().AddDate(0, 0, 7).Format("2006-01-02")
}

func today() string {
	return time.Now().Format("2006-01-02")
}

func TestCalendarCRUD(t *testing.T) {
	t.Run("ListCalendars", func(t *testing.T) {
		output := mustRunCLI(t, "calendar", "list")
		assertContains(t, output, "Calendars for")
	})

	t.Run("CreateCalendar", func(t *testing.T) {
		output := mustRunCLI(t, "calendar", "create", testCalendarName)
		assertContains(t, output, "Successfully created calendar")
		assertContains(t, output, testCalendarName)

		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "ID:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					calendarID = parts[1]
				}
			}
		}
		if calendarID == "" {
			t.Fatal("Failed to extract calendar ID from output")
		}
		t.Logf("Created calendar with ID: %s", calendarID)
	})

	t.Run("VerifyCalendarInList", func(t *testing.T) {
		if calendarID == "" {
			t.Skip("Calendar not created")
		}
		output := mustRunCLI(t, "calendar", "list")
		assertContains(t, output, testCalendarName)
	})

	t.Run("UpdateCalendarName", func(t *testing.T) {
		if calendarID == "" {
			t.Skip("Calendar not created")
		}
		newName := testCalendarName + "-renamed"
		output := mustRunCLI(t, "calendar", "update", calendarID, "-n", newName)
		assertContains(t, output, "Successfully updated calendar")

		listOutput := mustRunCLI(t, "calendar", "list")
		assertContains(t, listOutput, newName)

		mustRunCLI(t, "calendar", "update", calendarID, "-n", testCalendarName)
	})
}

func TestEventCRUD(t *testing.T) {
	if calendarID == "" {
		t.Skip("Calendar not created")
	}

	var eventUID string
	var event2UID string

	t.Run("CreateEvent", func(t *testing.T) {
		output := mustRunCLI(t, "event", "create",
			"-t", "Test Event 1",
			"-s", tomorrow()+" 10:00",
			"-e", tomorrow()+" 11:00",
			"-c", calendarID,
			"-l", "Test Location",
			"-d", "Test Description")
		assertContains(t, output, "Event created")

		if idx := strings.Index(output, "UID: "); idx != -1 {
			rest := output[idx+5:]
			if endIdx := strings.Index(rest, ")"); endIdx != -1 {
				eventUID = rest[:endIdx]
			}
		}
		if eventUID == "" {
			t.Fatal("Failed to extract event UID")
		}
		t.Logf("Created event with UID: %s", eventUID)
	})

	t.Run("CreateEventWithoutEndTime", func(t *testing.T) {
		output := mustRunCLI(t, "event", "create",
			"-t", "Test Event 2 - No End",
			"-s", tomorrow()+" 14:00",
			"-c", calendarID)
		assertContains(t, output, "Event created")

		if idx := strings.Index(output, "UID: "); idx != -1 {
			rest := output[idx+5:]
			if endIdx := strings.Index(rest, ")"); endIdx != -1 {
				event2UID = rest[:endIdx]
			}
		}
	})

	t.Run("ListEventsInCalendar", func(t *testing.T) {
		output := mustRunCLI(t, "event", "list",
			"-c", calendarID,
			"-s", today(),
			"-e", nextWeek())
		assertContains(t, output, "Test Event 1")
		assertContains(t, output, "Test Event 2 - No End")
	})

	t.Run("ListEventsJSON", func(t *testing.T) {
		output := mustRunCLI(t, "event", "list",
			"-c", calendarID,
			"-s", today(),
			"-e", nextWeek(),
			"-o", "json")
		assertContains(t, output, `"title": "Test Event 1"`)
		assertContains(t, output, `"location": "Test Location"`)

		var events []map[string]interface{}
		if err := json.Unmarshal([]byte(output), &events); err != nil {
			t.Errorf("Failed to parse JSON output: %v", err)
		}
	})

	t.Run("GetEventDetails", func(t *testing.T) {
		if eventUID == "" {
			t.Skip("Event not created")
		}
		output := mustRunCLI(t, "event", "get", eventUID, "-c", calendarID)
		assertContains(t, output, "Test Event 1")
		assertContains(t, output, "Test Location")
		assertContains(t, output, "Test Description")
	})

	t.Run("GetEventJSON", func(t *testing.T) {
		if eventUID == "" {
			t.Skip("Event not created")
		}
		output := mustRunCLI(t, "event", "get", eventUID, "-c", calendarID, "-o", "json")
		assertContains(t, output, `"title": "Test Event 1"`)
	})

	t.Run("UpdateEventTitle", func(t *testing.T) {
		if eventUID == "" {
			t.Skip("Event not created")
		}
		output := mustRunCLI(t, "event", "update", eventUID,
			"-c", calendarID,
			"-t", "Updated Event Title")
		assertContains(t, output, "Event updated")

		getOutput := mustRunCLI(t, "event", "get", eventUID, "-c", calendarID)
		assertContains(t, getOutput, "Updated Event Title")
	})

	t.Run("UpdateEventLocationDescription", func(t *testing.T) {
		if eventUID == "" {
			t.Skip("Event not created")
		}
		output := mustRunCLI(t, "event", "update", eventUID,
			"-c", calendarID,
			"-l", "New Location",
			"-d", "New Description")
		assertContains(t, output, "Event updated")
	})

	t.Run("UpdateEventTime", func(t *testing.T) {
		if eventUID == "" {
			t.Skip("Event not created")
		}
		output := mustRunCLI(t, "event", "update", eventUID,
			"-c", calendarID,
			"-s", tomorrow()+" 15:00",
			"-e", tomorrow()+" 16:00")
		assertContains(t, output, "Event updated")
	})

	t.Run("DeleteEvent", func(t *testing.T) {
		if eventUID == "" {
			t.Skip("Event not created")
		}
		output := mustRunCLI(t, "event", "delete", eventUID, "-c", calendarID)
		assertContains(t, output, "Event deleted")
	})

	t.Run("VerifyEventDeleted", func(t *testing.T) {
		output := mustRunCLI(t, "event", "list",
			"-c", calendarID,
			"-s", today(),
			"-e", nextWeek())
		assertNotContains(t, output, "Updated Event Title")
	})

	t.Run("DeleteSecondEvent", func(t *testing.T) {
		if event2UID == "" {
			t.Skip("Second event not created")
		}
		output := mustRunCLI(t, "event", "delete", event2UID, "-c", calendarID)
		assertContains(t, output, "Event deleted")
	})
}

func TestRecurringEvents(t *testing.T) {
	if calendarID == "" {
		t.Skip("Calendar not created")
	}

	var recurringEventUID string

	t.Run("CreateRecurringEvent", func(t *testing.T) {
		output := mustRunCLI(t, "event", "create",
			"-t", "Daily Standup",
			"-s", tomorrow()+" 09:00",
			"-e", tomorrow()+" 09:30",
			"-c", calendarID,
			"-r", "FREQ=DAILY;COUNT=5")
		assertContains(t, output, "Event created")

		if idx := strings.Index(output, "UID: "); idx != -1 {
			rest := output[idx+5:]
			if endIdx := strings.Index(rest, ")"); endIdx != -1 {
				recurringEventUID = rest[:endIdx]
			}
		}
		if recurringEventUID == "" {
			t.Fatal("Failed to extract recurring event UID")
		}
		t.Logf("Created recurring event with UID: %s", recurringEventUID)
	})

	t.Run("ListRecurringEvents", func(t *testing.T) {
		output := mustRunCLI(t, "event", "list",
			"-c", calendarID,
			"-s", today(),
			"-e", nextWeek())
		assertContains(t, output, "Daily Standup")
	})

	t.Run("UpdateRecurringWithoutSeriesFlag", func(t *testing.T) {
		if recurringEventUID == "" {
			t.Skip("Recurring event not created")
		}
		output, _ := runCLI(t, "event", "update", recurringEventUID,
			"-c", calendarID,
			"-t", "Updated Standup")
		assertContains(t, output, "recurring event")
	})

	t.Run("UpdateEntireRecurringSeries", func(t *testing.T) {
		if recurringEventUID == "" {
			t.Skip("Recurring event not created")
		}
		output := mustRunCLI(t, "event", "update", recurringEventUID,
			"-c", calendarID,
			"-t", "Team Standup",
			"-S")
		assertContains(t, output, "series updated")

		getOutput := mustRunCLI(t, "event", "get", recurringEventUID, "-c", calendarID)
		assertContains(t, getOutput, "Team Standup")
	})

	t.Run("DeleteRecurringWithoutSeriesFlag", func(t *testing.T) {
		if recurringEventUID == "" {
			t.Skip("Recurring event not created")
		}
		output, _ := runCLI(t, "event", "delete", recurringEventUID, "-c", calendarID)
		assertContains(t, output, "recurring event")
	})

	t.Run("DeleteEntireRecurringSeries", func(t *testing.T) {
		if recurringEventUID == "" {
			t.Skip("Recurring event not created")
		}
		output := mustRunCLI(t, "event", "delete", recurringEventUID, "-c", calendarID, "-S")
		assertContains(t, output, "series deleted")
	})
}

func TestErrorHandling(t *testing.T) {
	if calendarID == "" {
		t.Skip("Calendar not created")
	}

	t.Run("CreateEventWithoutTitle", func(t *testing.T) {
		output, err := runCLI(t, "event", "create",
			"-s", tomorrow()+" 10:00",
			"-c", calendarID)
		if err == nil {
			t.Error("Expected error for missing title")
		}
		assertContains(t, output, "title")
	})

	t.Run("CreateEventWithoutStart", func(t *testing.T) {
		output, err := runCLI(t, "event", "create",
			"-t", "No Start",
			"-c", calendarID)
		if err == nil {
			t.Error("Expected error for missing start time")
		}
		assertContains(t, output, "start")
	})

	t.Run("CreateEventWithoutCalendar", func(t *testing.T) {
		output, err := runCLI(t, "event", "create",
			"-t", "No Calendar",
			"-s", tomorrow()+" 10:00")
		if err == nil {
			t.Error("Expected error for missing calendar")
		}
		assertContains(t, output, "calendar")
	})

	t.Run("GetEventWithoutCalendar", func(t *testing.T) {
		output, err := runCLI(t, "event", "get", "fake-uid")
		if err == nil {
			t.Error("Expected error for missing calendar")
		}
		assertContains(t, output, "calendar")
	})

	t.Run("CreateEventWithInvalidDate", func(t *testing.T) {
		output, err := runCLI(t, "event", "create",
			"-t", "Bad Date",
			"-s", "not-a-date",
			"-c", calendarID)
		if err == nil {
			t.Error("Expected error for invalid date")
		}
		assertContains(t, output, "invalid")
	})

	t.Run("DeleteNonExistentEvent", func(t *testing.T) {
		output, err := runCLI(t, "event", "delete", "non-existent-uid-12345", "-c", calendarID)
		if err == nil {
			t.Error("Expected error for non-existent event")
		}
		if !strings.Contains(output, "failed") && !strings.Contains(output, "not found") && !strings.Contains(output, "error") {
			t.Errorf("Expected error message, got: %s", output)
		}
	})
}

func TestDateRanges(t *testing.T) {
	if calendarID == "" {
		t.Skip("Calendar not created")
	}

	var dateTestUID string

	t.Run("CreateDateRangeTestEvent", func(t *testing.T) {
		output := mustRunCLI(t, "event", "create",
			"-t", "Date Range Test Event",
			"-s", tomorrow()+" 12:00",
			"-e", tomorrow()+" 13:00",
			"-c", calendarID)
		assertContains(t, output, "Event created")

		if idx := strings.Index(output, "UID: "); idx != -1 {
			rest := output[idx+5:]
			if endIdx := strings.Index(rest, ")"); endIdx != -1 {
				dateTestUID = rest[:endIdx]
			}
		}
	})

	t.Run("ListWithRelativeDate0d2d", func(t *testing.T) {
		output := mustRunCLI(t, "event", "list",
			"-c", calendarID,
			"-s", "0d",
			"-e", "2d")
		assertContains(t, output, "Date Range Test Event")
	})

	t.Run("ListWithRelativeDate1w", func(t *testing.T) {
		output := mustRunCLI(t, "event", "list",
			"-c", calendarID,
			"-s", "0d",
			"-e", "1w")
		assertContains(t, output, "Date Range Test Event")
	})

	t.Run("CleanupDateRangeTestEvent", func(t *testing.T) {
		if dateTestUID == "" {
			t.Skip("Date range test event not created")
		}
		mustRunCLI(t, "event", "delete", dateTestUID, "-c", calendarID)
	})
}

func TestCalendarCleanup(t *testing.T) {
	if calendarID == "" {
		t.Skip("Calendar not created")
	}

	t.Run("DeleteTestCalendar", func(t *testing.T) {
		output := mustRunCLI(t, "calendar", "delete", calendarID)
		assertContains(t, output, "Successfully deleted")
	})

	t.Run("VerifyCalendarDeleted", func(t *testing.T) {
		output := mustRunCLI(t, "calendar", "list")
		assertNotContains(t, output, testCalendarName)
	})
}
