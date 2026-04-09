DB_URL ?= postgres://devops:devops_local@localhost:5432/devops_status?sslmode=disable

.PHONY: up down start stop migrate-up migrate-down seed reset

up:
	docker compose up -d

down:
	docker compose down

start: up
	@echo "Waiting for PostgreSQL to be ready..."
	@until docker exec devops-status-db pg_isready -U devops -d devops_status > /dev/null 2>&1; do sleep 1; done
	@echo "PostgreSQL is ready."

stop:
	docker compose stop

migrate-up: start
	@echo "Running migrations..."
	@docker exec -i devops-status-db psql -U devops -d devops_status < migrations/001_initial_schema.up.sql
	@echo "Migrations applied."

migrate-down:
	@echo "Rolling back migrations..."
	@docker exec -i devops-status-db psql -U devops -d devops_status < migrations/001_initial_schema.down.sql
	@echo "Migrations rolled back."

seed:
	@echo "Running seed data..."
	@docker exec -i devops-status-db psql -U devops -d devops_status < seeds/001_admin_user.sql
	@docker exec -i devops-status-db psql -U devops -d devops_status < seeds/002_sample_data.sql
	@echo "Seed data applied."

reset: migrate-down migrate-up seed
	@echo "Database reset complete."
