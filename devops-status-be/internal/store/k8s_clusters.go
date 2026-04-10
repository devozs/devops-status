package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type K8sCluster struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	EnvironmentID      *uuid.UUID `json:"environment_id,omitempty"`
	Endpoint           string     `json:"endpoint"`
	AuthMethod         string     `json:"auth_method"`
	CredentialRef      string     `json:"-"`
	DefaultNamespace   string     `json:"default_namespace"`
	NamespaceFilter    []string   `json:"namespace_filter"`
	K8sVersion         string     `json:"k8s_version"`
	Status             string     `json:"status"`
	LastCapabilityScan *time.Time `json:"last_capability_scan,omitempty"`
	APIVerifyOk        *bool      `json:"api_verify_ok"`
	APIVerifyCheckedAt *time.Time `json:"api_verify_checked_at,omitempty"`
	APIVerifyDetail    string     `json:"api_verify_detail,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type CreateK8sClusterInput struct {
	Name             string     `json:"name"`
	EnvironmentID    *uuid.UUID `json:"environment_id,omitempty"`
	Endpoint         string     `json:"endpoint"`
	AuthMethod       string     `json:"auth_method"`
	DefaultNamespace string     `json:"default_namespace"`
}

func (s *Store) ListK8sClusters(ctx context.Context) ([]K8sCluster, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, environment_id, endpoint, auth_method, default_namespace, k8s_version, status, last_capability_scan,
			api_verify_ok, api_verify_checked_at, api_verify_detail, created_at, updated_at
		 FROM k8s_clusters ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list k8s clusters: %w", err)
	}
	defer rows.Close()

	clusters := make([]K8sCluster, 0)
	for rows.Next() {
		var c K8sCluster
		var apiOk sql.NullBool
		var apiAt sql.NullTime
		var apiDetail sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.EnvironmentID, &c.Endpoint, &c.AuthMethod, &c.DefaultNamespace,
			&c.K8sVersion, &c.Status, &c.LastCapabilityScan,
			&apiOk, &apiAt, &apiDetail,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan k8s cluster: %w", err)
		}
		if apiOk.Valid {
			v := apiOk.Bool
			c.APIVerifyOk = &v
		}
		if apiAt.Valid {
			t := apiAt.Time
			c.APIVerifyCheckedAt = &t
		}
		if apiDetail.Valid {
			c.APIVerifyDetail = apiDetail.String
		}
		clusters = append(clusters, c)
	}
	return clusters, nil
}

func (s *Store) GetK8sClusterByID(ctx context.Context, id uuid.UUID) (*K8sCluster, error) {
	var c K8sCluster
	var apiOk sql.NullBool
	var apiAt sql.NullTime
	var apiDetail sql.NullString
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, environment_id, endpoint, auth_method, default_namespace, k8s_version, status, last_capability_scan,
			api_verify_ok, api_verify_checked_at, api_verify_detail, created_at, updated_at
		 FROM k8s_clusters WHERE id=$1`, id).
		Scan(&c.ID, &c.Name, &c.EnvironmentID, &c.Endpoint, &c.AuthMethod, &c.DefaultNamespace,
			&c.K8sVersion, &c.Status, &c.LastCapabilityScan,
			&apiOk, &apiAt, &apiDetail,
			&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get k8s cluster: %w", err)
	}
	if apiOk.Valid {
		v := apiOk.Bool
		c.APIVerifyOk = &v
	}
	if apiAt.Valid {
		t := apiAt.Time
		c.APIVerifyCheckedAt = &t
	}
	if apiDetail.Valid {
		c.APIVerifyDetail = apiDetail.String
	}
	return &c, nil
}

func (s *Store) CreateK8sCluster(ctx context.Context, input CreateK8sClusterInput) (*K8sCluster, error) {
	var c K8sCluster
	var apiOk sql.NullBool
	var apiAt sql.NullTime
	var apiDetail sql.NullString
	err := s.pool.QueryRow(ctx,
		`INSERT INTO k8s_clusters (name, environment_id, endpoint, auth_method, default_namespace)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, environment_id, endpoint, auth_method, default_namespace, k8s_version, status, last_capability_scan,
			api_verify_ok, api_verify_checked_at, api_verify_detail, created_at, updated_at`,
		input.Name, input.EnvironmentID, input.Endpoint, input.AuthMethod, input.DefaultNamespace).
		Scan(&c.ID, &c.Name, &c.EnvironmentID, &c.Endpoint, &c.AuthMethod, &c.DefaultNamespace,
			&c.K8sVersion, &c.Status, &c.LastCapabilityScan,
			&apiOk, &apiAt, &apiDetail,
			&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create k8s cluster: %w", err)
	}
	if apiOk.Valid {
		v := apiOk.Bool
		c.APIVerifyOk = &v
	}
	if apiAt.Valid {
		t := apiAt.Time
		c.APIVerifyCheckedAt = &t
	}
	if apiDetail.Valid {
		c.APIVerifyDetail = apiDetail.String
	}
	return &c, nil
}

func (s *Store) UpdateK8sClusterStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE k8s_clusters SET status=$1, updated_at=now() WHERE id=$2`, status, id)
	return err
}

func (s *Store) UpdateK8sClusterVersion(ctx context.Context, id uuid.UUID, version string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE k8s_clusters SET k8s_version=$1, last_capability_scan=now(), updated_at=now() WHERE id=$2`, version, id)
	return err
}

func (s *Store) DeleteK8sCluster(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM k8s_clusters WHERE id = $1`, id)
	return err
}

const maxAPIVerifyDetailLen = 2000

// UpdateK8sClusterAPIVerify persists the outcome of GET /version from the app server to the cluster API.
func (s *Store) UpdateK8sClusterAPIVerify(ctx context.Context, clusterID uuid.UUID, ok bool, detail string) error {
	detail = strings.TrimSpace(detail)
	if len(detail) > maxAPIVerifyDetailLen {
		detail = detail[:maxAPIVerifyDetailLen]
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE k8s_clusters SET api_verify_ok=$2, api_verify_checked_at=now(), api_verify_detail=$3, updated_at=now() WHERE id=$1`,
		clusterID, ok, detail)
	return err
}

// ClearK8sClusterAPIVerify resets stored API check state (e.g. after rotating the cluster token).
func (s *Store) ClearK8sClusterAPIVerify(ctx context.Context, clusterID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE k8s_clusters SET api_verify_ok=NULL, api_verify_checked_at=NULL, api_verify_detail=NULL, updated_at=now() WHERE id=$1`,
		clusterID)
	return err
}

type HandshakeToken struct {
	Token     string
	TokenHash string
	ExpiresAt time.Time
}

func (s *Store) CreateHandshakeToken(ctx context.Context, clusterID uuid.UUID, ttl time.Duration) (*HandshakeToken, error) {
	rawToken := make([]byte, 32)
	rand.Read(rawToken)
	token := hex.EncodeToString(rawToken)

	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash token: %w", err)
	}

	expiresAt := time.Now().Add(ttl)
	_, err = s.pool.Exec(ctx,
		`INSERT INTO k8s_cluster_handshakes (cluster_id, token_hash, status, expires_at)
		 VALUES ($1, $2, 'pending', $3)`,
		clusterID, string(hash), expiresAt)
	if err != nil {
		return nil, fmt.Errorf("create handshake: %w", err)
	}

	return &HandshakeToken{Token: token, TokenHash: string(hash), ExpiresAt: expiresAt}, nil
}

func (s *Store) ValidateHandshakeToken(ctx context.Context, clusterID uuid.UUID, token string) (bool, error) {
	var tokenHash string
	var expiresAt time.Time
	var hsID uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT id, token_hash, expires_at FROM k8s_cluster_handshakes
		 WHERE cluster_id=$1 AND status='pending' ORDER BY issued_at DESC LIMIT 1`, clusterID).
		Scan(&hsID, &tokenHash, &expiresAt)
	if err != nil {
		return false, fmt.Errorf("find handshake: %w", err)
	}

	if time.Now().After(expiresAt) {
		return false, nil
	}

	if bcrypt.CompareHashAndPassword([]byte(tokenHash), []byte(token)) != nil {
		return false, nil
	}

	now := time.Now()
	_, err = s.pool.Exec(ctx,
		`UPDATE k8s_cluster_handshakes SET status='registered', registered_at=$1, last_heartbeat=$1 WHERE id=$2`, now, hsID)
	if err != nil {
		return false, fmt.Errorf("update handshake: %w", err)
	}

	return true, nil
}

func (s *Store) UpdateHandshakeHeartbeat(ctx context.Context, clusterID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE k8s_cluster_handshakes SET last_heartbeat=now()
		 WHERE cluster_id=$1 AND status IN ('registered','connected') ORDER BY issued_at DESC LIMIT 1`, clusterID)
	return err
}

func (s *Store) RevokeHandshake(ctx context.Context, clusterID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE k8s_cluster_handshakes SET status='revoked', revoked_at=now()
		 WHERE cluster_id=$1 AND status IN ('pending','registered','connected')`, clusterID)
	return err
}
