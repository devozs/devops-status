package liveness

// Liveness check kinds (match service provider types + kubernetes).
const (
	CheckKubernetes     = "kubernetes"
	CheckPrometheus     = "prometheus"
	CheckGrafana        = "grafana"
	CheckElasticsearch  = "elasticsearch"
	CheckJenkins        = "jenkins"
	CheckArtifactory    = "artifactory"
	CheckRancher        = "rancher"
)
