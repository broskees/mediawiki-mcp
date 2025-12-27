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

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/yourusername/mediawiki-mcp/config"
	mcpServer "github.com/yourusername/mediawiki-mcp/internal/mcp"
)

func main() {
	// Load configuration
	cfg := config.Load()

	log.Printf("Starting MediaWiki MCP Server v1.0.0")
	log.Printf("Config: Port=%s, RateLimit=%.1f req/s, CacheTTL=%s",
		cfg.Port, cfg.RateLimit, cfg.CacheTTL)

	// Create MCP server
	server := mcpServer.NewServer(cfg)
	mcpSrv := server.GetMCPServer()

	// Create Streamable HTTP handler with stateless JSON responses
	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server {
			return mcpSrv
		},
		&mcp.StreamableHTTPOptions{
			Stateless:    true, // No session validation required
			JSONResponse: true, // Return application/json instead of text/event-stream
		},
	)

	// Register routes
	http.Handle("/mcp", handler)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Info endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "MediaWiki MCP Server v1.0.0\n")
		fmt.Fprintf(w, "MCP endpoint: /mcp\n")
		fmt.Fprintf(w, "Health check: /health\n")
	})

	// Start HTTP server
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	log.Printf("Server listening on :%s", cfg.Port)
	log.Printf("MCP endpoint: http://localhost:%s/mcp", cfg.Port)
	log.Printf("Health check: http://localhost:%s/health", cfg.Port)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server stopped")
}
