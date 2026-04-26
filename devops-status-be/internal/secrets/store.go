package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store encrypts values at rest in the secrets table.
type Store struct {
	pool *pgxpool.Pool
	key  []byte
}

func NewStore(pool *pgxpool.Pool, hexKey string) (*Store, error) {
	key, err := parseKey(hexKey)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool, key: key}, nil
}

// PrometheusCredentials is stored as JSON in a single secret row.
type PrometheusCredentials struct {
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	BearerToken string `json:"bearer_token,omitempty"`
}

// HTTPBasicCredentials is username/password JSON for service providers (Grafana, ES, Jenkins, Artifactory).
type HTTPBasicCredentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// RancherCredentials stores a Rancher API token for service provider probes.
type RancherCredentials struct {
	BearerToken string `json:"bearer_token,omitempty"`
}

func (s *Store) Write(ctx context.Context, ownerType string, ownerID uuid.UUID, key string, plaintext []byte) error {
	nonce, ct, err := encrypt(plaintext, s.key)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}
	now := time.Now()
	_, err = s.pool.Exec(ctx,
		`INSERT INTO secrets (owner_type, owner_id, key, cipher_text, nonce, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $6)
		 ON CONFLICT (owner_type, owner_id, key) DO UPDATE SET
		   cipher_text = EXCLUDED.cipher_text,
		   nonce = EXCLUDED.nonce,
		   updated_at = EXCLUDED.updated_at`,
		ownerType, ownerID, key, ct, nonce, now)
	return err
}

func (s *Store) Read(ctx context.Context, ownerType string, ownerID uuid.UUID, key string) ([]byte, error) {
	var nonce, ct []byte
	err := s.pool.QueryRow(ctx,
		`SELECT nonce, cipher_text FROM secrets WHERE owner_type=$1 AND owner_id=$2 AND key=$3`,
		ownerType, ownerID, key).Scan(&nonce, &ct)
	if err != nil {
		return nil, err
	}
	return decrypt(nonce, ct, s.key)
}

func (s *Store) Delete(ctx context.Context, ownerType string, ownerID uuid.UUID, key string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM secrets WHERE owner_type=$1 AND owner_id=$2 AND key=$3`,
		ownerType, ownerID, key)
	return err
}

func (s *Store) DeleteAllForOwner(ctx context.Context, ownerType string, ownerID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM secrets WHERE owner_type=$1 AND owner_id=$2`,
		ownerType, ownerID)
	return err
}

const OwnerTelemetry = "telemetry"
const OwnerServiceProvider = "service_provider"
const KeyPrometheusCreds = "prometheus_credentials"

const (
	KeyGrafanaCreds        = "grafana_credentials"
	KeyElasticsearchCreds  = "elasticsearch_credentials"
	KeyJenkinsCreds        = "jenkins_credentials"
	KeyArtifactoryCreds    = "artifactory_credentials"
	KeyRancherCreds        = "rancher_credentials"
	KeyHlctlCreds          = "hlctl_credentials"
)

// WritePrometheusCredentials stores username/password or bearer as JSON.
func (s *Store) WritePrometheusCredentials(ctx context.Context, telemetryID uuid.UUID, c PrometheusCredentials) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.Write(ctx, OwnerTelemetry, telemetryID, KeyPrometheusCreds, b)
}

// ReadPrometheusCredentials loads and unmarshals stored credentials.
func (s *Store) ReadPrometheusCredentials(ctx context.Context, telemetryID uuid.UUID) (PrometheusCredentials, error) {
	var out PrometheusCredentials
	b, err := s.Read(ctx, OwnerTelemetry, telemetryID, KeyPrometheusCreds)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		return out, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Store) HasPrometheusCredentials(ctx context.Context, telemetryID uuid.UUID) (bool, error) {
	var one int
	err := s.pool.QueryRow(ctx,
		`SELECT 1 FROM secrets WHERE owner_type=$1 AND owner_id=$2 AND key=$3 LIMIT 1`,
		OwnerTelemetry, telemetryID, KeyPrometheusCreds).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// WriteServiceProviderPrometheusCredentials stores Prometheus auth for a service provider.
func (s *Store) WriteServiceProviderPrometheusCredentials(ctx context.Context, providerID uuid.UUID, c PrometheusCredentials) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.Write(ctx, OwnerServiceProvider, providerID, KeyPrometheusCreds, b)
}

// ReadServiceProviderPrometheusCredentials loads credentials for verify/save merge.
func (s *Store) ReadServiceProviderPrometheusCredentials(ctx context.Context, providerID uuid.UUID) (PrometheusCredentials, error) {
	var out PrometheusCredentials
	b, err := s.Read(ctx, OwnerServiceProvider, providerID, KeyPrometheusCreds)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		return out, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Store) HasServiceProviderPrometheusCredentials(ctx context.Context, providerID uuid.UUID) (bool, error) {
	return s.HasServiceProviderSecretKey(ctx, providerID, KeyPrometheusCreds)
}

// HasServiceProviderSecretKey reports whether a secret row exists for this provider and key.
func (s *Store) HasServiceProviderSecretKey(ctx context.Context, providerID uuid.UUID, key string) (bool, error) {
	var one int
	err := s.pool.QueryRow(ctx,
		`SELECT 1 FROM secrets WHERE owner_type=$1 AND owner_id=$2 AND key=$3 LIMIT 1`,
		OwnerServiceProvider, providerID, key).Scan(&one)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// WriteServiceProviderHTTPBasic stores basic auth for a service provider integration.
func (s *Store) WriteServiceProviderHTTPBasic(ctx context.Context, providerID uuid.UUID, key string, c HTTPBasicCredentials) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.Write(ctx, OwnerServiceProvider, providerID, key, b)
}

// ReadServiceProviderHTTPBasic loads basic auth for verify/save merge.
func (s *Store) ReadServiceProviderHTTPBasic(ctx context.Context, providerID uuid.UUID, key string) (HTTPBasicCredentials, error) {
	var out HTTPBasicCredentials
	b, err := s.Read(ctx, OwnerServiceProvider, providerID, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		return out, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}

// WriteServiceProviderRancherCredentials stores a Rancher API token for a service provider.
func (s *Store) WriteServiceProviderRancherCredentials(ctx context.Context, providerID uuid.UUID, c RancherCredentials) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.Write(ctx, OwnerServiceProvider, providerID, KeyRancherCreds, b)
}

// ReadServiceProviderRancherCredentials loads the token for verify/save merge.
func (s *Store) ReadServiceProviderRancherCredentials(ctx context.Context, providerID uuid.UUID) (RancherCredentials, error) {
	var out RancherCredentials
	b, err := s.Read(ctx, OwnerServiceProvider, providerID, KeyRancherCreds)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, nil
		}
		return out, err
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return out, err
	}
	return out, nil
}
