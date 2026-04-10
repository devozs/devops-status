package incidentenrich

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/devops-status/be/internal/model"
	"github.com/google/uuid"
)

func TestBuildIssueMessage_firstUpdateWins(t *testing.T) {
	inc := model.Incident{Title: "ignored"}
	item := model.AdminIncidentListItem{TargetName: "API"}
	first := &model.IncidentUpdate{Message: "  We are investigating elevated errors.  "}
	got := BuildIssueMessage(inc, item, first, nil)
	if got != "We are investigating elevated errors." {
		t.Fatalf("got %q", got)
	}
}

func TestBuildIssueMessage_degradationQoS(t *testing.T) {
	deg, _ := json.Marshal(map[string]any{"kind": "qos", "pass_rate": 92.5, "qos_level": "degraded", "adapter": "http"})
	inc := model.Incident{Degradation: deg, Title: "x"}
	item := model.AdminIncidentListItem{TargetName: "Prod"}
	got := BuildIssueMessage(inc, item, nil, nil)
	want := "QoS degradation: pass rate 92.5%, level degraded (http). Affects Prod."
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildIssueMessage_qosHintFallback(t *testing.T) {
	inc := model.Incident{TargetType: "service"}
	item := model.AdminIncidentListItem{TargetName: "Edge"}
	h := 88.0
	got := BuildIssueMessage(inc, item, nil, &h)
	if !strings.Contains(got, "88.0") || !strings.Contains(got, "Edge") {
		t.Fatalf("got %q", got)
	}
}

func TestBuildResolutionMessage_open(t *testing.T) {
	inc := model.Incident{}
	got := BuildResolutionMessage(inc, "issue", nil, nil)
	if got == nil || !strings.Contains(*got, "open") {
		t.Fatalf("got %v", got)
	}
}

func TestBuildResolutionMessage_resolvedAutomatic(t *testing.T) {
	now := time.Now()
	sys := "system"
	inc := model.Incident{ResolvedAt: &now, ResolvedBy: &sys}
	first := &model.IncidentUpdate{ID: uuid.New(), Message: "Investigating"}
	last := &model.IncidentUpdate{ID: uuid.New(), Message: "Pass rate: 99.0% — recovered"}
	got := BuildResolutionMessage(inc, "Investigating", first, last)
	if got == nil {
		t.Fatal("nil")
	}
	if !strings.Contains(*got, "Pass rate") || !strings.Contains(*got, "automatically") {
		t.Fatalf("got %q", *got)
	}
}

func TestBuildResolutionMessage_resolvedDuplicateLastSkipped(t *testing.T) {
	now := time.Now()
	sys := "system"
	inc := model.Incident{ResolvedAt: &now, ResolvedBy: &sys}
	id := uuid.New()
	msg := "Same text"
	first := &model.IncidentUpdate{ID: id, Message: msg}
	last := &model.IncidentUpdate{ID: id, Message: msg}
	got := BuildResolutionMessage(inc, msg, first, last)
	if got == nil {
		t.Fatal("nil")
	}
	if strings.Count(*got, "Same text") > 0 {
		t.Fatalf("should not repeat issue text: %q", *got)
	}
	if !strings.Contains(*got, "automatically") {
		t.Fatalf("got %q", *got)
	}
}
