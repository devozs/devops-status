# Telemetry shell runner: Ubuntu with common CLI tools and kubectl (multi-cluster via KUBECONFIG).
FROM ubuntu:24.04

ARG KUBECTL_VERSION=v1.30.8
ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        wget \
        jq \
    && rm -rf /var/lib/apt/lists/*

RUN arch="$(dpkg --print-architecture)" \
    && case "$arch" in amd64) k_arch=amd64 ;; arm64) k_arch=arm64 ;; *) echo "unsupported arch: $arch" >&2; exit 1 ;; esac \
    && curl -fsSL -o /usr/local/bin/kubectl \
        "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${k_arch}/kubectl" \
    && chmod +x /usr/local/bin/kubectl

# hlctl (Habana cluster CLI): install script resolves latest tagged binary from Artifactory generic repo.
# BUILD_URL must be non-empty so the script drops ./hlctl in cwd (see download-latest.sh) instead of ~/.local/bin.
RUN cd /tmp \
    && export BUILD_URL=docker \
    && curl -fsSL "https://artifactory-kfs.habana-labs.com:443/artifactory/k8s-infra-generic-dev-local/hlctl/download-latest.sh" | bash \
    && mv /tmp/hlctl /usr/local/bin/hlctl

RUN mkdir -p /root/.kube

WORKDIR /workspace
