#!/bin/bash

ICLOUD_CLI="${ICLOUD_CLI:-./icloud}"
TEST_CALENDAR_NAME="icalendar-integration-test"
CALENDAR_ID=""
EVENT_UID=""
RECURRING_EVENT_UID=""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass_count=0
fail_count=0

log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((pass_count++))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((fail_count++))
}

assert_contains() {
    local output="$1"
    local expected="$2"
    local test_name="$3"
    if echo "$output" | grep -qe "$expected"; then
        log_pass "$test_name"
        return 0
    else
        log_fail "$test_name - Expected to contain: '$expected'"
        echo "Actual output: $output"
        return 1
    fi
}

assert_not_contains() {
    local output="$1"
    local unexpected="$2"
    local test_name="$3"
    if ! echo "$output" | grep -qe "$unexpected"; then
        log_pass "$test_name"
        return 0
    else
        log_fail "$test_name - Did not expect to contain: '$unexpected'"
        return 1
    fi
}

assert_exit_code() {
    local actual="$1"
    local expected="$2"
    local test_name="$3"
    if [ "$actual" -eq "$expected" ]; then
        log_pass "$test_name"
        return 0
    else
        log_fail "$test_name - Expected exit code $expected, got $actual"
        return 1
    fi
}

cleanup() {
    log_info "Cleaning up test calendar..."
    if [ -n "$CALENDAR_ID" ]; then
        $ICLOUD_CLI calendar delete "$CALENDAR_ID" -f 2>/dev/null || true
    fi
}

trap cleanup EXIT

echo "=============================================="
echo "  iCloud Calendar & Event Acceptance Tests"
echo "=============================================="
echo ""

log_info "Testing with CLI: $ICLOUD_CLI"
echo ""

# ----------------------------------------------
# Calendar Tests
# ----------------------------------------------
echo ">>> CALENDAR TESTS <<<"
echo ""

# Test 1: List calendars (initial)
log_info "Test: List calendars (initial)"
output=$($ICLOUD_CLI calendar list 2>&1) || true
assert_contains "$output" "Calendars for" "Calendar list shows account info"

# Test 2: Create test calendar
log_info "Test: Create calendar"
output=$($ICLOUD_CLI calendar create "$TEST_CALENDAR_NAME" 2>&1)
assert_contains "$output" "Successfully created calendar" "Calendar creation succeeds"
assert_contains "$output" "$TEST_CALENDAR_NAME" "Calendar creation shows correct name"

# Extract calendar ID from the output
CALENDAR_ID=$(echo "$output" | grep "ID:" | awk '{print $2}')
log_info "Created calendar with ID: $CALENDAR_ID"

# Test 3: List calendars (verify creation)
log_info "Test: Verify calendar appears in list"
output=$($ICLOUD_CLI calendar list 2>&1)
assert_contains "$output" "$TEST_CALENDAR_NAME" "New calendar appears in list"

# Test 4: Update calendar name
log_info "Test: Update calendar name"
NEW_NAME="${TEST_CALENDAR_NAME}-renamed"
output=$($ICLOUD_CLI calendar update "$CALENDAR_ID" -n "$NEW_NAME" 2>&1)
assert_contains "$output" "Successfully updated calendar" "Calendar update succeeds"

# Test 5: Verify calendar update
log_info "Test: Verify calendar name updated"
output=$($ICLOUD_CLI calendar list 2>&1)
assert_contains "$output" "$NEW_NAME" "Updated calendar name appears in list"

# Test 6: Update calendar name back
log_info "Test: Rename calendar back to original"
output=$($ICLOUD_CLI calendar update "$CALENDAR_ID" -n "$TEST_CALENDAR_NAME" 2>&1)
assert_contains "$output" "Successfully updated calendar" "Calendar rename back succeeds"

# Test 7: Delete calendar without force (should fail)
log_info "Test: Delete calendar without force flag"
output=$($ICLOUD_CLI calendar delete "$CALENDAR_ID" 2>&1) || true
assert_contains "$output" "use --force" "Delete without force prompts for confirmation"

echo ""
# ----------------------------------------------
# Event Tests
# ----------------------------------------------
echo ">>> EVENT TESTS <<<"
echo ""

# Get dates for testing
TODAY=$(date +%Y-%m-%d)
TOMORROW=$(date -v+1d +%Y-%m-%d 2>/dev/null || date -d "+1 day" +%Y-%m-%d)
NEXT_WEEK=$(date -v+7d +%Y-%m-%d 2>/dev/null || date -d "+7 days" +%Y-%m-%d)

# Test 8: Create simple event
log_info "Test: Create simple event"
output=$($ICLOUD_CLI event create \
    -t "Test Event 1" \
    -s "$TOMORROW 10:00" \
    -e "$TOMORROW 11:00" \
    -c "$CALENDAR_ID" \
    -l "Test Location" \
    -d "Test Description" 2>&1)
assert_contains "$output" "Event created" "Event creation succeeds"
EVENT_UID=$(echo "$output" | grep -oE "UID: [^)]+\)" | sed 's/UID: //' | sed 's/)//')
log_info "Created event with UID: $EVENT_UID"

# Test 9: Create event without end time (should default to 1 hour)
log_info "Test: Create event without end time"
output=$($ICLOUD_CLI event create \
    -t "Test Event 2 - No End" \
    -s "$TOMORROW 14:00" \
    -c "$CALENDAR_ID" 2>&1)
assert_contains "$output" "Event created" "Event creation without end time succeeds"
EVENT2_UID=$(echo "$output" | grep -oE "UID: [^)]+\)" | sed 's/UID: //' | sed 's/)//')

# Test 10: List events in test calendar
log_info "Test: List events in specific calendar"
output=$($ICLOUD_CLI event list -c "$CALENDAR_ID" -s "$TODAY" -e "$NEXT_WEEK" 2>&1)
assert_contains "$output" "Test Event 1" "Event list shows created event"
assert_contains "$output" "Test Event 2 - No End" "Event list shows second event"

# Test 11: List events with JSON output
log_info "Test: List events with JSON output"
output=$($ICLOUD_CLI event list -c "$CALENDAR_ID" -s "$TODAY" -e "$NEXT_WEEK" -o json 2>&1)
assert_contains "$output" '"title": "Test Event 1"' "JSON output contains event title"
assert_contains "$output" '"location": "Test Location"' "JSON output contains location"

# Test 12: Get event details
log_info "Test: Get event details"
output=$($ICLOUD_CLI event get "$EVENT_UID" -c "$CALENDAR_ID" 2>&1)
assert_contains "$output" "Test Event 1" "Get event shows title"
assert_contains "$output" "Test Location" "Get event shows location"
assert_contains "$output" "Test Description" "Get event shows description"

# Test 13: Get event with JSON output
log_info "Test: Get event with JSON output"
output=$($ICLOUD_CLI event get "$EVENT_UID" -c "$CALENDAR_ID" -o json 2>&1)
assert_contains "$output" '"title": "Test Event 1"' "JSON get shows title"

# Test 14: Update event title
log_info "Test: Update event title"
output=$($ICLOUD_CLI event update "$EVENT_UID" -c "$CALENDAR_ID" -t "Updated Event Title" 2>&1)
assert_contains "$output" "Event updated" "Event update succeeds"

# Test 15: Verify event update
log_info "Test: Verify event update"
output=$($ICLOUD_CLI event get "$EVENT_UID" -c "$CALENDAR_ID" 2>&1)
assert_contains "$output" "Updated Event Title" "Updated title appears"

# Test 16: Update event location and description
log_info "Test: Update event location and description"
output=$($ICLOUD_CLI event update "$EVENT_UID" -c "$CALENDAR_ID" -l "New Location" -d "New Description" 2>&1)
assert_contains "$output" "Event updated" "Event location/description update succeeds"

# Test 17: Update event time
log_info "Test: Update event time"
output=$($ICLOUD_CLI event update "$EVENT_UID" -c "$CALENDAR_ID" -s "$TOMORROW 15:00" -e "$TOMORROW 16:00" 2>&1)
assert_contains "$output" "Event updated" "Event time update succeeds"

# Test 18: Delete event without force
log_info "Test: Delete event without force"
output=$($ICLOUD_CLI event delete "$EVENT_UID" -c "$CALENDAR_ID" 2>&1) || true
assert_contains "$output" "--force" "Delete without force prompts confirmation"

# Test 19: Delete event with force
log_info "Test: Delete event with force"
output=$($ICLOUD_CLI event delete "$EVENT_UID" -c "$CALENDAR_ID" -f 2>&1)
assert_contains "$output" "Event deleted" "Event deletion succeeds"

# Test 20: Verify event deleted
log_info "Test: Verify event deleted"
output=$($ICLOUD_CLI event list -c "$CALENDAR_ID" -s "$TODAY" -e "$NEXT_WEEK" 2>&1)
assert_not_contains "$output" "Updated Event Title" "Deleted event no longer appears"

# Test 21: Delete second event
log_info "Test: Delete second event"
output=$($ICLOUD_CLI event delete "$EVENT2_UID" -c "$CALENDAR_ID" -f 2>&1)
assert_contains "$output" "Event deleted" "Second event deletion succeeds"

echo ""
# ----------------------------------------------
# Recurring Event Tests
# ----------------------------------------------
echo ">>> RECURRING EVENT TESTS <<<"
echo ""

# Test 22: Create recurring event (daily for 5 occurrences)
log_info "Test: Create recurring event"
output=$($ICLOUD_CLI event create \
    -t "Daily Standup" \
    -s "$TOMORROW 09:00" \
    -e "$TOMORROW 09:30" \
    -c "$CALENDAR_ID" \
    -r "FREQ=DAILY;COUNT=5" 2>&1)
assert_contains "$output" "Event created" "Recurring event creation succeeds"
RECURRING_EVENT_UID=$(echo "$output" | grep -oE "UID: [^)]+\)" | sed 's/UID: //' | sed 's/)//')
log_info "Created recurring event with UID: $RECURRING_EVENT_UID"

# Test 23: List recurring events
log_info "Test: List recurring events"
output=$($ICLOUD_CLI event list -c "$CALENDAR_ID" -s "$TODAY" -e "$NEXT_WEEK" 2>&1)
assert_contains "$output" "Daily Standup" "Recurring event appears in list"

# Test 24: Update recurring event (series) - requires --series flag
log_info "Test: Update recurring event without --series flag"
output=$($ICLOUD_CLI event update "$RECURRING_EVENT_UID" -c "$CALENDAR_ID" -t "Updated Standup" 2>&1) || true
assert_contains "$output" "recurring event" "Update without --series prompts for choice"

# Test 25: Update entire recurring series
log_info "Test: Update entire recurring series"
output=$($ICLOUD_CLI event update "$RECURRING_EVENT_UID" -c "$CALENDAR_ID" -t "Team Standup" -S 2>&1)
assert_contains "$output" "series updated" "Series update succeeds"

# Test 26: Verify recurring event update
log_info "Test: Verify recurring event update"
output=$($ICLOUD_CLI event get "$RECURRING_EVENT_UID" -c "$CALENDAR_ID" 2>&1)
assert_contains "$output" "Team Standup" "Updated series title appears"

# Test 27: Delete recurring event without flag
log_info "Test: Delete recurring event without --series flag"
output=$($ICLOUD_CLI event delete "$RECURRING_EVENT_UID" -c "$CALENDAR_ID" -f 2>&1) || true
assert_contains "$output" "recurring event" "Delete without --series prompts for choice"

# Test 28: Delete entire recurring series
log_info "Test: Delete entire recurring series"
output=$($ICLOUD_CLI event delete "$RECURRING_EVENT_UID" -c "$CALENDAR_ID" -f -S 2>&1)
assert_contains "$output" "series deleted" "Series deletion succeeds"

echo ""
# ----------------------------------------------
# Edge Cases and Error Handling
# ----------------------------------------------
echo ">>> ERROR HANDLING TESTS <<<"
echo ""

# Test 29: Create event without required fields
log_info "Test: Create event without title"
output=$($ICLOUD_CLI event create -s "$TOMORROW 10:00" -c "$CALENDAR_ID" 2>&1) || true
assert_contains "$output" "title" "Missing title error shown"

# Test 30: Create event without start time
log_info "Test: Create event without start time"
output=$($ICLOUD_CLI event create -t "No Start" -c "$CALENDAR_ID" 2>&1) || true
assert_contains "$output" "start" "Missing start time error shown"

# Test 31: Create event without calendar
log_info "Test: Create event without calendar"
output=$($ICLOUD_CLI event create -t "No Calendar" -s "$TOMORROW 10:00" 2>&1) || true
assert_contains "$output" "calendar" "Missing calendar error shown"

# Test 32: Get event without calendar
log_info "Test: Get event without calendar"
output=$($ICLOUD_CLI event get "fake-uid" 2>&1) || true
assert_contains "$output" "calendar" "Missing calendar error shown"

# Test 33: Invalid date format
log_info "Test: Create event with invalid date format"
output=$($ICLOUD_CLI event create -t "Bad Date" -s "not-a-date" -c "$CALENDAR_ID" 2>&1) || true
assert_contains "$output" "invalid" "Invalid date format error shown"

# Test 34: Delete non-existent event
log_info "Test: Delete non-existent event"
output=$($ICLOUD_CLI event delete "non-existent-uid-12345" -c "$CALENDAR_ID" -f 2>&1) || true
assert_contains "$output" "failed\|not found\|error" "Non-existent event error shown"

echo ""
# ----------------------------------------------
# Date Range Tests
# ----------------------------------------------
echo ">>> DATE RANGE TESTS <<<"
echo ""

# Create an event for date range tests
log_info "Creating event for date range tests..."
output=$($ICLOUD_CLI event create \
    -t "Date Range Test Event" \
    -s "$TOMORROW 12:00" \
    -e "$TOMORROW 13:00" \
    -c "$CALENDAR_ID" 2>&1)
DATE_TEST_UID=$(echo "$output" | grep -oE "UID: [^)]+\)" | sed 's/UID: //' | sed 's/)//')

# Test 35: List with relative date (1d)
log_info "Test: List events with relative date 1d"
output=$($ICLOUD_CLI event list -c "$CALENDAR_ID" -s "0d" -e "2d" 2>&1)
assert_contains "$output" "Date Range Test Event" "Relative date 1d works"

# Test 36: List with relative date (1w)
log_info "Test: List events with relative date 1w"
output=$($ICLOUD_CLI event list -c "$CALENDAR_ID" -s "0d" -e "1w" 2>&1)
assert_contains "$output" "Date Range Test Event" "Relative date 1w works"

# Clean up date range test event
$ICLOUD_CLI event delete "$DATE_TEST_UID" -c "$CALENDAR_ID" -f 2>&1 >/dev/null || true

echo ""
# ----------------------------------------------
# Calendar Cleanup
# ----------------------------------------------
echo ">>> CALENDAR CLEANUP <<<"
echo ""

# Test 37: Delete test calendar
log_info "Test: Delete test calendar with force"
output=$($ICLOUD_CLI calendar delete "$CALENDAR_ID" -f 2>&1)
assert_contains "$output" "Successfully deleted" "Calendar deletion succeeds"

# Clear calendar ID to prevent double cleanup
CALENDAR_ID=""

# Test 38: Verify calendar deleted
log_info "Test: Verify calendar deleted"
output=$($ICLOUD_CLI calendar list 2>&1)
assert_not_contains "$output" "$TEST_CALENDAR_NAME" "Deleted calendar no longer appears"

echo ""
echo "=============================================="
echo "  Test Summary"
echo "=============================================="
echo -e "${GREEN}Passed: $pass_count${NC}"
echo -e "${RED}Failed: $fail_count${NC}"
echo "=============================================="

if [ $fail_count -gt 0 ]; then
    exit 1
fi

exit 0
