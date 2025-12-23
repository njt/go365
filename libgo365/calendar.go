package libgo365

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Event represents a calendar event from Microsoft Graph
type Event struct {
	ID             string             `json:"id,omitempty"`
	Subject        string             `json:"subject,omitempty"`
	Start          *DateTimeTimeZone  `json:"start,omitempty"`
	End            *DateTimeTimeZone  `json:"end,omitempty"`
	IsAllDay       bool               `json:"isAllDay,omitempty"`
	Location       *Location          `json:"location,omitempty"`
	Organizer      *Recipient         `json:"organizer,omitempty"`
	Attendees      []*Attendee        `json:"attendees,omitempty"`
	ResponseStatus *ResponseStatus    `json:"responseStatus,omitempty"`
	Body           *ItemBody          `json:"body,omitempty"`
	OnlineMeeting  *OnlineMeetingInfo `json:"onlineMeeting,omitempty"`
	WebLink        string             `json:"webLink,omitempty"`
	CalendarID     string             `json:"calendarId,omitempty"` // Populated when using AllCalendars
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
}

// CalendarViewResponse represents the response from CalendarView with pagination info
type CalendarViewResponse struct {
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
	path := "/me/calendarView"
	if opts.CalendarID != "" {
		path = fmt.Sprintf("/me/calendars/%s/calendarView", opts.CalendarID)
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

// GetEvent retrieves a specific event by ID
func (c *Client) GetEvent(ctx context.Context, eventID string, calendarID string) (*Event, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID is required")
	}

	path := fmt.Sprintf("/me/events/%s", eventID)
	if calendarID != "" {
		path = fmt.Sprintf("/me/calendars/%s/events/%s", calendarID, eventID)
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
