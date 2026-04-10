package public

import (
	"errors"
	"net/http"
	"strings"

	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
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

type telemetryBreakdownRow struct {
	TelemetryID            string              `json:"telemetry_id"`
	Name                   string              `json:"name"`
	DisplayName            string              `json:"display_name,omitempty"`
	Status                 string              `json:"status"`
	UptimePct              float64             `json:"uptime_pct"`
	Days                   []store.DailyRollup `json:"days"`
	HasPerTelemetrySamples bool                `json:"has_per_telemetry_samples"`
}

// EnvironmentTelemetryBreakdown returns per-linked-telemetry operational rollups for a public environment (404 if missing or not public).
func (h *StatusHandler) EnvironmentTelemetryBreakdown(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	env, err := h.store.GetEnvironmentBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			handler.WriteError(w, http.StatusNotFound, "environment not found")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to load environment")
		return
	}
	if !env.IsPublic {
		handler.WriteError(w, http.StatusNotFound, "environment not found")
		return
	}

	links, err := h.store.ListEnvironmentTelemetryLinks(r.Context(), &env.ID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry links")
		return
	}

	rows := make([]telemetryBreakdownRow, 0, len(links))
	for _, link := range links {
		tel, err := h.store.GetTelemetryByID(r.Context(), link.TelemetryID)
		if err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry")
			return
		}
		recent, _ := h.store.ListRecentSamplesForEnvironmentTelemetry(r.Context(), env.ID, link.TelemetryID, "operational", 1)
		hasSamples := len(recent) > 0

		row := telemetryBreakdownRow{
			TelemetryID:            link.TelemetryID.String(),
			Name:                   tel.Name,
			HasPerTelemetrySamples: hasSamples,
			Status:                 "operational",
			UptimePct:              100,
		}
		if d := strings.TrimSpace(tel.DisplayName); d != "" {
			row.DisplayName = d
		}
		if hasSamples {
			rollups, _ := h.store.GetDailyRollupsForEnvironmentTelemetry(r.Context(), env.ID, link.TelemetryID, 90)
			row.Days = rollups
			if len(rollups) > 0 {
				var total float64
				for _, d := range rollups {
					total += d.AvailabilityPct
				}
				row.UptimePct = total / float64(len(rollups))
			}
			if len(recent) > 0 && !recent[0].Success {
				row.Status = "disruption"
			}
		}
		rows = append(rows, row)
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"environment": map[string]string{
			"id":   env.ID.String(),
			"name": env.Name,
			"slug": env.Slug,
		},
		"telemetries": rows,
	})
}

// ServiceTelemetryBreakdown returns per-linked-telemetry operational rollups for a public service (404 if missing or not public).
func (h *StatusHandler) ServiceTelemetryBreakdown(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	svc, err := h.store.GetServiceBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			handler.WriteError(w, http.StatusNotFound, "service not found")
			return
		}
		handler.WriteError(w, http.StatusInternalServerError, "failed to load service")
		return
	}
	if !svc.IsPublic {
		handler.WriteError(w, http.StatusNotFound, "service not found")
		return
	}

	links, err := h.store.ListServiceTelemetryLinks(r.Context(), &svc.ID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry links")
		return
	}

	rows := make([]telemetryBreakdownRow, 0, len(links))
	for _, link := range links {
		tel, err := h.store.GetTelemetryByID(r.Context(), link.TelemetryID)
		if err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "failed to load telemetry")
			return
		}
		recent, _ := h.store.ListRecentSamplesForServiceTelemetry(r.Context(), svc.ID, link.TelemetryID, "operational", 1)
		hasSamples := len(recent) > 0

		row := telemetryBreakdownRow{
			TelemetryID:            link.TelemetryID.String(),
			Name:                   tel.Name,
			HasPerTelemetrySamples: hasSamples,
			Status:                 "operational",
			UptimePct:              100,
		}
		if d := strings.TrimSpace(tel.DisplayName); d != "" {
			row.DisplayName = d
		}
		if hasSamples {
			rollups, _ := h.store.GetDailyRollupsForServiceTelemetry(r.Context(), svc.ID, link.TelemetryID, 90)
			row.Days = rollups
			if len(rollups) > 0 {
				var total float64
				for _, d := range rollups {
					total += d.AvailabilityPct
				}
				row.UptimePct = total / float64(len(rollups))
			}
			if len(recent) > 0 && !recent[0].Success {
				row.Status = "disruption"
			}
		}
		rows = append(rows, row)
	}

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"service": map[string]string{
			"id":   svc.ID.String(),
			"name": svc.Name,
			"slug": svc.Slug,
		},
		"telemetries": rows,
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
