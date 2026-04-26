-- Split closed incident status into auto vs manual; align timeline rows.
UPDATE incidents
SET status = 'auto_resolved'
WHERE status = 'resolved' AND (resolved_by = 'system' OR resolved_by IS NULL);

UPDATE incidents
SET status = 'manually_resolved'
WHERE status = 'resolved' AND resolved_by = 'admin';

UPDATE incident_updates u
SET status = 'auto_resolved'
FROM incidents i
WHERE u.incident_id = i.id
  AND u.status = 'resolved'
  AND i.status = 'auto_resolved';

UPDATE incident_updates u
SET status = 'manually_resolved'
FROM incidents i
WHERE u.incident_id = i.id
  AND u.status = 'resolved'
  AND i.status = 'manually_resolved';

ALTER TABLE incidents
    ADD CONSTRAINT incidents_status_check CHECK (
        status IN (
            'investigating',
            'identified',
            'monitoring',
            'auto_resolved',
            'manually_resolved'
        )
    );
