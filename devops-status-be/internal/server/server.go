package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/config"
	"github.com/devops-status/be/internal/engine"
	"github.com/devops-status/be/internal/scheduler"
	"github.com/devops-status/be/internal/store"
)

type Server struct {
	cfg       *config.Config
	store     *store.Store
	session   *auth.SessionManager
	scheduler *scheduler.Scheduler
	http      *http.Server
}

func New(cfg *config.Config) (*Server, error) {
	ctx := context.Background()

	db, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("initializing store: %w", err)
	}

	sm := auth.NewSessionManager(cfg.SessionKey)

	adapter.Register(adapter.NewHTTPAdapter())
	adapter.Register(adapter.NewPrometheusAdapter())
	adapter.Register(adapter.NewKubernetesAdapter())
	adapter.Register(adapter.NewCLIAdapter([]string{"curl", "kubectl", "go"}))

	incidentMgr := engine.NewIncidentManager(db)
	statusEval := engine.NewStatusEvaluator(db, incidentMgr)
	sched := scheduler.New(db, statusEval, incidentMgr, 5)

	s := &Server{
		cfg:       cfg,
		store:     db,
		session:   sm,
		scheduler: sched,
	}

	router := s.routes()

	s.http = &http.Server{
		Addr:         cfg.Addr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

func (s *Server) Run() error {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.scheduler.Start(ctx)

	go func() {
		slog.Info("server starting", "addr", s.http.Addr)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down...")

	cancel()
	s.scheduler.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := s.http.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	s.store.Close()
	slog.Info("server stopped")
	return nil
}
