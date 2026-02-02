package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"mcpist/server/internal/modules"
	"mcpist/server/internal/modules/airtable"
	"mcpist/server/internal/modules/confluence"
	"mcpist/server/internal/modules/github"
	"mcpist/server/internal/modules/google_calendar"
	"mcpist/server/internal/modules/google_tasks"
	"mcpist/server/internal/modules/jira"
	"mcpist/server/internal/modules/microsoft_todo"
	"mcpist/server/internal/modules/notion"
	"mcpist/server/internal/modules/supabase"
	"mcpist/server/internal/modules/todoist"
	"mcpist/server/internal/modules/trello"
)

// ToolAnnotations mirrors modules.ToolAnnotations for JSON export
type ToolAnnotations struct {
	ReadOnlyHint    *bool `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool `json:"openWorldHint,omitempty"`
}

// ToolDef represents a tool definition for tools.json
type ToolDef struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Descriptions map[string]string `json:"descriptions"`
	Annotations  *ToolAnnotations  `json:"annotations,omitempty"`
}

// ModuleDef represents a module definition for tools.json
type ModuleDef struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Descriptions map[string]string `json:"descriptions"`
	APIVersion   string            `json:"apiVersion"`
	Tools        []ToolDef         `json:"tools"`
}

// ToolExport represents the tools.json structure
type ToolExport struct {
	Modules []ModuleDef `json:"modules"`
}

// Service display names (Module.Name() returns lowercase id)
var serviceDisplayNames = map[string]string{
	"notion":          "Notion",
	"github":          "GitHub",
	"jira":            "Jira",
	"confluence":      "Confluence",
	"supabase":        "Supabase",
	"airtable":        "Airtable",
	"google_calendar": "Google Calendar",
	"google_tasks":    "Google Tasks",
	"microsoft_todo":  "Microsoft To Do",
	"todoist":         "Todoist",
	"trello":          "Trello",
}

func init() {
	// Register all modules
	modules.RegisterModule(notion.New())
	modules.RegisterModule(github.New())
	modules.RegisterModule(jira.New())
	modules.RegisterModule(confluence.New())
	modules.RegisterModule(supabase.New())
	modules.RegisterModule(airtable.New())
	modules.RegisterModule(google_calendar.New())
	modules.RegisterModule(google_tasks.New())
	modules.RegisterModule(microsoft_todo.New())
	modules.RegisterModule(todoist.New())
	modules.RegisterModule(trello.New())
}

func main() {
	outputDir := flag.String("output", "../console/src/lib", "Output directory for JSON files (default: ../console/src/lib)")
	flag.Parse()

	moduleNames := modules.ListModules()
	sort.Strings(moduleNames)

	exportTools(moduleNames, *outputDir)
}

func exportTools(moduleNames []string, outputDir string) {
	export := ToolExport{
		Modules: make([]ModuleDef, 0, len(moduleNames)),
	}

	for _, name := range moduleNames {
		m, _ := modules.GetModule(name)
		displayName := serviceDisplayNames[name]
		if displayName == "" {
			displayName = name
		}

		moduleDef := ModuleDef{
			ID:           name,
			Name:         displayName,
			Descriptions: m.Descriptions(),
			APIVersion:   m.APIVersion(),
			Tools:        make([]ToolDef, 0),
		}

		for _, tool := range m.Tools() {
			toolDef := ToolDef{
				ID:           tool.ID,
				Name:         tool.Name,
				Descriptions: tool.Descriptions,
			}
			if tool.Annotations != nil {
				toolDef.Annotations = &ToolAnnotations{
					ReadOnlyHint:    tool.Annotations.ReadOnlyHint,
					DestructiveHint: tool.Annotations.DestructiveHint,
					IdempotentHint:  tool.Annotations.IdempotentHint,
					OpenWorldHint:   tool.Annotations.OpenWorldHint,
				}
			}
			moduleDef.Tools = append(moduleDef.Tools, toolDef)
		}

		export.Modules = append(export.Modules, moduleDef)
	}

	output, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal tools: %v\n", err)
		os.Exit(1)
	}

	if outputDir == "" {
		fmt.Println(string(output))
	} else {
		path := filepath.Join(outputDir, "tools.json")
		if err := os.WriteFile(path, output, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Written: %s\n", path)
	}
}
