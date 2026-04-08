.PHONY: dev build run migrate sqlc frontend docker clean admin remote-admin local-db local-db-down local-dev

# Local PostgreSQL for development
local-db:
	docker compose -f docker-compose.local.yml up -d

local-db-down:
	docker compose -f docker-compose.local.yml down

# Development with local auth (starts DB, builds backend, starts both)
local-dev: local-db
	@echo "Waiting for PostgreSQL..."
	@until docker compose -f docker-compose.local.yml exec db pg_isready -U gosnag > /dev/null 2>&1; do sleep 0.5; done
	@echo "Building backend..."
	@go build -o gosnag ./cmd/gosnag
	@echo "Starting backend + frontend..."
	@DATABASE_URL="postgres://gosnag:gosnag@localhost:5432/gosnag?sslmode=disable" AUTH_MODE=local PORT=8099 BASE_URL=http://localhost:8099 ./gosnag &
	@cd web && GOSNAG_PORT=8099 VITE_PORT=5200 npm run dev

# Development: run backend + frontend with hot reload
dev:
	@echo "Starting backend..."
	@go run ./cmd/gosnag &
	@cd web && npm run dev

# Build everything
build: frontend
	go build -o gosnag ./cmd/gosnag

# Run the compiled binary
run: build
	./gosnag

# Build frontend only
frontend:
	cd web && npm ci && npm run build

# Run database migrations (requires DATABASE_URL)
migrate:
	go run ./cmd/gosnag migrate

# Regenerate sqlc code from SQL queries
sqlc:
	sqlc generate

# Docker build and run
docker:
	docker compose up --build

# Docker in background
docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

# Add admin user (Docker): make admin EMAIL=user@example.com
admin:
	@test -n "$(EMAIL)" || (echo "Usage: make admin EMAIL=user@example.com" && exit 1)
	docker compose exec db psql -U gosnag -c \
		"INSERT INTO users (id, email, name, role, created_at, updated_at) VALUES (gen_random_uuid(), '$(EMAIL)', '$(EMAIL)', 'admin', now(), now()) ON CONFLICT (email) DO UPDATE SET role = 'admin', updated_at = now();"
	@echo "✓ $(EMAIL) is now admin"

# Add admin user on remote EC2: make remote-admin EMAIL=user@example.com HOST=1.2.3.4
remote-admin:
	@test -n "$(EMAIL)" || (echo "Usage: make remote-admin EMAIL=user@example.com HOST=1.2.3.4" && exit 1)
	@test -n "$(HOST)" || (echo "Usage: make remote-admin EMAIL=user@example.com HOST=1.2.3.4" && exit 1)
	ssh -i ~/.ssh/gosnag-key.pem ec2-user@$(HOST) \
		"sudo docker compose -f /opt/gosnag/docker-compose.yml exec db psql -U gosnag -c \"INSERT INTO users (id, email, name, role, created_at, updated_at) VALUES (gen_random_uuid(), '$(EMAIL)', '$(EMAIL)', 'admin', now(), now()) ON CONFLICT (email) DO UPDATE SET role = 'admin', updated_at = now();\""
	@echo "✓ $(EMAIL) is now admin on $(HOST)"

# Staging: run with internal PostgreSQL on port 8081
staging-up:
	docker compose -f docker-compose.staging.yml up --build -d

staging-down:
	docker compose -f docker-compose.staging.yml down

staging-logs:
	docker compose -f docker-compose.staging.yml logs -f gosnag

# Sync production data to staging (requires staging running + prod DB access)
staging-sync:
	./scripts/staging-sync.sh

staging-sync-schema:
	./scripts/staging-sync.sh --schema-only

# Add admin to staging
staging-admin:
	@test -n "$(EMAIL)" || (echo "Usage: make staging-admin EMAIL=user@example.com" && exit 1)
	docker compose -f docker-compose.staging.yml exec db psql -U gosnag -c \
		"SET search_path TO gosnag, public; INSERT INTO users (id, email, name, role, created_at, updated_at) VALUES (gen_random_uuid(), '$(EMAIL)', '$(EMAIL)', 'admin', now(), now()) ON CONFLICT (email) DO UPDATE SET role = 'admin', updated_at = now();"
	@echo "$(EMAIL) is now admin on staging"

# Seed local project with sample events: make local-seed DSN=http://key@localhost:8099/5
local-seed:
	@test -n "$(DSN)" || (echo "Usage: make local-seed DSN=<project-dsn> [COUNT=200]" && exit 1)
	@./scripts/local-seed.sh "$(DSN)" "$(or $(COUNT),200)"

# Clean build artifacts
clean:
	rm -f gosnag
	rm -rf web/dist web/node_modules
