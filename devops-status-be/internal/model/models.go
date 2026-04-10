package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AdminUser struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	IsActive     bool       `json:"is_active"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Environment struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	Description   string     `json:"description"`
	EnvType       string     `json:"env_type"`
	IsPublic      bool       `json:"is_public"`
	Criticality   string     `json:"criticality"`
	K8sClusterID  *uuid.UUID `json:"k8s_cluster_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Service struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Slug               string     `json:"slug"`
	Description        string     `json:"description"`
	IsPublic           bool       `json:"is_public"`
	Criticality        string     `json:"criticality"`
	ServiceProviderID  *uuid.UUID `json:"service_provider_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type ServiceProvider struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	ImageURL     string    `json:"image_url"`
	ProviderType string    `json:"provider_type"`
	ConfigJSON   any       `json:"config_json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Telemetry struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	DisplayName     string    `json:"display_name,omitempty"`
	Adapter         string    `json:"adapter"`
	ConfigJSON      any       `json:"config_json"`
	QosThresholds   any       `json:"qos_thresholds,omitempty"`
	ExecutionTarget string    `json:"execution_target"`
	SecretRef       string    `json:"-"`
	TimeoutMs       int       `json:"timeout_ms"`
	Retries         int       `json:"retries"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Incident struct {
	ID                uuid.UUID       `json:"id"`
	TargetType        string          `json:"target_type"`
	TargetID          uuid.UUID       `json:"target_id"`
	Title             string          `json:"title"`
	Status            string          `json:"status"`
	Severity          string          `json:"severity"`
	StartedAt         time.Time       `json:"started_at"`
	ResolvedAt        *time.Time      `json:"resolved_at,omitempty"`
	SourceTelemetryID *uuid.UUID      `json:"source_telemetry_id,omitempty"`
	Degradation       json.RawMessage `json:"degradation,omitempty"`
	ResolvedBy        *string         `json:"resolved_by,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// AdminIncidentListItem is the admin incidents table row with resolved target and infra context.
type AdminIncidentListItem struct {
	Incident
	TargetName      string                 `json:"target_name"`
	TargetSlug      string                 `json:"target_slug"`
	Infra           *AdminIncidentInfra    `json:"infra,omitempty"`
	LinkedTelemetry []AdminLinkedTelemetry `json:"linked_telemetry"`
	Resolution      string                 `json:"resolution"` // open | automatic | manual | unknown
}

type AdminIncidentInfra struct {
	ServiceProvider *AdminInfraServiceProvider `json:"service_provider,omitempty"`
	K8sCluster      *AdminInfraK8sCluster      `json:"k8s_cluster,omitempty"`
}

type AdminInfraServiceProvider struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	ProviderType string    `json:"provider_type"`
}

type AdminInfraK8sCluster struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type AdminLinkedTelemetry struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Adapter     string    `json:"adapter"`
}

type IncidentUpdate struct {
	ID         uuid.UUID `json:"id"`
	IncidentID uuid.UUID `json:"incident_id"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	Author     string    `json:"author"`
	CreatedAt  time.Time `json:"created_at"`
}

type ServiceTelemetryLink struct {
	ID                           uuid.UUID `json:"id"`
	ServiceID                    uuid.UUID `json:"service_id"`
	TelemetryID                  uuid.UUID `json:"telemetry_id"`
	SampleIntervalSec            int       `json:"sample_interval_sec"`
	WindowSize                   int       `json:"window_size"`
	ConsecutiveFailuresToDown    int       `json:"consecutive_failures_to_down"`
	ConsecutiveSuccessToRecover  int       `json:"consecutive_success_to_recover"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}

type EnvironmentTelemetryLink struct {
	ID                           uuid.UUID `json:"id"`
	EnvironmentID                uuid.UUID `json:"environment_id"`
	TelemetryID                  uuid.UUID `json:"telemetry_id"`
	SampleIntervalSec            int       `json:"sample_interval_sec"`
	WindowSize                   int       `json:"window_size"`
	ConsecutiveFailuresToDown    int       `json:"consecutive_failures_to_down"`
	ConsecutiveSuccessToRecover  int       `json:"consecutive_success_to_recover"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID          uuid.UUID `json:"id"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Action      string    `json:"action"`
	EntityType  string    `json:"entity_type"`
	EntityID    *uuid.UUID `json:"entity_id,omitempty"`
	DetailsJSON any       `json:"details_json"`
	IPAddress   string    `json:"ip_address"`
	CreatedAt   time.Time `json:"created_at"`
}
