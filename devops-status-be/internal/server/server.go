package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/adapter/liveness"
	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/config"
	"github.com/devops-status/be/internal/hlctlbin"
	"github.com/devops-status/be/internal/engine"
	"github.com/devops-status/be/internal/scheduler"
	"github.com/devops-status/be/internal/secrets"
	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Server struct {
	cfg          *config.Config
	store        *store.Store
	secret       *secrets.Store
	session      *auth.SessionManager
	scheduler    *scheduler.Scheduler
	http         *http.Server
	hlctlEnabled bool
}

func New(cfg *config.Config) (*Server, error) {
	ctx := context.Background()

	db, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("initializing store: %w", err)
	}

	sec, err := secrets.NewStore(db.Pool(), cfg.SecretEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("secrets store: %w", err)
	}

	sm := auth.NewSessionManager(cfg.SessionKey, cfg.IsProd())

	hlctlEnabled := hlctlbin.Enabled()
	slog.Info("hlctl local probes", "enabled", hlctlEnabled)

	logMax := cfg.CLILogMaxBytes
	if logMax <= 0 {
		logMax = adapter.DefaultCLILogMaxBytes
	}
	imgs := adapter.CLIRunnerImages{
		Alpine:   cfg.CLIRunnerImageAlpine,
		Ubuntu24: cfg.CLIRunnerImageUbuntu,
		Toolkit:  cfg.CLIRunnerImageToolkit,
		Legacy:   cfg.CLIRunnerImage,
	}

	var backendRun adapter.BackendCLIRunner
	switch cfg.CLIBackendExecutor {
	case "k8s":
		cidStr := cfg.CLIBackendK8sCluster
		if cidStr == "" {
			backendRun = func(_ context.Context, _ string, _ adapter.CLIConfig) (*adapter.ProbeResult, error) {
				return &adapter.ProbeResult{
					Success:  false,
					ProbedAt: time.Now(),
					Error:    "CLI_BACKEND_EXECUTOR=k8s requires CLI_BACKEND_K8S_CLUSTER_ID",
				}, nil
			}
		} else {
			clusterUUID, perr := uuid.Parse(cidStr)
			if perr != nil {
				backendRun = func(_ context.Context, _ string, _ adapter.CLIConfig) (*adapter.ProbeResult, error) {
					return &adapter.ProbeResult{
						Success:  false,
						ProbedAt: time.Now(),
						Error:    "invalid CLI_BACKEND_K8S_CLUSTER_ID",
					}, nil
				}
			} else {
				backendRun = func(c context.Context, image string, cliCfg adapter.CLIConfig) (*adapter.ProbeResult, error) {
					_, token, terr := db.GetActiveClusterCredential(c, clusterUUID)
					if terr != nil {
						if errors.Is(terr, pgx.ErrNoRows) {
							return &adapter.ProbeResult{
								Success:  false,
								ProbedAt: time.Now(),
								Error:    "backend k8s cluster has no active API credential",
							}, nil
						}
						return &adapter.ProbeResult{
							Success:  false,
							ProbedAt: time.Now(),
							Error:    fmt.Sprintf("load cluster credential: %v", terr),
						}, nil
					}
					if token == "" {
						return &adapter.ProbeResult{
							Success:  false,
							ProbedAt: time.Now(),
							Error:    "backend k8s cluster token is empty",
						}, nil
					}
					cl, cerr := db.GetK8sClusterByID(c, clusterUUID)
					if cerr != nil {
						return &adapter.ProbeResult{
							Success:  false,
							ProbedAt: time.Now(),
							Error:    fmt.Sprintf("load cluster: %v", cerr),
						}, nil
					}
					if cl.Endpoint == "" {
						return &adapter.ProbeResult{
							Success:  false,
							ProbedAt: time.Now(),
							Error:    "backend k8s cluster has no endpoint",
						}, nil
					}
					return adapter.RunCLIAsK8sJob(c, cl.Endpoint, token, cfg.K8SInsecureSkipTLS, image, cliCfg, logMax)
				}
			}
		}
	default:
		dockerOpts := adapter.DockerRunOpts{Network: cfg.CLIDockerNetwork}
		backendRun = func(c context.Context, image string, cliCfg adapter.CLIConfig) (*adapter.ProbeResult, error) {
			return adapter.RunCLIAsDocker(c, image, cliCfg, dockerOpts, logMax)
		}
	}

	k8sRun := func(c context.Context, endpoint, token string, insecure bool, image string, cliCfg adapter.CLIConfig) (*adapter.ProbeResult, error) {
		return adapter.RunCLIAsK8sJob(c, endpoint, token, insecure, image, cliCfg, logMax)
	}

	adapter.Register(adapter.NewHTTPAdapter())
	adapter.Register(adapter.NewPrometheusAdapter())
	adapter.Register(adapter.NewKubernetesAdapter(imgs, k8sRun))
	adapter.Register(liveness.NewAdapter())
	adapter.Register(adapter.NewCLIAdapter([]string{"curl", "kubectl", "go", "echo"}, imgs, logMax, backendRun, k8sRun))
	adapter.Register(adapter.NewHLCTLAdapter(logMax))

	incidentMgr := engine.NewIncidentManager(db)
	statusEval := engine.NewStatusEvaluator(db, incidentMgr)
	sched := scheduler.New(db, sec, statusEval, incidentMgr, 5, cfg.K8SInsecureSkipTLS)

	s := &Server{
		cfg:          cfg,
		store:        db,
		secret:       sec,
		session:      sm,
		scheduler:    sched,
		hlctlEnabled: hlctlEnabled,
	}

	router := s.routes()

	// WriteTimeout 0: allow long SSE streams (telemetry Verify). Per-chunk deadlines use http.ResponseController.
	// IdleTimeout must exceed long-lived streaming responses; ReadTimeout still bounds the initial request read.
	s.http = &http.Server{
		Addr:         cfg.Addr(),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  2 * time.Hour,
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
