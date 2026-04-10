package admin

import (
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/k8sconnect"
	"github.com/devops-status/be/internal/store"
)

type K8sClustersHandler struct {
	store            *store.Store
	externalURL      string
	isDev            bool
	k8sInsecureTLS   bool
}

func NewK8sClustersHandler(s *store.Store, externalURL string, isDev bool, k8sInsecureTLS bool) *K8sClustersHandler {
	return &K8sClustersHandler{store: s, externalURL: externalURL, isDev: isDev, k8sInsecureTLS: k8sInsecureTLS}
}

func isLocalhostURL(u string) bool {
	return strings.Contains(u, "localhost") || strings.Contains(u, "127.0.0.1")
}

type k8sClusterListItem struct {
	store.K8sCluster
	HasAPICredential bool `json:"has_api_credential"`
}

func (h *K8sClustersHandler) List(w http.ResponseWriter, r *http.Request) {
	clusters, err := h.store.ListK8sClusters(r.Context())
	if err != nil {
		slog.Error("list k8s clusters", "error", err)
		handler.WriteError(w, http.StatusInternalServerError, "failed to list clusters")
		return
	}
	out := make([]k8sClusterListItem, 0, len(clusters))
	for _, c := range clusters {
		has, _ := h.store.HasActiveClusterCredential(r.Context(), c.ID)
		out = append(out, k8sClusterListItem{K8sCluster: c, HasAPICredential: has})
	}
	handler.WriteJSON(w, http.StatusOK, out)
}

func (h *K8sClustersHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	c, err := h.store.GetK8sClusterByID(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	handler.WriteJSON(w, http.StatusOK, c)
}

func (h *K8sClustersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input store.CreateK8sClusterInput
	if err := handler.DecodeJSON(r, &input); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if input.Name == "" || input.Endpoint == "" {
		handler.WriteError(w, http.StatusBadRequest, "name and endpoint are required")
		return
	}
	if input.AuthMethod == "" {
		input.AuthMethod = "service_account_token"
	}
	if input.DefaultNamespace == "" {
		input.DefaultNamespace = "default"
	}
	c, err := h.store.CreateK8sCluster(r.Context(), input)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to create cluster")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "create", "k8s_cluster", &c.ID, input, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusCreated, c)
}

func (h *K8sClustersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.store.DeleteK8sCluster(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to delete cluster")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "delete", "k8s_cluster", &id, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *K8sClustersHandler) GenerateOnboardingToken(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	cluster, err := h.store.GetK8sClusterByID(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "cluster not found")
		return
	}

	ht, err := h.store.CreateHandshakeToken(r.Context(), id, 30*time.Minute)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	manifest := generateOnboardingManifest(cluster.Name, id.String(), ht.Token, h.externalURL)

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "generate_onboarding_token", "k8s_cluster", &id, nil, r.RemoteAddr)

	command := onboardingShellPreamble() + "kubectl apply -f - <<'EOF'\n" + manifest + "EOF"

	instructions := buildOnboardingInstructions(h.externalURL, h.isDev)

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"token":        ht.Token,
		"expires_at":   ht.ExpiresAt,
		"manifest":     manifest,
		"command":      command,
		"external_url": h.externalURL,
		"is_dev":       h.isDev,
		"instructions": instructions,
	})
}

func (h *K8sClustersHandler) RegisterCallback(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Token                 string `json:"token"`
		ClusterBearerToken    string `json:"cluster_bearer_token"`
		ClusterBearerTokenB64 string `json:"cluster_bearer_token_b64"`
	}
	if err := handler.DecodeJSON(r, &req); err != nil || req.Token == "" {
		handler.WriteError(w, http.StatusBadRequest, "token is required")
		return
	}

	valid, err := h.store.ValidateHandshakeToken(r.Context(), id, req.Token)
	if err != nil || !valid {
		handler.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	bearer := strings.TrimSpace(req.ClusterBearerToken)
	if bearer == "" && req.ClusterBearerTokenB64 != "" {
		raw, decErr := base64.StdEncoding.DecodeString(strings.TrimSpace(req.ClusterBearerTokenB64))
		if decErr != nil {
			handler.WriteError(w, http.StatusBadRequest, "invalid cluster_bearer_token_b64")
			return
		}
		bearer = string(raw)
	}
	if bearer != "" {
		if err := h.store.RotateClusterCredential(r.Context(), id, "service_account_token", bearer); err != nil {
			handler.WriteError(w, http.StatusInternalServerError, "failed to store cluster credentials")
			return
		}
	}

	_ = h.store.UpdateK8sClusterStatus(r.Context(), id, "connected")

	handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "connected"})
}

func (h *K8sClustersHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	_ = h.store.UpdateHandshakeHeartbeat(r.Context(), id)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *K8sClustersHandler) SetClusterAPIToken(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		BearerToken string `json:"bearer_token"`
	}
	if err := handler.DecodeJSON(r, &body); err != nil || body.BearerToken == "" {
		handler.WriteError(w, http.StatusBadRequest, "bearer_token is required")
		return
	}
	if _, err := h.store.GetK8sClusterByID(r.Context(), id); err != nil {
		handler.WriteError(w, http.StatusNotFound, "cluster not found")
		return
	}
	if err := h.store.RotateClusterCredential(r.Context(), id, "service_account_token", body.BearerToken); err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to store token")
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "set_cluster_api_token", "k8s_cluster", &id, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *K8sClustersHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	_ = h.store.RevokeHandshake(r.Context(), id)
	_ = h.store.RevokeClusterCredentials(r.Context(), id)
	_ = h.store.UpdateK8sClusterStatus(r.Context(), id, "revoked")

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "revoke", "k8s_cluster", &id, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// VerifyConnectivity calls the cluster Kubernetes API GET /version using the stored service-account token
// (equivalent to confirming kubectl can reach the API with that credential).
func (h *K8sClustersHandler) VerifyConnectivity(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	cluster, err := h.store.GetK8sClusterByID(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusNotFound, "cluster not found")
		return
	}
	_, token, err := h.store.GetActiveClusterCredential(r.Context(), id)
	if err != nil || token == "" {
		handler.WriteJSON(w, http.StatusOK, k8sconnect.PingResult{
			OK:      false,
			Message: "No active API token stored. Complete onboarding or use Add API token.",
		})
		return
	}

	res := k8sconnect.PingVersion(r.Context(), cluster.Endpoint, token, h.k8sInsecureTLS)
	if res.OK && res.K8sVersion != "" && cluster.K8sVersion == "" {
		_ = h.store.UpdateK8sClusterVersion(r.Context(), id, res.K8sVersion)
	}
	_ = h.store.UpdateK8sClusterAPIVerify(r.Context(), id, res.OK, res.Message)
	handler.WriteJSON(w, http.StatusOK, res)
}

// onboardingJobSuffix makes a unique, DNS-safe Job name suffix from the handshake token (immutable Job spec).
func onboardingJobSuffix(token string) string {
	if len(token) < 8 {
		return "00000000"
	}
	return strings.ToLower(token[:8])
}

// normalizeExternalURLForOnboarding trims whitespace and trailing slashes so the Job does not build //api/... URLs.
func normalizeExternalURLForOnboarding(raw string) string {
	raw = strings.TrimSpace(raw)
	return strings.TrimRight(raw, "/")
}

// onboardingShellPreamble removes the legacy fixed-name Job so re-onboarding works after a DB reset
// when the cluster still has the old manifest applied.
func onboardingShellPreamble() string {
	return `# Remove legacy fixed-name Job if it still exists (e.g. DB reset while cluster resources remain).
kubectl delete job devops-status-register -n devops-status-system --ignore-not-found

`
}

func generateOnboardingManifest(_ string, clusterID, token, externalURL string) string {
	jobName := "devops-status-register-" + onboardingJobSuffix(token)
	baseURL := normalizeExternalURLForOnboarding(externalURL)
	return `---
apiVersion: v1
kind: Namespace
metadata:
  name: devops-status-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: devops-status-agent
  namespace: devops-status-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: devops-status-agent
rules:
  - apiGroups: [""]
    resources: ["nodes", "pods", "services", "namespaces"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets", "daemonsets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["batch"]
    resources: ["jobs", "cronjobs"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: devops-status-agent
subjects:
  - kind: ServiceAccount
    name: devops-status-agent
    namespace: devops-status-system
roleRef:
  kind: ClusterRole
  name: devops-status-agent
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: batch/v1
kind: Job
metadata:
  name: ` + jobName + `
  namespace: devops-status-system
  labels:
    app.kubernetes.io/managed-by: devops-status
    app.kubernetes.io/component: cluster-onboarding
spec:
  template:
    spec:
      serviceAccountName: devops-status-agent
      containers:
        - name: register
          image: curlimages/curl:latest
          command: ["sh", "-c"]
          args:
            - |
              set -eu
              HSK='` + token + `'
              SA_TOKEN="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
              B64=$(printf '%s' "$SA_TOKEN" | base64 | tr -d '\n')
              URL="${STATUS_API_URL}/api/admin/k8s-clusters/` + clusterID + `/register"
              echo "devops-status: POST ${URL}" >&2
              # -4: prefer IPv4 (exit 6 is often AAAA/IPv6 egress issues from pods)
              curl -4 --connect-timeout 25 --max-time 120 -Ssf -X POST \
                -H "Content-Type: application/json" \
                -d "{\"token\":\"${HSK}\",\"cluster_bearer_token_b64\":\"${B64}\"}" \
                "$URL" || {
                  c=$?
                  echo "devops-status: curl failed, exit ${c} (6=DNS, 7=connect, 22=HTTP 4xx/5xx, 28=timeout)" >&2
                  exit "${c}"
                }
              echo "devops-status: register succeeded" >&2
          env:
            - name: STATUS_API_URL
              value: "` + baseURL + `"
      restartPolicy: Never
  backoffLimit: 3
`
}

func buildOnboardingInstructions(externalURL string, isDev bool) []string {
	if !isDev {
		return []string{
			"Copy the command below and run it on the target Kubernetes cluster.",
			"It removes a legacy Job named devops-status-register if present (e.g. after resetting the status DB), then applies a new registration Job with a unique name for this handshake.",
			"The Job posts the handshake token and the in-cluster service-account token back to the status API so probes can call the Kubernetes API.",
			"The generated URL uses a DNS trailing dot on the hostname so pods treat it as an internet FQDN (not subject to cluster DNS search paths). If exit 6 persists, fix cluster/upstream DNS or egress; exit 7 is TCP blocked.",
			"To remove old onboarding Jobs later: kubectl delete jobs -n devops-status-system -l app.kubernetes.io/managed-by=devops-status,app.kubernetes.io/component=cluster-onboarding",
			"Return here after the Job completes to verify status.",
		}
	}

	if isLocalhostURL(externalURL) {
		return []string{
			"WARNING: Backend URL is " + externalURL + " which is NOT reachable from Kubernetes pods.",
			"Start the tunnel first: make dev-infra-up && make be-run (tunnel URL is auto-detected).",
			"Alternatively, use the \"Manual Register\" button and supply a bearer token obtained via kubectl.",
		}
	}

	return []string{
		"Copy the command below and run it on the target cluster (kubectl context must target that cluster).",
		"It deletes the legacy Job devops-status-register if it still exists, then applies a new registration Job (unique name per handshake — Kubernetes Jobs cannot be updated in place).",
		"The Job connects to your backend via Cloudflare tunnel (" + externalURL + ").",
		"The API URL uses a DNS trailing dot on the hostname so pods resolve it as a full internet name (avoids *.svc.cluster.local search confusing external hosts). If you still see curl exit 6, check CoreDNS / upstream DNS and egress; exit 7 means TCP connect failed.",
		"Optional cleanup of completed onboarding Jobs: kubectl delete jobs -n devops-status-system -l app.kubernetes.io/managed-by=devops-status,app.kubernetes.io/component=cluster-onboarding",
		"Return here after the Job completes to verify status; use \"Add API token\" if the Job failed.",
	}
}
