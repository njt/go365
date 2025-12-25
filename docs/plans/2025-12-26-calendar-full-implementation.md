# Calendar Full Feature Set Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add full calendar management: create events, find meeting times, respond to invitations, check free/busy, list calendars.

**Architecture:** Library functions in `libgo365/calendar.go`, CLI commands in `cmd/go365/main.go`. All new commands follow existing patterns with `--json` and `--markdown` flags.

**Tech Stack:** Go, Microsoft Graph API, spf13/cobra CLI, httptest for mocking.

---

## Phase 1: Quick Wins

### Task 1: Add `calendar calendars` CLI command

Library already has `ListCalendars()`. Just wire up the CLI.

**Files:**
- Modify: `cmd/go365/main.go`

**Step 1: Add calendarCalendarsCmd variable**

After `calendarGetCmd` (around line 697), add:

```go
var calendarCalendarsCmd = &cobra.Command{
	Use:   "calendars",
	Short: "List available calendars",
	Long:  `List all calendars available to the authenticated user`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		authConfig := libgo365.AuthConfig{
			TenantID: config.TenantID,
			ClientID: config.ClientID,
			Scopes:   config.Scopes,
		}

		auth, err := libgo365.NewAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %w", err)
		}

		ctx := context.Background()
		if !auth.IsAuthenticated(ctx) {
			return fmt.Errorf("not authenticated. Please run 'go365 login' first")
		}

		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		client := libgo365.NewClient(ctx, accessToken)
		jsonOutput, _ := cmd.Flags().GetBool("json")

		calendars, err := client.ListCalendars(ctx)
		if err != nil {
			return fmt.Errorf("failed to list calendars: %w", err)
		}

		if jsonOutput {
			listResp := output.FormatListResponse(calendars, len(calendars), "")
			return output.WriteJSON(os.Stdout, listResp)
		}

		if len(calendars) == 0 {
			fmt.Println("No calendars found")
			return nil
		}

		fmt.Println("Calendars:\n")
		for i, cal := range calendars {
			fmt.Printf("%d. %s\n", i+1, cal.Name)
			fmt.Printf("   ID: %s\n", cal.ID)
			if cal.Owner != nil {
				fmt.Printf("   Owner: %s\n", cal.Owner.Address)
			}
			fmt.Println()
		}

		return nil
	},
}
```

**Step 2: Register command and flags**

In `init()`, add after `calendarCmd.AddCommand(calendarGetCmd)`:

```go
calendarCalendarsCmd.Flags().Bool("json", false, "Output as JSON")
calendarCalendarsCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op)")
calendarCmd.AddCommand(calendarCalendarsCmd)
```

**Step 3: Build and verify**

Run: `go build -o go365 ./cmd/go365`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add cmd/go365/main.go
git commit -m "feat(calendar): add calendars subcommand to list available calendars"
```

---

### Task 2: Add `ListEvents` library function

**Files:**
- Modify: `libgo365/calendar.go`
- Modify: `libgo365/calendar_test.go`

**Step 1: Add types and function signature**

After `CalendarViewResponse` struct (line 79), add:

```go
// ListEventsOptions represents options for listing raw events
type ListEventsOptions struct {
	CalendarID string
	Top        int
	PageToken  string
	Filter     string // OData filter expression
}

// ListEventsResponse represents the response from ListEvents with pagination
type ListEventsResponse struct {
	Events        []*Event
	Count         int
	HasMore       bool
	NextPageToken string
}
```

**Step 2: Write failing test**

In `calendar_test.go`, add:

```go
func TestListEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		expectedPath := "/me/events"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		response := EventList{
			Value: []*Event{
				{ID: "event1", Subject: "Recurring Series Master"},
				{ID: "event2", Subject: "Single Event"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	resp, err := client.ListEvents(ctx, nil)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}

	if len(resp.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(resp.Events))
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./libgo365/... -run TestListEvents -v`
Expected: FAIL with "ListEvents not defined"

**Step 4: Implement ListEvents**

In `calendar.go`, after `GetEvent` function:

```go
// ListEvents retrieves raw events (including series masters for recurring)
func (c *Client) ListEvents(ctx context.Context, opts *ListEventsOptions) (*ListEventsResponse, error) {
	path := "/me/events"
	if opts != nil && opts.CalendarID != "" {
		path = fmt.Sprintf("/me/calendars/%s/events", opts.CalendarID)
	}

	params := url.Values{}
	if opts != nil {
		if opts.Top > 0 {
			params.Set("$top", fmt.Sprintf("%d", opts.Top))
		}
		if opts.PageToken != "" {
			params.Set("$skip", opts.PageToken)
		}
		if opts.Filter != "" {
			params.Set("$filter", opts.Filter)
		}
	}

	fullPath := path
	if len(params) > 0 {
		fullPath = path + "?" + params.Encode()
	}

	data, err := c.Get(ctx, fullPath)
	if err != nil {
		return nil, err
	}

	var eventList EventList
	if err := json.Unmarshal(data, &eventList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal events: %w", err)
	}

	nextPageToken := ExtractPageToken(eventList.NextLink)

	return &ListEventsResponse{
		Events:        eventList.Value,
		Count:         len(eventList.Value),
		HasMore:       eventList.NextLink != "",
		NextPageToken: nextPageToken,
	}, nil
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./libgo365/... -run TestListEvents -v`
Expected: PASS

**Step 6: Commit**

```bash
git add libgo365/calendar.go libgo365/calendar_test.go
git commit -m "feat(calendar): add ListEvents for raw events (series masters)"
```

---

### Task 3: Add `calendar events` CLI command

**Files:**
- Modify: `cmd/go365/main.go`

**Step 1: Add calendarEventsCmd variable**

```go
var calendarEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "List raw calendar events",
	Long:  `List raw events including series masters for recurring events. Unlike 'list', this doesn't expand recurring events into occurrences.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		authConfig := libgo365.AuthConfig{
			TenantID: config.TenantID,
			ClientID: config.ClientID,
			Scopes:   config.Scopes,
		}

		auth, err := libgo365.NewAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %w", err)
		}

		ctx := context.Background()
		if !auth.IsAuthenticated(ctx) {
			return fmt.Errorf("not authenticated. Please run 'go365 login' first")
		}

		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		client := libgo365.NewClient(ctx, accessToken)

		calendarID, _ := cmd.Flags().GetString("calendar-id")
		top, _ := cmd.Flags().GetInt("top")
		pageToken, _ := cmd.Flags().GetString("page-token")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		opts := &libgo365.ListEventsOptions{
			CalendarID: calendarID,
			Top:        top,
			PageToken:  pageToken,
		}

		resp, err := client.ListEvents(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list events: %w", err)
		}

		if jsonOutput {
			listResp := output.FormatListResponse(resp.Events, resp.Count, resp.NextPageToken)
			return output.WriteJSON(os.Stdout, listResp)
		}

		if len(resp.Events) == 0 {
			fmt.Println("No events found")
			return nil
		}

		for _, event := range resp.Events {
			fmt.Printf("ID: %s\n", event.ID)
			fmt.Printf("Subject: %s\n", event.Subject)
			if event.Start != nil {
				fmt.Printf("Start: %s\n", event.Start.DateTime)
			}
			if event.End != nil {
				fmt.Printf("End: %s\n", event.End.DateTime)
			}
			fmt.Println("---")
		}

		output.PrintNextPageHint(os.Stdout, resp.NextPageToken)
		return nil
	},
}
```

**Step 2: Register command and flags**

```go
calendarEventsCmd.Flags().String("calendar-id", "", "Query specific calendar")
calendarEventsCmd.Flags().Int("top", 0, "Limit number of results")
calendarEventsCmd.Flags().String("page-token", "", "Pagination token")
calendarEventsCmd.Flags().Bool("json", false, "Output as JSON")
calendarEventsCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op for list)")
calendarCmd.AddCommand(calendarEventsCmd)
```

**Step 3: Build and verify**

Run: `go build -o go365 ./cmd/go365`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add cmd/go365/main.go
git commit -m "feat(calendar): add events subcommand for raw event listing"
```

---

## Phase 2: Agent Scheduling

### Task 4: Add `CreateEvent` library function

**Files:**
- Modify: `libgo365/calendar.go`
- Modify: `libgo365/calendar_test.go`

**Step 1: Add Event fields for creation**

The Event struct already has the fields we need. Add `IsOnlineMeeting` field:

```go
// In Event struct, add after OnlineMeeting field:
IsOnlineMeeting bool `json:"isOnlineMeeting,omitempty"`
```

**Step 2: Write failing test**

```go
func TestCreateEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/me/events" {
			t.Errorf("Expected path /me/events, got %s", r.URL.Path)
		}

		var event Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		if event.Subject != "Test Meeting" {
			t.Errorf("Expected subject 'Test Meeting', got '%s'", event.Subject)
		}

		// Return created event with ID
		event.ID = "created-event-123"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(event)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	event := &Event{
		Subject: "Test Meeting",
		Start: &DateTimeTimeZone{
			DateTime: "2025-01-20T10:00:00",
			TimeZone: "Pacific/Auckland",
		},
		End: &DateTimeTimeZone{
			DateTime: "2025-01-20T10:30:00",
			TimeZone: "Pacific/Auckland",
		},
	}

	created, err := client.CreateEvent(ctx, event, "")
	if err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	if created.ID != "created-event-123" {
		t.Errorf("Expected ID 'created-event-123', got '%s'", created.ID)
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./libgo365/... -run TestCreateEvent -v`
Expected: FAIL

**Step 4: Implement CreateEvent**

```go
// CreateEvent creates a new calendar event
func (c *Client) CreateEvent(ctx context.Context, event *Event, calendarID string) (*Event, error) {
	if event == nil {
		return nil, fmt.Errorf("event is required")
	}
	if event.Subject == "" {
		return nil, fmt.Errorf("event subject is required")
	}

	path := "/me/events"
	if calendarID != "" {
		path = fmt.Sprintf("/me/calendars/%s/events", calendarID)
	}

	data, err := c.Post(ctx, path, event)
	if err != nil {
		return nil, err
	}

	var created Event
	if err := json.Unmarshal(data, &created); err != nil {
		return nil, fmt.Errorf("failed to unmarshal created event: %w", err)
	}

	return &created, nil
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./libgo365/... -run TestCreateEvent -v`
Expected: PASS

**Step 6: Commit**

```bash
git add libgo365/calendar.go libgo365/calendar_test.go
git commit -m "feat(calendar): add CreateEvent for creating calendar events"
```

---

### Task 5: Add `calendar create` CLI command

**Files:**
- Modify: `cmd/go365/main.go`
- Modify: `internal/dateparse/dateparse.go` (add ParseDuration)

**Step 1: Add ParseDuration helper**

In `internal/dateparse/dateparse.go`, add:

```go
// ParseDuration parses a duration string like "30m", "1h", "90m"
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
```

**Step 2: Add calendarCreateCmd**

```go
var calendarCreateCmd = &cobra.Command{
	Use:   "create <subject>",
	Short: "Create a calendar event",
	Long:  `Create a new calendar event with subject, time, and optional attendees.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		subject := args[0]

		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		authConfig := libgo365.AuthConfig{
			TenantID: config.TenantID,
			ClientID: config.ClientID,
			Scopes:   config.Scopes,
		}

		auth, err := libgo365.NewAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %w", err)
		}

		ctx := context.Background()
		if !auth.IsAuthenticated(ctx) {
			return fmt.Errorf("not authenticated. Please run 'go365 login' first")
		}

		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		client := libgo365.NewClient(ctx, accessToken)

		// Parse flags
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		durationStr, _ := cmd.Flags().GetString("duration")
		attendeesStr, _ := cmd.Flags().GetString("attendees")
		location, _ := cmd.Flags().GetString("location")
		body, _ := cmd.Flags().GetString("body")
		online, _ := cmd.Flags().GetBool("online")
		allDay, _ := cmd.Flags().GetBool("all-day")
		calendarID, _ := cmd.Flags().GetString("calendar-id")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		if startStr == "" {
			return fmt.Errorf("--start is required")
		}

		if endStr != "" && durationStr != "" {
			return fmt.Errorf("--end and --duration are mutually exclusive")
		}

		now := time.Now()
		startTime, err := dateparse.Parse(startStr, now)
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}

		var endTime time.Time
		if endStr != "" {
			endTime, err = dateparse.Parse(endStr, now)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
		} else if durationStr != "" {
			duration, err := dateparse.ParseDuration(durationStr)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			endTime = startTime.Add(duration)
		} else {
			// Default: 30 minutes
			endTime = startTime.Add(30 * time.Minute)
		}

		tz := startTime.Location().String()
		if tz == "Local" {
			tz = "Pacific/Auckland" // TODO: get from system
		}

		event := &libgo365.Event{
			Subject:  subject,
			IsAllDay: allDay,
			Start: &libgo365.DateTimeTimeZone{
				DateTime: startTime.Format("2006-01-02T15:04:05"),
				TimeZone: tz,
			},
			End: &libgo365.DateTimeTimeZone{
				DateTime: endTime.Format("2006-01-02T15:04:05"),
				TimeZone: tz,
			},
			IsOnlineMeeting: online,
		}

		if location != "" {
			event.Location = &libgo365.Location{DisplayName: location}
		}

		if body != "" {
			event.Body = &libgo365.ItemBody{
				ContentType: "Text",
				Content:     body,
			}
		}

		if attendeesStr != "" {
			emails := strings.Split(attendeesStr, ",")
			for _, email := range emails {
				email = strings.TrimSpace(email)
				if email != "" {
					event.Attendees = append(event.Attendees, &libgo365.Attendee{
						EmailAddress: &libgo365.EmailAddress{Address: email},
						Type:         "required",
					})
				}
			}
		}

		created, err := client.CreateEvent(ctx, event, calendarID)
		if err != nil {
			return fmt.Errorf("failed to create event: %w", err)
		}

		if jsonOutput {
			return output.WriteJSON(os.Stdout, created)
		}

		fmt.Printf("Created event: %s\n", created.Subject)
		fmt.Printf("ID: %s\n", created.ID)
		if created.Start != nil {
			fmt.Printf("Start: %s\n", created.Start.DateTime)
		}
		if created.End != nil {
			fmt.Printf("End: %s\n", created.End.DateTime)
		}
		if created.OnlineMeeting != nil && created.OnlineMeeting.JoinUrl != "" {
			fmt.Printf("Teams Link: %s\n", created.OnlineMeeting.JoinUrl)
		}

		return nil
	},
}
```

**Step 3: Add import for strings package** (if not present)

**Step 4: Register command and flags**

```go
calendarCreateCmd.Flags().String("start", "", "Start date/time (required, accepts natural language)")
calendarCreateCmd.Flags().String("end", "", "End date/time")
calendarCreateCmd.Flags().String("duration", "", "Duration (e.g., 30m, 1h) - alternative to --end")
calendarCreateCmd.Flags().String("attendees", "", "Comma-separated email addresses")
calendarCreateCmd.Flags().String("location", "", "Location")
calendarCreateCmd.Flags().String("body", "", "Description/agenda")
calendarCreateCmd.Flags().Bool("online", false, "Generate Teams meeting link")
calendarCreateCmd.Flags().Bool("all-day", false, "All-day event")
calendarCreateCmd.Flags().String("calendar-id", "", "Target calendar")
calendarCreateCmd.Flags().Bool("json", false, "Output as JSON")
calendarCreateCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op)")
calendarCmd.AddCommand(calendarCreateCmd)
```

**Step 5: Build and verify**

Run: `go build -o go365 ./cmd/go365`
Expected: Builds successfully

**Step 6: Commit**

```bash
git add cmd/go365/main.go internal/dateparse/dateparse.go
git commit -m "feat(calendar): add create subcommand for creating events"
```

---

### Task 6: Add `FindMeetingTimes` library function

**Files:**
- Modify: `libgo365/calendar.go`
- Modify: `libgo365/calendar_test.go`

**Step 1: Add types**

```go
// FindTimeOptions represents options for finding meeting times
type FindTimeOptions struct {
	Attendees           []string
	DurationMinutes     int
	StartDateTime       string
	EndDateTime         string
	MaxCandidates       int
	IsOrganizerOptional bool
}

// MeetingTimeSuggestion represents a suggested meeting time
type MeetingTimeSuggestion struct {
	Confidence          float64                    `json:"confidence"`
	MeetingTimeSlot     *TimeSlot                  `json:"meetingTimeSlot"`
	AttendeeAvailability []*AttendeeAvailability   `json:"attendeeAvailability"`
}

// TimeSlot represents a time slot
type TimeSlot struct {
	Start *DateTimeTimeZone `json:"start"`
	End   *DateTimeTimeZone `json:"end"`
}

// AttendeeAvailability represents an attendee's availability for a slot
type AttendeeAvailability struct {
	Attendee     *AttendeeBase `json:"attendee"`
	Availability string        `json:"availability"` // free, tentative, busy, oof, unknown
}

// AttendeeBase represents basic attendee info
type AttendeeBase struct {
	EmailAddress *EmailAddress `json:"emailAddress"`
}

// FindMeetingTimesResponse represents the response from findMeetingTimes
type FindMeetingTimesResponse struct {
	Suggestions       []*MeetingTimeSuggestion `json:"meetingTimeSuggestions"`
	EmptySuggestionsReason string               `json:"emptySuggestionsReason,omitempty"`
}
```

**Step 2: Write failing test**

```go
func TestFindMeetingTimes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/me/findMeetingTimes" {
			t.Errorf("Expected path /me/findMeetingTimes, got %s", r.URL.Path)
		}

		response := FindMeetingTimesResponse{
			Suggestions: []*MeetingTimeSuggestion{
				{
					Confidence: 100,
					MeetingTimeSlot: &TimeSlot{
						Start: &DateTimeTimeZone{DateTime: "2025-01-20T10:00:00", TimeZone: "Pacific/Auckland"},
						End:   &DateTimeTimeZone{DateTime: "2025-01-20T10:30:00", TimeZone: "Pacific/Auckland"},
					},
					AttendeeAvailability: []*AttendeeAvailability{
						{
							Attendee:     &AttendeeBase{EmailAddress: &EmailAddress{Address: "bob@example.com"}},
							Availability: "free",
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	opts := &FindTimeOptions{
		Attendees:       []string{"bob@example.com"},
		DurationMinutes: 30,
		StartDateTime:   "2025-01-20T00:00:00",
		EndDateTime:     "2025-01-27T00:00:00",
		MaxCandidates:   5,
	}

	resp, err := client.FindMeetingTimes(ctx, opts)
	if err != nil {
		t.Fatalf("FindMeetingTimes failed: %v", err)
	}

	if len(resp.Suggestions) != 1 {
		t.Errorf("Expected 1 suggestion, got %d", len(resp.Suggestions))
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./libgo365/... -run TestFindMeetingTimes -v`
Expected: FAIL

**Step 4: Implement FindMeetingTimes**

```go
// FindMeetingTimes finds available meeting times for attendees
func (c *Client) FindMeetingTimes(ctx context.Context, opts *FindTimeOptions) (*FindMeetingTimesResponse, error) {
	if opts == nil {
		return nil, fmt.Errorf("options are required")
	}
	if len(opts.Attendees) == 0 {
		return nil, fmt.Errorf("at least one attendee is required")
	}

	// Build request body
	type attendeeType struct {
		EmailAddress EmailAddress `json:"emailAddress"`
		Type         string       `json:"type"`
	}
	type timeConstraint struct {
		ActivityDomain string `json:"activityDomain"`
		TimeSlots      []struct {
			Start DateTimeTimeZone `json:"start"`
			End   DateTimeTimeZone `json:"end"`
		} `json:"timeSlots"`
	}
	type requestBody struct {
		Attendees          []attendeeType  `json:"attendees"`
		TimeConstraint     *timeConstraint `json:"timeConstraint,omitempty"`
		MeetingDuration    string          `json:"meetingDuration,omitempty"`
		MaxCandidates      int             `json:"maxCandidates,omitempty"`
		IsOrganizerOptional bool           `json:"isOrganizerOptional,omitempty"`
	}

	body := requestBody{
		MaxCandidates:       opts.MaxCandidates,
		IsOrganizerOptional: opts.IsOrganizerOptional,
	}

	for _, email := range opts.Attendees {
		body.Attendees = append(body.Attendees, attendeeType{
			EmailAddress: EmailAddress{Address: email},
			Type:         "required",
		})
	}

	if opts.DurationMinutes > 0 {
		body.MeetingDuration = fmt.Sprintf("PT%dM", opts.DurationMinutes)
	}

	if opts.StartDateTime != "" && opts.EndDateTime != "" {
		body.TimeConstraint = &timeConstraint{
			ActivityDomain: "work",
			TimeSlots: []struct {
				Start DateTimeTimeZone `json:"start"`
				End   DateTimeTimeZone `json:"end"`
			}{
				{
					Start: DateTimeTimeZone{DateTime: opts.StartDateTime, TimeZone: "UTC"},
					End:   DateTimeTimeZone{DateTime: opts.EndDateTime, TimeZone: "UTC"},
				},
			},
		}
	}

	data, err := c.Post(ctx, "/me/findMeetingTimes", body)
	if err != nil {
		return nil, err
	}

	var resp FindMeetingTimesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./libgo365/... -run TestFindMeetingTimes -v`
Expected: PASS

**Step 6: Commit**

```bash
git add libgo365/calendar.go libgo365/calendar_test.go
git commit -m "feat(calendar): add FindMeetingTimes for finding available slots"
```

---

### Task 7: Add `calendar find-time` CLI command

**Files:**
- Modify: `cmd/go365/main.go`

**Step 1: Add calendarFindTimeCmd**

```go
var calendarFindTimeCmd = &cobra.Command{
	Use:   "find-time",
	Short: "Find available meeting times",
	Long:  `Find available meeting times across attendees' calendars.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		authConfig := libgo365.AuthConfig{
			TenantID: config.TenantID,
			ClientID: config.ClientID,
			Scopes:   config.Scopes,
		}

		auth, err := libgo365.NewAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %w", err)
		}

		ctx := context.Background()
		if !auth.IsAuthenticated(ctx) {
			return fmt.Errorf("not authenticated. Please run 'go365 login' first")
		}

		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		client := libgo365.NewClient(ctx, accessToken)

		attendeesStr, _ := cmd.Flags().GetString("attendees")
		durationStr, _ := cmd.Flags().GetString("duration")
		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		maxResults, _ := cmd.Flags().GetInt("max-results")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		if attendeesStr == "" {
			return fmt.Errorf("--attendees is required")
		}

		attendees := strings.Split(attendeesStr, ",")
		for i := range attendees {
			attendees[i] = strings.TrimSpace(attendees[i])
		}

		// Parse duration (default 30m)
		duration := 30
		if durationStr != "" {
			d, err := time.ParseDuration(durationStr)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}
			duration = int(d.Minutes())
		}

		now := time.Now()
		var startTime, endTime time.Time

		if startStr == "" {
			startTime = now.Add(24 * time.Hour) // tomorrow
		} else {
			startTime, err = dateparse.Parse(startStr, now)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}
		}

		if endStr == "" {
			endTime = startTime.Add(7 * 24 * time.Hour) // +7 days
		} else {
			endTime, err = dateparse.Parse(endStr, now)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
		}

		if maxResults == 0 {
			maxResults = 5
		}

		opts := &libgo365.FindTimeOptions{
			Attendees:       attendees,
			DurationMinutes: duration,
			StartDateTime:   dateparse.FormatISO8601(startTime),
			EndDateTime:     dateparse.FormatISO8601(endTime),
			MaxCandidates:   maxResults,
		}

		resp, err := client.FindMeetingTimes(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to find meeting times: %w", err)
		}

		if jsonOutput {
			return output.WriteJSON(os.Stdout, resp)
		}

		if len(resp.Suggestions) == 0 {
			fmt.Println("No available times found")
			if resp.EmptySuggestionsReason != "" {
				fmt.Printf("Reason: %s\n", resp.EmptySuggestionsReason)
			}
			return nil
		}

		fmt.Printf("Found %d available slots for %dm meeting:\n\n", len(resp.Suggestions), duration)

		for i, suggestion := range resp.Suggestions {
			slot := suggestion.MeetingTimeSlot
			if slot == nil || slot.Start == nil {
				continue
			}
			fmt.Printf("%d. %s - %s\n", i+1, slot.Start.DateTime, slot.End.DateTime)
			for _, avail := range suggestion.AttendeeAvailability {
				if avail.Attendee != nil && avail.Attendee.EmailAddress != nil {
					fmt.Printf("   %s: %s\n", avail.Attendee.EmailAddress.Address, avail.Availability)
				}
			}
			fmt.Println()
		}

		return nil
	},
}
```

**Step 2: Register command and flags**

```go
calendarFindTimeCmd.Flags().String("attendees", "", "Comma-separated email addresses (required)")
calendarFindTimeCmd.Flags().String("duration", "30m", "Meeting duration (e.g., 30m, 1h)")
calendarFindTimeCmd.Flags().String("start", "", "Search window start (default: tomorrow)")
calendarFindTimeCmd.Flags().String("end", "", "Search window end (default: start + 7 days)")
calendarFindTimeCmd.Flags().Int("max-results", 5, "Maximum suggestions to return")
calendarFindTimeCmd.Flags().Bool("json", false, "Output as JSON")
calendarFindTimeCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op)")
calendarCmd.AddCommand(calendarFindTimeCmd)
```

**Step 3: Build and verify**

Run: `go build -o go365 ./cmd/go365`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add cmd/go365/main.go
git commit -m "feat(calendar): add find-time subcommand for finding meeting slots"
```

---

### Task 8: Add `GetSchedule` library function (free/busy)

**Files:**
- Modify: `libgo365/calendar.go`
- Modify: `libgo365/calendar_test.go`

**Step 1: Add types**

```go
// ScheduleItem represents a busy/free time block
type ScheduleItem struct {
	Status  string            `json:"status"` // busy, tentative, oof, free
	Start   *DateTimeTimeZone `json:"start"`
	End     *DateTimeTimeZone `json:"end"`
	Subject string            `json:"subject,omitempty"`
}

// ScheduleInfo represents schedule info for one user
type ScheduleInfo struct {
	ScheduleId      string          `json:"scheduleId"`
	AvailabilityView string         `json:"availabilityView"`
	ScheduleItems   []*ScheduleItem `json:"scheduleItems"`
	Error           *ScheduleError  `json:"error,omitempty"`
}

// ScheduleError represents an error getting schedule
type ScheduleError struct {
	Message string `json:"message"`
}

// GetScheduleResponse represents the response from getSchedule
type GetScheduleResponse struct {
	Value []*ScheduleInfo `json:"value"`
}
```

**Step 2: Write failing test**

```go
func TestGetSchedule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/me/calendar/getSchedule" {
			t.Errorf("Expected path /me/calendar/getSchedule, got %s", r.URL.Path)
		}

		response := GetScheduleResponse{
			Value: []*ScheduleInfo{
				{
					ScheduleId:       "bob@example.com",
					AvailabilityView: "0020000000",
					ScheduleItems: []*ScheduleItem{
						{
							Status: "busy",
							Start:  &DateTimeTimeZone{DateTime: "2025-01-20T10:00:00"},
							End:    &DateTimeTimeZone{DateTime: "2025-01-20T11:00:00"},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	resp, err := client.GetSchedule(ctx, []string{"bob@example.com"}, "2025-01-20T00:00:00", "2025-01-21T00:00:00")
	if err != nil {
		t.Fatalf("GetSchedule failed: %v", err)
	}

	if len(resp.Value) != 1 {
		t.Errorf("Expected 1 schedule, got %d", len(resp.Value))
	}

	if resp.Value[0].ScheduleId != "bob@example.com" {
		t.Errorf("Expected scheduleId 'bob@example.com', got '%s'", resp.Value[0].ScheduleId)
	}
}
```

**Step 3: Run test to verify it fails**

Run: `go test ./libgo365/... -run TestGetSchedule -v`
Expected: FAIL

**Step 4: Implement GetSchedule**

```go
// GetSchedule retrieves free/busy information for users
func (c *Client) GetSchedule(ctx context.Context, emails []string, startDateTime, endDateTime string) (*GetScheduleResponse, error) {
	if len(emails) == 0 {
		return nil, fmt.Errorf("at least one email is required")
	}
	if startDateTime == "" || endDateTime == "" {
		return nil, fmt.Errorf("start and end date/time are required")
	}

	type requestBody struct {
		Schedules        []string `json:"schedules"`
		StartTime        DateTimeTimeZone `json:"startTime"`
		EndTime          DateTimeTimeZone `json:"endTime"`
		AvailabilityViewInterval int `json:"availabilityViewInterval,omitempty"`
	}

	body := requestBody{
		Schedules: emails,
		StartTime: DateTimeTimeZone{DateTime: startDateTime, TimeZone: "UTC"},
		EndTime:   DateTimeTimeZone{DateTime: endDateTime, TimeZone: "UTC"},
		AvailabilityViewInterval: 30, // 30-minute slots
	}

	data, err := c.Post(ctx, "/me/calendar/getSchedule", body)
	if err != nil {
		return nil, err
	}

	var resp GetScheduleResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./libgo365/... -run TestGetSchedule -v`
Expected: PASS

**Step 6: Commit**

```bash
git add libgo365/calendar.go libgo365/calendar_test.go
git commit -m "feat(calendar): add GetSchedule for free/busy lookup"
```

---

### Task 9: Add `calendar free-busy` CLI command

**Files:**
- Modify: `cmd/go365/main.go`

**Step 1: Add calendarFreeBusyCmd**

```go
var calendarFreeBusyCmd = &cobra.Command{
	Use:   "free-busy <emails>",
	Short: "Check availability for users",
	Long:  `Check free/busy status for one or more users. Works for anyone in your organization.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		authConfig := libgo365.AuthConfig{
			TenantID: config.TenantID,
			ClientID: config.ClientID,
			Scopes:   config.Scopes,
		}

		auth, err := libgo365.NewAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %w", err)
		}

		ctx := context.Background()
		if !auth.IsAuthenticated(ctx) {
			return fmt.Errorf("not authenticated. Please run 'go365 login' first")
		}

		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		client := libgo365.NewClient(ctx, accessToken)

		// Parse emails from args (may be comma-separated or multiple args)
		var emails []string
		for _, arg := range args {
			parts := strings.Split(arg, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					emails = append(emails, p)
				}
			}
		}

		startStr, _ := cmd.Flags().GetString("start")
		endStr, _ := cmd.Flags().GetString("end")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		now := time.Now()
		var startTime, endTime time.Time

		if startStr == "" {
			startTime = now
		} else {
			startTime, err = dateparse.Parse(startStr, now)
			if err != nil {
				return fmt.Errorf("invalid start time: %w", err)
			}
		}

		if endStr == "" {
			endTime = startTime.Add(24 * time.Hour)
		} else {
			endTime, err = dateparse.Parse(endStr, now)
			if err != nil {
				return fmt.Errorf("invalid end time: %w", err)
			}
		}

		resp, err := client.GetSchedule(ctx, emails, dateparse.FormatISO8601(startTime), dateparse.FormatISO8601(endTime))
		if err != nil {
			return fmt.Errorf("failed to get schedule: %w", err)
		}

		if jsonOutput {
			return output.WriteJSON(os.Stdout, resp)
		}

		for _, schedule := range resp.Value {
			fmt.Printf("%s:\n", schedule.ScheduleId)
			if schedule.Error != nil {
				fmt.Printf("  Error: %s\n", schedule.Error.Message)
				continue
			}
			if len(schedule.ScheduleItems) == 0 {
				fmt.Println("  Free")
				continue
			}
			for _, item := range schedule.ScheduleItems {
				startDT := ""
				endDT := ""
				if item.Start != nil {
					startDT = item.Start.DateTime
				}
				if item.End != nil {
					endDT = item.End.DateTime
				}
				fmt.Printf("  %s: %s - %s\n", strings.Title(item.Status), startDT, endDT)
			}
			fmt.Println()
		}

		return nil
	},
}
```

**Step 2: Register command and flags**

```go
calendarFreeBusyCmd.Flags().String("start", "", "Start date/time (default: now)")
calendarFreeBusyCmd.Flags().String("end", "", "End date/time (default: start + 1 day)")
calendarFreeBusyCmd.Flags().Bool("json", false, "Output as JSON")
calendarFreeBusyCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op)")
calendarCmd.AddCommand(calendarFreeBusyCmd)
```

**Step 3: Build and verify**

Run: `go build -o go365 ./cmd/go365`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add cmd/go365/main.go
git commit -m "feat(calendar): add free-busy subcommand for checking availability"
```

---

## Phase 3: Invitation Triage

### Task 10: Add `calendar pending` CLI command

**Files:**
- Modify: `cmd/go365/main.go`

**Step 1: Add calendarPendingCmd**

This uses `ListEvents` with a filter for pending responses.

```go
var calendarPendingCmd = &cobra.Command{
	Use:   "pending",
	Short: "List pending invitations",
	Long:  `List calendar invitations awaiting your response.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		authConfig := libgo365.AuthConfig{
			TenantID: config.TenantID,
			ClientID: config.ClientID,
			Scopes:   config.Scopes,
		}

		auth, err := libgo365.NewAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %w", err)
		}

		ctx := context.Background()
		if !auth.IsAuthenticated(ctx) {
			return fmt.Errorf("not authenticated. Please run 'go365 login' first")
		}

		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		client := libgo365.NewClient(ctx, accessToken)
		jsonOutput, _ := cmd.Flags().GetBool("json")

		// Filter for events where responseStatus is notResponded or none
		opts := &libgo365.ListEventsOptions{
			Filter: "responseStatus/response eq 'notResponded' or responseStatus/response eq 'none'",
		}

		resp, err := client.ListEvents(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list events: %w", err)
		}

		if jsonOutput {
			listResp := output.FormatListResponse(resp.Events, resp.Count, resp.NextPageToken)
			return output.WriteJSON(os.Stdout, listResp)
		}

		if len(resp.Events) == 0 {
			fmt.Println("No pending invitations")
			return nil
		}

		fmt.Printf("%d pending invitation(s):\n\n", len(resp.Events))

		for i, event := range resp.Events {
			fmt.Printf("%d. [%s] \"%s\"\n", i+1, event.ID[:12]+"...", event.Subject)
			if event.Start != nil {
				fmt.Printf("   When: %s\n", event.Start.DateTime)
			}
			if event.Organizer != nil && event.Organizer.EmailAddress != nil {
				fmt.Printf("   From: %s\n", event.Organizer.EmailAddress.Address)
			}
			fmt.Println()
		}

		return nil
	},
}
```

**Step 2: Register command and flags**

```go
calendarPendingCmd.Flags().Bool("json", false, "Output as JSON")
calendarPendingCmd.Flags().Bool("markdown", false, "Convert HTML to Markdown (no-op)")
calendarCmd.AddCommand(calendarPendingCmd)
```

**Step 3: Build and verify**

Run: `go build -o go365 ./cmd/go365`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add cmd/go365/main.go
git commit -m "feat(calendar): add pending subcommand for listing invitations"
```

---

### Task 11: Add `RespondToEvent` library function

**Files:**
- Modify: `libgo365/calendar.go`
- Modify: `libgo365/calendar_test.go`

**Step 1: Write failing test**

```go
func TestRespondToEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/me/events/event123/accept" {
			t.Errorf("Expected path /me/events/event123/accept, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	err := client.RespondToEvent(ctx, "event123", "accept", "")
	if err != nil {
		t.Fatalf("RespondToEvent failed: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./libgo365/... -run TestRespondToEvent -v`
Expected: FAIL

**Step 3: Implement RespondToEvent**

```go
// RespondToEvent responds to a calendar invitation (accept, decline, tentativelyAccept)
func (c *Client) RespondToEvent(ctx context.Context, eventID, response, message string) error {
	if eventID == "" {
		return fmt.Errorf("event ID is required")
	}

	validResponses := map[string]string{
		"accept":     "accept",
		"decline":    "decline",
		"tentative":  "tentativelyAccept",
	}

	endpoint, ok := validResponses[response]
	if !ok {
		return fmt.Errorf("invalid response: %s (must be accept, decline, or tentative)", response)
	}

	path := fmt.Sprintf("/me/events/%s/%s", eventID, endpoint)

	var body interface{}
	if message != "" {
		body = map[string]interface{}{
			"comment":    message,
			"sendResponse": true,
		}
	} else {
		body = map[string]interface{}{
			"sendResponse": true,
		}
	}

	_, err := c.Post(ctx, path, body)
	return err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./libgo365/... -run TestRespondToEvent -v`
Expected: PASS

**Step 5: Commit**

```bash
git add libgo365/calendar.go libgo365/calendar_test.go
git commit -m "feat(calendar): add RespondToEvent for accepting/declining invites"
```

---

### Task 12: Add `calendar respond` CLI command

**Files:**
- Modify: `cmd/go365/main.go`

**Step 1: Add calendarRespondCmd**

```go
var calendarRespondCmd = &cobra.Command{
	Use:   "respond <event-id> <accept|decline|tentative>",
	Short: "Respond to a calendar invitation",
	Long:  `Accept, decline, or tentatively accept a calendar invitation.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := configMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		authConfig := libgo365.AuthConfig{
			TenantID: config.TenantID,
			ClientID: config.ClientID,
			Scopes:   config.Scopes,
		}

		auth, err := libgo365.NewAuthenticator(authConfig)
		if err != nil {
			return fmt.Errorf("failed to create authenticator: %w", err)
		}

		ctx := context.Background()
		if !auth.IsAuthenticated(ctx) {
			return fmt.Errorf("not authenticated. Please run 'go365 login' first")
		}

		accessToken, err := auth.GetAccessToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get access token: %w", err)
		}

		client := libgo365.NewClient(ctx, accessToken)

		respondAll, _ := cmd.Flags().GetBool("all")
		idsStr, _ := cmd.Flags().GetString("ids")
		message, _ := cmd.Flags().GetString("message")

		var eventIDs []string
		var response string

		if respondAll {
			if len(args) < 1 {
				return fmt.Errorf("response type required (accept, decline, or tentative)")
			}
			response = args[0]

			// Get all pending events
			opts := &libgo365.ListEventsOptions{
				Filter: "responseStatus/response eq 'notResponded' or responseStatus/response eq 'none'",
			}
			resp, err := client.ListEvents(ctx, opts)
			if err != nil {
				return fmt.Errorf("failed to list pending events: %w", err)
			}
			for _, e := range resp.Events {
				eventIDs = append(eventIDs, e.ID)
			}
		} else if idsStr != "" {
			if len(args) < 1 {
				return fmt.Errorf("response type required (accept, decline, or tentative)")
			}
			response = args[0]
			parts := strings.Split(idsStr, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					eventIDs = append(eventIDs, p)
				}
			}
		} else {
			if len(args) < 2 {
				return fmt.Errorf("usage: calendar respond <event-id> <accept|decline|tentative>")
			}
			eventIDs = []string{args[0]}
			response = args[1]
		}

		if len(eventIDs) == 0 {
			fmt.Println("No events to respond to")
			return nil
		}

		for _, eventID := range eventIDs {
			err := client.RespondToEvent(ctx, eventID, response, message)
			if err != nil {
				fmt.Printf("Failed to respond to %s: %v\n", eventID, err)
				continue
			}
			fmt.Printf("Responded '%s' to event %s\n", response, eventID)
		}

		return nil
	},
}
```

**Step 2: Register command and flags**

```go
calendarRespondCmd.Flags().String("message", "", "Optional response message")
calendarRespondCmd.Flags().Bool("all", false, "Respond to all pending invitations")
calendarRespondCmd.Flags().String("ids", "", "Comma-separated event IDs to respond to")
calendarCmd.AddCommand(calendarRespondCmd)
```

**Step 3: Build and verify**

Run: `go build -o go365 ./cmd/go365`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add cmd/go365/main.go
git commit -m "feat(calendar): add respond subcommand for accepting/declining invites"
```

---

## Phase 4: Cross-Calendar Access

### Task 13: Add `--user` flag to `calendar list` and `calendar get`

**Files:**
- Modify: `cmd/go365/main.go`
- Modify: `libgo365/calendar.go`

**Step 1: Add UserID to CalendarViewOptions**

In `libgo365/calendar.go`, modify `CalendarViewOptions`:

```go
type CalendarViewOptions struct {
	StartDateTime string
	EndDateTime   string
	CalendarID    string
	AllCalendars  bool
	Top           int
	PageToken     string
	UserID        string // Email or user ID for accessing another user's calendar
}
```

**Step 2: Modify calendarViewSingle to support UserID**

Update the path construction:

```go
func (c *Client) calendarViewSingle(ctx context.Context, opts *CalendarViewOptions) (*CalendarViewResponse, error) {
	var path string
	if opts.UserID != "" {
		if opts.CalendarID != "" {
			path = fmt.Sprintf("/users/%s/calendars/%s/calendarView", opts.UserID, opts.CalendarID)
		} else {
			path = fmt.Sprintf("/users/%s/calendarView", opts.UserID)
		}
	} else {
		if opts.CalendarID != "" {
			path = fmt.Sprintf("/me/calendars/%s/calendarView", opts.CalendarID)
		} else {
			path = "/me/calendarView"
		}
	}
	// ... rest unchanged
}
```

**Step 3: Add --user flag to calendarListCmd**

In `init()`:

```go
calendarListCmd.Flags().String("user", "", "View another user's calendar (email or ID)")
```

**Step 4: Use --user flag in calendarListCmd RunE**

Add after other flag parsing:

```go
userID, _ := cmd.Flags().GetString("user")
// ...
opts := &libgo365.CalendarViewOptions{
	// ... existing fields
	UserID: userID,
}
```

**Step 5: Similarly update GetEvent and calendarGetCmd**

Add UserID to function signature or options struct, update path construction.

**Step 6: Build and verify**

Run: `go build -o go365 ./cmd/go365`
Expected: Builds successfully

**Step 7: Run all tests**

Run: `go test ./...`
Expected: All tests pass

**Step 8: Commit**

```bash
git add libgo365/calendar.go cmd/go365/main.go
git commit -m "feat(calendar): add --user flag for cross-calendar access"
```

---

## Final Verification

### Task 14: Run full test suite and build

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 2: Build final binary**

Run: `go build -o go365 ./cmd/go365`
Expected: Builds successfully

**Step 3: Verify help output**

Run: `./go365 calendar --help`
Expected: Shows all new subcommands (calendars, events, create, find-time, free-busy, pending, respond)

**Step 4: Commit any final fixes**

If needed.

---

## Summary

| Task | Component | Estimated Complexity |
|------|-----------|---------------------|
| 1 | calendar calendars CLI | Low |
| 2 | ListEvents library | Low |
| 3 | calendar events CLI | Low |
| 4 | CreateEvent library | Medium |
| 5 | calendar create CLI | Medium |
| 6 | FindMeetingTimes library | Medium |
| 7 | calendar find-time CLI | Medium |
| 8 | GetSchedule library | Medium |
| 9 | calendar free-busy CLI | Low |
| 10 | calendar pending CLI | Low |
| 11 | RespondToEvent library | Low |
| 12 | calendar respond CLI | Medium |
| 13 | --user cross-calendar flag | Medium |
| 14 | Final verification | Low |

Total: 14 tasks across 4 phases.
