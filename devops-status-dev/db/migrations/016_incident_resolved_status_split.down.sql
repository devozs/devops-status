ALTER TABLE incidents DROP CONSTRAINT IF EXISTS incidents_status_check;

UPDATE incidents
SET status = 'resolved'
WHERE status IN ('auto_resolved', 'manually_resolved');

UPDATE incident_updates
SET status = 'resolved'
WHERE status IN ('auto_resolved', 'manually_resolved');
