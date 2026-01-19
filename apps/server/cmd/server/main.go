package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"mcpist/server/internal/auth"
	"mcpist/server/internal/mcp"
	"mcpist/server/internal/middleware"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/modules/notion"
)

func init() {
	// Load .env for local development
	// Try multiple paths for flexibility
	_ = godotenv.Load(".env")           // Current directory
	_ = godotenv.Load("../../.env")     // From cmd/server/
	_ = godotenv.Load("../../../.env")  // From apps/server/ to monorepo root

	// Register modules
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

	// Initialize auth middleware
	authMiddleware := auth.NewMiddlewareFromEnv()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Instance-ID", instanceID)
		w.Header().Set("X-Instance-Region", instanceRegion)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","instance":"%s","region":"%s"}`, instanceID, instanceRegion)
	})

	// MCP endpoint with CORS and auth middleware
	mcpHandler := mcp.NewHandler()
	http.Handle("/mcp", middleware.CORS(authMiddleware.Authenticate(mcpHandler)))

	log.Printf("Starting MCP server on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
