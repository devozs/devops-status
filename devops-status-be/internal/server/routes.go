package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	adminhandler "github.com/devops-status/be/internal/handler/admin"
	publichandler "github.com/devops-status/be/internal/handler/public"
	"github.com/devops-status/be/internal/middleware"
)

func (s *Server) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.RequestLogger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{s.cfg.FrontendURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health checks
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.store.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy"}`))
			return
		}
		w.Write([]byte(`{"status":"healthy"}`))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.store.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"ready":false}`))
			return
		}
		w.Write([]byte(`{"ready":true}`))
	})

	// Public API
	statusH := publichandler.NewStatusHandler(s.store)
	incidentsH := publichandler.NewIncidentsHandler(s.store)
	historyH := publichandler.NewHistoryHandler(s.store)

	r.Route("/api/public", func(r chi.Router) {
		r.Get("/status/summary", statusH.Summary)
		r.Get("/services", statusH.ListServices)
		r.Get("/environments", statusH.ListEnvironments)
		r.Get("/environments/{slug}/telemetry-breakdown", statusH.EnvironmentTelemetryBreakdown)
		r.Get("/services/{slug}/telemetry-breakdown", statusH.ServiceTelemetryBreakdown)
		r.Get("/services/{slug}/history", historyH.ServiceHistory)
		r.Get("/environments/{slug}/history", historyH.EnvironmentHistory)
		r.Get("/incidents", incidentsH.List)
		r.Get("/incidents/{id}", incidentsH.Get)
	})

	// K8s agent registration callback (no admin session required; token-based)
	k8sH := adminhandler.NewK8sClustersHandler(s.store, s.cfg.ExternalURL, s.cfg.IsDev(), s.cfg.K8SInsecureSkipTLS)
	r.Post("/api/admin/k8s-clusters/{id}/register", k8sH.RegisterCallback)
	r.Post("/api/admin/k8s-clusters/{id}/heartbeat", k8sH.Heartbeat)

	// Admin auth (no session required for login)
	authH := adminhandler.NewAuthHandler(s.store, s.session)
	r.Post("/api/admin/auth/login", authH.Login)

	// Admin API (session required)
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.RequireAuth(s.session))
		r.Use(middleware.RequireRole("admin"))

		if !s.cfg.IsDev() {
			r.Use(middleware.CSRFProtection)
		}

		r.Post("/auth/logout", authH.Logout)
		r.Get("/auth/me", authH.Me)

		metaH := adminhandler.NewMetaHandler(s.cfg.ExternalURL, s.cfg.IsDev())
		r.Get("/meta", metaH.Get)

		// Service providers
		spH := adminhandler.NewServiceProvidersHandler(s.store, s.secret)
		r.Route("/service-providers", func(r chi.Router) {
			r.Get("/check-name", spH.CheckNameAvailable)
			r.Post("/verify", spH.VerifyReachability)
			r.Get("/", spH.List)
			r.Post("/", spH.Create)
			r.Get("/{id}", spH.Get)
			r.Put("/{id}", spH.Update)
			r.Delete("/{id}", spH.Delete)
		})

		// Services CRUD
		svcH := adminhandler.NewServicesHandler(s.store)
		r.Route("/services", func(r chi.Router) {
			r.Get("/", svcH.List)
			r.Get("/check-slug", svcH.CheckSlugAvailable)
			r.Post("/", svcH.Create)
			r.Post("/{id}/telemetry-links", svcH.CreateTelemetryLink)
			r.Patch("/{id}/telemetry-links/{linkId}", svcH.PatchTelemetryLink)
			r.Delete("/{id}/telemetry-links/{linkId}", svcH.DeleteTelemetryLink)
			r.Get("/{id}/telemetry-samples", svcH.ListTelemetrySamples)
			r.Get("/{id}", svcH.Get)
			r.Put("/{id}", svcH.Update)
			r.Delete("/{id}", svcH.Delete)
		})

		// Environments CRUD
		envH := adminhandler.NewEnvironmentsHandler(s.store)
		r.Route("/environments", func(r chi.Router) {
			r.Get("/", envH.List)
			r.Get("/check-slug", envH.CheckSlugAvailable)
			r.Post("/", envH.Create)
			r.Post("/{id}/telemetry-links", envH.CreateTelemetryLink)
			r.Patch("/{id}/telemetry-links/{linkId}", envH.PatchTelemetryLink)
			r.Delete("/{id}/telemetry-links/{linkId}", envH.DeleteTelemetryLink)
			r.Get("/{id}/telemetry-samples", envH.ListTelemetrySamples)
			r.Get("/{id}", envH.Get)
			r.Put("/{id}", envH.Update)
			r.Delete("/{id}", envH.Delete)
		})

		telLinksSumH := adminhandler.NewTelemetryLinksSummaryHandler(s.store)
		r.Get("/telemetry-links", telLinksSumH.ListAll)

		// Telemetry CRUD
		telH := adminhandler.NewTelemetryHandler(s.store, s.secret)
		r.Route("/telemetry", func(r chi.Router) {
			r.Get("/", telH.List)
			r.Get("/check-name", telH.CheckNameAvailable)
			r.Post("/", telH.Create)
			r.Get("/{id}", telH.Get)
			r.Put("/{id}", telH.Update)
			r.Delete("/{id}", telH.Delete)
		})

		// K8s Clusters — register /{id}/… routes before /{id} so chi never treats a suffix as an {id} value.
		r.Route("/k8s-clusters", func(r chi.Router) {
			r.Get("/", k8sH.List)
			r.Post("/", k8sH.Create)
			r.Post("/{id}/onboard", k8sH.GenerateOnboardingToken)
			r.Post("/{id}/api-token", k8sH.SetClusterAPIToken)
			r.Get("/{id}/verify", k8sH.VerifyConnectivity)
			r.Post("/{id}/verify", k8sH.VerifyConnectivity)
			r.Post("/{id}/revoke", k8sH.Revoke)
			r.Get("/{id}", k8sH.Get)
			r.Delete("/{id}", k8sH.Delete)
		})

		// Incidents
		incAdminH := adminhandler.NewIncidentsHandler(s.store)
		r.Route("/incidents", func(r chi.Router) {
			r.Get("/", incAdminH.List)
			r.Get("/{id}", incAdminH.Get)
			r.Put("/{id}", incAdminH.Update)
		})

		// Probe Test
		probeTestH := adminhandler.NewProbeTestHandler(s.store, s.secret, s.cfg.K8SInsecureSkipTLS)
		r.Post("/probes/test", probeTestH.Test)
	})

	return r
}
