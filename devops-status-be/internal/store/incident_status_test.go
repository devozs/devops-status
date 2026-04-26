package store

import "testing"

func TestIncidentStatusIsClosed(t *testing.T) {
	if !IncidentStatusIsClosed(IncidentStatusAutoResolved) || !IncidentStatusIsClosed(IncidentStatusManuallyResolved) {
		t.Fatal("expected closed statuses")
	}
	if IncidentStatusIsClosed("investigating") || IncidentStatusIsClosed("monitoring") {
		t.Fatal("expected open statuses")
	}
	if !IncidentStatusIsClosed("resolved") {
		t.Fatal("legacy resolved should count as closed for compatibility")
	}
}
