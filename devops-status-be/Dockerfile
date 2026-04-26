# syntax=docker/dockerfile:1
# Build from the repository root (context = "."), e.g.:
#   docker build -f devops-status-be/Dockerfile -t devops-status-management .

FROM golang:1.24-bookworm AS build
WORKDIR /src

COPY devops-status-be/go.mod devops-status-be/go.sum ./
RUN go mod download

COPY devops-status-be/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# Migrations ship in-image (single source: devops-status-dev/db/migrations). Alpine + psql for migrate.sh.
# kubectl + hlctl: HLCTL service provider and telemetry run locally in the management process (see internal/hlctlbin, internal/adapter/hlctl.go).
FROM alpine:3.21
ARG KUBECTL_VERSION=v1.30.8
RUN apk add --no-cache ca-certificates postgresql-client bash curl jq \
  && arch="$(uname -m)" \
  && case "$arch" in x86_64) k_arch=amd64 ;; aarch64) k_arch=arm64 ;; *) echo "unsupported arch: $arch" >&2; exit 1 ;; esac \
  && curl -fsSL -o /usr/local/bin/kubectl \
      "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${k_arch}/kubectl" \
  && chmod +x /usr/local/bin/kubectl \
  && cd /tmp && export BUILD_URL=docker \
  && curl -fsSL "https://artifactory-kfs.habana-labs.com:443/artifactory/k8s-infra-generic-dev-local/hlctl/download-latest.sh" | bash \
  && mv /tmp/hlctl /usr/local/bin/hlctl \
  && chmod +x /usr/local/bin/hlctl \
  && addgroup -g 65532 -S nonroot && adduser -u 65532 -S -G nonroot -D nonroot

COPY --from=build /server /server
COPY devops-status-dev/db/migrations/*.up.sql /migrations/
COPY devops-status-dev/db/seeds/*.sql /seeds/
COPY devops-status-be/docker/migrate.sh /migrate.sh
COPY devops-status-be/docker/seed.sh /seed.sh
COPY devops-status-be/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /migrate.sh /seed.sh /entrypoint.sh \
  && chown -R nonroot:nonroot /migrations /seeds /server /migrate.sh /seed.sh /entrypoint.sh

USER nonroot:nonroot
WORKDIR /
EXPOSE 8080
ENTRYPOINT ["/entrypoint.sh"]
