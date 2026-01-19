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

	// Log registered modules
	log.Printf("Registered modules: %v", modules.ListModules())

	// Initialize auth middleware
	authMiddleware := auth.NewMiddlewareFromEnv()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// MCP endpoint with CORS and auth middleware
	mcpHandler := mcp.NewHandler()
	http.Handle("/mcp", middleware.CORS(authMiddleware.Authenticate(mcpHandler)))

	log.Printf("Starting MCP server on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
