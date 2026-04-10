-- Service provider: optional image URL, case-insensitive unique display name.
UPDATE service_providers
SET name = LEFT(TRIM(host) || ':' || port::text, 255)
WHERE TRIM(COALESCE(name, '')) = '';

ALTER TABLE service_providers
    ADD COLUMN IF NOT EXISTS image_url VARCHAR(2000) NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS service_providers_name_lower_uidx ON service_providers (LOWER(TRIM(name)));
