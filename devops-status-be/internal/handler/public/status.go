package public

import (
	"net/http"

	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/store"
)

type StatusHandler struct {
	store *store.Store
}

func NewStatusHandler(s *store.Store) *StatusHandler {
	return &StatusHandler{store: s}
}

type serviceStatus struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Slug        string             `json:"slug"`
	Description string             `json:"description"`
	Status      string             `json:"status"`
	UptimePct   float64            `json:"uptime_pct"`
	Days        []store.DailyRollup `json:"days"`
}

type envStatus struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Slug        string             `json:"slug"`
	Description string             `json:"description"`
	EnvType     string             `json:"env_type"`
	Status      string             `json:"status"`
	UptimePct   float64            `json:"uptime_pct"`
	Days        []store.DailyRollup `json:"days"`
}

func (h *StatusHandler) Summary(w http.ResponseWriter, r *http.Request) {
	services, err := h.store.ListServices(r.Context(), true)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load services")
		return
	}

	environments, err := h.store.ListEnvironments(r.Context(), true)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load environments")
		return
	}

	overallStatus := "operational"
	overallMessage := "All Systems Operational"

	var svcStatuses []serviceStatus
	for _, svc := range services {
		rollups, _ := h.store.GetDailyRollups(r.Context(), "service", svc.ID, 90)
		status := "operational"
		uptimePct := 100.0

		if len(rollups) > 0 {
			total := 0.0
			for _, d := range rollups {
				total += d.AvailabilityPct
			}
			uptimePct = total / float64(len(rollups))
		}

		recent, _ := h.store.GetRecentSamples(r.Context(), "service", svc.ID, "operational", 3)
		if len(recent) > 0 && !recent[0].Success {
			status = "disruption"
			overallStatus = "disruption"
			overallMessage = "Disruption with some DevOps services"
		}

		svcStatuses = append(svcStatuses, serviceStatus{
			ID: svc.ID.String(), Name: svc.Name, Slug: svc.Slug,
			Description: svc.Description, Status: status, UptimePct: uptimePct, Days: rollups,
		})
	}

	var envStatuses []envStatus
	for _, env := range environments {
		rollups, _ := h.store.GetDailyRollups(r.Context(), "environment", env.ID, 90)
		status := "operational"
		uptimePct := 100.0

		if len(rollups) > 0 {
			total := 0.0
			for _, d := range rollups {
				total += d.AvailabilityPct
			}
			uptimePct = total / float64(len(rollups))
		}

		recent, _ := h.store.GetRecentSamples(r.Context(), "environment", env.ID, "operational", 3)
		if len(recent) > 0 && !recent[0].Success {
			status = "disruption"
			overallStatus = "disruption"
			overallMessage = "Disruption with some DevOps services"
		}

		envStatuses = append(envStatuses, envStatus{
			ID: env.ID.String(), Name: env.Name, Slug: env.Slug,
			Description: env.Description, EnvType: env.EnvType,
			Status: status, UptimePct: uptimePct, Days: rollups,
		})
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"status":       overallStatus,
		"message":      overallMessage,
		"services":     svcStatuses,
		"environments": envStatuses,
	})
}

func (h *StatusHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.store.ListServices(r.Context(), true)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load services")
		return
	}
	handler.WriteJSON(w, http.StatusOK, services)
}

func (h *StatusHandler) ListEnvironments(w http.ResponseWriter, r *http.Request) {
	environments, err := h.store.ListEnvironments(r.Context(), true)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load environments")
		return
	}
	handler.WriteJSON(w, http.StatusOK, environments)
}
