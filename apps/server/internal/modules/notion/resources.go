package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mcpist/server/internal/modules"
	gen "mcpist/server/pkg/notionapi/gen"
)

// resourceDefinitions returns available Notion resources
func resourceDefinitions() []modules.Resource {
	return []modules.Resource{
		{
			URI:         "notion://pages/{page_id}",
			Name:        "Notion Page",
			Description: "Get the content of a Notion page",
			MimeType:    "application/json",
		},
		{
			URI:         "notion://databases/{database_id}",
			Name:        "Notion Database Schema",
			Description: "Get the schema of a Notion database",
			MimeType:    "application/json",
		},
		{
			URI:         "notion://databases/{database_id}/rows",
			Name:        "Notion Database Rows",
			Description: "Get rows from a Notion database",
			MimeType:    "application/json",
		},
		{
			URI:         "notion://blocks/{block_id}/children",
			Name:        "Notion Block Children",
			Description: "Get child blocks of a Notion block",
			MimeType:    "application/json",
		},
	}
}

// readResource reads a Notion resource by URI
func readResource(ctx context.Context, uri string) (string, error) {
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
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	// Get page metadata
	pageRes, err := c.GetPage(ctx, gen.GetPageParams{PageID: pageID})
	if err != nil {
		return "", err
	}

	// Get page content (blocks)
	p := gen.GetBlockChildrenParams{BlockID: pageID}
	p.PageSize.SetTo(100)
	blocksRes, err := c.GetBlockChildren(ctx, p)
	if err != nil {
		return "", err
	}

	// Combine into single response
	pageJSON, _ := json.Marshal(pageRes)
	blocksJSON, _ := json.Marshal(blocksRes)

	var pageData, blocksData any
	json.Unmarshal(pageJSON, &pageData)
	json.Unmarshal(blocksJSON, &blocksData)

	result := map[string]any{
		"page":   pageData,
		"blocks": blocksData,
	}
	return toJSON(result)
}

func readDatabaseSchema(ctx context.Context, databaseID string) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	res, err := c.GetDatabase(ctx, gen.GetDatabaseParams{DatabaseID: databaseID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func readDatabaseRows(ctx context.Context, databaseID string) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}

	body := map[string]any{"page_size": 100}
	bodyJSON, _ := json.Marshal(body)
	var req gen.QueryDatabaseRequest
	json.Unmarshal(bodyJSON, &req)

	res, err := c.QueryDatabase(ctx, &req, gen.QueryDatabaseParams{DatabaseID: databaseID})
	if err != nil {
		return "", err
	}
	return toJSON(res)
}

func readBlockChildren(ctx context.Context, blockID string) (string, error) {
	c, err := newOgenClient(ctx)
	if err != nil {
		return "", err
	}
	p := gen.GetBlockChildrenParams{BlockID: blockID}
	p.PageSize.SetTo(100)
	res, err := c.GetBlockChildren(ctx, p)
	if err != nil {
		return "", err
	}
	return toJSON(res)
}
