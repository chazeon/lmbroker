package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"lmbroker/internal/broker"
	"lmbroker/internal/config"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Initialize the logger.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Load configuration.
	cfg, err := config.Load("config.toml")
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("configuration loaded successfully", "log_level", cfg.LogLevel)

	// Create a new broker instance.
	brk := broker.New(cfg)

	// Create a new ServeMux to register our routes.
	mux := http.NewServeMux()

	// Register the health check endpoint.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	// Register Prometheus metrics handler.
	mux.Handle("/metrics", promhttp.Handler())

	// Register the main broker handlers from the plan.
	mux.HandleFunc("/v1/chat/completions", brk.HandleChatCompletions)
	mux.HandleFunc("/v1/messages", brk.HandleChatCompletions) // Anthropic format
	mux.HandleFunc("/v1/embeddings", brk.HandleEmbeddings)

	// Start the server.
	address := cfg.Server.Address()
	slog.Info("starting server", "address", address, "host", cfg.Server.Host, "port", cfg.Server.Port)
	if err := http.ListenAndServe(address, mux); err != nil {
		slog.Error("server failed to start", "error", err, "address", address)
		os.Exit(1)
	}
}
