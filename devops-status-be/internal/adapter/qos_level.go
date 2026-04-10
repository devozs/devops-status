package adapter

import (
	"encoding/json"
	"strings"
)

// QoSLevelFromProbe maps probe outcome + thresholds to green|yellow|red. operationalOk must be true for latency/Prometheus rules.
func QoSLevelFromProbe(ad string, qosBytes []byte, result *ProbeResult, operationalOk bool) string {
	if result == nil {
		return ""
	}
	if len(qosBytes) == 0 || string(qosBytes) == "null" || string(qosBytes) == "{}" {
		return ""
	}
	if ad == "cli" {
		if result.Metadata != nil {
			if lvl, _ := result.Metadata["qos_level"].(string); lvl != "" {
				return strings.ToLower(lvl)
			}
		}
		return ""
	}
	if !operationalOk {
		return ""
	}
	switch ad {
	case "http", "kubernetes":
		var t struct {
			GreenMaxMs  int `json:"green_max_ms"`
			YellowMaxMs int `json:"yellow_max_ms"`
			RedMaxMs    int `json:"red_max_ms"`
		}
		if json.Unmarshal(qosBytes, &t) != nil || t.YellowMaxMs <= 0 {
			return ""
		}
		l := result.LatencyMs
		if l <= t.GreenMaxMs {
			return "green"
		}
		if l <= t.YellowMaxMs {
			return "yellow"
		}
		return "red"
	case "prometheus":
		var t struct {
			GreenOperator   string  `json:"green_operator"`
			GreenThreshold  float64 `json:"green_threshold"`
			YellowOperator  string  `json:"yellow_operator"`
			YellowThreshold float64 `json:"yellow_threshold"`
		}
		if json.Unmarshal(qosBytes, &t) != nil {
			return ""
		}
		v := result.RawValue
		if CompareValue(v, t.GreenThreshold, t.GreenOperator) {
			return "green"
		}
		if CompareValue(v, t.YellowThreshold, t.YellowOperator) {
			return "yellow"
		}
		return "red"
	default:
		return ""
	}
}

// HasCLIPQoSThresholds reports whether qos JSON uses the CLI triple-command shape (not metric value_kind).
func HasCLIPQoSThresholds(qosBytes []byte) bool {
	if len(qosBytes) == 0 || string(qosBytes) == "null" || string(qosBytes) == "{}" {
		return false
	}
	var t struct {
		GreenCommand  string `json:"green_command"`
		YellowCommand string `json:"yellow_command"`
		RedCommand    string `json:"red_command"`
		ValueKind     string `json:"value_kind"`
	}
	if json.Unmarshal(qosBytes, &t) != nil {
		return false
	}
	if strings.TrimSpace(t.ValueKind) != "" {
		return false
	}
	return t.GreenCommand != "" && t.YellowCommand != "" && t.RedCommand != ""
}
