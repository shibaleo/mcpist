package dropbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// doPost sends a JSON-RPC POST request to the Dropbox API and returns the raw response body.
func doPost(ctx context.Context, path string, body any) (string, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", dropboxAPIBase+path, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	creds := getCredentials(ctx)
	if creds != nil {
		req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

// doContentDownload handles Dropbox content download endpoints.
// Parameters are sent via Dropbox-API-Arg header, response body is file content.
func doContentDownload(ctx context.Context, path string, apiArg any) (string, error) {
	argJSON, err := json.Marshal(apiArg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal API arg: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", dropboxContentBase+path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	creds := getCredentials(ctx)
	if creds != nil {
		req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	}
	req.Header.Set("Dropbox-API-Arg", string(argJSON))

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Read content (limit to 1MB)
	const maxSize = 1 * 1024 * 1024
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}
	truncated := len(content) > maxSize
	if truncated {
		content = content[:maxSize]
	}

	// Metadata comes back in Dropbox-API-Result header
	apiResult := resp.Header.Get("Dropbox-API-Result")

	result := map[string]any{
		"content":   string(content),
		"truncated": truncated,
	}
	if apiResult != "" {
		result["metadata"] = json.RawMessage(apiResult)
	}
	return toJSON(result)
}

// doContentUpload handles Dropbox content upload endpoints.
// Parameters are sent via Dropbox-API-Arg header, request body is file content.
func doContentUpload(ctx context.Context, path string, apiArg any, content string) (string, error) {
	argJSON, err := json.Marshal(apiArg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal API arg: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", dropboxContentBase+path, strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	creds := getCredentials(ctx)
	if creds != nil {
		req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
	}
	req.Header.Set("Dropbox-API-Arg", string(argJSON))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed (status %d): %s", resp.StatusCode, string(respBody))
	}
	return string(respBody), nil
}
