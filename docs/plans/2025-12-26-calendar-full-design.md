# Calendar Full Feature Set Design

**Date:** 2025-12-26
**Status:** Approved
**Covers:** bd-21 (scheduling), bd-22 (invitations), bd-23 (cross-calendar), bd-24 (raw events), bd-25 (list calendars)

## Command Structure

```
calendar
├── list              # (existing) View events in date range
├── get               # (existing) Get single event
├── calendars         # List available calendars
├── events            # List raw events (series masters)
├── create            # Create event
├── find-time         # Find available meeting slots
├── pending           # List pending invitations
├── respond           # Accept/decline/tentative
└── free-busy         # Check someone's availability
```

All commands support `--json`, `--markdown`, and pagination flags.

## Commands

### `calendar create`

Create a new calendar event.

```bash
calendar create "Weekly sync" --start "next monday 10am" --attendees bob@company.com,alice@company.com
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--start` | Yes | - | Natural language or ISO 8601 |
| `--end` | No | start + 30min | Natural language or ISO 8601 |
| `--duration` | No | 30m | Alternative to --end (e.g., "1h", "90m") |
| `--attendees` | No | none | Comma-separated emails |
| `--location` | No | - | Location string |
| `--body` | No | - | Description/agenda |
| `--online` | No | false | Generate Teams meeting link |
| `--all-day` | No | false | All-day event |
| `--calendar-id` | No | primary | Target calendar |

- Subject is positional arg (required)
- `--duration` and `--end` are mutually exclusive
- `--online` uses Graph's `isOnlineMeeting: true`
- Returns created event (respects `--json`)

### `calendar find-time`

Find available meeting slots across attendees.

```bash
calendar find-time --attendees bob@company.com --duration 30m --start "next week" --end "next friday"
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--attendees` | Yes | - | Comma-separated emails |
| `--duration` | No | 30m | Meeting length |
| `--start` | No | tomorrow | Search window start |
| `--end` | No | start + 7 days | Search window end |
| `--max-results` | No | 5 | Max suggestions to return |

Human-readable output:
```
Found 3 available slots for 30m meeting:

1. Mon Jan 6, 10:00-10:30 AM
   Bob: free, Alice: free

2. Mon Jan 6, 2:00-2:30 PM
   Bob: free, Alice: free
```

Uses Graph's `findMeetingTimes` endpoint. Respects working hours.

### `calendar pending`

List pending invitations requiring response.

```bash
calendar pending
calendar pending --json
```

Output:
```
3 pending invitations:

1. [AAMk...xyz] "Q1 Planning" - Mon Jan 6, 2:00 PM
   From: alice@company.com

2. [AAMk...abc] "1:1 with Bob" - Tue Jan 7, 10:00 AM
   From: bob@company.com
```

Filters for `responseStatus.response == "notResponded"` or `"none"`.

### `calendar respond`

Respond to meeting invitations.

```bash
# Single event
calendar respond <event-id> accept
calendar respond <event-id> decline --message "Conflict with another meeting"
calendar respond <event-id> tentative

# Batch
calendar respond --all accept
calendar respond --ids AAMk...xyz,AAMk...abc decline
```

| Flag | Description |
|------|-------------|
| `--message` | Optional response message |
| `--all` | Apply to all pending invitations |
| `--ids` | Comma-separated event IDs |

Uses Graph's `/events/{id}/accept`, `/decline`, `/tentativelyAccept` endpoints.

### `calendar free-busy`

Check availability for users.

```bash
calendar free-busy bob@company.com --start "tomorrow" --end "next friday"
calendar free-busy bob@company.com,alice@company.com --start "next monday 9am" --end "next monday 5pm"
```

| Flag | Required | Default |
|------|----------|---------|
| `--start` | No | now |
| `--end` | No | start + 1 day |

Output:
```
bob@company.com:
  Busy: Mon Jan 6, 9:00-10:00 AM
  Busy: Mon Jan 6, 2:00-3:30 PM

alice@company.com:
  Busy: Mon Jan 6, 10:00-11:00 AM
  Tentative: Mon Jan 6, 3:00-4:00 PM
```

Uses Graph's `getSchedule` endpoint. Works for anyone in org.

### `calendar calendars`

List available calendars.

```bash
calendar calendars
calendar calendars --json
```

Output:
```
Calendars:

1. Calendar (default)
   ID: AAMkAD...primary

2. Work
   ID: AAMkAD...work
```

### `calendar events`

List raw events (series masters for recurring).

```bash
calendar events --calendar-id AAMkAD...work
calendar events --top 20 --json
```

Different from `list`:
- `list` = calendarView API (expands recurring into instances)
- `events` = events API (shows series masters)

Use case: Modifying recurring series requires series master ID.

### Cross-Calendar Access

View shared calendars via `--user` flag:

```bash
calendar list --user bob@company.com --start "today"
calendar get <event-id> --user bob@company.com
```

Uses `/users/{id}/calendarView`. If permission denied: "Calendar not shared with you. Use 'free-busy' to check availability."

## Implementation Order

**Phase 1 - Quick wins:**
1. `calendar calendars` - library exists, CLI wiring only
2. `calendar events` - GET `/me/events`

**Phase 2 - Agent scheduling:**
3. `calendar create` - POST `/me/events`
4. `calendar find-time` - POST `/me/findMeetingTimes`
5. `calendar free-busy` - POST `/me/calendar/getSchedule`

**Phase 3 - Invitation triage:**
6. `calendar pending` - GET with responseStatus filter
7. `calendar respond` - POST to accept/decline/tentativelyAccept

**Phase 4 - Cross-calendar:**
8. `--user` flag on `list` and `get`

## Library Additions

```go
// calendar.go additions

func (c *Client) CreateEvent(ctx context.Context, event *Event) (*Event, error)

func (c *Client) FindMeetingTimes(ctx context.Context, opts *FindTimeOptions) (*MeetingTimeSuggestions, error)

func (c *Client) GetSchedule(ctx context.Context, emails []string, start, end string) (*ScheduleResponse, error)

func (c *Client) RespondToEvent(ctx context.Context, eventID, response, message string) error

func (c *Client) ListEvents(ctx context.Context, opts *ListEventsOptions) (*EventListResponse, error)
```

## Graph API Endpoints

| Command | Method | Endpoint |
|---------|--------|----------|
| create | POST | `/me/events` or `/me/calendars/{id}/events` |
| find-time | POST | `/me/findMeetingTimes` |
| free-busy | POST | `/me/calendar/getSchedule` |
| respond | POST | `/me/events/{id}/accept` (or decline/tentativelyAccept) |
| calendars | GET | `/me/calendars` |
| events | GET | `/me/events` or `/me/calendars/{id}/events` |
| list --user | GET | `/users/{id}/calendarView` |
