package adapter

import (
	"encoding/json"
)

const maxCLIStoredMetadataBytes = 256 * 1024

// TrimCLIProbeMetadata ensures metadata_json does not exceed a safe size for Postgres rows.
func TrimCLIProbeMetadata(meta map[string]any) map[string]any {
	if meta == nil {
		return nil
	}
	b, err := json.Marshal(meta)
	if err != nil || len(b) <= maxCLIStoredMetadataBytes {
		return meta
	}
	cloned := make(map[string]any, len(meta))
	for k, v := range meta {
		cloned[k] = v
	}
	delete(cloned, "cli_qos_attempts")
	cloned["cli_metadata_truncated"] = true
	b2, err := json.Marshal(cloned)
	if err != nil || len(b2) <= maxCLIStoredMetadataBytes {
		return cloned
	}
	return map[string]any{
		"cli_metadata_truncated": true,
		"error":                  "probe metadata too large after trim",
	}
}
