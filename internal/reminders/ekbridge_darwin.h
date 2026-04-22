#ifndef ICLOUD_CLI_EKBRIDGE_H
#define ICLOUD_CLI_EKBRIDGE_H

// Each function returns a heap-allocated C string (JSON on success, NULL on
// error). On error, *errOut is set to a heap-allocated error string. The
// caller MUST free() both the return value and *errOut when they are
// non-NULL.

char* EKRequestAccess(char** errOut);
char* EKListLists(char** errOut);
char* EKListReminders(const char* listID, int includeCompleted, char** errOut);
char* EKGetReminder(const char* uid, char** errOut);

// Create a new reminder. jsonInput must be an object with at least "title".
// Optional: "list_id", "notes", "url", "priority", "due" (object with "year",
// "month", "day" and optional "hour", "minute"). Returns the full created
// reminder as JSON.
char* EKCreateReminder(const char* jsonInput, char** errOut);

// Update a reminder by UID. jsonInput is an object whose keys indicate which
// fields to change. Presence-only semantics: a key that is not present is
// left untouched. Recognized keys: "title", "notes", "url", "priority",
// "list_id", "due" (same shape as create), "clear_due" (bool).
char* EKUpdateReminder(const char* uid, const char* jsonInput, char** errOut);

// Mark a reminder complete (1) or uncomplete (0). Apple updates
// completionDate automatically.
char* EKSetCompleted(const char* uid, int completed, char** errOut);

#endif
