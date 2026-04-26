// Package incidentenrich builds display rows for incidents (admin and public APIs).
package incidentenrich

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/devops-status/be/internal/model"
	"github.com/devops-status/be/internal/store"
	"github.com/google/uuid"
)

// Resolution returns open | automatic | manual | unknown for UI.
func Resolution(inc model.Incident) string {
	if inc.ResolvedAt == nil {
		return "open"
	}
	switch inc.Status {
	case store.IncidentStatusAutoResolved:
		return "automatic"
	case store.IncidentStatusManuallyResolved:
		return "manual"
	}
	if inc.ResolvedBy != nil {
		switch *inc.ResolvedBy {
		case "system":
			return "automatic"
		case "admin":
			return "manual"
		}
	}
	return "unknown"
}

// QosPassRateFromDegradation extracts pass rate when degradation JSON is QoS kind.
func QosPassRateFromDegradation(deg json.RawMessage) *float64 {
	if len(deg) == 0 {
		return nil
	}
	var m struct {
		Kind     string  `json:"kind"`
		PassRate float64 `json:"pass_rate"`
	}
	if json.Unmarshal(deg, &m) != nil || m.Kind != "qos" {
		return nil
	}
	return &m.PassRate
}

// List joins services, environments, providers, clusters, and telemetry for each incident.
func List(ctx context.Context, s *store.Store, incidents []model.Incident) ([]model.AdminIncidentListItem, error) {
	svcSeen := make(map[uuid.UUID]struct{})
	envSeen := make(map[uuid.UUID]struct{})
	var svcIDs, envIDs []uuid.UUID
	for _, inc := range incidents {
		switch inc.TargetType {
		case "service":
			if _, ok := svcSeen[inc.TargetID]; !ok {
				svcSeen[inc.TargetID] = struct{}{}
				svcIDs = append(svcIDs, inc.TargetID)
			}
		case "environment":
			if _, ok := envSeen[inc.TargetID]; !ok {
				envSeen[inc.TargetID] = struct{}{}
				envIDs = append(envIDs, inc.TargetID)
			}
		}
	}

	svcs, err := s.ListServicesByIDs(ctx, svcIDs)
	if err != nil {
		return nil, err
	}
	envs, err := s.ListEnvironmentsByIDs(ctx, envIDs)
	if err != nil {
		return nil, err
	}

	var provIDs, clusterIDs []uuid.UUID
	provSeen := make(map[uuid.UUID]struct{})
	clusterSeen := make(map[uuid.UUID]struct{})
	for _, svc := range svcs {
		if svc.ServiceProviderID != nil {
			if _, ok := provSeen[*svc.ServiceProviderID]; !ok {
				provSeen[*svc.ServiceProviderID] = struct{}{}
				provIDs = append(provIDs, *svc.ServiceProviderID)
			}
		}
	}
	for _, e := range envs {
		if e.K8sClusterID != nil {
			if _, ok := clusterSeen[*e.K8sClusterID]; !ok {
				clusterSeen[*e.K8sClusterID] = struct{}{}
				clusterIDs = append(clusterIDs, *e.K8sClusterID)
			}
		}
	}

	provs, err := s.ListServiceProvidersByIDs(ctx, provIDs)
	if err != nil {
		return nil, err
	}
	clusters, err := s.ListK8sClustersByIDs(ctx, clusterIDs)
	if err != nil {
		return nil, err
	}

	linkedRows, err := s.ListLinkedTelemetryForTargets(ctx, svcIDs, envIDs)
	if err != nil {
		return nil, err
	}
	linkedByTarget := make(map[uuid.UUID][]model.AdminLinkedTelemetry)
	for _, row := range linkedRows {
		linkedByTarget[row.TargetID] = append(linkedByTarget[row.TargetID], model.AdminLinkedTelemetry{
			ID:          row.TelemetryID,
			Name:        row.Name,
			DisplayName: row.DisplayName,
			Adapter:     row.Adapter,
		})
	}
	for tid := range linkedByTarget {
		sort.Slice(linkedByTarget[tid], func(i, j int) bool {
			return linkedByTarget[tid][i].DisplayName < linkedByTarget[tid][j].DisplayName
		})
	}

	var incIDs []uuid.UUID
	for _, inc := range incidents {
		incIDs = append(incIDs, inc.ID)
	}
	hints, err := s.ListLatestQoSPassRateHints(ctx, incIDs)
	if err != nil {
		return nil, err
	}

	bookends, err := s.ListIncidentUpdateBookends(ctx, incIDs)
	if err != nil {
		return nil, err
	}

	out := make([]model.AdminIncidentListItem, 0, len(incidents))
	for _, inc := range incidents {
		item := model.AdminIncidentListItem{
			Incident:        inc,
			LinkedTelemetry: nil,
			Resolution:      Resolution(inc),
		}
		if pr := QosPassRateFromDegradation(inc.Degradation); pr != nil {
			item.QosPassRatePercent = pr
		} else if v, ok := hints[inc.ID]; ok {
			item.QosPassRatePercent = &v
		}
		switch inc.TargetType {
		case "service":
			if svc, ok := svcs[inc.TargetID]; ok {
				item.TargetName = svc.Name
				item.TargetSlug = svc.Slug
				if svc.ServiceProviderID != nil {
					if p, ok := provs[*svc.ServiceProviderID]; ok {
						item.Infra = &model.AdminIncidentInfra{
							ServiceProvider: &model.AdminInfraServiceProvider{
								ID:           p.ID,
								Name:         p.Name,
								ProviderType: p.ProviderType,
							},
						}
					}
				}
			}
			item.LinkedTelemetry = linkedByTarget[inc.TargetID]
		case "environment":
			if e, ok := envs[inc.TargetID]; ok {
				item.TargetName = e.Name
				item.TargetSlug = e.Slug
				if e.K8sClusterID != nil {
					if c, ok := clusters[*e.K8sClusterID]; ok {
						item.Infra = &model.AdminIncidentInfra{
							K8sCluster: &model.AdminInfraK8sCluster{ID: c.ID, Name: c.Name},
						}
					}
				}
			}
			item.LinkedTelemetry = linkedByTarget[inc.TargetID]
		default:
			item.TargetName = inc.TargetType
		}
		if item.LinkedTelemetry == nil {
			item.LinkedTelemetry = []model.AdminLinkedTelemetry{}
		}

		be := bookends[inc.ID]
		issue := BuildIssueMessage(inc, item, be.First, item.QosPassRatePercent)
		item.IssueMessage = &issue
		item.ResolutionMessage = BuildResolutionMessage(inc, issue, be.First, be.Last)

		out = append(out, item)
	}
	return out, nil
}
