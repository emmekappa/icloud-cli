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
│   ├── caldav/client.go         # CalDAV client wrapper for iCloud
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
icloud account add [--email EMAIL --password PASS]  # Uses APPLE_EMAIL/APPLE_APP_SPECIFIC_PASSWORD env vars
icloud account list
icloud account set-default <account>
icloud account remove <account>

# Calendar management
icloud calendar list [-a ACCOUNT]
icloud calendar create <name> [-a ACCOUNT]
icloud calendar delete <calendar-id> [-f]
icloud calendar update <calendar-id> --name <new-name>

# Event management
icloud event list [-a ACCOUNT] [-s START_DATE] [-e END_DATE] [-c CALENDAR_ID] [-o FORMAT]
  # Dates: YYYY-MM-DD or relative (1d, 1w, -2d)
  # Output: tsv (default) or json
icloud event create -t TITLE -s START -c CALENDAR_ID [-e END] [-l LOCATION] [-d DESCRIPTION]
  # Start/End: YYYY-MM-DD HH:MM (e.g., 2024-01-15 14:30)
icloud event update <event-uid> -c CALENDAR_ID [-t TITLE] [-s START] [-e END] [-l LOCATION] [-d DESCRIPTION]
icloud event delete <event-uid> -c CALENDAR_ID [-f]
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
