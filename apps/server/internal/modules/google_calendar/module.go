package google_calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/store"
)

const (
	googleCalendarAPIBase = "https://www.googleapis.com/calendar/v3"
	googleCalendarVersion = "v3"
	googleTokenURL        = "https://oauth2.googleapis.com/token"
	tokenRefreshBuffer    = 5 * 60 // Refresh 5 minutes before expiry
)

var client = httpclient.New()

// GoogleCalendarModule implements the Module interface for Google Calendar API
type GoogleCalendarModule struct{}

// New creates a new GoogleCalendarModule instance
func New() *GoogleCalendarModule {
	return &GoogleCalendarModule{}
}

// Name returns the module name
func (m *GoogleCalendarModule) Name() string {
	return "google_calendar"
}

// Description returns the module description
func (m *GoogleCalendarModule) Description() string {
	return "Google Calendar API - List, create, update, and delete events"
}

// APIVersion returns the Google Calendar API version
func (m *GoogleCalendarModule) APIVersion() string {
	return googleCalendarVersion
}

// Tools returns all available tools
func (m *GoogleCalendarModule) Tools() []modules.Tool {
	return toolDefinitions
}

// ExecuteTool executes a tool by name and returns JSON response
func (m *GoogleCalendarModule) ExecuteTool(ctx context.Context, name string, params map[string]any) (string, error) {
	handler, ok := toolHandlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, params)
}

// Resources returns all available resources (none for Google Calendar)
func (m *GoogleCalendarModule) Resources() []modules.Resource {
	return nil
}

// ReadResource reads a resource by URI (not implemented)
func (m *GoogleCalendarModule) ReadResource(ctx context.Context, uri string) (string, error) {
	return "", fmt.Errorf("resources not supported")
}

// Prompts returns all available prompts (none for Google Calendar)
func (m *GoogleCalendarModule) Prompts() []modules.Prompt {
	return nil
}

// GetPrompt generates a prompt with arguments (not implemented)
func (m *GoogleCalendarModule) GetPrompt(ctx context.Context, name string, args map[string]any) (string, error) {
	return "", fmt.Errorf("prompts not supported")
}

// =============================================================================
// Token and Headers
// =============================================================================

func getCredentials(ctx context.Context) *store.Credentials {
	authCtx := middleware.GetAuthContext(ctx)
	if authCtx == nil {
		log.Printf("[google_calendar] No auth context")
		return nil
	}
	credentials, err := store.GetTokenStore().GetModuleToken(ctx, authCtx.UserID, "google_calendar")
	if err != nil {
		log.Printf("[google_calendar] GetModuleToken error: %v", err)
		return nil
	}
	log.Printf("[google_calendar] Got credentials: auth_type=%s, has_access_token=%v", credentials.AuthType, credentials.AccessToken != "")

	// Check if token needs refresh (OAuth2 only)
	if credentials.AuthType == store.AuthTypeOAuth2 && credentials.RefreshToken != "" {
		if needsRefresh(credentials) {
			log.Printf("[google_calendar] Token expired or expiring soon, refreshing...")
			refreshed, err := refreshToken(ctx, authCtx.UserID, credentials)
			if err != nil {
				log.Printf("[google_calendar] Token refresh failed: %v", err)
				// Return original credentials and let the API call fail
				return credentials
			}
			log.Printf("[google_calendar] Token refreshed successfully")
			return refreshed
		}
	}

	return credentials
}

// needsRefresh checks if the token is expired or will expire soon
func needsRefresh(creds *store.Credentials) bool {
	if creds.ExpiresAt == 0 {
		// No expiry information, assume token is valid
		return false
	}
	now := time.Now().Unix()
	// Refresh if expired or expiring within buffer period
	return now >= (creds.ExpiresAt - tokenRefreshBuffer)
}

// refreshToken exchanges the refresh token for a new access token
func refreshToken(ctx context.Context, userID string, creds *store.Credentials) (*store.Credentials, error) {
	// Get OAuth app credentials (client_id, client_secret)
	oauthApp, err := store.GetTokenStore().GetOAuthAppCredentials(ctx, "google")
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth app credentials: %w", err)
	}

	// Exchange refresh token for new access token
	data := url.Values{}
	data.Set("client_id", oauthApp.ClientID)
	data.Set("client_secret", oauthApp.ClientSecret)
	data.Set("refresh_token", creds.RefreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", googleTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Update credentials with new access token
	newCreds := &store.Credentials{
		AuthType:     store.AuthTypeOAuth2,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: creds.RefreshToken, // Keep the same refresh token
		ExpiresAt:    time.Now().Unix() + tokenResp.ExpiresIn,
	}

	// Save updated credentials to Vault
	err = store.GetTokenStore().UpdateModuleToken(ctx, userID, "google_calendar", newCreds)
	if err != nil {
		log.Printf("[google_calendar] Failed to save refreshed token: %v", err)
		// Continue anyway, the token is still valid for this request
	}

	return newCreds, nil
}

func headers(ctx context.Context) map[string]string {
	creds := getCredentials(ctx)
	if creds == nil {
		return map[string]string{}
	}

	h := map[string]string{
		"Accept": "application/json",
	}

	// OAuth2 uses Bearer token
	if creds.AuthType == store.AuthTypeOAuth2 {
		h["Authorization"] = "Bearer " + creds.AccessToken
	}

	return h
}

// =============================================================================
// Tool Definitions
// =============================================================================

var toolDefinitions = []modules.Tool{
	// Calendars
	{
		Name:        "list_calendars",
		Description: "List all calendars accessible to the user.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type:       "object",
			Properties: map[string]modules.Property{},
		},
	},
	{
		Name:        "get_calendar",
		Description: "Get details of a specific calendar.",
		Annotations: modules.AnnotateReadOnly,
		InputSchema: modules.InputSchema{
			Type: "object",
			Properties: map[string]modules.Property{
				"calendar_id": {Type: "string", Description: "Calendar ID. Use 'primary' for the user's primary calendar."},
			},
			Required: []string{"calendar_id"},
		},
	},
	// Events
	{
		Name:        "list_events",
		Description: "List events from a calendar within a time range.",
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
		Name:        "get_event",
		Description: "Get details of a specific event.",
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
		Name:        "create_event",
		Description: "Create a new event in a calendar.",
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
		Name:        "update_event",
		Description: "Update an existing event.",
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
		Name:        "delete_event",
		Description: "Delete an event from a calendar.",
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
	// Quick add
	{
		Name:        "quick_add",
		Description: "Create an event based on a simple text string (e.g., 'Meeting with John tomorrow at 3pm').",
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
	endpoint := googleCalendarAPIBase + "/users/me/calendarList"
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getCalendar(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	endpoint := fmt.Sprintf("%s/calendars/%s", googleCalendarAPIBase, url.PathEscape(calendarID))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

// =============================================================================
// Events
// =============================================================================

func listEvents(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)

	query := url.Values{}

	// Time range defaults
	now := time.Now().UTC()
	timeMin := now.Format(time.RFC3339)
	timeMax := now.AddDate(0, 0, 7).Format(time.RFC3339)

	if tm, ok := params["time_min"].(string); ok && tm != "" {
		timeMin = tm
	}
	if tm, ok := params["time_max"].(string); ok && tm != "" {
		timeMax = tm
	}
	query.Set("timeMin", timeMin)
	query.Set("timeMax", timeMax)

	// Max results
	maxResults := 50
	if mr, ok := params["max_results"].(float64); ok {
		maxResults = int(mr)
	}
	query.Set("maxResults", fmt.Sprintf("%d", maxResults))

	// Single events (expand recurring)
	singleEvents := true
	if se, ok := params["single_events"].(bool); ok {
		singleEvents = se
	}
	query.Set("singleEvents", fmt.Sprintf("%t", singleEvents))

	// Order by
	orderBy := "startTime"
	if ob, ok := params["order_by"].(string); ok && ob != "" {
		orderBy = ob
	}
	if singleEvents {
		query.Set("orderBy", orderBy)
	}

	// Search query
	if q, ok := params["q"].(string); ok && q != "" {
		query.Set("q", q)
	}

	endpoint := fmt.Sprintf("%s/calendars/%s/events?%s", googleCalendarAPIBase, url.PathEscape(calendarID), query.Encode())
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func getEvent(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	eventID, _ := params["event_id"].(string)

	endpoint := fmt.Sprintf("%s/calendars/%s/events/%s", googleCalendarAPIBase, url.PathEscape(calendarID), url.PathEscape(eventID))
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func createEvent(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	summary, _ := params["summary"].(string)
	startTime, _ := params["start_time"].(string)
	endTime, _ := params["end_time"].(string)

	// Build event body
	event := map[string]interface{}{
		"summary": summary,
	}

	if desc, ok := params["description"].(string); ok && desc != "" {
		event["description"] = desc
	}
	if loc, ok := params["location"].(string); ok && loc != "" {
		event["location"] = loc
	}

	// Timezone
	timezone := "UTC"
	if tz, ok := params["timezone"].(string); ok && tz != "" {
		timezone = tz
	}

	// All-day event vs timed event
	allDay, _ := params["all_day"].(bool)
	if allDay {
		event["start"] = map[string]string{"date": startTime}
		event["end"] = map[string]string{"date": endTime}
	} else {
		event["start"] = map[string]string{"dateTime": startTime, "timeZone": timezone}
		event["end"] = map[string]string{"dateTime": endTime, "timeZone": timezone}
	}

	// Attendees
	if attendees, ok := params["attendees"].([]interface{}); ok && len(attendees) > 0 {
		attendeeList := make([]map[string]string, len(attendees))
		for i, a := range attendees {
			if email, ok := a.(string); ok {
				attendeeList[i] = map[string]string{"email": email}
			}
		}
		event["attendees"] = attendeeList
	}

	endpoint := fmt.Sprintf("%s/calendars/%s/events", googleCalendarAPIBase, url.PathEscape(calendarID))
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), event)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func updateEvent(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	eventID, _ := params["event_id"].(string)

	// First, get the existing event
	getEndpoint := fmt.Sprintf("%s/calendars/%s/events/%s", googleCalendarAPIBase, url.PathEscape(calendarID), url.PathEscape(eventID))
	existingBody, err := client.DoJSON("GET", getEndpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	var event map[string]interface{}
	if err := json.Unmarshal(existingBody, &event); err != nil {
		return "", fmt.Errorf("failed to parse existing event: %w", err)
	}

	// Update fields
	if summary, ok := params["summary"].(string); ok && summary != "" {
		event["summary"] = summary
	}
	if desc, ok := params["description"].(string); ok {
		event["description"] = desc
	}
	if loc, ok := params["location"].(string); ok {
		event["location"] = loc
	}

	// Update times
	allDay, hasAllDay := params["all_day"].(bool)
	if startTime, ok := params["start_time"].(string); ok && startTime != "" {
		if hasAllDay && allDay {
			event["start"] = map[string]string{"date": startTime}
		} else {
			event["start"] = map[string]string{"dateTime": startTime}
		}
	}
	if endTime, ok := params["end_time"].(string); ok && endTime != "" {
		if hasAllDay && allDay {
			event["end"] = map[string]string{"date": endTime}
		} else {
			event["end"] = map[string]string{"dateTime": endTime}
		}
	}

	// PUT to update
	respBody, err := client.DoJSON("PUT", getEndpoint, headers(ctx), event)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}

func deleteEvent(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	eventID, _ := params["event_id"].(string)

	endpoint := fmt.Sprintf("%s/calendars/%s/events/%s", googleCalendarAPIBase, url.PathEscape(calendarID), url.PathEscape(eventID))

	// DoJSON handles DELETE requests - Google Calendar API returns 204 No Content on success
	_, err := client.DoJSON("DELETE", endpoint, headers(ctx), nil)
	if err != nil {
		// Check if it's a 204 No Content (success for DELETE)
		if apiErr, ok := err.(*httpclient.APIError); ok && apiErr.StatusCode == 204 {
			return `{"success": true, "message": "Event deleted"}`, nil
		}
		return "", err
	}

	return `{"success": true, "message": "Event deleted"}`, nil
}

func quickAdd(ctx context.Context, params map[string]any) (string, error) {
	calendarID, _ := params["calendar_id"].(string)
	text, _ := params["text"].(string)

	endpoint := fmt.Sprintf("%s/calendars/%s/events/quickAdd?text=%s", googleCalendarAPIBase, url.PathEscape(calendarID), url.QueryEscape(text))
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSON(respBody), nil
}
