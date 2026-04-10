DROP INDEX IF EXISTS service_providers_name_lower_uidx;
ALTER TABLE service_providers DROP COLUMN IF EXISTS image_url;
