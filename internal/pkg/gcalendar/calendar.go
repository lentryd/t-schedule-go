// Package gcalendar wraps the Google Calendar API for creating per-student
// calendars and syncing events, mirroring src/utils/calendar.ts.
package gcalendar

import (
	"context"
	"fmt"
	"time"

	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var scopes = []string{
	calendar.CalendarScope,
	calendar.CalendarEventsScope,
}

// Client wraps the Google Calendar service.
type Client struct {
	svc *calendar.Service
}

// New creates a Client authenticated with the given service account
// credentials file.
func New(ctx context.Context, credentialsPath string) (*Client, error) {
	svc, err := calendar.NewService(ctx, option.WithAuthCredentialsFile(option.ServiceAccount, credentialsPath), option.WithScopes(scopes...))
	if err != nil {
		return nil, err
	}
	return &Client{svc: svc}, nil
}

const defaultDescription = "Generated and updating by @t_schedule_bot"

// CreateCalendar creates a calendar with the given summary and opens default
// (reader) access to it, mirroring createCalendar().
func (c *Client) CreateCalendar(ctx context.Context, summary string) (string, error) {
	cal := &calendar.Calendar{
		Summary:     summary,
		TimeZone:    "Europe/Moscow",
		Description: "Сгенерировано и обновляется @t_schedule_bot",
	}
	if cal.Summary == "" {
		cal.Summary = "New Schedule"
	}

	created, err := c.svc.Calendars.Insert(cal).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	if created.Id == "" {
		return "", fmt.Errorf("gcalendar: calendar id not found")
	}

	if _, err := c.CreateDefaultRule(ctx, created.Id); err != nil {
		_ = c.DeleteCalendar(ctx, created.Id)
		return "", fmt.Errorf("gcalendar: unable to open calendar access: %w", err)
	}

	return created.Id, nil
}

// DeleteCalendar deletes a calendar.
func (c *Client) DeleteCalendar(ctx context.Context, calendarID string) error {
	return c.svc.Calendars.Delete(calendarID).Context(ctx).Do()
}

// CreateDefaultRule makes the calendar publicly readable.
func (c *Client) CreateDefaultRule(ctx context.Context, calendarID string) (string, error) {
	return c.createRule(ctx, calendarID, &calendar.AclRule{
		Role:  "reader",
		Scope: &calendar.AclRuleScope{Type: "default"},
	})
}

// CreateUserRule grants a specific user writer access to the calendar,
// mirroring createRule(calendarId, email).
func (c *Client) CreateUserRule(ctx context.Context, calendarID, email string) (string, error) {
	return c.createRule(ctx, calendarID, &calendar.AclRule{
		Role:  "writer",
		Scope: &calendar.AclRuleScope{Type: "user", Value: email},
	})
}

func (c *Client) createRule(ctx context.Context, calendarID string, rule *calendar.AclRule) (string, error) {
	created, err := c.svc.Acl.Insert(calendarID, rule).SendNotifications(false).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	if created.Id == "" {
		return "", fmt.Errorf("gcalendar: rule id not found")
	}
	return created.Id, nil
}

// ListEvents lists events within [start, end) (defaulting to this month
// through next month, like listEvent() in calendar.ts).
func (c *Client) ListEvents(ctx context.Context, calendarID string) ([]*calendar.Event, error) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 2, -1)
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, end.Location())

	resp, err := c.svc.Events.List(calendarID).
		TimeMin(start.Format(time.RFC3339)).
		TimeMax(end.Format(time.RFC3339)).
		MaxResults(400).
		SingleEvents(true).
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

// CreateEvent creates a calendar event.
func (c *Client) CreateEvent(ctx context.Context, calendarID string, event *calendar.Event) (string, error) {
	event.Id = ""
	event.Etag = ""
	applyEventDefaults(event)

	created, err := c.svc.Events.Insert(calendarID, event).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	if created.Id == "" {
		return "", fmt.Errorf("gcalendar: event id not found")
	}
	return created.Id, nil
}

// UpdateEvent updates an existing calendar event; event.Id must be set.
func (c *Client) UpdateEvent(ctx context.Context, calendarID string, event *calendar.Event) (*calendar.Event, error) {
	eventID := event.Id
	if eventID == "" {
		return nil, fmt.Errorf("gcalendar: event id not found")
	}
	event.Id = ""
	event.Etag = ""
	applyEventDefaults(event)

	return c.svc.Events.Update(calendarID, eventID, event).Context(ctx).Do()
}

// DeleteEvent deletes an event by ID.
func (c *Client) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	return c.svc.Events.Delete(calendarID, eventID).Context(ctx).Do()
}

// applyEventDefaults mirrors the {...DEFAULT_EVENT_OPTIONS, ...params} spread
// in calendar.ts: defaults only fill in fields the caller left unset, they do
// not override an explicit empty string. Since ScheduleFormat (format.go)
// always populates Summary/Location/Description/ColorId, this only matters
// for the (rarely used) Start/TimeZone-less callers.
func applyEventDefaults(event *calendar.Event) {
	if event.Start == nil {
		now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
		event.Start = &calendar.EventDateTime{DateTime: now, TimeZone: "Europe/Moscow"}
	}
	if event.End == nil {
		now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
		event.End = &calendar.EventDateTime{DateTime: now, TimeZone: "Europe/Moscow"}
	}
}
