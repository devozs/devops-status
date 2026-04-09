package admin

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/devops-status/be/internal/auth"
	"github.com/devops-status/be/internal/handler"
	"github.com/devops-status/be/internal/store"
)

type K8sClustersHandler struct {
	store *store.Store
}

func NewK8sClustersHandler(s *store.Store) *K8sClustersHandler {
	return &K8sClustersHandler{store: s}
}

func (h *K8sClustersHandler) List(w http.ResponseWriter, r *http.Request) {
	clusters, err := h.store.ListK8sClusters(r.Context())
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list clusters")
		return
	}
	handler.WriteJSON(w, http.StatusOK, clusters)
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

	manifest := generateOnboardingManifest(cluster.Name, id.String(), ht.Token)

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "generate_onboarding_token", "k8s_cluster", &id, nil, r.RemoteAddr)

	handler.WriteJSON(w, http.StatusOK, map[string]any{
		"token":      ht.Token,
		"expires_at": ht.ExpiresAt,
		"manifest":   manifest,
		"instructions": []string{
			"1. Copy the manifest below and save as devops-status-agent.yaml",
			"2. Run: kubectl apply -f devops-status-agent.yaml",
			"3. The agent will register with the status platform automatically",
			"4. Return here to verify the connection status",
		},
	})
}

func (h *K8sClustersHandler) RegisterCallback(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Token string `json:"token"`
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

func (h *K8sClustersHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	_ = h.store.RevokeHandshake(r.Context(), id)
	_ = h.store.UpdateK8sClusterStatus(r.Context(), id, "revoked")

	userID, _ := auth.UserIDFromContext(r.Context())
	_ = h.store.CreateAuditLog(r.Context(), &userID, "revoke", "k8s_cluster", &id, nil, r.RemoteAddr)
	handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func generateOnboardingManifest(clusterName, clusterID, token string) string {
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
  name: devops-status-register
  namespace: devops-status-system
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
              curl -s -X POST \
                -H "Content-Type: application/json" \
                -d '{"token":"` + token + `"}' \
                ${STATUS_API_URL}/api/admin/k8s-clusters/` + clusterID + `/register
          env:
            - name: STATUS_API_URL
              value: "http://YOUR_STATUS_APP_URL:8080"
      restartPolicy: Never
  backoffLimit: 3
`
}
