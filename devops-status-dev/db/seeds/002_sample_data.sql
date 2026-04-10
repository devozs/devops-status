-- Sample environments
INSERT INTO environments (name, slug, description, env_type, is_public, criticality) VALUES
('Production', 'production', 'Main production environment', 'prod', true, 'critical'),
('Staging', 'staging', 'Pre-production staging environment', 'staging', true, 'standard'),
('Development', 'development', 'Internal development environment', 'dev', false, 'low')
ON CONFLICT (slug) DO NOTHING;

-- Sample services
INSERT INTO services (name, slug, description, is_public, criticality) VALUES
('Gaudi Provisioning', 'gaudi-provisioning', 'Gaudi accelerator resource provisioning via Kubernetes', true, 'critical'),
('CPU Provisioning', 'cpu-provisioning', 'CPU resource provisioning via Kubernetes', true, 'critical'),
('CLI Tool', 'cli-tool', 'Developer CLI tool for resource management', true, 'standard'),
('Monitoring Stack', 'monitoring-stack', 'Prometheus and Grafana monitoring infrastructure', true, 'standard'),
('API Gateway', 'api-gateway', 'Central API gateway for platform services', true, 'critical')
ON CONFLICT (slug) DO NOTHING;

-- Sample membership (assign services to production environment)
INSERT INTO environment_service_membership (environment_id, service_id)
SELECT e.id, s.id
FROM environments e, services s
WHERE e.slug = 'production' AND s.slug IN ('gaudi-provisioning', 'cpu-provisioning', 'cli-tool', 'monitoring-stack', 'api-gateway')
ON CONFLICT (environment_id, service_id) DO NOTHING;
