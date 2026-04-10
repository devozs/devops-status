package incidentenrich

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/devops-status/be/internal/model"
)

// BuildIssueMessage prefers the first incident update, then degradation / QoS context, then title.
func BuildIssueMessage(inc model.Incident, item model.AdminIncidentListItem, first *model.IncidentUpdate, qosHint *float64) string {
	if first != nil {
		if m := strings.TrimSpace(first.Message); m != "" {
			return m
		}
	}
	if line := degradationHumanLine(inc.Degradation); line != "" {
		if item.TargetName != "" {
			return line + " Affects " + item.TargetName + "."
		}
		return line
	}
	if qosHint != nil {
		tgt := item.TargetName
		if tgt == "" {
			tgt = inc.TargetType
		}
		return "Pass rate approximately " + strconv.FormatFloat(*qosHint, 'f', 1, 64) +
			"% (from health checks). Affects " + tgt + "."
	}
	if t := strings.TrimSpace(inc.Title); t != "" {
		return t
	}
	return "An incident was recorded for this target."
}

// BuildResolutionMessage describes how the incident ended, optionally including the latest update text.
func BuildResolutionMessage(inc model.Incident, issueText string, first, last *model.IncidentUpdate) *string {
	if inc.ResolvedAt == nil {
		s := "This incident is still open. Updates will be posted as the situation changes."
		return &s
	}
	clause := resolvedHowClause(inc)
	issueTrim := strings.TrimSpace(issueText)

	var lastMsg string
	if last != nil {
		lastMsg = strings.TrimSpace(last.Message)
	}
	sameAsIssue := lastMsg != "" && lastMsg == issueTrim

	var b strings.Builder
	if lastMsg != "" && !sameAsIssue {
		b.WriteString(lastMsg)
		if !strings.HasSuffix(lastMsg, ".") && !strings.HasSuffix(lastMsg, "!") && !strings.HasSuffix(lastMsg, "?") {
			b.WriteString(".")
		}
		b.WriteString(" ")
	}
	b.WriteString(clause)
	out := strings.TrimSpace(b.String())
	if out == "" {
		return nil
	}
	return &out
}

func resolvedHowClause(inc model.Incident) string {
	switch Resolution(inc) {
	case "automatic":
		return "The incident was resolved automatically when health checks recovered."
	case "manual":
		return "The incident was resolved manually by an operator."
	case "unknown":
		return "The incident is now marked resolved."
	default:
		return "The incident is now resolved."
	}
}

func degradationHumanLine(deg json.RawMessage) string {
	if len(deg) == 0 {
		return ""
	}
	var m struct {
		Kind     string  `json:"kind"`
		PassRate float64 `json:"pass_rate"`
		QoSLevel string  `json:"qos_level"`
		Adapter  string  `json:"adapter"`
	}
	if json.Unmarshal(deg, &m) != nil {
		return ""
	}
	switch m.Kind {
	case "qos":
		ad := strings.TrimSpace(m.Adapter)
		if ad == "" {
			ad = "probe"
		}
		lvl := strings.TrimSpace(m.QoSLevel)
		if lvl == "" {
			lvl = "unknown"
		}
		return "QoS degradation: pass rate " + strconv.FormatFloat(m.PassRate, 'f', 1, 64) +
			"%, level " + lvl + " (" + ad + ")."
	case "operational":
		ad := strings.TrimSpace(m.Adapter)
		if ad == "" {
			ad = "telemetry"
		}
		return "Operational issue detected via " + ad + "."
	default:
		return ""
	}
}
