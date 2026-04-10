ENV_FILE ?= .env.dev
include $(ENV_FILE)
-include .env.dev.local
export

DEV_DIR := devops-status-dev
DB_DIR := $(DEV_DIR)/db
# Tunnel forwards to the Go server on the host (same as PORT from .env.dev).
TUNNEL_TARGET_PORT := $(if $(strip $(PORT)),$(PORT),8080)

.PHONY: help dev-ensure-pg-volume db-up db-down db-migrate db-seed db-reset tunnel-up tunnel-down tunnel-url dev-infra-up infra-down dev-cleanup-containers be-run be-build fe-install fe-dev dev

help:
	@echo "DevOps Status - Development Commands"
	@echo ""
	@echo "Infrastructure ($(DEV_DIR)/docker-compose.yml):"
	@echo "  make db-up          Start PostgreSQL only"
	@echo "  make db-down          Stop PostgreSQL (tunnel keeps running if up)"
	@echo "  make tunnel-up        Start Cloudflare quick tunnel -> host:8080 (public URL for K8s)"
	@echo "  make tunnel-down      Stop tunnel"
	@echo "  make tunnel-url       Write EXTERNAL_URL to .env.dev.local from tunnel logs"
	@echo "  make dev-infra-up     Start PostgreSQL + tunnel (prod-like dev)"
	@echo "  make infra-down       Stop all compose services in $(DEV_DIR)"
	@echo "  make dev-cleanup-containers  Remove devops-status-db / devops-status-tunnel if name conflict (one-time / after moving compose)"
	@echo ""
	@echo "Database:"
	@echo "  make db-migrate   Run migrations ($(DB_DIR)/migrations)"
	@echo "  make db-seed      Run seed data"
	@echo "  make db-reset     Drop + migrate + seed"
	@echo ""
	@echo "Backend (runs on host):"
	@echo "  make be-run       Run Go API server"
	@echo "  make be-build     Build Go API binary"
	@echo ""
	@echo "Frontend (runs on host):"
	@echo "  make fe-install   Install npm dependencies"
	@echo "  make fe-dev       Run Nuxt dev server"
	@echo ""
	@echo "Prod-like dev:"
	@echo "  ./run.sh            Infra + be-run (bg) + fe-dev — safe full restart from repo root"
	@echo "  ./run.sh --kill     Stop API/FE listeners + docker infra only (no start)"
	@echo "  Or manually:"
	@echo "  1) make dev-infra-up && make db-migrate && make db-seed"
	@echo "  2) make be-run   (auto-detects tunnel URL)"
	@echo "  3) make fe-dev"

# --- Infrastructure (Docker) ---
# Ensure Postgres data volume exists (external in compose; same name as legacy devops-status-dev-db project).
dev-ensure-pg-volume:
	@docker volume inspect devops-status-dev-db_pg_data >/dev/null 2>&1 || docker volume create devops-status-dev-db_pg_data

db-up: dev-ensure-pg-volume
	cd $(DEV_DIR) && docker compose up -d postgres

db-down:
	cd $(DEV_DIR) && docker compose stop postgres

tunnel-up:
	cd $(DEV_DIR) && TUNNEL_TARGET_PORT=$(TUNNEL_TARGET_PORT) docker compose up -d tunnel

tunnel-down:
	cd $(DEV_DIR) && docker compose stop tunnel

tunnel-url:
	@./$(DEV_DIR)/tunnel/sync-external-url.sh

dev-infra-up: dev-ensure-pg-volume
	cd $(DEV_DIR) && TUNNEL_TARGET_PORT=$(TUNNEL_TARGET_PORT) docker compose up -d postgres tunnel

infra-down:
	cd $(DEV_DIR) && docker compose down

# Remove fixed container names left over from an older compose project path (Docker name conflict).
dev-cleanup-containers:
	-docker rm -f devops-status-db devops-status-tunnel
	@echo "Done. Run: make dev-infra-up"

# --- Database ---
db-migrate:
	@echo "Waiting for PostgreSQL to be ready..."
	@n=0; until docker exec devops-status-db pg_isready -U devops -d devops_status > /dev/null 2>&1; do \
		n=$$((n+1)); if [ "$$n" -ge 90 ]; then echo "error: Postgres not ready after 90s (is devops-status-db running?)"; exit 1; fi; \
		sleep 0.33; \
	done
	@echo "Running migrations..."
	@set -e; for f in $$(ls $(DB_DIR)/migrations/*.up.sql | sort); do \
		echo "  $$f"; docker exec -i devops-status-db psql -U devops -d devops_status < "$$f"; \
	done
	@echo "Migrations applied."

db-seed:
	@echo "Running seed data..."
	@docker exec -i devops-status-db psql -U devops -d devops_status < $(DB_DIR)/seeds/001_admin_user.sql
	@docker exec -i devops-status-db psql -U devops -d devops_status < $(DB_DIR)/seeds/002_sample_data.sql
	@echo "Seed data applied."

db-reset:
	@echo "Rolling back migrations..."
	@set -e; for f in $$(ls $(DB_DIR)/migrations/*.down.sql | sort -r); do \
		echo "  $$f"; docker exec -i devops-status-db psql -U devops -d devops_status < "$$f" || true; \
	done
	@echo "Migrations rolled back."
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker exec devops-status-db pg_isready -U devops -d devops_status > /dev/null 2>&1; do sleep 1; done
	@echo "Running migrations..."
	@set -e; for f in $$(ls $(DB_DIR)/migrations/*.up.sql | sort); do \
		echo "  $$f"; docker exec -i devops-status-db psql -U devops -d devops_status < "$$f"; \
	done
	@echo "Migrations applied."
	@echo "Running seed data..."
	@docker exec -i devops-status-db psql -U devops -d devops_status < $(DB_DIR)/seeds/001_admin_user.sql
	@docker exec -i devops-status-db psql -U devops -d devops_status < $(DB_DIR)/seeds/002_sample_data.sql
	@echo "Database reset complete."

# --- Backend ---
be-run:
	@lsof -t -i:8080 | xargs kill -9 2>/dev/null || true
	@TUNNEL_URL=""; \
	if [ "$(ENVIRONMENT)" = "development" ]; then \
		if docker ps --format '{{.Names}}' 2>/dev/null | grep -qx devops-status-tunnel; then \
			for i in $$(seq 1 40); do \
				TUNNEL_URL=$$(docker logs devops-status-tunnel 2>&1 | grep -oE 'https://[a-zA-Z0-9.-]+\.trycloudflare\.com' | tail -1 || true); \
				[ -n "$$TUNNEL_URL" ] && break; \
				sleep 0.25; \
			done; \
		fi; \
		if [ -n "$$TUNNEL_URL" ]; then \
			echo "[dev] Tunnel detected: $$TUNNEL_URL"; \
			export EXTERNAL_URL="$$TUNNEL_URL"; \
		else \
			echo "[dev] No tunnel running. EXTERNAL_URL=$(EXTERNAL_URL) (K8s onboarding Jobs will fail if this is localhost)"; \
		fi; \
	fi; \
	cd devops-status-be && go run ./cmd/server

be-build:
	cd devops-status-be && go build -o bin/server ./cmd/server

# --- Frontend ---
fe-install:
	cd devops-status-fe && npm install

# Uses bash + scripts/dev-node-env.sh so npm works under nvm/mise when make runs from an IDE.
fe-dev:
	bash scripts/fe-dev.sh

# --- Combined (informational) ---
dev:
	@echo "Prod-like local dev:"
	@echo "  ./run.sh   — after first-time: make db-migrate && make db-seed"
	@echo "  Or:"
	@echo "  1) make dev-infra-up && make db-migrate && make db-seed"
	@echo "  2) make be-run   (auto-detects tunnel URL from devops-status-tunnel container)"
	@echo "  3) make fe-install && make fe-dev"
	@echo ""
	@echo "DB only (no tunnel): make db-up && make db-migrate && make db-seed"
