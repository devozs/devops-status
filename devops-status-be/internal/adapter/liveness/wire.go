package liveness

import "encoding/json"

// ResolvedWire is produced by store.ResolveTelemetryProbeConfig for adapter "liveness".
// LivenessAdapter.Probe only accepts this shape.
type ResolvedWire struct {
	LivenessCheck string          `json:"liveness_check"`
	Config        json.RawMessage `json:"config"`
}
