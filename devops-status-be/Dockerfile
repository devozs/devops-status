# syntax=docker/dockerfile:1
# Build from the repository root (context = "."), e.g.:
#   docker build -f devops-status-be/Dockerfile -t devops-status-be .

FROM golang:1.24-bookworm AS build
WORKDIR /src

COPY devops-status-be/go.mod devops-status-be/go.sum ./
RUN go mod download

COPY devops-status-be/ .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /server /server
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
