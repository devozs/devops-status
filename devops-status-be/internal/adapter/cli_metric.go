package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var cliFirstFloatRe = regexp.MustCompile(`[-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?`)

// HasCLIMetricQoS is true when config uses cli_shell and qos uses value_kind (number|text).
func HasCLIMetricQoS(configJSON, qosBytes []byte) bool {
	if len(configJSON) == 0 || len(qosBytes) == 0 {
		return false
	}
	var qos struct {
		ValueKind string `json:"value_kind"`
	}
	if json.Unmarshal(qosBytes, &qos) != nil || strings.TrimSpace(qos.ValueKind) == "" {
		return false
	}
	var cfg struct {
		CLIShell string `json:"cli_shell"`
	}
	if json.Unmarshal(configJSON, &cfg) != nil {
		return false
	}
	return strings.TrimSpace(cfg.CLIShell) != ""
}

// CLIMetricValueKind returns "number", "text", or "" if not a metric qos payload.
func CLIMetricValueKind(qosBytes []byte) string {
	var qos struct {
		ValueKind string `json:"value_kind"`
	}
	if json.Unmarshal(qosBytes, &qos) != nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(qos.ValueKind)) {
	case "number":
		return "number"
	case "text":
		return "text"
	default:
		return ""
	}
}

// CompareText applies eq or neq; value and ref are compared after trim.
func CompareText(value, ref, op string) bool {
	value = strings.TrimSpace(value)
	ref = strings.TrimSpace(ref)
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "eq":
		return value == ref
	case "neq":
		return value != ref
	default:
		return false
	}
}

// RunCLIMetricShellProbe runs one arbitrary-shell CLI probe and applies metric QoS (number or text).
func RunCLIMetricShellProbe(ctx context.Context, a Adapter, cfgRaw, qosRaw json.RawMessage) (*ProbeResult, error) {
	var cfg CLIConfig
	if err := json.Unmarshal(cfgRaw, &cfg); err != nil {
		return nil, fmt.Errorf("parse cli config: %w", err)
	}
	var qos struct {
		ValueKind        string  `json:"value_kind"`
		NumericValuePick string  `json:"numeric_value_pick,omitempty"`
		GreenOperator    string  `json:"green_operator"`
		GreenThreshold   float64 `json:"green_threshold"`
		YellowOperator   string  `json:"yellow_operator"`
		YellowThreshold  float64 `json:"yellow_threshold"`
		GreenText        string  `json:"green_text"`
		YellowText       string  `json:"yellow_text"`
	}
	if err := json.Unmarshal(qosRaw, &qos); err != nil {
		return nil, fmt.Errorf("parse qos: %w", err)
	}

	r, err := a.Probe(ctx, cfgRaw)
	if err != nil {
		return nil, err
	}
	if r.Metadata == nil {
		r.Metadata = map[string]any{}
	}

	stdout, _ := r.Metadata["cli_stdout"].(string)
	exitCode := -1
	switch e := r.Metadata["exit_code"].(type) {
	case int:
		exitCode = e
	case float64:
		exitCode = int(e)
	}

	exitOk := exitCode == cfg.SuccessExit

	vk := strings.ToLower(strings.TrimSpace(qos.ValueKind))
	r.Metadata["value_kind"] = vk

	switch vk {
	case "number":
		r.Metadata["metric_green_band"] = formatMetricNumericBand(qos.GreenOperator, qos.GreenThreshold)
		r.Metadata["metric_yellow_band"] = formatMetricNumericBand(qos.YellowOperator, qos.YellowThreshold)
		v, parsed, src := parseCLINumericOut(stdout, cfg.ParseJSON, qos.NumericValuePick)
		if parsed {
			r.Metadata["metric_value_source"] = src
		}
		operational := exitOk && parsed
		if operational {
			r.RawValue = v
			r.Success = true
			r.Metadata["qos_level"] = cliMetricNumberQoSLevel(qos.GreenOperator, qos.GreenThreshold, qos.YellowOperator, qos.YellowThreshold, v)
			if r.Error != "" && strings.Contains(r.Error, "exit code") {
				r.Error = ""
			}
		} else {
			r.Success = false
			r.RawValue = 0
			r.Metadata["qos_level"] = "red"
			if exitOk && !parsed {
				r.Error = "could not parse a numeric value from probe stdout"
			}
		}
	case "text":
		r.Metadata["metric_value_source"] = "text"
		r.Metadata["metric_green_band"] = formatMetricTextBand(qos.GreenOperator, qos.GreenText)
		r.Metadata["metric_yellow_band"] = formatMetricTextBand(qos.YellowOperator, qos.YellowText)
		textVal := strings.TrimSpace(stdout)
		r.Metadata["cli_text_value"] = textVal
		operational := exitOk && textVal != ""
		if operational {
			r.Success = true
			r.Metadata["qos_level"] = cliMetricTextQoSLevel(qos.GreenOperator, qos.GreenText, qos.YellowOperator, qos.YellowText, textVal)
			if r.Error != "" && strings.Contains(r.Error, "exit code") {
				r.Error = ""
			}
		} else {
			r.Success = false
			r.Metadata["qos_level"] = "red"
			if exitOk && textVal == "" {
				r.Error = "probe stdout is empty after trim (text mode requires non-empty output)"
			}
		}
	default:
		return nil, fmt.Errorf("unsupported value_kind %q", qos.ValueKind)
	}

	return r, nil
}

func formatMetricNumericBand(op string, threshold float64) string {
	return strings.TrimSpace(op) + " " + strconv.FormatFloat(threshold, 'f', -1, 64)
}

func formatMetricTextBand(op, ref string) string {
	return strings.TrimSpace(op) + " " + strconv.Quote(strings.TrimSpace(ref))
}

// parseCLINumericOut returns the parsed float, whether parsing succeeded, and when successful
// a short source tag for verify UI: "json", "first_number", or "last_number".
func parseCLINumericOut(stdout string, parseJSON bool, numericValuePick string) (float64, bool, string) {
	s := strings.TrimSpace(stdout)
	if s == "" {
		return 0, false, ""
	}
	if parseJSON {
		var st CLIStructuredOutput
		if json.Unmarshal([]byte(s), &st) == nil {
			return st.Value, true, "json"
		}
	}
	matches := cliFirstFloatRe.FindAllString(s, -1)
	if len(matches) == 0 {
		return 0, false, ""
	}
	pick := strings.ToLower(strings.TrimSpace(numericValuePick))
	m := matches[0]
	src := "first_number"
	if pick == "last" {
		m = matches[len(matches)-1]
		src = "last_number"
	}
	v, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return 0, false, ""
	}
	return v, true, src
}

func cliMetricNumberQoSLevel(gOp string, gTh float64, yOp string, yTh, v float64) string {
	if CompareValue(v, gTh, gOp) {
		return "green"
	}
	if CompareValue(v, yTh, yOp) {
		return "yellow"
	}
	return "red"
}

func cliMetricTextQoSLevel(gOp, gText, yOp, yText, value string) string {
	if CompareText(value, gText, gOp) {
		return "green"
	}
	if CompareText(value, yText, yOp) {
		return "yellow"
	}
	return "red"
}
