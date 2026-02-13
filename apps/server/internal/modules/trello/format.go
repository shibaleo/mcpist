package trello

import (
	"encoding/json"
	"fmt"
	"strings"
)

// =============================================================================
// Compact formatters per tool
// =============================================================================

func formatCompact(toolName, jsonStr string) string {
	switch toolName {
	case "list_boards":
		return boardsToCSV(jsonStr)
	case "get_board":
		return boardToCompact(jsonStr)
	case "get_lists":
		return listsToCSV(jsonStr)
	case "get_cards":
		return cardsToCSV(jsonStr)
	case "get_card":
		return cardToCompact(jsonStr)
	case "get_checklists":
		return checklistsToCSV(jsonStr)
	case "get_checklist_items":
		return checkItemsToCSV(jsonStr)
	case "create_card", "update_card", "move_card":
		return pickKeys(jsonStr, "id", "name", "idList")
	case "create_checklist":
		return pickKeys(jsonStr, "id", "name", "idCard")
	case "add_checklist_item", "update_checklist_item":
		return pickKeys(jsonStr, "id", "name", "state")
	default:
		return jsonStr
	}
}

// pickKeys extracts specified keys from a JSON object.
func pickKeys(jsonStr string, keys ...string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	result := make(map[string]any, len(keys))
	for _, k := range keys {
		if v, ok := data[k]; ok && v != nil {
			result[k] = v
		}
	}
	out, err := json.Marshal(result)
	if err != nil {
		return jsonStr
	}
	return string(out)
}

// boardsToCSV: id,name,closed
func boardsToCSV(jsonStr string) string {
	var boards []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &boards); err != nil {
		return jsonStr
	}
	if len(boards) == 0 {
		return "# 0 boards"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,closed\n")
	for _, b := range boards {
		sb.WriteString(fmt.Sprintf("%s,%s,%v\n",
			csvEscape(str(b, "id")),
			csvEscape(str(b, "name")),
			b["closed"],
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// boardToCompact: single board summary
func boardToCompact(jsonStr string) string {
	var b map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &b); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(b, "name")))
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", str(b, "id")))
	if url := str(b, "url"); url != "" {
		sb.WriteString(fmt.Sprintf("- **URL**: %s\n", url))
	}
	if desc := str(b, "desc"); desc != "" {
		sb.WriteString(fmt.Sprintf("- **Description**: %s\n", desc))
	}
	if closed, ok := b["closed"].(bool); ok && closed {
		sb.WriteString("- **Status**: Closed\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// listsToCSV: id,name,closed
func listsToCSV(jsonStr string) string {
	var lists []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &lists); err != nil {
		return jsonStr
	}
	if len(lists) == 0 {
		return "# 0 lists"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,closed\n")
	for _, l := range lists {
		sb.WriteString(fmt.Sprintf("%s,%s,%v\n",
			csvEscape(str(l, "id")),
			csvEscape(str(l, "name")),
			l["closed"],
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// cardsToCSV: id,name,idList,due,closed
func cardsToCSV(jsonStr string) string {
	var cards []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &cards); err != nil {
		return jsonStr
	}
	if len(cards) == 0 {
		return "# 0 cards"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,idList,due,closed\n")
	for _, c := range cards {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%v\n",
			csvEscape(str(c, "id")),
			csvEscape(str(c, "name")),
			str(c, "idList"),
			str(c, "due"),
			c["closed"],
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// cardToCompact: single card with checklists
func cardToCompact(jsonStr string) string {
	var c map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &c); err != nil {
		return jsonStr
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n", str(c, "name")))
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", str(c, "id")))
	sb.WriteString(fmt.Sprintf("- **List**: %s\n", str(c, "idList")))
	sb.WriteString(fmt.Sprintf("- **Board**: %s\n", str(c, "idBoard")))
	if due := str(c, "due"); due != "" {
		sb.WriteString(fmt.Sprintf("- **Due**: %s\n", due))
	}
	if closed, ok := c["closed"].(bool); ok && closed {
		sb.WriteString("- **Status**: Archived\n")
	}
	if desc := str(c, "desc"); desc != "" {
		sb.WriteString(fmt.Sprintf("\n## Description\n%s\n", desc))
	}

	// Labels
	if labels, ok := c["labels"].([]any); ok && len(labels) > 0 {
		names := make([]string, 0, len(labels))
		for _, l := range labels {
			if lm, ok := l.(map[string]any); ok {
				name := str(lm, "name")
				if name == "" {
					name = str(lm, "color")
				}
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			sb.WriteString(fmt.Sprintf("- **Labels**: %s\n", strings.Join(names, ", ")))
		}
	}

	// Checklists as CSV
	if checklists, ok := c["checklists"].([]any); ok && len(checklists) > 0 {
		sb.WriteString("\n```csv\nchecklist_id,checklist_name,item_id,item_name,state\n")
		for _, cl := range checklists {
			clm, ok := cl.(map[string]any)
			if !ok {
				continue
			}
			clID := str(clm, "id")
			clName := str(clm, "name")
			items, ok := clm["checkItems"].([]any)
			if !ok || len(items) == 0 {
				sb.WriteString(fmt.Sprintf("%s,%s,,,\n", csvEscape(clID), csvEscape(clName)))
				continue
			}
			for _, item := range items {
				im, ok := item.(map[string]any)
				if !ok {
					continue
				}
				sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
					csvEscape(clID),
					csvEscape(clName),
					csvEscape(str(im, "id")),
					csvEscape(str(im, "name")),
					str(im, "state"),
				))
			}
		}
		sb.WriteString("```\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// checklistsToCSV: checklist_id,checklist_name,item_id,item_name,state
func checklistsToCSV(jsonStr string) string {
	var checklists []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &checklists); err != nil {
		return jsonStr
	}
	if len(checklists) == 0 {
		return "# 0 checklists"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nchecklist_id,checklist_name,item_id,item_name,state\n")
	for _, cl := range checklists {
		clID := str(cl, "id")
		clName := str(cl, "name")
		items, ok := cl["checkItems"].([]any)
		if !ok || len(items) == 0 {
			// Checklist with no items: show checklist row only
			sb.WriteString(fmt.Sprintf("%s,%s,,,\n", csvEscape(clID), csvEscape(clName)))
			continue
		}
		for _, item := range items {
			im, ok := item.(map[string]any)
			if !ok {
				continue
			}
			sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s\n",
				csvEscape(clID),
				csvEscape(clName),
				csvEscape(str(im, "id")),
				csvEscape(str(im, "name")),
				str(im, "state"),
			))
		}
	}
	sb.WriteString("```")
	return sb.String()
}

// checkItemsToCSV: id,name,state
func checkItemsToCSV(jsonStr string) string {
	var items []map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		return jsonStr
	}
	if len(items) == 0 {
		return "# 0 items"
	}
	var sb strings.Builder
	sb.WriteString("```csv\nid,name,state\n")
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("%s,%s,%s\n",
			csvEscape(str(item, "id")),
			csvEscape(str(item, "name")),
			str(item, "state"),
		))
	}
	sb.WriteString("```")
	return sb.String()
}

// =============================================================================
// Helpers
// =============================================================================

func str(obj map[string]any, key string) string {
	if v, ok := obj[key].(string); ok {
		return v
	}
	return ""
}

func csvEscape(s string) string {
	if s == "" {
		return ""
	}
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}
