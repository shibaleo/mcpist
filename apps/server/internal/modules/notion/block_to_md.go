package notion

import (
	"encoding/json"
	"fmt"
	"strings"
)

// =============================================================================
// Markdown Converter for Notion Blocks
// Converts Notion block JSON to readable Markdown format
// =============================================================================

// BlocksToMarkdown converts Notion blocks JSON response to Markdown
func BlocksToMarkdown(jsonStr string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	results, ok := data["results"].([]any)
	if !ok {
		return ""
	}

	var sb strings.Builder
	var listStack []string // Track nested list types

	for _, item := range results {
		block, ok := item.(map[string]any)
		if !ok {
			continue
		}

		blockType := getString(block, "type")
		md := blockToMarkdown(block, blockType, &listStack)
		if md != "" {
			sb.WriteString(md)
		}
	}

	// Add pagination info if has_more
	if hasMore, ok := data["has_more"].(bool); ok && hasMore {
		if cursor, ok := data["next_cursor"].(string); ok {
			sb.WriteString(fmt.Sprintf("\n---\n[more: %s]\n", cursor))
		}
	}

	return strings.TrimSuffix(sb.String(), "\n")
}

// blockToMarkdown converts a single block to Markdown
func blockToMarkdown(block map[string]any, blockType string, listStack *[]string) string {
	content, ok := block[blockType].(map[string]any)
	if !ok && blockType != "unsupported" {
		return ""
	}

	switch blockType {
	// Headings
	case "heading_1":
		return fmt.Sprintf("# %s\n\n", extractRichTextMD(content))
	case "heading_2":
		return fmt.Sprintf("## %s\n\n", extractRichTextMD(content))
	case "heading_3":
		return fmt.Sprintf("### %s\n\n", extractRichTextMD(content))

	// Paragraph
	case "paragraph":
		text := extractRichTextMD(content)
		if text == "" {
			return "\n"
		}
		return fmt.Sprintf("%s\n\n", text)

	// Lists
	case "bulleted_list_item":
		return fmt.Sprintf("- %s\n", extractRichTextMD(content))
	case "numbered_list_item":
		return fmt.Sprintf("1. %s\n", extractRichTextMD(content))
	case "to_do":
		checked := false
		if c, ok := content["checked"].(bool); ok {
			checked = c
		}
		checkbox := "[ ]"
		if checked {
			checkbox = "[x]"
		}
		return fmt.Sprintf("- %s %s\n", checkbox, extractRichTextMD(content))

	// Quote & Callout
	case "quote":
		text := extractRichTextMD(content)
		lines := strings.Split(text, "\n")
		var quoted []string
		for _, line := range lines {
			quoted = append(quoted, "> "+line)
		}
		return strings.Join(quoted, "\n") + "\n\n"
	case "callout":
		icon := extractIcon(content)
		text := extractRichTextMD(content)
		if icon != "" {
			return fmt.Sprintf("> %s %s\n\n", icon, text)
		}
		return fmt.Sprintf("> %s\n\n", text)

	// Code
	case "code":
		lang := ""
		if l, ok := content["language"].(string); ok {
			lang = l
		}
		text := extractRichTextMD(content)
		return fmt.Sprintf("```%s\n%s\n```\n\n", lang, text)

	// Divider
	case "divider":
		return "---\n\n"

	// Toggle
	case "toggle":
		text := extractRichTextMD(content)
		hasChildren := false
		if hc, ok := block["has_children"].(bool); ok {
			hasChildren = hc
		}
		if hasChildren {
			return fmt.Sprintf("<details>\n<summary>%s</summary>\n\n[nested content]\n</details>\n\n", text)
		}
		return fmt.Sprintf("<details>\n<summary>%s</summary>\n</details>\n\n", text)

	// Image
	case "image":
		url := extractFileURL(content)
		caption := extractCaption(content)
		if caption != "" {
			return fmt.Sprintf("![%s](%s)\n\n", caption, url)
		}
		return fmt.Sprintf("![](%s)\n\n", url)

	// Bookmark & Link
	case "bookmark":
		url := ""
		if u, ok := content["url"].(string); ok {
			url = u
		}
		caption := extractCaption(content)
		if caption != "" {
			return fmt.Sprintf("[%s](%s)\n\n", caption, url)
		}
		return fmt.Sprintf("<%s>\n\n", url)

	case "link_preview":
		url := ""
		if u, ok := content["url"].(string); ok {
			url = u
		}
		return fmt.Sprintf("<%s>\n\n", url)

	// Embed & Video
	case "embed", "video":
		url := extractFileURL(content)
		return fmt.Sprintf("<%s>\n\n", url)

	// File & PDF
	case "file", "pdf":
		url := extractFileURL(content)
		caption := extractCaption(content)
		if caption != "" {
			return fmt.Sprintf("[%s](%s)\n\n", caption, url)
		}
		return fmt.Sprintf("[file](%s)\n\n", url)

	// Table (simplified - just indicate table exists)
	case "table":
		return "[table]\n\n"
	case "table_row":
		cells := extractTableRow(content)
		return "| " + strings.Join(cells, " | ") + " |\n"

	// Child page/database
	case "child_page":
		title := ""
		if t, ok := content["title"].(string); ok {
			title = t
		}
		return fmt.Sprintf("**%s**\n\n", title)
	case "child_database":
		title := ""
		if t, ok := content["title"].(string); ok {
			title = t
		}
		return fmt.Sprintf("**%s**\n\n", title)

	// Synced block
	case "synced_block":
		return "[synced block]\n\n"

	// Column
	case "column_list":
		return "" // Column containers don't render
	case "column":
		return "" // Columns don't render directly

	// Equation
	case "equation":
		expr := ""
		if e, ok := content["expression"].(string); ok {
			expr = e
		}
		return fmt.Sprintf("$$%s$$\n\n", expr)

	// Breadcrumb & TOC
	case "breadcrumb":
		return "[breadcrumb]\n"
	case "table_of_contents":
		return "[table of contents]\n\n"

	// Link to page
	case "link_to_page":
		return "[link to page]\n\n"

	// Unsupported
	case "unsupported":
		return "[unsupported block]\n\n"

	default:
		return fmt.Sprintf("[%s]\n\n", blockType)
	}
}

// =============================================================================
// Helper functions for Markdown extraction
// =============================================================================

// extractRichTextMD extracts rich text with Markdown formatting
func extractRichTextMD(content map[string]any) string {
	arr, ok := content["rich_text"].([]any)
	if !ok {
		return ""
	}

	var parts []string
	for _, item := range arr {
		textObj, ok := item.(map[string]any)
		if !ok {
			continue
		}

		plainText := ""
		if pt, ok := textObj["plain_text"].(string); ok {
			plainText = pt
		}

		// Apply annotations
		if annotations, ok := textObj["annotations"].(map[string]any); ok {
			plainText = applyAnnotations(plainText, annotations)
		}

		// Handle links
		if href, ok := textObj["href"].(string); ok && href != "" {
			plainText = fmt.Sprintf("[%s](%s)", plainText, href)
		}

		parts = append(parts, plainText)
	}

	return strings.Join(parts, "")
}

// applyAnnotations applies Markdown formatting based on Notion annotations
func applyAnnotations(text string, annotations map[string]any) string {
	if text == "" {
		return text
	}

	// Bold
	if bold, ok := annotations["bold"].(bool); ok && bold {
		text = "**" + text + "**"
	}

	// Italic
	if italic, ok := annotations["italic"].(bool); ok && italic {
		text = "*" + text + "*"
	}

	// Strikethrough
	if strike, ok := annotations["strikethrough"].(bool); ok && strike {
		text = "~~" + text + "~~"
	}

	// Code
	if code, ok := annotations["code"].(bool); ok && code {
		text = "`" + text + "`"
	}

	// Underline (not standard MD, use HTML)
	if underline, ok := annotations["underline"].(bool); ok && underline {
		text = "<u>" + text + "</u>"
	}

	return text
}

// extractIcon extracts emoji or icon from callout
func extractIcon(content map[string]any) string {
	icon, ok := content["icon"].(map[string]any)
	if !ok {
		return ""
	}

	if emoji, ok := icon["emoji"].(string); ok {
		return emoji
	}

	return ""
}

// extractFileURL extracts URL from file/image block
func extractFileURL(content map[string]any) string {
	// Try external URL first
	if external, ok := content["external"].(map[string]any); ok {
		if url, ok := external["url"].(string); ok {
			return url
		}
	}

	// Try file URL (uploaded to Notion)
	if file, ok := content["file"].(map[string]any); ok {
		if url, ok := file["url"].(string); ok {
			return url
		}
	}

	// Direct URL field
	if url, ok := content["url"].(string); ok {
		return url
	}

	return ""
}

// extractCaption extracts caption from block
func extractCaption(content map[string]any) string {
	if caption, ok := content["caption"].([]any); ok {
		return extractPlainText(caption)
	}
	return ""
}

// extractTableRow extracts cells from table_row block
func extractTableRow(content map[string]any) []string {
	cells, ok := content["cells"].([]any)
	if !ok {
		return nil
	}

	var result []string
	for _, cell := range cells {
		if cellArr, ok := cell.([]any); ok {
			result = append(result, extractPlainText(cellArr))
		}
	}
	return result
}
