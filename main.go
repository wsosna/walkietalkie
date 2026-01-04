package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"walkietalkie/internal/config"
	"walkietalkie/internal/handler"
	"walkietalkie/internal/service"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg := config.Load()

	// Initialize business logic service
	svc := service.New(cfg.OpenAIAPIKey, cfg.Model)

	// Initialize chat handler
	chatHandler := handler.NewChat(svc)

	// Setup router
	mux := http.NewServeMux()
	mux.HandleFunc("/chat", chatHandler.Handle)

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/", fs)

	// Create server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Server shutting down gracefully...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}
