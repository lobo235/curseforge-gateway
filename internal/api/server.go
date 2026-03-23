package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/lobo235/curseforge-gateway/internal/curseforge"
)

// curseforgeClient is the interface the Server uses to communicate with CurseForge.
type curseforgeClient interface {
	Ping(ctx context.Context) error
	GetModpack(ctx context.Context, projectID int) (*curseforge.Project, error)
	GetMod(ctx context.Context, projectID int) (*curseforge.Project, error)
	GetFiles(ctx context.Context, projectID int) ([]curseforge.File, error)
}

// Server holds the dependencies for the HTTP server.
type Server struct {
	curseforge curseforgeClient
	apiKey     string
	version    string
	log        *slog.Logger
}

// NewServer creates a Server wired to the given CurseForge client, API key, version string, and logger.
func NewServer(client curseforgeClient, apiKey, version string, log *slog.Logger) *Server {
	return &Server{
		curseforge: client,
		apiKey:     apiKey,
		version:    version,
		log:        log,
	}
}

// Handler builds and returns the root http.Handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	auth := bearerAuth(s.apiKey)

	// /health is unauthenticated — used by Nomad container health checks
	mux.HandleFunc("GET /health", s.healthHandler())

	// Authenticated routes
	mux.Handle("GET /modpacks/{projectID}", auth(http.HandlerFunc(s.getModpackHandler())))
	mux.Handle("GET /modpacks/{projectID}/files", auth(http.HandlerFunc(s.getModpackFilesHandler())))
	mux.Handle("GET /mods/{projectID}", auth(http.HandlerFunc(s.getModHandler())))
	mux.Handle("GET /mods/{projectID}/files", auth(http.HandlerFunc(s.getModFilesHandler())))

	return requestLogger(s.log)(mux)
}

// Run starts the HTTP server and blocks until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.log.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
