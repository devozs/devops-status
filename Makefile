.PHONY: help db-up db-down db-migrate db-seed db-reset be-run be-build fe-install fe-dev dev

help:
	@echo "DevOps Status - Development Commands"
	@echo ""
	@echo "Database (runs in Docker):"
	@echo "  make db-up        Start PostgreSQL container"
	@echo "  make db-down      Stop PostgreSQL container"
	@echo "  make db-migrate   Run database migrations"
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
	@echo "Combined:"
	@echo "  make dev          Start DB + BE + FE (requires tmux or separate terminals)"

# --- Database ---
db-up:
	cd devops-status-dev-db && docker compose up -d

db-down:
	cd devops-status-dev-db && docker compose down

db-migrate:
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker exec devops-status-db pg_isready -U devops -d devops_status > /dev/null 2>&1; do sleep 1; done
	@echo "Running migrations..."
	@docker exec -i devops-status-db psql -U devops -d devops_status < devops-status-dev-db/migrations/001_initial_schema.up.sql
	@echo "Migrations applied."

db-seed:
	@echo "Running seed data..."
	@docker exec -i devops-status-db psql -U devops -d devops_status < devops-status-dev-db/seeds/001_admin_user.sql
	@docker exec -i devops-status-db psql -U devops -d devops_status < devops-status-dev-db/seeds/002_sample_data.sql
	@echo "Seed data applied."

db-reset:
	@echo "Rolling back migrations..."
	@docker exec -i devops-status-db psql -U devops -d devops_status < devops-status-dev-db/migrations/001_initial_schema.down.sql
	@echo "Migrations rolled back."
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker exec devops-status-db pg_isready -U devops -d devops_status > /dev/null 2>&1; do sleep 1; done
	@echo "Running migrations..."
	@docker exec -i devops-status-db psql -U devops -d devops_status < devops-status-dev-db/migrations/001_initial_schema.up.sql
	@echo "Migrations applied."
	@echo "Running seed data..."
	@docker exec -i devops-status-db psql -U devops -d devops_status < devops-status-dev-db/seeds/001_admin_user.sql
	@docker exec -i devops-status-db psql -U devops -d devops_status < devops-status-dev-db/seeds/002_sample_data.sql
	@echo "Database reset complete."

# --- Backend ---
be-run:
	lsof -t -i:8080 | xargs kill -9 2>/dev/null || true
	cd devops-status-be && go run ./cmd/server

be-build:
	cd devops-status-be && go build -o bin/server ./cmd/server

# --- Frontend ---
fe-install:
	cd devops-status-fe && npm install

fe-dev:
	cd devops-status-fe && lsof -t -i:3000 | xargs kill -9 2>/dev/null || true && clear && npm run dev

# --- Combined (informational) ---
dev:
	@echo "Run in separate terminals:"
	@echo "  Terminal 1: make db-up && make db-migrate && make db-seed"
	@echo "  Terminal 2: make be-run"
	@echo "  Terminal 3: make fe-install && make fe-dev"
