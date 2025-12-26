package libgo365

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Event represents a calendar event from Microsoft Graph
type Event struct {
	ID              string             `json:"id,omitempty"`
	Subject         string             `json:"subject,omitempty"`
	Start           *DateTimeTimeZone  `json:"start,omitempty"`
	End             *DateTimeTimeZone  `json:"end,omitempty"`
	IsAllDay        bool               `json:"isAllDay,omitempty"`
	Location        *Location          `json:"location,omitempty"`
	Organizer       *Recipient         `json:"organizer,omitempty"`
	Attendees       []*Attendee        `json:"attendees,omitempty"`
	ResponseStatus  *ResponseStatus    `json:"responseStatus,omitempty"`
	Body            *ItemBody          `json:"body,omitempty"`
	OnlineMeeting   *OnlineMeetingInfo `json:"onlineMeeting,omitempty"`
	IsOnlineMeeting bool               `json:"isOnlineMeeting,omitempty"`
	WebLink         string             `json:"webLink,omitempty"`
	CalendarID      string             `json:"calendarId,omitempty"` // Populated when using AllCalendars
}

// DateTimeTimeZone represents a date/time with timezone from Graph API
type DateTimeTimeZone struct {
	DateTime string `json:"dateTime,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
}

// Location represents an event location
type Location struct {
	DisplayName string `json:"displayName,omitempty"`
}

// Attendee represents a meeting attendee
type Attendee struct {
	EmailAddress *EmailAddress   `json:"emailAddress,omitempty"`
	Status       *ResponseStatus `json:"status,omitempty"`
	Type         string          `json:"type,omitempty"` // required, optional, resource
}

// ResponseStatus represents a response to a meeting
type ResponseStatus struct {
	Response string `json:"response,omitempty"` // none, organizer, accepted, tentativelyAccepted, declined
	Time     string `json:"time,omitempty"`
}

// OnlineMeetingInfo represents online meeting details
type OnlineMeetingInfo struct {
	JoinUrl string `json:"joinUrl,omitempty"`
}

// Calendar represents a calendar from Microsoft Graph
type Calendar struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Owner *EmailAddress `json:"owner,omitempty"`
}

// CalendarViewOptions represents options for listing calendar events
type CalendarViewOptions struct {
	StartDateTime string // ISO 8601 format
	EndDateTime   string // ISO 8601 format
	CalendarID    string // Empty = default calendar
	AllCalendars  bool   // Query all calendars
	Top           int
	PageToken     string
	UserID        string // Email or user ID for accessing another user's calendar
}

// CalendarViewResponse represents the response from CalendarView with pagination info
type CalendarViewResponse struct {
	Events        []*Event
	Count         int
	HasMore       bool
	NextPageToken string
}

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

// EventList represents a list of events returned by Graph API
type EventList struct {
	Value    []*Event `json:"value"`
	NextLink string   `json:"@odata.nextLink,omitempty"`
}

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
	Confidence           float64                  `json:"confidence"`
	MeetingTimeSlot      *TimeSlot                `json:"meetingTimeSlot"`
	AttendeeAvailability []*AttendeeAvailability  `json:"attendeeAvailability"`
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
	Suggestions            []*MeetingTimeSuggestion `json:"meetingTimeSuggestions"`
	EmptySuggestionsReason string                   `json:"emptySuggestionsReason,omitempty"`
}

// ScheduleItem represents a busy/free time block
type ScheduleItem struct {
	Status  string            `json:"status"` // busy, tentative, oof, free
	Start   *DateTimeTimeZone `json:"start"`
	End     *DateTimeTimeZone `json:"end"`
	Subject string            `json:"subject,omitempty"`
}

// ScheduleInfo represents schedule info for one user
type ScheduleInfo struct {
	ScheduleId       string          `json:"scheduleId"`
	AvailabilityView string          `json:"availabilityView"`
	ScheduleItems    []*ScheduleItem `json:"scheduleItems"`
	Error            *ScheduleError  `json:"error,omitempty"`
}

// ScheduleError represents an error getting schedule
type ScheduleError struct {
	Message string `json:"message"`
}

// GetScheduleResponse represents the response from getSchedule
type GetScheduleResponse struct {
	Value []*ScheduleInfo `json:"value"`
}

// CalendarList represents a list of calendars returned by Graph API
type CalendarList struct {
	Value []*Calendar `json:"value"`
}

// CalendarView retrieves events from the calendar view (expands recurring events)
func (c *Client) CalendarView(ctx context.Context, opts *CalendarViewOptions) (*CalendarViewResponse, error) {
	if opts == nil {
		return nil, fmt.Errorf("options are required")
	}
	if opts.StartDateTime == "" || opts.EndDateTime == "" {
		return nil, fmt.Errorf("startDateTime and endDateTime are required")
	}

	if opts.AllCalendars {
		return c.calendarViewAllCalendars(ctx, opts)
	}

	return c.calendarViewSingle(ctx, opts)
}

// calendarViewSingle retrieves events from a single calendar
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

	params := url.Values{}
	params.Set("startDateTime", opts.StartDateTime)
	params.Set("endDateTime", opts.EndDateTime)

	if opts.Top > 0 {
		params.Set("$top", fmt.Sprintf("%d", opts.Top))
	}

	if opts.PageToken != "" {
		// PageToken contains the skip value
		params.Set("$skip", opts.PageToken)
	}

	data, err := c.Get(ctx, path+"?"+params.Encode())
	if err != nil {
		return nil, err
	}

	var eventList EventList
	if err := json.Unmarshal(data, &eventList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal events: %w", err)
	}

	nextPageToken := ExtractPageToken(eventList.NextLink)

	return &CalendarViewResponse{
		Events:        eventList.Value,
		Count:         len(eventList.Value),
		HasMore:       eventList.NextLink != "",
		NextPageToken: nextPageToken,
	}, nil
}

// calendarViewAllCalendars retrieves events from all user's calendars
func (c *Client) calendarViewAllCalendars(ctx context.Context, opts *CalendarViewOptions) (*CalendarViewResponse, error) {
	// First, get all calendars
	calendars, err := c.ListCalendars(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	var allEvents []*Event

	// Query each calendar
	for _, cal := range calendars {
		calOpts := &CalendarViewOptions{
			StartDateTime: opts.StartDateTime,
			EndDateTime:   opts.EndDateTime,
			CalendarID:    cal.ID,
			Top:           opts.Top,
		}

		resp, err := c.calendarViewSingle(ctx, calOpts)
		if err != nil {
			// Log but continue with other calendars
			continue
		}

		// Add calendar ID to each event
		for _, event := range resp.Events {
			event.CalendarID = cal.ID
		}

		allEvents = append(allEvents, resp.Events...)
	}

	// Note: Pagination is not supported for all-calendars mode
	// because we're aggregating across multiple calendars
	return &CalendarViewResponse{
		Events:  allEvents,
		Count:   len(allEvents),
		HasMore: false,
	}, nil
}

// ListCalendars retrieves all calendars for the user
func (c *Client) ListCalendars(ctx context.Context) ([]*Calendar, error) {
	data, err := c.Get(ctx, "/me/calendars")
	if err != nil {
		return nil, err
	}

	var calendarList CalendarList
	if err := json.Unmarshal(data, &calendarList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal calendars: %w", err)
	}

	return calendarList.Value, nil
}

// RespondToEvent responds to a calendar invitation (accept, decline, tentativelyAccept)
func (c *Client) RespondToEvent(ctx context.Context, eventID, response, message string) error {
	if eventID == "" {
		return fmt.Errorf("event ID is required")
	}

	validResponses := map[string]string{
		"accept":    "accept",
		"decline":   "decline",
		"tentative": "tentativelyAccept",
	}

	endpoint, ok := validResponses[response]
	if !ok {
		return fmt.Errorf("invalid response: %s (must be accept, decline, or tentative)", response)
	}

	path := fmt.Sprintf("/me/events/%s/%s", eventID, endpoint)

	var body interface{}
	if message != "" {
		body = map[string]interface{}{
			"comment":      message,
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

// GetSchedule retrieves free/busy information for users
func (c *Client) GetSchedule(ctx context.Context, emails []string, startDateTime, endDateTime string) (*GetScheduleResponse, error) {
	if len(emails) == 0 {
		return nil, fmt.Errorf("at least one email is required")
	}
	if startDateTime == "" || endDateTime == "" {
		return nil, fmt.Errorf("start and end date/time are required")
	}

	type requestBody struct {
		Schedules                []string         `json:"schedules"`
		StartTime                DateTimeTimeZone `json:"startTime"`
		EndTime                  DateTimeTimeZone `json:"endTime"`
		AvailabilityViewInterval int              `json:"availabilityViewInterval,omitempty"`
	}

	body := requestBody{
		Schedules:                emails,
		StartTime:                DateTimeTimeZone{DateTime: startDateTime, TimeZone: "UTC"},
		EndTime:                  DateTimeTimeZone{DateTime: endDateTime, TimeZone: "UTC"},
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
		Attendees           []attendeeType  `json:"attendees"`
		TimeConstraint      *timeConstraint `json:"timeConstraint,omitempty"`
		MeetingDuration     string          `json:"meetingDuration,omitempty"`
		MaxCandidates       int             `json:"maxCandidates,omitempty"`
		IsOrganizerOptional bool            `json:"isOrganizerOptional,omitempty"`
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

// GetEventOptions represents options for getting an event
type GetEventOptions struct {
	EventID    string
	CalendarID string
	UserID     string
}

// GetEvent retrieves a specific event by ID
func (c *Client) GetEvent(ctx context.Context, eventID string, calendarID string) (*Event, error) {
	return c.GetEventWithOptions(ctx, &GetEventOptions{
		EventID:    eventID,
		CalendarID: calendarID,
	})
}

// GetEventWithOptions retrieves a specific event with additional options
func (c *Client) GetEventWithOptions(ctx context.Context, opts *GetEventOptions) (*Event, error) {
	if opts == nil || opts.EventID == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	var path string
	if opts.UserID != "" {
		if opts.CalendarID != "" {
			path = fmt.Sprintf("/users/%s/calendars/%s/events/%s", opts.UserID, opts.CalendarID, opts.EventID)
		} else {
			path = fmt.Sprintf("/users/%s/events/%s", opts.UserID, opts.EventID)
		}
	} else {
		if opts.CalendarID != "" {
			path = fmt.Sprintf("/me/calendars/%s/events/%s", opts.CalendarID, opts.EventID)
		} else {
			path = fmt.Sprintf("/me/events/%s", opts.EventID)
		}
	}

	data, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return &event, nil
}
