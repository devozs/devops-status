package incidentenrich

import (
	"testing"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
)

func TestResolution(t *testing.T) {
	now := time.Now()
	sys := "system"
	adm := "admin"
	id := uuid.New()

	tests := []struct {
		name string
		inc  model.Incident
		want string
	}{
		{"open", model.Incident{ID: id}, "open"},
		{"automatic", model.Incident{ID: id, ResolvedAt: &now, ResolvedBy: &sys}, "automatic"},
		{"manual", model.Incident{ID: id, ResolvedAt: &now, ResolvedBy: &adm}, "manual"},
		{"automatic_by_status", model.Incident{ID: id, ResolvedAt: &now, Status: store.IncidentStatusAutoResolved}, "automatic"},
		{"manual_by_status", model.Incident{ID: id, ResolvedAt: &now, Status: store.IncidentStatusManuallyResolved}, "manual"},
		{"status_over_resolved_by", model.Incident{ID: id, ResolvedAt: &now, ResolvedBy: &adm, Status: store.IncidentStatusAutoResolved}, "automatic"},
		{"unknown", model.Incident{ID: id, ResolvedAt: &now}, "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if g := Resolution(tc.inc); g != tc.want {
				t.Fatalf("got %q want %q", g, tc.want)
			}
		})
	}
}
