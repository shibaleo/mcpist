package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"mcpist/server/internal/mcp"
	"mcpist/server/internal/modules"
	"mcpist/server/internal/modules/notion"
)

func init() {
	// Register modules
	modules.RegisterModule(notion.New())
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	// Log registered modules
	log.Printf("Registered modules: %v", modules.ListModules())

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// MCP endpoint
	http.Handle("/mcp", mcp.NewHandler())

	log.Printf("Starting MCP server on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
