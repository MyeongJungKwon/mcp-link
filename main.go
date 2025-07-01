package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/anyisalin/mcp-openapi-to-mcp-adapter/utils"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "mcp-link",
		Usage: "Convert OpenAPI to MCP compatible endpoints",
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start the MCP Link server",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Value:   8080,
						Usage:   "Port to listen on",
					},
					&cli.StringFlag{
						Name:    "host",
						Aliases: []string{"H"},
						Value:   "0.0.0.0",
						Usage:   "Host to listen on",
					},
				},
				Action: func(c *cli.Context) error {
					return runServer(c.String("host"), c.Int("port"))
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runServer(host string, port int) error {
	// Railway의 PORT 환경변수 사용 (Railway 배포시 자동 할당)
	envPort := os.Getenv("PORT")
	if envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}

	// Create server address
	addr := fmt.Sprintf("%s:%d", host, port)

	// Configure the SSE server
	ss := utils.NewSSEServer()

	// Create a custom handler that wraps the SSE server with additional routes
	handler := &customHandler{sseServer: ss}

	// Create HTTP server with CORS middleware
	corsHandler := corsMiddleware(handler)
	server := &http.Server{
		Addr:    addr,
		Handler: corsHandler,
	}

	// Channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		fmt.Printf("Starting server on %s\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v\n", err)
		}
	}()

	// Wait for interrupt signal
	<-stop

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown the server
	fmt.Println("Shutting down server...")
	if err := ss.Shutdown(ctx); err != nil {
		log.Fatalf("Error shutting down SSE server: %v\n", err)
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Error shutting down HTTP server: %v\n", err)
	}

	fmt.Println("Server gracefully stopped")
	return nil
}

// customHandler wraps the SSE server and adds additional routes
type customHandler struct {
	sseServer http.Handler
}

func (h *customHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle /connect-api endpoint
	if r.URL.Path == "/connect-api" {
		if r.Method == http.MethodGet {
			http.ServeFile(w, r, "static/connect-api.html")
			return
		}
	}

	// Handle root path for health checks
	if r.URL.Path == "/" {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{
				"status": "healthy",
				"service": "MCP Link Server",
				"version": "1.0.0",
				"endpoints": {
					"sse": "/sse",
					"message": "/message",
					"connect-api": "/connect-api"
				},
				"description": "Convert Any OpenAPI V3 API to MCP Server"
			}`)
			return
		}
	}

	// Handle status endpoint
	if r.URL.Path == "/status" || r.URL.Path == "/health" {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status": "ok", "timestamp": "%s"}`, time.Now().UTC().Format(time.RFC3339))
			return
		}
	}

	// Delegate to SSE server for all other requests
	h.sseServer.ServeHTTP(w, r)
}

// corsMiddleware adds CORS headers to allow requests from any origin
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Pass the request to the next handler
		next.ServeHTTP(w, r)
	})
}