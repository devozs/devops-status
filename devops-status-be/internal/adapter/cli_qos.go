package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RunCLIPQoSProbe runs green, then yellow, then red CLI commands; returns first success with qos_level in metadata, or failure.
func RunCLIPQoSProbe(ctx context.Context, a Adapter, mergedCfg json.RawMessage, qosRaw json.RawMessage) (*ProbeResult, error) {
	var base CLIConfig
	if err := json.Unmarshal(mergedCfg, &base); err != nil {
		return nil, err
	}
	var q struct {
		GreenCommand  string   `json:"green_command"`
		GreenArgs     []string `json:"green_args"`
		YellowCommand string   `json:"yellow_command"`
		YellowArgs    []string `json:"yellow_args"`
		RedCommand    string   `json:"red_command"`
		RedArgs       []string `json:"red_args"`
	}
	if err := json.Unmarshal(qosRaw, &q); err != nil {
		return nil, err
	}
	start := time.Now()
	steps := []struct {
		cmd, level string
		args       []string
	}{
		{q.GreenCommand, "green", q.GreenArgs},
		{q.YellowCommand, "yellow", q.YellowArgs},
		{q.RedCommand, "red", q.RedArgs},
	}
	attempts := make([]map[string]any, 0, len(steps))
	for _, st := range steps {
		if st.cmd == "" {
			continue
		}
		sub := base
		sub.Command = st.cmd
		sub.Args = st.args
		b, mErr := json.Marshal(sub)
		att := map[string]any{
			"qos_level": st.level,
			"command":   st.cmd,
			"args":      st.args,
		}
		if mErr != nil {
			att["marshal_error"] = mErr.Error()
			attempts = append(attempts, att)
			continue
		}
		r, pErr := a.Probe(ctx, b)
		if pErr != nil {
			att["probe_error"] = pErr.Error()
			attempts = append(attempts, att)
			continue
		}
		att["success"] = r.Success
		att["latency_ms"] = r.LatencyMs
		if r.Error != "" {
			att["error"] = r.Error
		}
		if r.Trace != "" {
			att["trace"] = r.Trace
		}
		if r.Metadata != nil {
			if v, ok := r.Metadata["exit_code"]; ok {
				att["exit_code"] = v
			}
			if v, ok := r.Metadata["cli_stdout"]; ok {
				att["cli_stdout"] = v
			}
			if v, ok := r.Metadata["cli_stderr"]; ok {
				att["cli_stderr"] = v
			}
			if v, ok := r.Metadata["cli_runner"]; ok {
				att["cli_runner"] = v
			}
			if v, ok := r.Metadata["cli_image"]; ok {
				att["cli_image"] = v
			}
			if v, ok := r.Metadata["job"]; ok {
				att["job"] = v
			}
		}
		attempts = append(attempts, att)
		if r.Success {
			if r.Metadata == nil {
				r.Metadata = map[string]any{}
			}
			r.Metadata["qos_level"] = st.level
			r.Metadata["cli_qos_attempts"] = attempts
			r.LatencyMs = int(time.Since(start).Milliseconds())
			r.ProbedAt = start
			return r, nil
		}
	}
	summary := qosFailureSummary(attempts)
	return &ProbeResult{
		Success:   false,
		ProbedAt:  start,
		LatencyMs: int(time.Since(start).Milliseconds()),
		Error:     summary,
		Trace:     qosFailureTrace(attempts),
		Metadata: map[string]any{
			"qos_level":        "red",
			"cli_qos_attempts": attempts,
		},
	}, nil
}

func qosFailureSummary(attempts []map[string]any) string {
	if len(attempts) == 0 {
		return "all QoS CLI commands failed (no attempts)"
	}
	var parts []string
	for _, a := range attempts {
		lvl, _ := a["qos_level"].(string)
		if pe, ok := a["probe_error"].(string); ok && pe != "" {
			parts = append(parts, fmt.Sprintf("%s: probe error: %s", lvl, pe))
			continue
		}
		if err, ok := a["error"].(string); ok && err != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", lvl, err))
			continue
		}
		if ok, _ := a["success"].(bool); !ok {
			parts = append(parts, fmt.Sprintf("%s: failed (no error string)", lvl))
		}
	}
	if len(parts) == 0 {
		return "all QoS CLI commands failed"
	}
	return "all QoS CLI commands failed: " + strings.Join(parts, "; ")
}

func qosFailureTrace(attempts []map[string]any) string {
	var b strings.Builder
	for i, a := range attempts {
		if i > 0 {
			b.WriteString("\n\n---\n\n")
		}
		lvl, _ := a["qos_level"].(string)
		b.WriteString(fmt.Sprintf("[%s]\n", lvl))
		if t, ok := a["trace"].(string); ok && t != "" {
			b.WriteString(t)
			continue
		}
		if out, ok := a["cli_stdout"].(string); ok && out != "" {
			b.WriteString(out)
		}
		if err, ok := a["cli_stderr"].(string); ok && err != "" {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
				b.WriteString("\n")
			}
			b.WriteString("stderr: ")
			b.WriteString(err)
		}
	}
	s := b.String()
	return truncate(s, DefaultCLILogMaxBytes)
}
