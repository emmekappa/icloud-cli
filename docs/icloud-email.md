# ADR: iCloud Email Integration

**Status**: Proposed  
**Date**: 2024-12-26  
**Author**: icloud-cli team

## Context

The iCloud CLI currently supports calendar management via CalDAV. Users have requested email functionality including the ability to list, search, send, reply, and manage drafts. This ADR evaluates the feasibility and proposes an implementation approach.

## Decision

We will implement email functionality using **IMAP** for reading/managing emails and **SMTP** for sending emails, leveraging the `emersion/go-imap` and `emersion/go-smtp` libraries.

## Feasibility Assessment

### Highly Feasible

| Factor | Assessment |
|--------|------------|
| Protocol Support | iCloud fully supports IMAP and SMTP |
| Authentication | App-specific passwords (already implemented) work for email |
| Go Libraries | Mature libraries from same author as `go-webdav` |
| Credential Reuse | Existing account config can be reused |

## Technical Specification

### iCloud Mail Server Details

| Service | Server | Port | Encryption |
|---------|--------|------|------------|
| IMAP (incoming) | `imap.mail.me.com` | 993 | SSL/TLS |
| SMTP (outgoing) | `smtp.mail.me.com` | 587 | STARTTLS |

**Authentication Notes:**
- IMAP username: email prefix (e.g., `johnappleseed`) or full address
- SMTP username: full email address (e.g., `johnappleseed@icloud.com`)
- Password: app-specific password (same as CalDAV)

### Recommended Libraries

| Library | Purpose | Notes |
|---------|---------|-------|
| `github.com/emersion/go-imap/v2` | IMAP client | v2 is the modern version |
| `github.com/emersion/go-smtp` | SMTP client | For sending emails |
| `github.com/emersion/go-sasl` | SASL authentication | Required for SMTP auth |
| `github.com/emersion/go-message` | MIME/mail parsing | Email composition and parsing |

These are all from the same author (`emersion`) who wrote `go-webdav`, ensuring consistency and quality.

## Proposed CLI Commands

### Email Listing & Reading

```bash
# List emails in a mailbox (default: INBOX)
icloud email list [-a ACCOUNT] [-m MAILBOX] [-n LIMIT] [-o FORMAT]
  # MAILBOX: INBOX, Sent Messages, Drafts, Trash, Junk, Archive
  # LIMIT: number of emails to fetch (default: 20)
  # FORMAT: tsv (default) or json

# List all mailboxes/folders
icloud email mailbox list [-a ACCOUNT]

# Read a specific email
icloud email get <message-id> [-a ACCOUNT] [-m MAILBOX] [-o FORMAT]
  # FORMAT: text (default), json, or raw (full MIME)
```

### Email Search

```bash
# Search emails
icloud email search <query> [-a ACCOUNT] [-m MAILBOX] [-n LIMIT] [-o FORMAT]
  # Supports IMAP search criteria:
  # - FROM:<address>
  # - TO:<address>
  # - SUBJECT:<text>
  # - BODY:<text>
  # - SINCE:YYYY-MM-DD
  # - BEFORE:YYYY-MM-DD
  # - UNSEEN (unread only)
  # - FLAGGED (starred)

# Examples:
icloud email search "FROM:john@example.com UNSEEN"
icloud email search "SUBJECT:invoice SINCE:2024-01-01"
```

### Email Sending

```bash
# Send a new email
icloud email send -t TO -s SUBJECT [-c CC] [-b BCC] [-B BODY] [-f FILE] [-a ACCOUNT]
  # TO, CC, BCC: comma-separated addresses
  # BODY: inline body text (use -f for file input)
  # FILE: read body from file (supports .txt, .html)

# Examples:
icloud email send -t "john@example.com" -s "Hello" -B "Message body"
icloud email send -t "john@example.com" -s "Report" -f ./report.html
```

### Email Reply

```bash
# Reply to an email
icloud email reply <message-id> -m MAILBOX [-B BODY] [-f FILE] [--all] [-a ACCOUNT]
  # --all: reply to all recipients
  # Automatically sets In-Reply-To and References headers
  # Prefixes subject with "Re: " if not present

# Example:
icloud email reply abc123 -m INBOX -B "Thanks for your email!"
```

### Draft Management

```bash
# Create a draft
icloud email draft create -t TO -s SUBJECT [-c CC] [-B BODY] [-f FILE] [-a ACCOUNT]

# List drafts
icloud email draft list [-a ACCOUNT] [-o FORMAT]

# Update a draft
icloud email draft update <message-id> [-t TO] [-s SUBJECT] [-c CC] [-B BODY] [-a ACCOUNT]

# Delete a draft
icloud email draft delete <message-id> [-a ACCOUNT] [-f]

# Send a draft
icloud email draft send <message-id> [-a ACCOUNT]
```

### Email Management

```bash
# Move email to folder
icloud email move <message-id> -m SOURCE -d DESTINATION [-a ACCOUNT]

# Delete email (move to Trash)
icloud email delete <message-id> -m MAILBOX [-a ACCOUNT] [-f]

# Mark as read/unread
icloud email mark <message-id> -m MAILBOX --read|--unread [-a ACCOUNT]

# Flag/unflag (star)
icloud email flag <message-id> -m MAILBOX --set|--unset [-a ACCOUNT]
```

## Proposed Project Structure

```
├── internal/
│   ├── caldav/           # Existing CalDAV client
│   └── email/            # New email package
│       ├── client.go     # IMAP/SMTP client initialization
│       ├── imap.go       # IMAP operations (list, search, read, move, delete)
│       ├── smtp.go       # SMTP operations (send)
│       ├── message.go    # Email message struct and parsing
│       ├── draft.go      # Draft-specific operations
│       └── helpers.go    # MIME encoding, date parsing, etc.
├── cmd/icloud/
│   ├── email.go          # Email command group
│   ├── email_list.go     # List/search subcommands
│   ├── email_send.go     # Send/reply subcommands
│   └── email_draft.go    # Draft management subcommands
```

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1)
- [x] Add `go-imap/v2`, `go-smtp`, `go-sasl`, `go-message` dependencies
- [x] Create `internal/email/client.go` with IMAP/SMTP connection handling
- [x] Add email server configuration to existing account config
- [x] Implement `email mailbox list` command

### Phase 2: Reading Emails (Week 1-2)
- [x] Implement `email list` command
- [x] Implement `email get` command with text/json/raw output
- [x] Implement `email search` command with IMAP search criteria

### Phase 3: Sending Emails (Week 2)
- [x] Implement `email send` command
- [x] Implement `email reply` command with proper threading headers
- [x] Support HTML and plain text bodies
- [ ] Handle attachments (stretch goal)

### Phase 4: Draft & Management (Week 3)
- [x] Implement draft CRUD operations
- [x] Implement `email move`, `email delete`
- [x] Implement `email mark`, `email flag`

### Phase 5: Polish (Week 3-4)
- [x] Add comprehensive error handling
- [ ] Add tests with mock IMAP/SMTP servers
- [x] Documentation and examples

## Data Structures

### Email Message

```go
type Email struct {
    MessageID   string    `json:"message_id"`
    UID         uint32    `json:"uid"`
    Mailbox     string    `json:"mailbox"`
    From        []Address `json:"from"`
    To          []Address `json:"to"`
    CC          []Address `json:"cc,omitempty"`
    BCC         []Address `json:"bcc,omitempty"`
    Subject     string    `json:"subject"`
    Date        time.Time `json:"date"`
    Body        string    `json:"body,omitempty"`        // Plain text
    HTMLBody    string    `json:"html_body,omitempty"`   // HTML version
    Seen        bool      `json:"seen"`
    Flagged     bool      `json:"flagged"`
    Answered    bool      `json:"answered"`
    InReplyTo   string    `json:"in_reply_to,omitempty"`
    References  []string  `json:"references,omitempty"`
    Attachments []Attachment `json:"attachments,omitempty"`
}

type Address struct {
    Name    string `json:"name,omitempty"`
    Address string `json:"address"`
}

type Attachment struct {
    Filename    string `json:"filename"`
    ContentType string `json:"content_type"`
    Size        int64  `json:"size"`
}
```

## Known Limitations & Considerations

### Protocol Limitations
1. **No POP support**: iCloud only supports IMAP (not a problem for our use case)
2. **No push notifications**: Would need IMAP IDLE for real-time updates
3. **Rate limiting**: Apple may rate-limit aggressive polling

### Authentication Gotchas
1. **Username inconsistency**: IMAP uses email prefix, SMTP uses full address
2. **Password limit**: Maximum 25 app-specific passwords per Apple account
3. **Password revocation**: Changing Apple password revokes all app-specific passwords

### Implementation Considerations
1. **Connection pooling**: Consider keeping IMAP connections alive for performance
2. **Large mailboxes**: Implement pagination for mailboxes with many emails
3. **MIME complexity**: Email parsing can be complex; rely heavily on `go-message`
4. **Encoding**: Handle various character encodings in email subjects/bodies

## Security Considerations

1. **Credentials**: Reuse existing secure credential storage from CalDAV
2. **TLS required**: Both IMAP (port 993) and SMTP (port 587) require encryption
3. **No plaintext**: Never send credentials over unencrypted connections
4. **Sensitive data**: Email content may contain sensitive information; consider memory handling

## Alternatives Considered

### 1. Apple Mail API (Private)
- **Rejected**: No public API; would require reverse-engineering
- Risk of breaking changes and potential ToS violations

### 2. JMAP (JSON Meta Application Protocol)
- **Rejected**: iCloud doesn't support JMAP
- Would be preferred if available (more modern than IMAP)

### 3. Third-party email services
- **Rejected**: Out of scope; this tool is specifically for iCloud

## Success Metrics

1. All five core operations work reliably (list, search, send, reply, draft)
2. Performance: List 100 emails in < 2 seconds
3. Compatibility: Works with all iCloud email address formats (@icloud.com, @me.com, @mac.com)
4. Error handling: Clear error messages for authentication and connection issues

## References

- [Apple iCloud Mail Server Settings](https://support.apple.com/en-us/102525)
- [go-imap v2 Documentation](https://github.com/emersion/go-imap)
- [go-smtp Documentation](https://github.com/emersion/go-smtp)
- [go-message Documentation](https://github.com/emersion/go-message)
- [RFC 3501 - IMAP](https://datatracker.ietf.org/doc/html/rfc3501)
- [RFC 5321 - SMTP](https://datatracker.ietf.org/doc/html/rfc5321)
