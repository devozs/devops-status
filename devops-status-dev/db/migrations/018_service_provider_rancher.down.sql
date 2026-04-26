-- Revert to provider types without Rancher (fails if any row has provider_type = 'rancher').
ALTER TABLE service_providers DROP CONSTRAINT IF EXISTS service_providers_provider_type_check;

ALTER TABLE service_providers
    ADD CONSTRAINT service_providers_provider_type_check
        CHECK (provider_type IN (
            'tcp',
            'prometheus',
            'grafana',
            'elasticsearch',
            'jenkins',
            'artifactory',
            'dns'
        ));
