.PHONY: dev dev:skip-tauri frontend:dev backend:dev tauri:dev build tauri:build tauri:build:debug clean setup lint typecheck supabase:types help

# Default target
help:
	@echo "CMS Desktop - Available commands:"
	@echo ""
	@echo "  make dev              Start all services (Tauri + Vite + Go)"
	@echo "  make dev:skip-tauri   Start Vite + Go (no Tauri)"
	@echo "  make frontend:dev     Start Vite dev server only"
	@echo "  make backend:dev      Start Go Fiber server only"
	@echo "  make tauri:dev        Start Tauri dev window"
	@echo ""
	@echo "  make build            Build frontend for production"
	@echo "  make tauri:build      Build desktop app for production"
	@echo "  make tauri:build:debug  Build desktop app (debug mode)"
	@echo ""
	@echo "  make clean            Remove build artifacts"
	@echo "  make setup            Install all dependencies"
	@echo "  make lint             Run ESLint"
	@echo "  make typecheck        Run TypeScript type check"
	@echo ""
	@echo "  make supabase:types   Generate TypeScript types from Supabase"
	@echo ""

# Development
dev:
	@npm run dev

dev:skip-tauri:
	@npm run dev:skip-tauri

frontend:dev:
	@cd frontend && npm run dev

backend:dev:
	@cd backend && go run main.go

tauri:dev:
	@cd src-tauri && cargo tauri dev

# Building
build:
	@npm run frontend:build

tauri:build:
	@npm run tauri:build

tauri:build:debug:
	@npm run tauri:build:debug

# Maintenance
clean:
	@npm run clean
	@echo "Cleaned all build artifacts"

setup:
	@npm install
	@cd frontend && npm install
	@cd backend && go mod download
	@echo "All dependencies installed"

lint:
	@cd frontend && npm run lint

typecheck:
	@cd frontend && npm run typecheck

# Supabase
supabase:types:
	@cd frontend && npx supabase gen types typescript --project-id YOUR_PROJECT_ID > src/types/supabase.ts
	@echo "Supabase types generated"

# Go tools
go:tidy:
	@cd backend && go mod tidy

go:fmt:
	@cd backend && go fmt ./...

go:vet:
	@cd backend && go vet ./...

# Rust tools
rust:fmt:
	@cd src-tauri && cargo fmt

rust:check:
	@cd src-tauri && cargo check

# Full dev workflow
install:setup
	@echo "Ready to develop! Run 'make dev' to start"
