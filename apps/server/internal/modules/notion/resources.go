package notion

import (
	"context"
	"fmt"
	"strings"

	"mcpist/server/internal/httpclient"
	"mcpist/server/internal/modules"
)

// resourceDefinitions returns available Notion resources
func resourceDefinitions() []modules.Resource {
	return []modules.Resource{
		{
			URI:         "notion://pages/{page_id}",
			Name:        "Notion Page",
			Description: "Notionページの内容を取得",
			MimeType:    "application/json",
		},
		{
			URI:         "notion://databases/{database_id}",
			Name:        "Notion Database Schema",
			Description: "Notionデータベースのスキーマを取得",
			MimeType:    "application/json",
		},
		{
			URI:         "notion://databases/{database_id}/rows",
			Name:        "Notion Database Rows",
			Description: "Notionデータベースの行を取得",
			MimeType:    "application/json",
		},
		{
			URI:         "notion://blocks/{block_id}/children",
			Name:        "Notion Block Children",
			Description: "Notionブロックの子ブロックを取得",
			MimeType:    "application/json",
		},
	}
}

// readResource reads a Notion resource by URI
func readResource(ctx context.Context, uri string) (string, error) {
	// Parse URI: notion://pages/{page_id}, notion://databases/{database_id}, etc.
	if !strings.HasPrefix(uri, "notion://") {
		return "", fmt.Errorf("invalid URI scheme: %s", uri)
	}

	path := strings.TrimPrefix(uri, "notion://")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		return "", fmt.Errorf("invalid URI format: %s", uri)
	}

	resourceType := parts[0]
	resourceID := parts[1]

	switch resourceType {
	case "pages":
		return readPage(ctx, resourceID)
	case "databases":
		if len(parts) == 3 && parts[2] == "rows" {
			return readDatabaseRows(ctx, resourceID)
		}
		return readDatabaseSchema(ctx, resourceID)
	case "blocks":
		if len(parts) == 3 && parts[2] == "children" {
			return readBlockChildren(ctx, resourceID)
		}
		return "", fmt.Errorf("invalid blocks URI: %s", uri)
	default:
		return "", fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

func readPage(ctx context.Context, pageID string) (string, error) {
	// Get page metadata
	endpoint := fmt.Sprintf("%s/pages/%s", notionAPIBase, pageID)
	pageData, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	// Get page content (blocks)
	blocksEndpoint := fmt.Sprintf("%s/blocks/%s/children?page_size=100", notionAPIBase, pageID)
	blocksData, err := client.DoJSON("GET", blocksEndpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}

	// Combine into single response
	result := map[string]interface{}{
		"page":   pageData,
		"blocks": blocksData,
	}

	return httpclient.PrettyJSONFromInterface(result), nil
}

func readDatabaseSchema(ctx context.Context, databaseID string) (string, error) {
	endpoint := fmt.Sprintf("%s/databases/%s", notionAPIBase, databaseID)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSONFromInterface(respBody), nil
}

func readDatabaseRows(ctx context.Context, databaseID string) (string, error) {
	endpoint := fmt.Sprintf("%s/databases/%s/query", notionAPIBase, databaseID)
	body := map[string]interface{}{
		"page_size": 100,
	}
	respBody, err := client.DoJSON("POST", endpoint, headers(ctx), body)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSONFromInterface(respBody), nil
}

func readBlockChildren(ctx context.Context, blockID string) (string, error) {
	endpoint := fmt.Sprintf("%s/blocks/%s/children?page_size=100", notionAPIBase, blockID)
	respBody, err := client.DoJSON("GET", endpoint, headers(ctx), nil)
	if err != nil {
		return "", err
	}
	return httpclient.PrettyJSONFromInterface(respBody), nil
}
