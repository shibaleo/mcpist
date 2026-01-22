package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"mcpist/server/internal/entitlement"
	"mcpist/server/internal/mcp"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/modules/notion"
)

func init() {
	// Register modules
	// Environment variables are loaded via dotenv-cli in package.json scripts
	modules.RegisterModule(notion.New())
}

func main() {
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
	log.Printf("Registered modules: %v", modules.ListModules())
	log.Printf("Instance: %s (region: %s)", instanceID, instanceRegion)

	// Initialize entitlement store and authorizer
	entitlementStore := entitlement.NewStore()
	authorizer := entitlement.NewAuthorizer(entitlementStore)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Instance-ID", instanceID)
		w.Header().Set("X-Instance-Region", instanceRegion)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","instance":"%s","region":"%s"}`, instanceID, instanceRegion)
	})

	// MCP endpoint with CORS and authorization middleware
	// Note: Authentication is handled by Cloudflare Worker, not Go Server
	// Worker sets X-User-ID and X-Gateway-Secret headers
	mcpHandler := mcp.NewHandler(entitlementStore)
	http.Handle("/mcp", middleware.CORS(authorizer.Authorize(mcpHandler)))

	log.Printf("Starting MCP server on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
