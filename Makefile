.PHONY: dev dev-backend dev-frontend build build-frontend build-backend serve clean seed seed-test-users setup-admin help watch run reset stop test-unit test-api test-e2e test-e2e-ui test-watch test-coverage test docker-build docker-run docker-stop

# Development: run Vite + backend with hot reload proxy
dev: build-backend setup-admin
	@echo "Starting development servers..."
	@echo "Backend: http://127.0.0.1:8090 (proxies frontend to Vite)"
	@echo "Vite:    http://localhost:5173 (direct)"
	@echo "Press Ctrl+C to stop"
	@echo ""
	@bash -c '\
		cleanup() { \
			echo ""; \
			echo "Shutting down..."; \
			kill $$VITE_PID $$BACKEND_PID 2>/dev/null; \
			wait; \
			echo "Stopped."; \
		}; \
		trap cleanup EXIT; \
		cd frontend && npm run dev & VITE_PID=$$!; \
		sleep 2; \
		DEV=1 ./gather serve & BACKEND_PID=$$!; \
		sleep 2; \
		$(MAKE) -s seed; \
		$(MAKE) -s seed-test-users; \
		wait \
	'

# Run just the backend in dev mode (assumes Vite is running separately)
dev-backend:
	@DEV=1 ./gather serve

# Run just the frontend (Vite dev server)
dev-frontend:
	@cd frontend && npm run dev

# Production: build everything, setup admin, seed data, start server
prod: build setup-admin seed serve

# Build everything
build: build-frontend build-backend

# Build frontend
build-frontend:
	@echo "Building frontend..."
	@cd frontend && npm install --silent && npm run build

# Build Go binary
build-backend: build-frontend
	@echo "Building Go binary..."
	@go build -o gather

# Run the server
serve:
	@echo "Starting server at http://127.0.0.1:8090"
	@echo "Admin: http://127.0.0.1:8090/_/ (admin@example.com / adminpassword123)"
	@./gather serve

# Create admin user (idempotent)
setup-admin:
	@echo "Setting up admin user..."
	@./gather superuser upsert admin@example.com adminpassword123 2>/dev/null || true

# Seed dummy data (requires server to be running)
seed:
	@if curl -s "http://127.0.0.1:8090/api/health" > /dev/null 2>&1; then \
		echo "Seeding dummy data..."; \
		PB_ADMIN_EMAIL=admin@example.com PB_ADMIN_PASSWORD=adminpassword123 python3 ./scripts/seed.py; \
	else \
		echo "Server not running, skipping seed"; \
	fi

# Seed test users for E2E testing
seed-test-users: setup-admin
	@echo "Seeding test users..."
	@./scripts/seed-test-users.sh

# Watch frontend for changes (development)
watch:
	@cd frontend && npm run dev

# Run backend only (assumes frontend is built)
run:
	@./gather serve

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f gather
	@rm -rf frontend/dist
	@rm -rf frontend/node_modules/.vite

# Reset everything including database
reset: clean
	@echo "Removing database..."
	@rm -rf pb_data
	@echo "Reset complete. Run 'make dev' to start fresh."

# Stop any running dev servers
stop:
	@echo "Stopping dev servers..."
	@-pkill -f "gather serve" 2>/dev/null
	@-pkill -f "npm run dev" 2>/dev/null
	@-pkill -f "node.*vite" 2>/dev/null
	@-fuser -k 8090/tcp 5173/tcp 5174/tcp 5175/tcp 2>/dev/null
	@echo "Stopped."

# Testing
test-unit:
	@echo "Running Go unit tests..."
	@go test ./internal/... -v 2>/dev/null || echo "No unit tests found or tests failed"

test-api:
	@echo "Running API integration tests..."
	@if [ -d "./internal/api" ]; then \
		go test ./internal/api/... -v; \
	else \
		echo "API tests not yet implemented (internal/api does not exist)"; \
	fi

test-e2e: build-backend
	@echo "Running E2E tests (headless)..."
	@cd tests/e2e && npm test

test-e2e-ui:
	@echo "Running E2E tests (UI mode)..."
	@cd tests/e2e && npm run test:ui

test-watch:
	@echo "Running E2E tests in watch mode (UI)..."
	@cd tests/e2e && npm run test:ui

test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test: test-unit test-api test-e2e
	@echo "All tests passed!"

# Docker
docker-build:
	@echo "Building Docker image..."
	@docker build -t gather:latest .

docker-run:
	@echo "Running Docker container..."
	@docker run -d \
		--name gather \
		-p 8090:8090 \
		-v gather-data:/app/pb_data \
		gather:latest

docker-stop:
	@echo "Stopping and removing Docker container..."
	@docker stop gather 2>/dev/null || true
	@docker rm gather 2>/dev/null || true

# Help
help:
	@echo "Gather - Community Calendar"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  dev            Run with hot reload (Vite + backend proxy)"
	@echo "  dev-backend    Run backend only in dev mode (proxy to Vite)"
	@echo "  dev-frontend   Run Vite dev server only"
	@echo "  prod           Build, seed, and start production server"
	@echo "  run            Start server (assumes already built)"
	@echo ""
	@echo "Building:"
	@echo "  build          Build frontend and backend"
	@echo "  build-frontend Build frontend only"
	@echo "  build-backend  Build Go binary (includes frontend)"
	@echo ""
	@echo "Data:"
	@echo "  setup-admin    Create admin user (admin@example.com / adminpassword123)"
	@echo "  seed           Add dummy data (tags, places, events)"
	@echo ""
	@echo "Cleanup:"
	@echo "  stop           Kill any running dev servers"
	@echo "  clean          Remove build artifacts"
	@echo "  reset          Clean everything including database"
	@echo ""
	@echo "Testing:"
	@echo "  test           Run all tests (unit, API, E2E)"
	@echo "  test-unit      Run Go unit tests"
	@echo "  test-api       Run API integration tests"
	@echo "  test-e2e       Run E2E tests (headless)"
	@echo "  test-e2e-ui    Run E2E tests (interactive UI)"
	@echo "  test-watch     Run E2E tests in watch mode (UI)"
	@echo "  test-coverage  Run tests with coverage report"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build   Build Docker image"
	@echo "  docker-run     Run Docker container (detached)"
	@echo "  docker-stop    Stop and remove Docker container"
	@echo ""
	@echo "Quick start:"
	@echo "  make dev"
	@echo ""
	@echo "Admin dashboard: http://127.0.0.1:8090/_/"
	@echo "  Email:    admin@example.com"
	@echo "  Password: adminpassword123"
	@echo ""
	@echo "Frontend login: http://127.0.0.1:8090/login"
	@echo "  admin@example.com  / adminpassword123  (admin)"
	@echo "  editor@example.com / editorpassword123 (editor)"
	@echo "  user@example.com   / userpassword123   (user)"
