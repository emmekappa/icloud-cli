# AGENTS.md - iCloud CLI

## Project Overview

A Go-based CLI tool for interacting with iCloud services via CalDAV. Currently supports calendar management (list, create, update, delete) with plans to extend to email functionality.

## Tech Stack

- **Language**: Go 1.24+
- **CLI Framework**: [cobra](https://github.com/spf13/cobra) for command structure
- **CalDAV Client**: [go-webdav](https://github.com/emersion/go-webdav) for iCloud CalDAV API
- **UUID Generation**: [google/uuid](https://github.com/google/uuid)

## Project Structure

```
├── main.go                      # Entry point
├── cmd/icloud/                  # CLI commands (cobra)
│   ├── root.go                  # Root command setup
│   ├── account.go               # Account management commands
│   ├── calendar.go              # Calendar management commands
│   └── event.go                 # Event management commands
├── internal/
│   ├── caldav/                  # CalDAV client wrapper for iCloud
│   │   ├── client.go            # Client struct, NewClient(), FindCalendarHomeSet()
│   │   ├── calendar.go          # Calendar struct and CRUD operations
│   │   ├── event.go             # Event struct and CRUD operations
│   │   └── helpers.go           # XML/iCal escaping, time parsing, ICS building
│   └── config/config.go         # Config management (~/.config/icloud-cli/)
```

## Key Commands

```bash
# Build
go build -o icloud .

# Run
./icloud <command>

# Test (no tests yet)
go test ./...

# Tidy dependencies
go mod tidy
```

## CLI Usage

```bash
# Account management
icloud account add [-e EMAIL] [-p PASSWORD] [-n NAME] [--alias ALIAS]
  # Can use env vars: APPLE_EMAIL, APPLE_APP_SPECIFIC_PASSWORD
icloud account list
icloud account set-default <account>
icloud account remove <account>

# Calendar management
icloud calendar list [-a ACCOUNT]
icloud calendar create <name> [-a ACCOUNT]
icloud calendar delete <calendar-id> [-a ACCOUNT] [-f]
icloud calendar update <calendar-id> -n <new-name> [-a ACCOUNT]

# Event management
icloud event list [-a ACCOUNT] [-s START] [-e END] [-c CALENDAR_ID] [-o FORMAT]
  # Dates: YYYY-MM-DD or relative (1d, 1w, -2d, 1m)
  # Output: tsv (default) or json
  # Defaults to current week if no dates specified
icloud event get <event-uid> -c CALENDAR_ID [-a ACCOUNT] [-o FORMAT]
icloud event create -t TITLE -s START -c CALENDAR_ID [-e END] [-l LOCATION] [-d DESCRIPTION] [-a ACCOUNT]
  # Start/End: YYYY-MM-DD HH:MM or YYYY-MM-DDTHH:MM
  # End defaults to 1 hour after start if not specified
icloud event update <event-uid> -c CALENDAR_ID [-t TITLE] [-s START] [-e END] [-l LOCATION] [-d DESCRIPTION] [-a ACCOUNT]
icloud event delete <event-uid> -c CALENDAR_ID [-a ACCOUNT] [-f]
```

## Configuration

- Config stored at `~/.config/icloud-cli/config.json`
- Credentials can be provided via environment variables: `APPLE_EMAIL`, `APPLE_APP_SPECIFIC_PASSWORD`
- Supports multiple accounts with alias-based identification

## Code Conventions

### Error Handling
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Validate inputs early in command handlers
- Provide user-friendly error messages with action suggestions

### Command Structure (Cobra)
- Commands defined as `*cobra.Command` package variables
- Subcommands registered in `init()` functions
- Use `RunE` for commands that return errors
- Use `PersistentFlags` for flags inherited by subcommands

### CalDAV Operations
- All CalDAV operations require context for cancellation (`signalContext()`)
- iCloud endpoint: `https://caldav.icloud.com`
- XML requests use MKCALENDAR, PROPPATCH for calendar operations
- Always escape XML content with `escapeXML()` helper

### Naming
- Use camelCase for Go identifiers
- Command names use lowercase with hyphens (e.g., `set-default`)
- Package names match directory names

## Planned Features (from REQ.md)

- [x] Event management: list events
- [x] Event management: create, update, delete events
- [ ] Email support
