package model

import (
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
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	EnvType     string    `json:"env_type"`
	IsPublic    bool      `json:"is_public"`
	Criticality string    `json:"criticality"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Service struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"is_public"`
	Criticality string    `json:"criticality"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EnvironmentServiceMembership struct {
	ID            uuid.UUID `json:"id"`
	EnvironmentID uuid.UUID `json:"environment_id"`
	ServiceID     uuid.UUID `json:"service_id"`
	CreatedAt     time.Time `json:"created_at"`
}

type DataSource struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	DSType     string    `json:"ds_type"`
	Adapter    string    `json:"adapter"`
	ConfigJSON any       `json:"config_json"`
	SecretRef  string    `json:"-"`
	TimeoutMs  int       `json:"timeout_ms"`
	Retries    int       `json:"retries"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Incident struct {
	ID         uuid.UUID  `json:"id"`
	TargetType string     `json:"target_type"`
	TargetID   uuid.UUID  `json:"target_id"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	Severity   string     `json:"severity"`
	StartedAt  time.Time  `json:"started_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type IncidentUpdate struct {
	ID         uuid.UUID `json:"id"`
	IncidentID uuid.UUID `json:"incident_id"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	Author     string    `json:"author"`
	CreatedAt  time.Time `json:"created_at"`
}

type ServiceProbeBinding struct {
	ID                           uuid.UUID `json:"id"`
	ServiceID                    uuid.UUID `json:"service_id"`
	ProbeKind                    string    `json:"probe_kind"`
	DataSourceID                 uuid.UUID `json:"data_source_id"`
	SampleIntervalSec            int       `json:"sample_interval_sec"`
	WindowSize                   int       `json:"window_size"`
	ConsecutiveFailuresToDown    int       `json:"consecutive_failures_to_down"`
	ConsecutiveSuccessToRecover  int       `json:"consecutive_success_to_recover"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}

type EnvironmentProbeBinding struct {
	ID                           uuid.UUID `json:"id"`
	EnvironmentID                uuid.UUID `json:"environment_id"`
	ProbeKind                    string    `json:"probe_kind"`
	DataSourceID                 uuid.UUID `json:"data_source_id"`
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
