# Calendar MVP Design

## Summary

Add `calendar list` and `calendar get` commands for reading calendar events. Uses Graph API calendar view (expands recurring events) with natural language date parsing.

## Commands

### calendar list

List events from calendar view (time-range based, expands recurring).

```bash
go365 calendar list [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--start <date>` | Start date/time (default: today). Accepts natural language. |
| `--end <date>` | End date/time (default: start + 1 day) |
| `--days <N>` | Shorthand: N days from start (overrides --end) |
| `--calendar-id <id>` | Query specific calendar (default: primary) |
| `--all-calendars` | Query all user's calendars |
| `--json` | Output as JSON |
| `--markdown` | Convert HTML body to Markdown (no-op for list) |
| `--top <N>` | Limit results |
| `--page-token <t>` | Pagination cursor |

**Date parsing** uses [tj/go-naturaldate](https://github.com/tj/go-naturaldate):
- `today`, `tomorrow`, `yesterday`
- `next week`, `last month`, `next Tuesday`
- `December 25th at 7:30am`
- `5 days ago`, `in 3 weeks`
- ISO 8601: `2025-01-15` or `2025-01-15T09:00:00`

**Examples:**

```bash
go365 calendar list                              # Today
go365 calendar list --days 7                     # Next 7 days
go365 calendar list --start "next monday" --end "friday"
go365 calendar list --start tomorrow --days 3
go365 calendar list --all-calendars --days 7 --json
```

### calendar get

Get specific event details.

```bash
go365 calendar get <event-id> [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--markdown` | Convert HTML body to Markdown |
| `--calendar-id <id>` | Calendar containing the event (default: primary) |

## Output Format

### calendar list (human-readable)

```
ID: AAMkAGI2...
Subject: Weekly Team Standup
Start: 2025-01-15T09:00:00+13:00
End: 2025-01-15T09:30:00+13:00
Location: Teams Meeting
Organizer: Jane Smith <jane@example.com>
Response: accepted
---
ID: AAMkAGI3...
Subject: Lunch with Client
Start: 2025-01-15T12:00:00+13:00
End: 2025-01-15T13:00:00+13:00
AllDay: false
Location: Cafe Central
Organizer: You
Response: organizer
---

Next page: --page-token eyJza2lw...
```

### calendar list --json

```json
{
  "value": [
    {
      "id": "AAMkAGI2...",
      "subject": "Weekly Team Standup",
      "start": {"dateTime": "2025-01-15T09:00:00", "timeZone": "Pacific/Auckland"},
      "end": {"dateTime": "2025-01-15T09:30:00", "timeZone": "Pacific/Auckland"},
      "isAllDay": false,
      "location": {"displayName": "Teams Meeting"},
      "organizer": {"emailAddress": {"name": "Jane Smith", "address": "jane@example.com"}},
      "responseStatus": {"response": "accepted"}
    }
  ],
  "@odata.count": 5,
  "hasMore": true,
  "nextPageToken": "..."
}
```

### calendar get (additional fields)

- `body` - Event description (with `--markdown` conversion for HTML)
- `attendees` - List with name, email, response status
- `onlineMeeting` - Teams/Zoom link details
- `recurrence` - Pattern if recurring event
- `calendarId` - When using `--all-calendars`

## Implementation

### New Dependency

```
github.com/tj/go-naturaldate
```

### Files

| File | Purpose |
|------|---------|
| `libgo365/calendar.go` | Event types, CalendarView, GetEvent methods |
| `libgo365/calendar_test.go` | Mock server tests |
| `cmd/go365/main.go` | calendar command with list/get subcommands |
| `internal/dateparse/dateparse.go` | Natural language date parsing wrapper |

### Types (libgo365/calendar.go)

```go
type Event struct {
    ID               string              `json:"id"`
    Subject          string              `json:"subject"`
    Start            *DateTimeTimeZone   `json:"start"`
    End              *DateTimeTimeZone   `json:"end"`
    IsAllDay         bool                `json:"isAllDay"`
    Location         *Location           `json:"location"`
    Organizer        *Recipient          `json:"organizer"`
    Attendees        []*Attendee         `json:"attendees,omitempty"`
    ResponseStatus   *ResponseStatus     `json:"responseStatus"`
    Body             *ItemBody           `json:"body,omitempty"`
    OnlineMeeting    *OnlineMeetingInfo  `json:"onlineMeeting,omitempty"`
    Recurrence       *PatternedRecurrence `json:"recurrence,omitempty"`
    CalendarID       string              `json:"calendarId,omitempty"`
}

type DateTimeTimeZone struct {
    DateTime string `json:"dateTime"`
    TimeZone string `json:"timeZone"`
}

type Location struct {
    DisplayName string `json:"displayName"`
}

type Attendee struct {
    EmailAddress *EmailAddress    `json:"emailAddress"`
    Status       *ResponseStatus  `json:"status"`
    Type         string           `json:"type"` // required, optional, resource
}

type ResponseStatus struct {
    Response string `json:"response"` // none, organizer, accepted, tentativelyAccepted, declined
}

type OnlineMeetingInfo struct {
    JoinUrl string `json:"joinUrl"`
}

type CalendarViewOptions struct {
    StartDateTime string
    EndDateTime   string
    CalendarID    string
    AllCalendars  bool
    Top           int
    PageToken     string
}

type CalendarViewResponse struct {
    Events        []*Event
    Count         int
    HasMore       bool
    NextPageToken string
}
```

### Graph API Endpoints

- List events: `GET /me/calendarView?startDateTime=...&endDateTime=...`
- List from specific calendar: `GET /me/calendars/{id}/calendarView?...`
- List all calendars: `GET /me/calendars` then query each
- Get event: `GET /me/events/{id}` or `GET /me/calendars/{calId}/events/{id}`

### Scopes

Requires `Calendars.Read` scope (covered by `https://graph.microsoft.com/.default`).

## Testing

- Natural language date parsing → ISO 8601 conversion
- Mock HTTP tests for CalendarView and GetEvent
- `--all-calendars` aggregation across multiple calendars
- Pagination token extraction and continuation
- `--json` and `--markdown` output formatting

## Future Work (tracked in beads)

- **bd-21**: Scheduling meetings (create events, find times, free/busy)
- **bd-22**: Managing invitations (accept/decline/tentative)
- **bd-23**: Cross-calendar visibility (other people's calendars)
- **bd-24**: List raw events (series masters)
- **bd-25**: List available calendars
