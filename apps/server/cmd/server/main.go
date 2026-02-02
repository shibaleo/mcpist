package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"mcpist/server/internal/mcp"
	"mcpist/server/internal/middleware"
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
	"mcpist/server/internal/observability"
	"mcpist/server/internal/store"
)

func init() {
	// Register modules
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
	// Initialize observability (Loki)
	observability.Init()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8089"
	}

	// Instance identification for LB
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = "local"
	}
	instanceRegion := os.Getenv("INSTANCE_REGION")
	if instanceRegion == "" {
		instanceRegion = "local"
	}

	// Log registered modules
	moduleNames := modules.ListModules()
	log.Printf("Registered modules: %v", moduleNames)
	log.Printf("Instance: %s (region: %s)", instanceID, instanceRegion)

	// Sync modules to database (ensures all registered modules exist)
	moduleStore := store.NewModuleStore()
	if err := moduleStore.SyncModules(moduleNames); err != nil {
		log.Printf("Warning: Failed to sync modules to database: %v", err)
	}

	// Initialize stores and authorizer
	userStore := store.NewUserStore()
	authorizer := middleware.NewAuthorizer(userStore)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Instance-ID", instanceID)
		w.Header().Set("X-Instance-Region", instanceRegion)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","instance":"%s","region":"%s"}`, instanceID, instanceRegion)
	})

	// MCP endpoint with authorization middleware
	// Note: Authentication is handled by Cloudflare Worker, not Go Server
	// Worker sets X-User-ID and X-Gateway-Secret headers
	mcpHandler := mcp.NewHandler(userStore)
	http.Handle("/mcp", authorizer.Authorize(mcpHandler))

	log.Printf("Starting MCP server on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
