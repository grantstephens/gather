.PHONY: dev build build-frontend build-backend serve clean seed setup-admin help watch run reset

# Default target: build everything, setup admin, seed data, start server
dev: build setup-admin seed serve

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

# Seed dummy data (requires server to be running or starts it temporarily)
seed:
	@echo "Seeding dummy data..."
	@if curl -s "http://127.0.0.1:8090/api/health" > /dev/null 2>&1; then \
		./scripts/seed.sh; \
	else \
		./gather serve & PID=$$!; \
		sleep 2; \
		./scripts/seed.sh; \
		kill $$PID 2>/dev/null; \
	fi

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

# Help
help:
	@echo "Gather - Community Calendar"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  dev            Build, setup admin, seed data, and start server (default)"
	@echo "  watch          Run Vite dev server for frontend hot-reload"
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
	@echo "  clean          Remove build artifacts"
	@echo "  reset          Clean everything including database"
	@echo ""
	@echo "Quick start:"
	@echo "  make dev"
	@echo ""
	@echo "Admin dashboard: http://127.0.0.1:8090/_/"
	@echo "  Email:    admin@example.com"
	@echo "  Password: adminpassword123"
