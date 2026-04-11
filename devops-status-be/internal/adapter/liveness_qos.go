package adapter

import (
	"encoding/json"
	"strings"
)

// LivenessQoSLatency matches qos_thresholds.latency for adapter liveness.
type LivenessQoSLatency struct {
	GreenMaxMs  int `json:"green_max_ms"`
	YellowMaxMs int `json:"yellow_max_ms"`
	RedMaxMs    int `json:"red_max_ms"`
}

// LivenessQoSValue matches qos_thresholds.value (optional); Prometheus-style bands on RawValue.
type LivenessQoSValue struct {
	GreenOperator   string  `json:"green_operator"`
	GreenThreshold  float64 `json:"green_threshold"`
	YellowOperator  string  `json:"yellow_operator"`
	YellowThreshold float64 `json:"yellow_threshold"`
}

type livenessQoSWire struct {
	Latency LivenessQoSLatency `json:"latency"`
	Value   *LivenessQoSValue  `json:"value,omitempty"`
}

func parseLivenessQoS(qosBytes []byte) (w livenessQoSWire, ok bool) {
	if len(qosBytes) == 0 || string(qosBytes) == "null" || string(qosBytes) == "{}" {
		return w, false
	}
	if json.Unmarshal(qosBytes, &w) != nil {
		return w, false
	}
	if w.Latency.YellowMaxMs <= 0 {
		return w, false
	}
	return w, true
}

func qosRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "green":
		return 0
	case "yellow":
		return 1
	case "red":
		return 2
	default:
		return 2
	}
}

func qosWorst(a, b string) string {
	if qosRank(a) >= qosRank(b) {
		return a
	}
	return b
}

func livenessLatencyLevel(ms int, lat LivenessQoSLatency) string {
	if ms <= lat.GreenMaxMs {
		return "green"
	}
	if ms <= lat.YellowMaxMs {
		return "yellow"
	}
	return "red"
}

func livenessSignalLevel(success bool, raw float64, val *LivenessQoSValue) string {
	if val == nil || strings.TrimSpace(val.GreenOperator) == "" || strings.TrimSpace(val.YellowOperator) == "" {
		if success {
			return "green"
		}
		return "red"
	}
	if CompareValue(raw, val.GreenThreshold, val.GreenOperator) {
		return "green"
	}
	if CompareValue(raw, val.YellowThreshold, val.YellowOperator) {
		return "yellow"
	}
	return "red"
}

// LivenessPerSampleQoSLevel computes green|yellow|red for one probe (worst of latency vs signal).
func LivenessPerSampleQoSLevel(qosBytes []byte, latencyMs int, rawValue float64, success bool) (level string, ok bool) {
	w, ok := parseLivenessQoS(qosBytes)
	if !ok {
		return "", false
	}
	latLvl := livenessLatencyLevel(latencyMs, w.Latency)
	sigLvl := livenessSignalLevel(success, rawValue, w.Value)
	return qosWorst(latLvl, sigLvl), true
}

// BuildSyntheticMetricQoSFromLivenessValue maps qos_thresholds.value (numeric bands) into CLI metric QoS JSON
// so RunCLIMetricShellProbe can parse stdout when liveness kubernetes uses cli_shell.
func BuildSyntheticMetricQoSFromLivenessValue(qosBytes []byte) (json.RawMessage, bool) {
	var outer struct {
		Value *LivenessQoSValue `json:"value"`
	}
	if json.Unmarshal(qosBytes, &outer) != nil || outer.Value == nil {
		return nil, false
	}
	v := outer.Value
	if strings.TrimSpace(v.GreenOperator) == "" || strings.TrimSpace(v.YellowOperator) == "" {
		return nil, false
	}
	out, err := json.Marshal(map[string]any{
		"value_kind":        "number",
		"green_operator":    v.GreenOperator,
		"green_threshold":   v.GreenThreshold,
		"yellow_operator":   v.YellowOperator,
		"yellow_threshold":  v.YellowThreshold,
	})
	if err != nil {
		return nil, false
	}
	return out, true
}
