package incidentenrich

import (
	"encoding/json"
	"testing"
)

func TestQosPassRateFromDegradation(t *testing.T) {
	raw := json.RawMessage(`{"kind":"qos","pass_rate":72.5,"adapter":"kubernetes"}`)
	got := QosPassRateFromDegradation(raw)
	if got == nil || *got != 72.5 {
		t.Fatalf("got %#v want 72.5", got)
	}
	if QosPassRateFromDegradation(json.RawMessage(`{"kind":"operational"}`)) != nil {
		t.Fatal("operational should not yield pass rate")
	}
	if QosPassRateFromDegradation(nil) != nil {
		t.Fatal("nil should yield nil")
	}
}
