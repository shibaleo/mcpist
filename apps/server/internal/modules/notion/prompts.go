package notion

import (
	"context"
	"fmt"
	"strings"

	"mcpist/server/internal/modules"
)

// promptDefinitions returns available Notion prompts
func promptDefinitions() []modules.Prompt {
	return []modules.Prompt{
		{
			Name:        "notion_create_meeting_notes",
			Description: "会議メモページの作成テンプレート",
			Arguments: []modules.PromptArgument{
				{Name: "title", Description: "会議タイトル", Required: true},
				{Name: "date", Description: "会議日時 (YYYY-MM-DD)", Required: true},
				{Name: "attendees", Description: "参加者（カンマ区切り）", Required: false},
				{Name: "parent_page_id", Description: "親ページID", Required: true},
			},
		},
		{
			Name:        "notion_create_task",
			Description: "タスクページの作成テンプレート",
			Arguments: []modules.PromptArgument{
				{Name: "title", Description: "タスクタイトル", Required: true},
				{Name: "description", Description: "タスク詳細", Required: false},
				{Name: "due_date", Description: "期限 (YYYY-MM-DD)", Required: false},
				{Name: "priority", Description: "優先度 (high/medium/low)", Required: false},
				{Name: "database_id", Description: "タスクデータベースID", Required: true},
			},
		},
		{
			Name:        "notion_weekly_report",
			Description: "週次レポートページの作成テンプレート",
			Arguments: []modules.PromptArgument{
				{Name: "week_start", Description: "週の開始日 (YYYY-MM-DD)", Required: true},
				{Name: "accomplishments", Description: "今週の成果", Required: false},
				{Name: "challenges", Description: "課題・問題点", Required: false},
				{Name: "next_week_plans", Description: "来週の予定", Required: false},
				{Name: "parent_page_id", Description: "親ページID", Required: true},
			},
		},
	}
}

// getPrompt generates a prompt template with arguments
func getPrompt(ctx context.Context, name string, args map[string]any) (string, error) {
	switch name {
	case "notion_create_meeting_notes":
		return generateMeetingNotesPrompt(args)
	case "notion_create_task":
		return generateTaskPrompt(args)
	case "notion_weekly_report":
		return generateWeeklyReportPrompt(args)
	default:
		return "", fmt.Errorf("unknown prompt: %s", name)
	}
}

func generateMeetingNotesPrompt(args map[string]any) (string, error) {
	title, _ := args["title"].(string)
	date, _ := args["date"].(string)
	attendees, _ := args["attendees"].(string)
	parentPageID, _ := args["parent_page_id"].(string)

	if title == "" || date == "" || parentPageID == "" {
		return "", fmt.Errorf("title, date, and parent_page_id are required")
	}

	prompt := fmt.Sprintf(`Create a meeting notes page in Notion with the following structure:

**Title:** %s - %s

**Blocks to create:**
1. Heading 2: "参加者"
   - Bulleted list: %s

2. Heading 2: "アジェンダ"
   - Numbered list: (empty items for user to fill)

3. Heading 2: "議事メモ"
   - Paragraph: (empty for notes)

4. Heading 2: "アクションアイテム"
   - To-do items: (empty checkboxes)

5. Heading 2: "次回予定"
   - Paragraph: (empty)

**Instructions:**
Use the Notion API to:
1. Create a new page with parent_page_id: %s
2. Add the blocks above using append_blocks
`, title, date, formatAttendees(attendees), parentPageID)

	return prompt, nil
}

func generateTaskPrompt(args map[string]any) (string, error) {
	title, _ := args["title"].(string)
	description, _ := args["description"].(string)
	dueDate, _ := args["due_date"].(string)
	priority, _ := args["priority"].(string)
	databaseID, _ := args["database_id"].(string)

	if title == "" || databaseID == "" {
		return "", fmt.Errorf("title and database_id are required")
	}

	if priority == "" {
		priority = "medium"
	}

	prompt := fmt.Sprintf(`Create a task in Notion database with the following properties:

**Database ID:** %s

**Properties to set:**
- Name/Title: %s
- Status: Not Started
- Priority: %s
%s%s

**Instructions:**
Use the Notion API create_page tool with:
- parent_database_id: %s
- title: %s
- properties: Set appropriate property values based on the database schema

First, use get_database to check the exact property names and types.
`, databaseID, title, priority,
		conditionalLine("- Due Date: ", dueDate),
		conditionalLine("- Description: ", description),
		databaseID, title)

	return prompt, nil
}

func generateWeeklyReportPrompt(args map[string]any) (string, error) {
	weekStart, _ := args["week_start"].(string)
	accomplishments, _ := args["accomplishments"].(string)
	challenges, _ := args["challenges"].(string)
	nextWeekPlans, _ := args["next_week_plans"].(string)
	parentPageID, _ := args["parent_page_id"].(string)

	if weekStart == "" || parentPageID == "" {
		return "", fmt.Errorf("week_start and parent_page_id are required")
	}

	prompt := fmt.Sprintf(`Create a weekly report page in Notion:

**Title:** Weekly Report - Week of %s

**Structure:**
1. Heading 2: "今週の成果"
%s

2. Heading 2: "課題・問題点"
%s

3. Heading 2: "来週の予定"
%s

4. Heading 2: "メトリクス"
   - Table or callout for KPIs

**Instructions:**
Use the Notion API to:
1. Create a new page with parent_page_id: %s
2. Add the content blocks above
`, weekStart,
		formatListItems(accomplishments),
		formatListItems(challenges),
		formatListItems(nextWeekPlans),
		parentPageID)

	return prompt, nil
}

// Helper functions

func formatAttendees(attendees string) string {
	if attendees == "" {
		return "(参加者を追加)"
	}
	parts := strings.Split(attendees, ",")
	var result []string
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return strings.Join(result, ", ")
}

func conditionalLine(prefix, value string) string {
	if value == "" {
		return ""
	}
	return prefix + value + "\n"
}

func formatListItems(items string) string {
	if items == "" {
		return "   - (追加予定)"
	}
	parts := strings.Split(items, ",")
	var result []string
	for _, p := range parts {
		result = append(result, "   - "+strings.TrimSpace(p))
	}
	return strings.Join(result, "\n")
}
