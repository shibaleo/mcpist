package observability

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type LokiClient struct {
	url        string
	username   string
	apiKey     string
	httpClient *http.Client
	enabled    bool
}

// Loki Push API format
type lokiPushRequest struct {
	Streams []lokiStream `json:"streams"`
}

type lokiStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}

var defaultClient *LokiClient

func Init() {
	url := os.Getenv("GRAFANA_LOKI_URL")
	username := os.Getenv("GRAFANA_LOKI_USER")
	apiKey := os.Getenv("GRAFANA_LOKI_API_KEY")

	if url == "" || username == "" || apiKey == "" {
		log.Println("Loki not configured, logging disabled")
		defaultClient = &LokiClient{enabled: false}
		return
	}

	defaultClient = &LokiClient{
		url:        url + "/loki/api/v1/push",
		username:   username,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		enabled:    true,
	}
	log.Println("Loki client initialized")
}

func Push(labels map[string]string, data map[string]any) {
	if defaultClient == nil || !defaultClient.enabled {
		return
	}

	go defaultClient.push(labels, data)
}

func (c *LokiClient) push(labels map[string]string, data map[string]any) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app"] = "mcpist-dev"

	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("Loki: failed to marshal data: %v", err)
		return
	}

	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)

	req := lokiPushRequest{
		Streams: []lokiStream{
			{
				Stream: labels,
				Values: [][]string{
					{timestamp, string(dataJSON)},
				},
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		log.Printf("Loki: failed to marshal request: %v", err)
		return
	}

	httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(body))
	if err != nil {
		log.Printf("Loki: failed to create request: %v", err)
		return
	}

	httpReq.SetBasicAuth(c.username, c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("Loki: failed to send: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Loki: unexpected status code: %d", resp.StatusCode)
		return
	}
}

// LogToolCall logs a tool call to Loki
func LogToolCall(requestID, module, tool string, durationMs int64, status string, errMsg string) {
	labels := map[string]string{
		"module": module,
		"tool":   tool,
		"status": status,
	}

	data := map[string]any{
		"request_id":  requestID,
		"duration_ms": durationMs,
		"module":      module,
		"tool":        tool,
		"status":      status,
	}

	if errMsg != "" {
		data["error"] = errMsg
	}

	Push(labels, data)
	log.Printf("Tool call: request=%s module=%s tool=%s duration=%dms status=%s", requestID, module, tool, durationMs, status)
}

// LogRequest logs an incoming request to Loki
func LogRequest(method, path string, statusCode int, durationMs int64) {
	labels := map[string]string{
		"type":   "request",
		"method": method,
		"path":   path,
	}

	data := map[string]any{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": durationMs,
	}

	Push(labels, data)
	log.Printf("Request: %s %s status=%d duration=%dms", method, path, statusCode, durationMs)
}

// LogError logs an error to Loki
func LogError(context string, err error) {
	labels := map[string]string{
		"type":  "error",
		"level": "error",
	}

	data := map[string]any{
		"context": context,
		"error":   fmt.Sprintf("%v", err),
	}

	Push(labels, data)
	log.Printf("Error: context=%s error=%v", context, err)
}

// LogSecurityEvent logs a security-related event to Loki (Layer 3: Detection)
func LogSecurityEvent(requestID, userID, event string, details map[string]any) {
	labels := map[string]string{
		"type":           "security",
		"level":          "warn",
		"event":          event,
		"maybe_attacked": "true",
	}

	data := map[string]any{
		"request_id": requestID,
		"user_id":    userID,
		"event":      event,
	}
	for k, v := range details {
		data[k] = v
	}

	Push(labels, data)
	log.Printf("WARN: security event=%s user=%s request=%s details=%v", event, userID, requestID, details)
}
