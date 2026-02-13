package google_drive

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// =============================================================================
// Module-local HTTP helpers for endpoints that cannot be modeled by ogen:
//   - upload_file     (multipart/related upload, different base URL)
//   - update_file_content (media upload, different base URL)
//   - read_file       (binary/text download)
//   - export_file     (export download)
//   - empty_trash     (DELETE 204, no response body)
// =============================================================================

const (
	driveAPIBase   = "https://www.googleapis.com/drive/v3"
	driveUploadURL = "https://www.googleapis.com/upload/drive/v3"
	fileFields     = "id,name,mimeType,size,createdTime,modifiedTime,parents,webViewLink,iconLink,trashed"
	maxReadSize    = 1 * 1024 * 1024 // 1MB
)

// doUploadFile uploads a new file via multipart/related.
func doUploadFile(ctx context.Context, token string, name, content, mimeType, parentID string) (string, error) {
	if mimeType == "" {
		mimeType = "text/plain"
	}

	metadata := map[string]any{"name": name, "mimeType": mimeType}
	if parentID != "" {
		metadata["parents"] = []string{parentID}
	}

	boundary := "========multipart_boundary========"
	var body strings.Builder
	body.WriteString("--" + boundary + "\r\nContent-Type: application/json; charset=UTF-8\r\n\r\n")
	metaJSON, _ := json.Marshal(metadata)
	body.Write(metaJSON)
	body.WriteString("\r\n--" + boundary + "\r\nContent-Type: " + mimeType + "\r\n\r\n")
	body.WriteString(content)
	body.WriteString("\r\n--" + boundary + "--")

	endpoint := fmt.Sprintf("%s/files?uploadType=multipart&fields=%s", driveUploadURL, fileFields)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(body.String()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "multipart/related; boundary="+boundary)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed: status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}

// doUpdateFileContent updates file content via media upload.
func doUpdateFileContent(ctx context.Context, token, fileID, content string) (string, error) {
	// Get current MIME type
	metaReq, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/files/%s?fields=mimeType", driveAPIBase, url.PathEscape(fileID)), nil)
	metaReq.Header.Set("Authorization", "Bearer "+token)
	metaResp, err := http.DefaultClient.Do(metaReq)
	if err != nil {
		return "", err
	}
	defer metaResp.Body.Close()

	var meta struct {
		MimeType string `json:"mimeType"`
	}
	json.NewDecoder(metaResp.Body).Decode(&meta)
	mimeType := meta.MimeType
	if mimeType == "" {
		mimeType = "text/plain"
	}

	endpoint := fmt.Sprintf("%s/files/%s?uploadType=media&fields=%s", driveUploadURL, url.PathEscape(fileID), fileFields)
	req, err := http.NewRequestWithContext(ctx, "PATCH", endpoint, strings.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", mimeType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to update file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("update failed: status %d", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}

// doReadFile reads file content (handles Google Workspace export automatically).
func doReadFile(ctx context.Context, token, fileID string) (string, error) {
	// Get metadata to determine MIME type
	metaReq, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/files/%s?fields=mimeType,name", driveAPIBase, url.PathEscape(fileID)), nil)
	metaReq.Header.Set("Authorization", "Bearer "+token)
	metaResp, err := http.DefaultClient.Do(metaReq)
	if err != nil {
		return "", err
	}
	defer metaResp.Body.Close()

	var meta struct {
		MimeType string `json:"mimeType"`
		Name     string `json:"name"`
	}
	json.NewDecoder(metaResp.Body).Decode(&meta)

	var endpoint string
	switch meta.MimeType {
	case "application/vnd.google-apps.document":
		endpoint = fmt.Sprintf("%s/files/%s/export?mimeType=text/plain", driveAPIBase, url.PathEscape(fileID))
	case "application/vnd.google-apps.spreadsheet":
		endpoint = fmt.Sprintf("%s/files/%s/export?mimeType=text/csv", driveAPIBase, url.PathEscape(fileID))
	case "application/vnd.google-apps.presentation":
		endpoint = fmt.Sprintf("%s/files/%s/export?mimeType=text/plain", driveAPIBase, url.PathEscape(fileID))
	default:
		endpoint = fmt.Sprintf("%s/files/%s?alt=media", driveAPIBase, url.PathEscape(fileID))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("read failed: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReadSize+1))
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	result := map[string]any{
		"name":      meta.Name,
		"mime_type": meta.MimeType,
		"content":   string(data),
		"truncated": len(data) > maxReadSize,
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}

// doExportFile exports a Google Workspace file to the specified MIME type.
func doExportFile(ctx context.Context, token, fileID, mimeType string) (string, error) {
	endpoint := fmt.Sprintf("%s/files/%s/export?mimeType=%s", driveAPIBase, url.PathEscape(fileID), url.QueryEscape(mimeType))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to export file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("export failed: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReadSize+1))
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	result := map[string]any{
		"mime_type": mimeType,
		"content":   string(data),
		"truncated": len(data) > maxReadSize,
	}
	b, _ := json.Marshal(result)
	return string(b), nil
}

// doEmptyTrash permanently deletes all trashed files.
func doEmptyTrash(ctx context.Context, token string) (string, error) {
	endpoint := driveAPIBase + "/files/trash"

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to empty trash: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("empty trash failed: status %d", resp.StatusCode)
	}

	return `{"success":true,"message":"Trash emptied successfully"}`, nil
}
