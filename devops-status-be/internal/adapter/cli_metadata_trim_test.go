package adapter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTrimCLIProbeMetadata(t *testing.T) {
	small := map[string]any{"cli_stdout": "ok"}
	if got := TrimCLIProbeMetadata(small); len(got) != 1 {
		t.Fatalf("expected passthrough: %#v", got)
	}
	huge := map[string]any{
		"cli_qos_attempts": strings.Repeat("x", maxCLIStoredMetadataBytes+100),
	}
	got := TrimCLIProbeMetadata(huge)
	if got["cli_metadata_truncated"] != true {
		t.Fatalf("expected truncation flag: %#v", got)
	}
	if _, ok := got["cli_qos_attempts"]; ok {
		t.Fatal("expected attempts removed")
	}
	b, _ := json.Marshal(got)
	if len(b) > maxCLIStoredMetadataBytes+1024 {
		t.Fatalf("still too large: %d", len(b))
	}
}
