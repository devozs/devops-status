package store

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/devops-status/be/internal/model"
)

// BuildResourceTopologyForService loads service, optional provider, and linked telemetries.
func (s *Store) BuildResourceTopologyForService(ctx context.Context, svcID uuid.UUID) (*model.ResourceTopology, error) {
	svc, err := s.GetServiceByID(ctx, svcID)
	if err != nil {
		return nil, err
	}

	var infra *model.ResourceTopologyInfra
	if svc.ServiceProviderID != nil {
		p, err := s.GetServiceProviderByID(ctx, *svc.ServiceProviderID)
		if err != nil {
			return nil, err
		}
		infra = &model.ResourceTopologyInfra{
			Type:         "service_provider",
			ID:           p.ID,
			Name:         p.Name,
			ProviderType: p.ProviderType,
		}
	}

	links, err := s.ListServiceTelemetryLinks(ctx, &svcID)
	if err != nil {
		return nil, err
	}

	rows := make([]model.ResourceTopologyTelemetryRow, 0, len(links))
	for _, link := range links {
		tel, err := s.GetTelemetryByID(ctx, link.TelemetryID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, model.ResourceTopologyTelemetryRow{
			Telemetry: model.ResourceTopologyTelemetryRef{
				ID:          tel.ID,
				Name:        tel.Name,
				DisplayName: strings.TrimSpace(tel.DisplayName),
				Adapter:     tel.Adapter,
			},
			Link: model.ResourceTopologyLinkTuning{
				SampleIntervalSec:           link.SampleIntervalSec,
				WindowSize:                  link.WindowSize,
				ConsecutiveFailuresToDown:   link.ConsecutiveFailuresToDown,
				ConsecutiveSuccessToRecover: link.ConsecutiveSuccessToRecover,
			},
		})
	}

	return &model.ResourceTopology{
		Kind: "service",
		Resource: model.ResourceTopologyResource{
			ID:   svc.ID,
			Name: svc.Name,
			Slug: svc.Slug,
		},
		Infra:       infra,
		Telemetries: rows,
	}, nil
}

// BuildResourceTopologyForEnvironment loads environment, optional cluster, and linked telemetries.
func (s *Store) BuildResourceTopologyForEnvironment(ctx context.Context, envID uuid.UUID) (*model.ResourceTopology, error) {
	env, err := s.GetEnvironmentByID(ctx, envID)
	if err != nil {
		return nil, err
	}

	var infra *model.ResourceTopologyInfra
	if env.K8sClusterID != nil {
		c, err := s.GetK8sClusterByID(ctx, *env.K8sClusterID)
		if err != nil {
			return nil, err
		}
		infra = &model.ResourceTopologyInfra{
			Type: "k8s_cluster",
			ID:   c.ID,
			Name: c.Name,
		}
	}

	links, err := s.ListEnvironmentTelemetryLinks(ctx, &envID)
	if err != nil {
		return nil, err
	}

	rows := make([]model.ResourceTopologyTelemetryRow, 0, len(links))
	for _, link := range links {
		tel, err := s.GetTelemetryByID(ctx, link.TelemetryID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, model.ResourceTopologyTelemetryRow{
			Telemetry: model.ResourceTopologyTelemetryRef{
				ID:          tel.ID,
				Name:        tel.Name,
				DisplayName: strings.TrimSpace(tel.DisplayName),
				Adapter:     tel.Adapter,
			},
			Link: model.ResourceTopologyLinkTuning{
				SampleIntervalSec:           link.SampleIntervalSec,
				WindowSize:                  link.WindowSize,
				ConsecutiveFailuresToDown:   link.ConsecutiveFailuresToDown,
				ConsecutiveSuccessToRecover: link.ConsecutiveSuccessToRecover,
			},
		})
	}

	return &model.ResourceTopology{
		Kind: "environment",
		Resource: model.ResourceTopologyResource{
			ID:   env.ID,
			Name: env.Name,
			Slug: env.Slug,
		},
		Infra:       infra,
		Telemetries: rows,
	}, nil
}
