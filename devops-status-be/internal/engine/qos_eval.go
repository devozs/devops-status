package engine

import (
	"encoding/json"
	"strings"

	adapt "github.com/devops-status/be/internal/adapter"
	"github.com/devops-status/be/internal/store"
)

func (e *StatusEvaluator) evaluateQoSWithThresholds(samples []store.SampleRecord, ad string, qosBytes []byte) (level string, passRate float64) {
	if len(samples) == 0 {
		return "", 0
	}

	// CLI / Kubernetes shell metric QoS: same worst-band aggregation as Prometheus on RawValue (number) or cli_text_value (text).
	if (ad == "cli" || ad == "kubernetes") && len(qosBytes) > 0 {
		var cliQ struct {
			ValueKind       string  `json:"value_kind"`
			GreenOperator   string  `json:"green_operator"`
			GreenThreshold  float64 `json:"green_threshold"`
			YellowOperator  string  `json:"yellow_operator"`
			YellowThreshold float64 `json:"yellow_threshold"`
			GreenText       string  `json:"green_text"`
			YellowText      string  `json:"yellow_text"`
		}
		if json.Unmarshal(qosBytes, &cliQ) == nil {
			vk := strings.ToLower(strings.TrimSpace(cliQ.ValueKind))
			if vk == "number" && strings.TrimSpace(cliQ.GreenOperator) != "" && strings.TrimSpace(cliQ.YellowOperator) != "" {
				g, y, r := 0, 0, 0
				for _, s := range samples {
					if s.RawValue == nil {
						continue
					}
					v := *s.RawValue
					if adapt.CompareValue(v, cliQ.GreenThreshold, cliQ.GreenOperator) {
						g++
					} else if adapt.CompareValue(v, cliQ.YellowThreshold, cliQ.YellowOperator) {
						y++
					} else {
						r++
					}
				}
				n := g + y + r
				if n == 0 {
					return "red", 0
				}
				if r > 0 {
					return "red", float64(g+y) / float64(n) * 100
				}
				if y > 0 {
					return "yellow", float64(g) / float64(n) * 100
				}
				return "green", 100
			}
			if vk == "text" && strings.TrimSpace(cliQ.GreenOperator) != "" && strings.TrimSpace(cliQ.YellowOperator) != "" {
				g, y, r := 0, 0, 0
				for _, s := range samples {
					if s.MetadataJSON == nil {
						continue
					}
					tv, _ := s.MetadataJSON["cli_text_value"].(string)
					tv = strings.TrimSpace(tv)
					if tv == "" {
						continue
					}
					if adapt.CompareText(tv, cliQ.GreenText, cliQ.GreenOperator) {
						g++
					} else if adapt.CompareText(tv, cliQ.YellowText, cliQ.YellowOperator) {
						y++
					} else {
						r++
					}
				}
				n := g + y + r
				if n == 0 {
					return "red", 0
				}
				if r > 0 {
					return "red", float64(g+y) / float64(n) * 100
				}
				if y > 0 {
					return "yellow", float64(g) / float64(n) * 100
				}
				return "green", 100
			}
		}
	}

	// Liveness: per-sample worst of latency bands vs optional value bands (or success-only signal).
	if ad == "liveness" && len(qosBytes) > 0 {
		g, y, r := 0, 0, 0
		for _, s := range samples {
			if s.LatencyMs == nil {
				r++
				continue
			}
			raw := 0.0
			if s.RawValue != nil {
				raw = *s.RawValue
			}
			lvl, ok := adapt.LivenessPerSampleQoSLevel(qosBytes, *s.LatencyMs, raw, s.Success)
			if !ok {
				r++
				continue
			}
			switch lvl {
			case "green":
				g++
			case "yellow":
				y++
			default:
				r++
			}
		}
		n := g + y + r
		if n == 0 {
			return "red", 0
		}
		if r > 0 {
			return "red", float64(g+y) / float64(n) * 100
		}
		if y > 0 {
			return "yellow", float64(g) / float64(n) * 100
		}
		return "green", 100
	}

	// Legacy CLI triple probe: qos_level in metadata only.
	if len(qosBytes) > 0 {
		var ct struct {
			GreenCommand string `json:"green_command"`
			ValueKind    string `json:"value_kind"`
		}
		if json.Unmarshal(qosBytes, &ct) == nil && ct.GreenCommand != "" && strings.TrimSpace(ct.ValueKind) == "" {
			return worstQoSLevelFromMetadata(samples), 0
		}
	}

	// Latency thresholds (HTTP / Kubernetes QoS).
	var latT struct {
		GreenMaxMs  int `json:"green_max_ms"`
		YellowMaxMs int `json:"yellow_max_ms"`
		RedMaxMs    int `json:"red_max_ms"`
	}
	if len(qosBytes) > 0 && json.Unmarshal(qosBytes, &latT) == nil && latT.YellowMaxMs > 0 &&
		(ad == "http" || ad == "kubernetes") {
		g, y, r := 0, 0, 0
		for _, s := range samples {
			if s.LatencyMs == nil {
				continue
			}
			ms := *s.LatencyMs
			if ms <= latT.GreenMaxMs {
				g++
			} else if ms <= latT.YellowMaxMs {
				y++
			} else {
				r++
			}
		}
		n := g + y + r
		if n == 0 {
			return "red", 0
		}
		if r > 0 {
			return "red", float64(g+y) / float64(n) * 100
		}
		if y > 0 {
			return "yellow", float64(g) / float64(n) * 100
		}
		return "green", 100
	}

	// Prometheus metric thresholds.
	var promT struct {
		GreenOperator   string  `json:"green_operator"`
		GreenThreshold  float64 `json:"green_threshold"`
		YellowOperator  string  `json:"yellow_operator"`
		YellowThreshold float64 `json:"yellow_threshold"`
	}
	if len(qosBytes) > 0 && json.Unmarshal(qosBytes, &promT) == nil && promT.GreenOperator != "" && ad == "prometheus" {
		g, y, r := 0, 0, 0
		for _, s := range samples {
			if s.RawValue == nil {
				continue
			}
			v := *s.RawValue
			if adapt.CompareValue(v, promT.GreenThreshold, promT.GreenOperator) {
				g++
			} else if adapt.CompareValue(v, promT.YellowThreshold, promT.YellowOperator) {
				y++
			} else {
				r++
			}
		}
		n := g + y + r
		if n == 0 {
			return "red", 0
		}
		if r > 0 {
			return "red", float64(g+y) / float64(n) * 100
		}
		if y > 0 {
			return "yellow", float64(g) / float64(n) * 100
		}
		return "green", 100
	}

	// Fallback: pass rate of success bool (legacy behaviour).
	successCount := 0
	for _, s := range samples {
		if s.Success {
			successCount++
		}
	}
	passRate = float64(successCount) / float64(len(samples)) * 100
	if passRate >= 99 {
		return "green", passRate
	}
	if passRate >= 95 {
		return "yellow", passRate
	}
	return "red", passRate
}

func worstQoSLevelFromMetadata(samples []store.SampleRecord) string {
	rank := map[string]int{"green": 0, "yellow": 1, "red": 2}
	found := false
	w := -1
	worst := "green"
	for _, s := range samples {
		if s.MetadataJSON == nil {
			continue
		}
		lvl, _ := s.MetadataJSON["qos_level"].(string)
		lvl = strings.ToLower(strings.TrimSpace(lvl))
		rk, ok := rank[lvl]
		if !ok {
			continue
		}
		found = true
		if rk > w {
			w = rk
			worst = lvl
		}
	}
	if !found {
		return "red"
	}
	return worst
}
