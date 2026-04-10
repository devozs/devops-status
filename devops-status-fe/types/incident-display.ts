/**
 * Shared enriched incident shape from GET /api/public/incidents and /api/admin/incidents.
 */
export type {
  AdminIncidentListItem as IncidentDisplayItem,
  AdminIncidentDetailResponse as PublicIncidentDetailResponse,
  IncidentUpdate,
  IncidentDegradation,
  AdminIncidentInfra as IncidentInfra,
  AdminLinkedTelemetry as LinkedTelemetry,
} from './admin-incidents'
