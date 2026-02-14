package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mcpist/server/internal/mcp"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/modules/airtable"
	"mcpist/server/internal/modules/asana"
	"mcpist/server/internal/modules/confluence"
	"mcpist/server/internal/modules/dropbox"
	"mcpist/server/internal/modules/github"
	"mcpist/server/internal/modules/grafana"
	"mcpist/server/internal/modules/google_calendar"
	"mcpist/server/internal/modules/google_docs"
	"mcpist/server/internal/modules/google_drive"
	"mcpist/server/internal/modules/google_apps_script"
	"mcpist/server/internal/modules/google_sheets"
	"mcpist/server/internal/modules/google_tasks"
	"mcpist/server/internal/modules/jira"
	"mcpist/server/internal/modules/microsoft_todo"
	"mcpist/server/internal/modules/notion"
	"mcpist/server/internal/modules/postgresql"
	"mcpist/server/internal/modules/supabase"
	"mcpist/server/internal/modules/ticktick"
	"mcpist/server/internal/modules/todoist"
	"mcpist/server/internal/modules/trello"
	"mcpist/server/internal/observability"
	"mcpist/server/internal/broker"
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
	modules.RegisterModule(google_docs.New())
	modules.RegisterModule(google_drive.New())
	modules.RegisterModule(google_sheets.New())
	modules.RegisterModule(google_apps_script.New())
	modules.RegisterModule(google_tasks.New())
	modules.RegisterModule(microsoft_todo.New())
	modules.RegisterModule(postgresql.New())
	modules.RegisterModule(ticktick.New())
	modules.RegisterModule(todoist.New())
	modules.RegisterModule(trello.New())
	modules.RegisterModule(asana.New())
	modules.RegisterModule(grafana.New())
	modules.RegisterModule(dropbox.New())
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
	moduleStore := broker.NewModuleStore()
	if err := moduleStore.SyncModules(moduleNames); err != nil {
		log.Printf("Warning: Failed to sync modules to database: %v", err)
	}

	// Initialize stores and authorizer
	userStore := broker.NewUserStore()
	authorizer := middleware.NewAuthorizer(userStore)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Instance-ID", instanceID)
		w.Header().Set("X-Instance-Region", instanceRegion)

		dbStatus := "ok"
		if err := userStore.HealthCheck(); err != nil {
			dbStatus = "unavailable"
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"degraded","instance":"%s","region":"%s","db":"%s"}`, instanceID, instanceRegion, dbStatus)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","instance":"%s","region":"%s","db":"%s"}`, instanceID, instanceRegion, dbStatus)
	})

	// MCP endpoint with authorization + rate limit middleware
	// Note: Authentication is handled by Cloudflare Worker, not Go Server
	// Worker sets X-User-ID and X-Gateway-Secret headers
	// Chain: Recovery → Authorize → RateLimit(10 req/sec per user) → MCPHandler
	rateLimiter := middleware.NewRateLimiter(10)
	mcpHandler := mcp.NewHandler(userStore)
	http.Handle("/mcp", middleware.Recovery(authorizer.Authorize(rateLimiter.Middleware(mcpHandler))))

	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", port),
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting MCP server on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal %s, shutting down gracefully...", sig)

	// Give in-flight requests up to 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Printf("Server stopped")
}
