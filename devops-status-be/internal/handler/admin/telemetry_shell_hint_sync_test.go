package admin

import (
	"testing"

	"github.com/google/uuid"
)

func TestFindShellHintSlot(t *testing.T) {
	id := uuid.New().String()
	obj := map[string]any{
		"job_env_hint_id": id,
		"env":             map[string]any{"A": "1"},
	}
	if findShellHintSlot(obj, id) != "job_env_hint_id" {
		t.Fatal()
	}
	if findShellHintSlot(obj, "other") != "" {
		t.Fatal()
	}
}

func TestApplyShellHintBodyToConfigObject(t *testing.T) {
	obj := map[string]any{}
	if err := applyShellHintBodyToConfigObject(obj, "job_env", "K=v\n"); err != nil {
		t.Fatal(err)
	}
	envVal, ok := obj["env"]
	if !ok {
		t.Fatal("missing env")
	}
	switch e := envVal.(type) {
	case map[string]string:
		if e["K"] != "v" {
			t.Fatal(e)
		}
	default:
		t.Fatalf("want map[string]string, got %T", envVal)
	}
}
