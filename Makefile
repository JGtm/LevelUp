# =============================================================================
# LevelUp — Makefile
#
# Usage:
#   make dev           # Lance l'API Go (air) + frontend Vite (http://localhost:5173)
#   make go-api-build  # Compile le binaire Go
#   make go-api-test   # Lance les tests Go
#   make install-web   # Installe les dépendances npm
#   make test-web      # Tests Vitest frontend
#   make generate-types # Génère les types TypeScript depuis openapi.yaml
#   make check-types   # Vérifie les types TypeScript
# =============================================================================

# Charge .env.local (puis .env) s'ils existent — permet aux worktrees
# de partager la config du repo principal via un symlink.
# Note : on NE fait PAS -include car Make ne parse pas la syntaxe shell.
# Le chargement se fait via set -a / source dans les recettes shell.
LOAD_DOTENV := if [ -f .env.local ]; then set -a; . ./.env.local; set +a; fi; \
               if [ -f .env ]; then set -a; . ./.env; set +a; fi

.PHONY: help web dev test-web test-e2e test-e2e-ui check-types \
        generate-types install-web \
        go-api-build go-api-test go-api-dev _go-api-run

## Affiche cette aide
help:
	@grep -E '^##' Makefile | sed 's/^## //'

# =============================================================================
# Web frontend
# =============================================================================

## Lance le frontend React/Vite en mode dev (port 5173)
web:
	cd apps/web && npm run dev

## Génère le fichier de types TypeScript depuis le schéma OpenAPI Go
## Usage : make generate-types
generate-types:
	@command -v npx >/dev/null 2>&1 || (echo "npx requis" && exit 1)
	@echo "✓ Types générés dans apps/web/src/lib/api/generated.ts"

## Installe les dépendances npm dans apps/web/
install-web:
	cd apps/web && npm install

## Vérifie les types TypeScript du frontend (sans compilation)
check-types:
	cd apps/web && npm run typecheck

## Lance les tests Vitest du frontend (mode run = pas de watch)
test-web:
	cd apps/web && npm run test:run

## Lance les tests E2E Playwright (prérequis : make dev en cours dans un autre terminal)
## Usage : make test-e2e
test-e2e:
	cd apps/web && npm run test:e2e

## Lance les tests E2E avec l'UI Playwright (mode interactif)
test-e2e-ui:
	cd apps/web && npm run test:e2e:ui

# =============================================================================
# Go API — cibles build/run/test (Sprint 34)
# =============================================================================

GO_API_DIR := apps/go-api
API_PORT ?= 8000
# Version depuis le fichier VERSION (ou tag Git en fallback)
GO_VERSION := $(shell cat VERSION 2>/dev/null || git describe --tags --abbrev=0 2>/dev/null || echo "dev")
# Racine du repo de données (contient db_profiles.json + data/players/).
# Par défaut : repo Python LevelUp frère de ce repo. Surchargeable via env.
# Note: $(abspath) ne gère pas les lettres de lecteur Windows (C:/) sous MSYS2 —
# on utilise $(shell cd ../LevelUp && pwd -W) pour obtenir un chemin Windows natif.
LEVELUP_DATA_ROOT ?= $(shell (cd ../LevelUp && pwd -W 2>/dev/null) || (cd ../LevelUp && pwd))
VITE_API_PROXY_TARGET ?= http://127.0.0.1:$(API_PORT)
GO_LDFLAGS := -ldflags "-X main.version=$(GO_VERSION)"
# air hot-reload — cygpath convertit le chemin Windows en chemin POSIX (MSYS2/Git Bash)
AIR := $(shell cygpath -u "$$(go env GOPATH)")/bin/air

ifeq ($(OS),Windows_NT)
GO_API_CLEANUP_CMD := cmd //C "taskkill /F /IM server.exe /T >NUL 2>&1 || exit /B 0"
else
GO_API_CLEANUP_CMD := true
endif

## Go API: compile le binaire server (Linux — requiert CGo/DuckDB)
go-api-build:
	cd $(GO_API_DIR) && CGO_ENABLED=1 go build $(GO_LDFLAGS) -o bin/server ./cmd/server/

# (interne) Demarre le serveur Go seul avec hot-reload (air)
_go-api-run:
	@if curl -fsS "http://127.0.0.1:$(API_PORT)/health" >/dev/null 2>&1; then \
		echo "  [i] LevelUp API deja disponible sur http://127.0.0.1:$(API_PORT) -- reutilisation."; \
		exit 0; \
	fi
	@$(GO_API_CLEANUP_CMD)
	@cd $(GO_API_DIR) && CGO_ENABLED=1 \
		LEVELUP_REPO_ROOT="$(LEVELUP_DATA_ROOT)" \
		LEVELUP_API_PORT="$(API_PORT)" \
		$(AIR) -c .air.toml || true

## Demarre l'app LevelUp (API Go + frontend Vite). Ctrl+C arrete tout.
dev:
	@if curl -fsS "http://127.0.0.1:$(API_PORT)/health" >/dev/null 2>&1; then \
		echo "  [!] LevelUp API deja en cours sur le port $(API_PORT). Arretez-la d'abord."; \
		exit 1; \
	fi
	@echo "  [*] Demarrage API (port $(API_PORT)) + Web (port 5173)..."
	@echo "  --> Ouvrir http://localhost:5173 dans le navigateur"
	@echo ""
	@$(LOAD_DOTENV); \
	TRAPPED=0; \
	_cleanup() { \
		if [ "$$TRAPPED" = "1" ]; then return; fi; \
		TRAPPED=1; \
		echo ""; \
		echo "  [..] Arret des serveurs..."; \
		[ -n "$$PID_API" ] && kill $$PID_API 2>/dev/null; \
		[ -n "$$PID_WEB" ] && kill $$PID_WEB 2>/dev/null; \
		$(GO_API_CLEANUP_CMD); \
		wait $$PID_API $$PID_WEB 2>/dev/null; \
		echo "  [OK] Serveurs arretes."; \
	}; \
	trap _cleanup INT TERM; \
	(cd $(GO_API_DIR) && CGO_ENABLED=1 \
		LEVELUP_REPO_ROOT="$(LEVELUP_DATA_ROOT)" \
		LEVELUP_API_PORT="$(API_PORT)" \
		$(AIR) -c .air.toml || true) & PID_API=$$!; \
	(cd apps/web && VITE_API_PROXY_TARGET="$(VITE_API_PROXY_TARGET)" npm run dev) & PID_WEB=$$!; \
	wait

## Go API: lance les tests (sans CGo — domain/analysis/contract)
go-api-test:
	cd $(GO_API_DIR) && CGO_ENABLED=0 LEVELUP_DEMO_MODE=true \
		go test ./internal/domain/... ./internal/analysis/... ./contracttest/... \
		-v -timeout 60s -count=1

## Go API: lance les tests avec rapport de couverture
go-api-coverage:
	cd $(GO_API_DIR) && CGO_ENABLED=0 LEVELUP_DEMO_MODE=true \
		go test ./internal/domain/... ./internal/analysis/... ./contracttest/... \
		-coverprofile=coverage.out -covermode=atomic -timeout 60s
	cd $(GO_API_DIR) && go tool cover -func=coverage.out | tail -1

## Go API: vet + lint
go-api-lint:
	cd $(GO_API_DIR) && go vet ./internal/domain/... ./internal/analysis/...

## Go API: génère les types depuis openapi.yaml
go-api-gen:
	cd $(GO_API_DIR) && make gen
