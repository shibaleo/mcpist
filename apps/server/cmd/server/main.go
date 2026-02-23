package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mcpist/server/internal/auth"
	"mcpist/server/internal/broker"
	"mcpist/server/internal/db"
	"mcpist/server/internal/mcp"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/ogenserver"
	gen "mcpist/server/internal/ogenserver/gen"
	"mcpist/server/internal/modules/airtable"
	"mcpist/server/internal/modules/asana"
	"mcpist/server/internal/modules/confluence"
	"mcpist/server/internal/modules/dropbox"
	"mcpist/server/internal/modules/github"
	"mcpist/server/internal/modules/google_apps_script"
	"mcpist/server/internal/modules/google_calendar"
	"mcpist/server/internal/modules/google_docs"
	"mcpist/server/internal/modules/google_drive"
	"mcpist/server/internal/modules/google_sheets"
	"mcpist/server/internal/modules/google_tasks"
	"mcpist/server/internal/modules/grafana"
	"mcpist/server/internal/modules/jira"
	"mcpist/server/internal/modules/microsoft_todo"
	"mcpist/server/internal/modules/notion"
	"mcpist/server/internal/modules/postgresql"
	"mcpist/server/internal/modules/supabase"
	"mcpist/server/internal/modules/ticktick"
	"mcpist/server/internal/modules/todoist"
	"mcpist/server/internal/modules/trello"
	"mcpist/server/internal/observability"
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

	// Initialize database
	database := db.Open()
	log.Printf("Database connected")

	// Initialize credential encryption (panics if CREDENTIAL_ENCRYPTION_KEY not set)
	db.InitEncryptionKey()
	log.Printf("Credential encryption initialized")

	// Initialize Ed25519 signing keys for JWT API keys
	if err := auth.Init(); err != nil {
		log.Fatalf("Failed to initialize auth keys: %v", err)
	}

	// Initialize brokers with GORM DB
	broker.InitTokenBroker(database)
	userStore := broker.NewUserBroker(database)

	// Sync modules+tools to database (non-blocking: log errors but don't abort)
	syncEntries := buildSyncEntries(moduleNames)
	if err := userStore.SyncModules(syncEntries); err != nil {
		log.Printf("WARNING: SyncModules failed: %v", err)
	}

	workerJwksURL := os.Getenv("WORKER_JWKS_URL")
	if workerJwksURL == "" {
		log.Fatal("WORKER_JWKS_URL is not set. Set it via environment variable or .env.dev")
	}
	gatewayVerifier := auth.NewGatewayVerifier(workerJwksURL)
	authorizer := middleware.NewAuthorizer(userStore, database, gatewayVerifier)

	// Create router (Go 1.22+ method-aware patterns)
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
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

	// MCP endpoint with authorization + rate limit + transport middleware
	rateLimiter := middleware.NewRateLimiter(10)
	mcpHandler := mcp.NewHandler(userStore)
	mux.Handle("/v1/mcp", middleware.Recovery(authorizer.Authorize(rateLimiter.Middleware(middleware.Transport(mcpHandler)))))

	// REST endpoints (ogen-generated server)
	ogenHandler := ogenserver.NewHandler(database)
	ogenSecurity := ogenserver.NewSecurityHandler(gatewayVerifier, database)
	ogenSrv, err := gen.NewServer(ogenHandler, ogenSecurity)
	if err != nil {
		log.Fatalf("Failed to create ogen server: %v", err)
	}
	mux.Handle("/v1/", ogenSrv)

	// Stripe webhook (outside ogen — needs raw body + Stripe signature)
	mux.HandleFunc("POST /v1/stripe/webhook", ogenserver.NewStripeWebhookHandler(database))

	// JWKS endpoint (public, for API key verification)
	mux.HandleFunc("GET /.well-known/jwks.json", handleJWKS)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: mux,
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

// buildSyncEntries collects module+tool data from the Go registry for DB sync.
func buildSyncEntries(moduleNames []string) []broker.SyncModuleEntry {
	type syncTool struct {
		ID           string            `json:"id"`
		Name         string            `json:"name"`
		Descriptions map[string]string `json:"descriptions,omitempty"`
		Annotations  interface{}       `json:"annotations,omitempty"`
	}

	entries := make([]broker.SyncModuleEntry, 0, len(moduleNames))
	for _, name := range moduleNames {
		m, ok := modules.GetModule(name)
		if !ok {
			continue
		}

		tools := m.Tools()
		syncTools := make([]syncTool, 0, len(tools))
		for _, t := range tools {
			syncTools = append(syncTools, syncTool{
				ID:           t.ID,
				Name:         t.Name,
				Descriptions: t.Descriptions,
				Annotations:  t.Annotations,
			})
		}

		entries = append(entries, broker.SyncModuleEntry{
			Name:         name,
			Status:       "active",
			Descriptions: m.Descriptions(),
			Tools:        syncTools,
		})
	}
	return entries
}

// handleJWKS serves the JWKS endpoint for API key verification.
func handleJWKS(w http.ResponseWriter, r *http.Request) {
	kp := auth.GetKeyPair()
	w.Header().Set("Content-Type", "application/json")
	if kp == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{}})
		return
	}
	jwk := map[string]interface{}{
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   base64.RawURLEncoding.EncodeToString(kp.PublicKey),
		"kid": kp.KID,
		"use": "sig",
		"alg": "EdDSA",
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{jwk}})
}
