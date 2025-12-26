# AGENTS.md - iCloud CLI

## Project Overview

A Go-based CLI tool for interacting with iCloud services. Supports calendar management via CalDAV and email management via IMAP/SMTP.

## Tech Stack

- **Language**: Go 1.24+
- **CLI Framework**: [cobra](https://github.com/spf13/cobra) for command structure
- **CalDAV Client**: [go-webdav](https://github.com/emersion/go-webdav) for iCloud CalDAV API
- **IMAP Client**: [go-imap/v2](https://github.com/emersion/go-imap) for email reading
- **SMTP Client**: [go-smtp](https://github.com/emersion/go-smtp) for email sending
- **UUID Generation**: [google/uuid](https://github.com/google/uuid)

## Project Structure

```
├── main.go                      # Entry point
├── cmd/icloud/                  # CLI commands (cobra)
│   ├── root.go                  # Root command setup
│   ├── account.go               # Account management commands
│   ├── calendar.go              # Calendar management commands
│   ├── event.go                 # Event management commands
│   ├── email.go                 # Email command group and mailbox list
│   ├── email_list.go            # Email list, get, search commands
│   ├── email_send.go            # Email send and reply commands
│   ├── email_draft.go           # Draft management commands
│   └── email_manage.go          # Email move, delete, mark, flag commands
├── internal/
│   ├── caldav/                  # CalDAV client wrapper for iCloud
│   │   ├── client.go            # Client struct, NewClient(), FindCalendarHomeSet()
│   │   ├── calendar.go          # Calendar struct and CRUD operations
│   │   ├── event.go             # Event struct and CRUD operations
│   │   └── helpers.go           # XML/iCal escaping, time parsing, ICS building
│   ├── email/                   # Email client wrapper for iCloud
│   │   ├── client.go            # IMAP/SMTP connection handling
│   │   ├── message.go           # Email, Address, Attachment structs
│   │   ├── imap.go              # IMAP operations (list, fetch, search, move, delete, mark, flag)
│   │   ├── smtp.go              # SMTP operations (send)
│   │   └── draft.go             # Draft-specific operations
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

# Email management
icloud email mailbox list [-a ACCOUNT]

icloud email list [-a ACCOUNT] [-m MAILBOX] [-n LIMIT] [-o FORMAT]
  # MAILBOX: INBOX (default), Sent Messages, Drafts, Trash, Junk, Archive
  # LIMIT: number of emails (default: 20)
  # FORMAT: tsv (default) or json

icloud email get <uid> -m MAILBOX [-a ACCOUNT] [-o FORMAT]

icloud email search <query> [-a ACCOUNT] [-m MAILBOX] [-n LIMIT] [-o FORMAT]
  # Search criteria: FROM:, TO:, SUBJECT:, BODY:, SINCE:YYYY-MM-DD, BEFORE:YYYY-MM-DD, UNSEEN, FLAGGED

icloud email send -t TO -s SUBJECT [-c CC] [-b BCC] [-B BODY] [-f FILE] [-a ACCOUNT]

icloud email reply <uid> -m MAILBOX [-B BODY] [-f FILE] [--all] [-a ACCOUNT]

icloud email move <uid> -m SOURCE -d DESTINATION [-a ACCOUNT]
icloud email delete <uid> -m MAILBOX [-a ACCOUNT] [-f]
icloud email mark <uid> -m MAILBOX --read|--unread [-a ACCOUNT]
icloud email flag <uid> -m MAILBOX --set|--unset [-a ACCOUNT]

# Draft management
icloud email draft create -t TO -s SUBJECT [-c CC] [-B BODY] [-a ACCOUNT]
icloud email draft list [-a ACCOUNT] [-n LIMIT] [-o FORMAT]
icloud email draft update <uid> [-t TO] [-s SUBJECT] [-c CC] [-B BODY] [-a ACCOUNT]
icloud email draft delete <uid> [-a ACCOUNT] [-f]
icloud email draft send <uid> [-a ACCOUNT]
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

### Email Operations
- IMAP server: `imap.mail.me.com:993` (SSL/TLS)
- SMTP server: `smtp.mail.me.com:587` (STARTTLS)
- IMAP username: email prefix (e.g., `johnappleseed`)
- SMTP username: full email address (e.g., `johnappleseed@icloud.com`)
- Password: app-specific password (same as CalDAV)

### Naming
- Use camelCase for Go identifiers
- Command names use lowercase with hyphens (e.g., `set-default`)
- Package names match directory names

## Planned Features (from REQ.md)

- [x] Event management: list events
- [x] Event management: create, update, delete events
- [x] Email support: list, search, send, reply, drafts, move, delete, mark, flag
