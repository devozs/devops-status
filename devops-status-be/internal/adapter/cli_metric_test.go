package adapter

import (
	"math"
	"testing"
)

func TestParseCLINumericOutFirstLast(t *testing.T) {
	s := "noise 1 noise 2.6"
	v, ok, src := parseCLINumericOut(s, false, "first")
	if !ok || src != "first_number" || math.Abs(v-1) > 1e-9 {
		t.Fatalf("first: got (%v, %v, %q), want (1, true, first_number)", v, ok, src)
	}
	v, ok, src = parseCLINumericOut(s, false, "last")
	if !ok || src != "last_number" || math.Abs(v-2.6) > 1e-9 {
		t.Fatalf("last: got (%v, %v, %q), want (2.6, true, last_number)", v, ok, src)
	}
	v, ok, src = parseCLINumericOut(s, false, "")
	if !ok || src != "first_number" || math.Abs(v-1) > 1e-9 {
		t.Fatalf("default: got (%v, %v, %q), want (1, true, first_number)", v, ok, src)
	}
	v, ok, src = parseCLINumericOut(s, false, "FIRST")
	if !ok || src != "first_number" || math.Abs(v-1) > 1e-9 {
		t.Fatalf("FIRST: got (%v, %v, %q), want (1, true, first_number)", v, ok, src)
	}
}

func TestParseCLINumericOutJSONIgnoresPick(t *testing.T) {
	s := `{"value": 99}`
	v, ok, src := parseCLINumericOut(s, true, "last")
	if !ok || src != "json" || math.Abs(v-99) > 1e-9 {
		t.Fatalf("json: got (%v, %v, %q), want (99, true, json)", v, ok, src)
	}
}
