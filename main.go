package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hornetSecurity/internal/handler"
	"hornetSecurity/internal/repository"
	"hornetSecurity/internal/service"
)

type app struct {
	logger *slog.Logger
	port   string
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	port := handler.GetPort()

	a := &app{logger: logger, port: port}
	if err := a.run(); err != nil {
		logger.Error("failed to run application", "error", err)
		os.Exit(1)
	}
}

func (a *app) run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repo := repository.NewInMemoryDocumentRepository()
	serv := service.NewDocumentService(repo)
	docHandler := handler.NewDocumentHandler(serv, a.logger)

	mux := http.NewServeMux()
	docHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:              ":" + a.port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		a.logger.Info("starting server", "addr", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
	}

	a.logger.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
