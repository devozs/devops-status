package store

import "testing"

func TestParseQoSPassRateFromUpdateMessage(t *testing.T) {
	v, ok := ParseQoSPassRateFromUpdateMessage("Quality of service degraded. Pass rate: 72.5%")
	if !ok || v != 72.5 {
		t.Fatalf("got %v %v", v, ok)
	}
	_, ok = ParseQoSPassRateFromUpdateMessage("no rate here")
	if ok {
		t.Fatal("expected no match")
	}
}
