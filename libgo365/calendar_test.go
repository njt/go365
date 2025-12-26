package libgo365

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCalendarView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header with Bearer token")
		}

		// Check query parameters
		startDateTime := r.URL.Query().Get("startDateTime")
		endDateTime := r.URL.Query().Get("endDateTime")

		if startDateTime == "" || endDateTime == "" {
			t.Error("Expected startDateTime and endDateTime query parameters")
		}

		response := EventList{
			Value: []*Event{
				{
					ID:       "event1",
					Subject:  "Team Meeting",
					IsAllDay: false,
					Start: &DateTimeTimeZone{
						DateTime: "2025-01-15T09:00:00",
						TimeZone: "Pacific/Auckland",
					},
					End: &DateTimeTimeZone{
						DateTime: "2025-01-15T10:00:00",
						TimeZone: "Pacific/Auckland",
					},
					Location: &Location{
						DisplayName: "Conference Room A",
					},
					Organizer: &Recipient{
						EmailAddress: &EmailAddress{
							Name:    "Jane Smith",
							Address: "jane@example.com",
						},
					},
					ResponseStatus: &ResponseStatus{
						Response: "accepted",
					},
				},
				{
					ID:       "event2",
					Subject:  "Lunch",
					IsAllDay: false,
					Start: &DateTimeTimeZone{
						DateTime: "2025-01-15T12:00:00",
						TimeZone: "Pacific/Auckland",
					},
					End: &DateTimeTimeZone{
						DateTime: "2025-01-15T13:00:00",
						TimeZone: "Pacific/Auckland",
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
	opts := &CalendarViewOptions{
		StartDateTime: "2025-01-15T00:00:00Z",
		EndDateTime:   "2025-01-16T00:00:00Z",
	}

	resp, err := client.CalendarView(ctx, opts)
	if err != nil {
		t.Fatalf("CalendarView failed: %v", err)
	}

	if len(resp.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(resp.Events))
	}

	if resp.Events[0].Subject != "Team Meeting" {
		t.Errorf("Expected subject 'Team Meeting', got '%s'", resp.Events[0].Subject)
	}

	if resp.Events[0].Location.DisplayName != "Conference Room A" {
		t.Errorf("Expected location 'Conference Room A', got '%s'", resp.Events[0].Location.DisplayName)
	}
}

func TestCalendarViewWithCalendarID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/me/calendars/cal123/calendarView"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		response := EventList{
			Value: []*Event{
				{ID: "event1", Subject: "Event in specific calendar"},
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
	opts := &CalendarViewOptions{
		StartDateTime: "2025-01-15T00:00:00Z",
		EndDateTime:   "2025-01-16T00:00:00Z",
		CalendarID:    "cal123",
	}

	resp, err := client.CalendarView(ctx, opts)
	if err != nil {
		t.Fatalf("CalendarView failed: %v", err)
	}

	if len(resp.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(resp.Events))
	}
}

func TestCalendarViewWithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := EventList{
			Value: []*Event{
				{ID: "event1", Subject: "Event 1"},
			},
			NextLink: "https://graph.microsoft.com/v1.0/me/calendarView?$skip=10",
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
	opts := &CalendarViewOptions{
		StartDateTime: "2025-01-15T00:00:00Z",
		EndDateTime:   "2025-01-16T00:00:00Z",
	}

	resp, err := client.CalendarView(ctx, opts)
	if err != nil {
		t.Fatalf("CalendarView failed: %v", err)
	}

	if !resp.HasMore {
		t.Error("Expected HasMore=true")
	}

	if resp.NextPageToken != "10" {
		t.Errorf("Expected NextPageToken='10', got '%s'", resp.NextPageToken)
	}
}

func TestCalendarViewMissingOptions(t *testing.T) {
	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     "http://localhost",
		accessToken: "test-token",
	}

	ctx := context.Background()

	// Nil options
	_, err := client.CalendarView(ctx, nil)
	if err == nil {
		t.Error("Expected error for nil options")
	}

	// Missing dates
	_, err = client.CalendarView(ctx, &CalendarViewOptions{})
	if err == nil {
		t.Error("Expected error for missing dates")
	}
}

func TestGetEvent(t *testing.T) {
	eventID := "event123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		expectedPath := "/me/events/" + eventID
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		event := Event{
			ID:      eventID,
			Subject: "Important Meeting",
			Start: &DateTimeTimeZone{
				DateTime: "2025-01-15T14:00:00",
				TimeZone: "Pacific/Auckland",
			},
			End: &DateTimeTimeZone{
				DateTime: "2025-01-15T15:00:00",
				TimeZone: "Pacific/Auckland",
			},
			Body: &ItemBody{
				ContentType: "HTML",
				Content:     "<p>Meeting agenda here</p>",
			},
			Attendees: []*Attendee{
				{
					EmailAddress: &EmailAddress{
						Name:    "John Doe",
						Address: "john@example.com",
					},
					Status: &ResponseStatus{
						Response: "accepted",
					},
					Type: "required",
				},
			},
			OnlineMeeting: &OnlineMeetingInfo{
				JoinUrl: "https://teams.microsoft.com/l/meetup-join/...",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(event)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	event, err := client.GetEvent(ctx, eventID, "")
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}

	if event.ID != eventID {
		t.Errorf("Expected event ID %s, got %s", eventID, event.ID)
	}

	if event.Subject != "Important Meeting" {
		t.Errorf("Expected subject 'Important Meeting', got '%s'", event.Subject)
	}

	if event.Body == nil || event.Body.Content != "<p>Meeting agenda here</p>" {
		t.Error("Expected body content")
	}

	if len(event.Attendees) != 1 {
		t.Errorf("Expected 1 attendee, got %d", len(event.Attendees))
	}

	if event.OnlineMeeting == nil || event.OnlineMeeting.JoinUrl == "" {
		t.Error("Expected online meeting info")
	}
}

func TestGetEventWithCalendarID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/me/calendars/cal123/events/event456"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		event := Event{ID: "event456", Subject: "Test"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(event)
	}))
	defer server.Close()

	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     server.URL,
		accessToken: "test-token",
	}

	ctx := context.Background()
	_, err := client.GetEvent(ctx, "event456", "cal123")
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}
}

func TestGetEventEmptyID(t *testing.T) {
	client := &Client{
		httpClient:  &http.Client{},
		baseURL:     "http://localhost",
		accessToken: "test-token",
	}

	ctx := context.Background()
	_, err := client.GetEvent(ctx, "", "")
	if err == nil {
		t.Error("Expected error for empty event ID")
	}
}

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

func TestListCalendars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/calendars" {
			t.Errorf("Expected path /me/calendars, got %s", r.URL.Path)
		}

		response := CalendarList{
			Value: []*Calendar{
				{
					ID:   "cal1",
					Name: "Calendar",
					Owner: &EmailAddress{
						Name:    "User",
						Address: "user@example.com",
					},
				},
				{
					ID:   "cal2",
					Name: "Work Calendar",
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
	calendars, err := client.ListCalendars(ctx)
	if err != nil {
		t.Fatalf("ListCalendars failed: %v", err)
	}

	if len(calendars) != 2 {
		t.Errorf("Expected 2 calendars, got %d", len(calendars))
	}

	if calendars[0].Name != "Calendar" {
		t.Errorf("Expected name 'Calendar', got '%s'", calendars[0].Name)
	}
}
