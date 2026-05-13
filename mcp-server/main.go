package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func main() {
	listenAddr := flag.String("listen", ":3666", "Listen address for SSE transport (empty string uses stdio)")
	flag.Parse()

	baseURL := os.Getenv("VIKUNJA_API_URL")
	if baseURL == "" {
		log.Fatal("VIKUNJA_API_URL environment variable is required")
	}
	token := os.Getenv("VIKUNJA_API_TOKEN")
	if token == "" {
		log.Fatal("VIKUNJA_API_TOKEN environment variable is required")
	}

	client := NewClient(baseURL, token)

	srv := mcpserver.NewMCPServer(
		"vikunja-mcp",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
	)

	registerTools(srv, &toolDeps{client: client})

	log.Printf("Vikunja MCP server starting (tools: %d registered)", countTools(srv))

	if *listenAddr == "" {
		log.Println("Using stdio transport")
		if err := mcpserver.ServeStdio(srv); err != nil {
			log.Fatalf("stdio server error: %v", err)
		}
		return
	}

	sseServer := mcpserver.NewSSEServer(
		srv,
		mcpserver.WithBaseURL("http://localhost"+*listenAddr),
	)

	mux := http.NewServeMux()
	mux.Handle("/sse", sseServer.SSEHandler())
	mux.Handle("/message", sseServer.MessageHandler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	httpServer := &http.Server{
		Addr:    *listenAddr,
		Handler: mux,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		sseServer.Shutdown(context.Background())
	}()

	log.Printf("SSE server listening on %s", *listenAddr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func countTools(srv *mcpserver.MCPServer) int {
	return len(srv.ListTools())
}
