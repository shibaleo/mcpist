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
	"mcpist/server/internal/modules/jira"
	"mcpist/server/internal/modules/notion"
	"mcpist/server/internal/modules/supabase"
)

// Service represents a service definition for services.json
// Only contains information available from Module interface (get_module_schema)
type Service struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	APIVersion  string `json:"apiVersion"`
}

// ServiceExport represents the services.json structure
type ServiceExport struct {
	Services []Service `json:"services"`
}

// ToolDef represents a tool definition for tools.json
type ToolDef struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Dangerous      bool   `json:"dangerous"`
	DefaultEnabled bool   `json:"defaultEnabled"`
}

// ModuleDef represents a module definition for tools.json
type ModuleDef struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	APIVersion  string    `json:"apiVersion"`
	Tools       []ToolDef `json:"tools"`
}

// ToolExport represents the tools.json structure
type ToolExport struct {
	Modules []ModuleDef `json:"modules"`
}

// Service display names (Module.Name() returns lowercase id)
var serviceDisplayNames = map[string]string{
	"notion":     "Notion",
	"github":     "GitHub",
	"jira":       "Jira",
	"confluence": "Confluence",
	"supabase":   "Supabase",
	"airtable":   "Airtable",
}

func init() {
	// Register all modules
	modules.RegisterModule(notion.New())
	modules.RegisterModule(github.New())
	modules.RegisterModule(jira.New())
	modules.RegisterModule(confluence.New())
	modules.RegisterModule(supabase.New())
	modules.RegisterModule(airtable.New())
}

func main() {
	outputDir := flag.String("output", "../console/src/lib", "Output directory for JSON files (default: ../console/src/lib)")
	format := flag.String("format", "both", "Output format: services, tools, or both")
	flag.Parse()

	moduleNames := modules.ListModules()
	sort.Strings(moduleNames)

	switch *format {
	case "services":
		exportServices(moduleNames, *outputDir)
	case "tools":
		exportTools(moduleNames, *outputDir)
	case "both":
		exportServices(moduleNames, *outputDir)
		exportTools(moduleNames, *outputDir)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", *format)
		os.Exit(1)
	}
}

func exportServices(moduleNames []string, outputDir string) {
	export := ServiceExport{
		Services: make([]Service, 0, len(moduleNames)),
	}

	for _, name := range moduleNames {
		m, _ := modules.GetModule(name)
		displayName := serviceDisplayNames[name]
		if displayName == "" {
			displayName = name
		}

		service := Service{
			ID:          name,
			Name:        displayName,
			Description: m.Description(),
			APIVersion:  m.APIVersion(),
		}

		export.Services = append(export.Services, service)
	}

	output, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal services: %v\n", err)
		os.Exit(1)
	}

	if outputDir == "" {
		fmt.Println(string(output))
	} else {
		path := filepath.Join(outputDir, "services.json")
		if err := os.WriteFile(path, output, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Written: %s\n", path)
	}
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
			ID:          name,
			Name:        displayName,
			Description: m.Description(),
			APIVersion:  m.APIVersion(),
			Tools:       make([]ToolDef, 0),
		}

		for _, tool := range m.Tools() {
			toolDef := ToolDef{
				ID:             tool.Name,
				Name:           tool.Name,
				Description:    tool.Description,
				Dangerous:      tool.Dangerous,
				DefaultEnabled: !tool.Dangerous, // Dangerous tools are disabled by default
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
