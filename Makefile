# PiNAS — atalhos
.PHONY: help install dev-backend dev-frontend build up down logs ps clean

help:
	@echo "PiNAS — make targets"
	@echo "  make install         instala deps no host (sudo)"
	@echo "  make dev-backend     roda backend Go em dev (precisa Go 1.23)"
	@echo "  make dev-frontend    roda Vite dev server (precisa Node 22)"
	@echo "  make build           build do frontend + imagens Docker"
	@echo "  make up              docker compose up -d"
	@echo "  make down            docker compose down"
	@echo "  make logs            docker compose logs -f"
	@echo "  make ps              docker compose ps"
	@echo "  make backup          rsync snapshot (sudo)"

install:
	sudo ./scripts/install.sh

dev-backend:
	cd backend && \
		PINAS_ENV=development \
		PINAS_DB_PATH=$$PWD/.dev/db/pinas.db \
		PINAS_DATA_DIR=$$PWD/.dev/data \
		PINAS_THUMBS_DIR=$$PWD/.dev/thumbs \
		PINAS_LOGS_DIR=$$PWD/.dev/logs \
		PINAS_SECRETS_DIR=$$PWD/.dev/secrets \
		PINAS_ADMIN_PASSWORD=devpass1 \
		PINAS_ALLOWED_ORIGIN=http://localhost:5173 \
		go run ./cmd/pinas

dev-frontend:
	cd frontend && npm install && npm run dev

build:
	cd frontend && npm ci && npm run build
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps

backup:
	sudo ./scripts/backup.sh

clean:
	rm -rf backend/.dev frontend/build frontend/.svelte-kit frontend/node_modules
