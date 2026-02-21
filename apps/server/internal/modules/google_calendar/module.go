package google_calendar

import (
	"context"
	"fmt"
	"log"
	"time"

	"mcpist/server/internal/broker"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/pkg/googlecalendarapi"
	gen "mcpist/server/pkg/googlecalendarapi/gen"
)

const (
	googleCalendarVersion = "v3"
)

var toJSON = modules.ToJSON

// GoogleCalendarModule implements the Module interface for Google Calendar API
type GoogleCalendarModule struct{}

func New() *GoogleCalendarModule { return &GoogleCalendarModule{} }

var moduleDescriptions = modules.LocalizedText{
	"en-US": "Google Calendar API - List, create, update, and delete events",
	"ja-JP": "Google Calendar API - イベントの一覧表示、作成、更新、削除",
}

func (m *GoogleCalendarModule) Name() string                        { return "google_calendar" }
func (m *GoogleCalendarModule) Descriptions() modules.LocalizedText { return moduleDescriptions }
func (m *GoogleCalendarModule) Description() string {
	return moduleDescriptions["en-US"]
}
func (m *GoogleCalendarModule) APIVersion() string                                        { return googleCalendarVersion }
func (m *GoogleCalendarModule) Tools() []modules.Tool                                     { return toolDefinitions }
func (m *GoogleCalendarModule) Resources() []modules.Resource                             { return nil }
func (m *GoogleCalendarModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

func (m *GoogleCalendarModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// ToCompact converts JSON result to compact format.
func (m *GoogleCalendarModule) ToCompact(toolName string, jsonResult string) string {
	return formatCompact(toolName, jsonResult)
}

// =============================================================================
// Token and Client
// =============================================================================

func getCredentials(ctx context.Context) *broker.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_calendar] No auth context")
		return nil
	}
	credentials, err := broker.GetTokenBroker().GetModuleToken(ctx, authCtx.UserID, "google_calendar")
	if err != nil {
		log.Printf("[google_calendar] GetModuleToken error: %v", err)
		return nil
	}
	return credentials
}

func newOgenClient(ctx context.Context) (*gen.Client, error) {
	creds := getCredentials(ctx)
	if creds == nil {
		return nil, fmt.Errorf("no credentials available")
	}
	return googlecalendarapi.NewClient(creds.AccessToken)
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	{
		ID:   "google_calendar:list_calendars",
		Name: "list_calendars",
		Descriptions: modules.LocalizedText{
			"en-US": "List all calendars accessible to the user.",
			"ja-JP": "ユーザーがアクセス可能なすべてのカレンダーを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		ID:   "google_calendar:get_calendar",
		Name: "get_calendar",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific calendar.",
			"ja-JP": "特定のカレンダーの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"calendar_id": {Type: "string", Description: "Calendar ID. Use 'primary' for the user's primary calendar."},
			},
			Required: []string{"calendar_id"},
		},
	},
	{
		ID:   "google_calendar:list_events",
		Name: "list_events",
		Descriptions: modules.LocalizedText{
			"en-US": "List events from a calendar within a time range.",
			"ja-JP": "時間範囲内のカレンダーからイベントを一覧表示します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"calendar_id":   {Type: "string", Description: "Calendar ID. Use 'primary' for the user's primary calendar."},
				"time_min":      {Type: "string", Description: "Start time (RFC3339 format, e.g., '2024-01-01T00:00:00Z'). Defaults to now."},
				"time_max":      {Type: "string", Description: "End time (RFC3339 format). Defaults to 7 days from now."},
				"max_results":   {Type: "number", Description: "Maximum number of events to return. Default: 50"},
				"single_events": {Type: "boolean", Description: "Expand recurring events into instances. Default: true"},
				"order_by":      {Type: "string", Description: "Order by 'startTime' or 'updated'. Default: startTime"},
				"q":             {Type: "string", Description: "Free text search query"},
			},
			Required: []string{"calendar_id"},
		},
	},
	{
		ID:   "google_calendar:get_event",
		Name: "get_event",
		Descriptions: modules.LocalizedText{
			"en-US": "Get details of a specific event.",
			"ja-JP": "特定のイベントの詳細を取得します。",
		},
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"calendar_id": {Type: "string", Description: "Calendar ID"},
				"event_id":    {Type: "string", Description: "Event ID"},
			},
			Required: []string{"calendar_id", "event_id"},
		},
	},
	{
		ID:   "google_calendar:create_event",
		Name: "create_event",
		Descriptions: modules.LocalizedText{
			"en-US": "Create a new event in a calendar.",
			"ja-JP": "カレンダーに新しいイベントを作成します。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"calendar_id": {Type: "string", Description: "Calendar ID. Use 'primary' for the user's primary calendar."},
				"summary":     {Type: "string", Description: "Event title"},
				"description": {Type: "string", Description: "Event description"},
				"location":    {Type: "string", Description: "Event location"},
				"start_time":  {Type: "string", Description: "Start time (RFC3339 format, e.g., '2024-01-15T09:00:00+09:00')"},
				"end_time":    {Type: "string", Description: "End time (RFC3339 format)"},
				"all_day":     {Type: "boolean", Description: "If true, create an all-day event (use date format 'YYYY-MM-DD' for start/end)"},
				"attendees":   {Type: "array", Description: "List of attendee email addresses"},
				"timezone":    {Type: "string", Description: "Timezone (e.g., 'Asia/Tokyo'). Default: UTC"},
			},
			Required: []string{"calendar_id", "summary", "start_time", "end_time"},
		},
	},
	{
		ID:   "google_calendar:update_event",
		Name: "update_event",
		Descriptions: modules.LocalizedText{
			"en-US": "Update an existing event.",
			"ja-JP": "既存のイベントを更新します。",
		},
		Annotations: modules.AnnotateUpdate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"calendar_id": {Type: "string", Description: "Calendar ID"},
				"event_id":    {Type: "string", Description: "Event ID"},
				"summary":     {Type: "string", Description: "New event title"},
				"description": {Type: "string", Description: "New event description"},
				"location":    {Type: "string", Description: "New event location"},
				"start_time":  {Type: "string", Description: "New start time (RFC3339 format)"},
				"end_time":    {Type: "string", Description: "New end time (RFC3339 format)"},
				"all_day":     {Type: "boolean", Description: "If true, update to an all-day event"},
			},
			Required: []string{"calendar_id", "event_id"},
		},
	},
	{
		ID:   "google_calendar:delete_event",
		Name: "delete_event",
		Descriptions: modules.LocalizedText{
			"en-US": "Delete an event from a calendar.",
			"ja-JP": "カレンダーからイベントを削除します。",
		},
		Annotations: modules.AnnotateDelete,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"calendar_id": {Type: "string", Description: "Calendar ID"},
				"event_id":    {Type: "string", Description: "Event ID"},
			},
			Required: []string{"calendar_id", "event_id"},
		},
	},
	{
		ID:   "google_calendar:quick_add",
		Name: "quick_add",
		Descriptions: modules.LocalizedText{
			"en-US": "Create an event based on a simple text string (e.g., 'Meeting with John tomorrow at 3pm').",
			"ja-JP": "シンプルなテキスト文字列に基づいてイベントを作成します（例：'明日午後3時にJohnとミーティング'）。",
		},
		Annotations: modules.AnnotateCreate,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"calendar_id": {Type: "string", Description: "Calendar ID. Use 'primary' for the user's primary calendar."},
				"text":        {Type: "string", Description: "Text describing the event to create"},
			},
			Required: []string{"calendar_id", "text"},
		},
	},
}

// =============================================================================
// Tool Handlers
// =============================================================================

type toolHandler func(ctx context.Context, params map[string]any) (string, error)

var toolHandlers = map[string]toolHandler{
	"list_calendars": listCalendars,
	"get_calendar":   getCalendar,
	"list_events":    listEvents,
	"get_event":      getEvent,
	"create_event":   createEvent,
	"update_event":   updateEvent,
	"delete_event":   deleteEvent,
	"quick_add":      quickAdd,
}

// =============================================================================
// Calendars
// =============================================================================

func listCalendars(ctx context.Context, params map[string]any) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListCalendars(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getCalendar(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetCalendar(ctx, gen.GetCalendarParams{CalendarId: calendarID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

// =============================================================================
// Events
// =============================================================================

func listEvents(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)

	now := time.Now().UTC()
	timeMin := now.Format(time.RFC3339)
	timeMax := now.AddDate(0, 0, 7).Format(time.RFC3339)

	if tm, ok := params["time_min"].(string); ok && tm != "" {
		timeMin = tm
	}
	if tm, ok := params["time_max"].(string); ok && tm != "" {
		timeMax = tm
	}

	maxResults := 50
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	}

	singleEvents := true
	if se, ok := params["single_events"].(bool); ok {
		singleEvents = se
	}

	p := gen.ListEventsParams{
		CalendarId:   calendarID,
		TimeMin:      gen.NewOptString(timeMin),
		TimeMax:      gen.NewOptString(timeMax),
		MaxResults:   gen.NewOptInt(maxResults),
		SingleEvents: gen.NewOptBool(singleEvents),
	}

	if singleEvents {
		orderBy := "startTime"
		if ob, ok := params["order_by"].(string); ok && ob != "" {
			orderBy = ob
		}
		p.OrderBy = gen.NewOptString(orderBy)
	}

	if q, ok := params["q"].(string); ok && q != "" {
		p.Q = gen.NewOptString(q)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.ListEvents(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func getEvent(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	eventID, _ := params["event_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetEvent(ctx, gen.GetEventParams{CalendarId: calendarID, EventId: eventID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func createEvent(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	summary, _ := params["summary"].(string)
	startTime, _ := params["start_time"].(string)
	endTime, _ := params["end_time"].(string)

	req := &gen.EventRequest{
		Summary: gen.NewOptNilString(summary),
	}

	if desc, ok := params["description"].(string); ok && desc != "" {
		req.Description = gen.NewOptNilString(desc)
	}
	if loc, ok := params["location"].(string); ok && loc != "" {
		req.Location = gen.NewOptNilString(loc)
	}

	timezone := "UTC"
	if tz, ok := params["timezone"].(string); ok && tz != "" {
		timezone = tz
	}

	allDay, _ := params["all_day"].(bool)
	if allDay {
		req.Start = gen.NewOptEventDateTime(gen.EventDateTime{Date: gen.NewOptNilString(startTime)})
		req.End = gen.NewOptEventDateTime(gen.EventDateTime{Date: gen.NewOptNilString(endTime)})
	} else {
		req.Start = gen.NewOptEventDateTime(gen.EventDateTime{DateTime: gen.NewOptNilString(startTime), TimeZone: gen.NewOptNilString(timezone)})
		req.End = gen.NewOptEventDateTime(gen.EventDateTime{DateTime: gen.NewOptNilString(endTime), TimeZone: gen.NewOptNilString(timezone)})
	}

	if attendees, ok := params["attendees"].([]interface{}); ok && len(attendees) > 0 {
		list := make([]gen.EventAttendee, 0, len(attendees))
		for _, a := range attendees {
			if email, ok := a.(string); ok {
				list = append(list, gen.EventAttendee{Email: gen.NewOptString(email)})
			}
		}
		req.Attendees = gen.NewOptNilEventAttendeeArray(list)
	}

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.CreateEvent(ctx, req, gen.CreateEventParams{CalendarId: calendarID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func updateEvent(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	eventID, _ := params["event_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	// Get existing event first
	existing, err := c.GetEvent(ctx, gen.GetEventParams{CalendarId: calendarID, EventId: eventID})
	if err != nil {
		return "", err
	}

	// Build update request from existing event
	req := &gen.EventRequest{
		Summary:     existing.Summary,
		Description: existing.Description,
		Location:    existing.Location,
		Start:       existing.Start,
		End:         existing.End,
		Attendees:   existing.Attendees,
	}

	// Override with provided params
	if summary, ok := params["summary"].(string); ok && summary != "" {
		req.Summary = gen.NewOptNilString(summary)
	}
	if desc, ok := params["description"].(string); ok {
		req.Description = gen.NewOptNilString(desc)
	}
	if loc, ok := params["location"].(string); ok {
		req.Location = gen.NewOptNilString(loc)
	}

	allDay, hasAllDay := params["all_day"].(bool)
	if startTime, ok := params["start_time"].(string); ok && startTime != "" {
		if hasAllDay && allDay {
			req.Start = gen.NewOptEventDateTime(gen.EventDateTime{Date: gen.NewOptNilString(startTime)})
		} else {
			req.Start = gen.NewOptEventDateTime(gen.EventDateTime{DateTime: gen.NewOptNilString(startTime)})
		}
	}
	if endTime, ok := params["end_time"].(string); ok && endTime != "" {
		if hasAllDay && allDay {
			req.End = gen.NewOptEventDateTime(gen.EventDateTime{Date: gen.NewOptNilString(endTime)})
		} else {
			req.End = gen.NewOptEventDateTime(gen.EventDateTime{DateTime: gen.NewOptNilString(endTime)})
		}
	}

	res, err := c.UpdateEvent(ctx, req, gen.UpdateEventParams{CalendarId: calendarID, EventId: eventID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func deleteEvent(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	eventID, _ := params["event_id"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	err = c.DeleteEvent(ctx, gen.DeleteEventParams{CalendarId: calendarID, EventId: eventID})
	if err != nil {
		return "", err
	}
	return `{"success":true,"message":"Event deleted"}`, nil
}

func quickAdd(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	text, _ := params["text"].(string)

	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.QuickAdd(ctx, gen.QuickAddParams{CalendarId: calendarID, Text: text})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}
