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
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
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
		r.Get("/services/{slug}/history", historyH.ServiceHistory)
		r.Get("/environments/{slug}/history", historyH.EnvironmentHistory)
		r.Get("/incidents", incidentsH.List)
		r.Get("/incidents/{id}", incidentsH.Get)
	})

	// K8s agent registration callback (no admin session required; token-based)
	k8sH := adminhandler.NewK8sClustersHandler(s.store)
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

		// Services CRUD
		svcH := adminhandler.NewServicesHandler(s.store)
		r.Route("/services", func(r chi.Router) {
			r.Get("/", svcH.List)
			r.Post("/", svcH.Create)
			r.Get("/{id}", svcH.Get)
			r.Put("/{id}", svcH.Update)
			r.Delete("/{id}", svcH.Delete)
		})

		// Environments CRUD
		envH := adminhandler.NewEnvironmentsHandler(s.store)
		r.Route("/environments", func(r chi.Router) {
			r.Get("/", envH.List)
			r.Post("/", envH.Create)
			r.Get("/{id}", envH.Get)
			r.Put("/{id}", envH.Update)
			r.Delete("/{id}", envH.Delete)
			r.Get("/{id}/members", envH.ListMembers)
			r.Post("/{id}/members", envH.AddMember)
			r.Delete("/{id}/members/{serviceId}", envH.RemoveMember)
		})

		// Data Sources CRUD
		dsH := adminhandler.NewDataSourcesHandler(s.store)
		r.Route("/data-sources", func(r chi.Router) {
			r.Get("/", dsH.List)
			r.Post("/", dsH.Create)
			r.Get("/{id}", dsH.Get)
			r.Put("/{id}", dsH.Update)
			r.Delete("/{id}", dsH.Delete)
		})

		// Bindings
		bindH := adminhandler.NewBindingsHandler(s.store)
		r.Route("/bindings", func(r chi.Router) {
			r.Get("/services", bindH.ListServiceBindings)
			r.Post("/services/{serviceId}", bindH.CreateServiceBinding)
			r.Delete("/services/{id}", bindH.DeleteServiceBinding)
			r.Get("/environments", bindH.ListEnvironmentBindings)
			r.Post("/environments/{envId}", bindH.CreateEnvironmentBinding)
			r.Delete("/environments/{id}", bindH.DeleteEnvironmentBinding)
		})

		// K8s Clusters
		r.Route("/k8s-clusters", func(r chi.Router) {
			r.Get("/", k8sH.List)
			r.Post("/", k8sH.Create)
			r.Get("/{id}", k8sH.Get)
			r.Delete("/{id}", k8sH.Delete)
			r.Post("/{id}/onboard", k8sH.GenerateOnboardingToken)
			r.Post("/{id}/revoke", k8sH.Revoke)
		})

		// Incidents
		incAdminH := adminhandler.NewIncidentsHandler(s.store)
		r.Route("/incidents", func(r chi.Router) {
			r.Get("/", incAdminH.List)
			r.Get("/{id}", incAdminH.Get)
			r.Put("/{id}", incAdminH.Update)
		})

		// Probe Test
		probeTestH := adminhandler.NewProbeTestHandler()
		r.Post("/probes/test", probeTestH.Test)
	})

	return r
}
