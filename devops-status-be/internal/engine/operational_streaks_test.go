package engine

import (
	"testing"

	"github.com/devops-status/be/internal/store"
)

func TestOperationalStreaks_newestTrailingSuccesses(t *testing.T) {
	// Slice order matches GetRecentSamplesForProbe: newest first.
	samples := []store.SampleRecord{
		{Success: true},
		{Success: true},
		{Success: false},
	}
	cf, cs := operationalStreaks(samples)
	if cf != 0 || cs != 2 {
		t.Fatalf("got cf=%d cs=%d want cf=0 cs=2", cf, cs)
	}
}

func TestOperationalStreaks_newestTrailingFails(t *testing.T) {
	samples := []store.SampleRecord{
		{Success: false},
		{Success: false},
		{Success: true},
	}
	cf, cs := operationalStreaks(samples)
	if cf != 2 || cs != 0 {
		t.Fatalf("got cf=%d cs=%d want cf=2 cs=0", cf, cs)
	}
}

func TestHasQoSBytes(t *testing.T) {
	if !hasQoSBytes([]byte(`{"green_max_ms":1}`)) {
		t.Fatal("expected qos bytes")
	}
	if hasQoSBytes([]byte(`{}`)) || hasQoSBytes(nil) || hasQoSBytes([]byte("null")) {
		t.Fatal("expected empty qos")
	}
}
