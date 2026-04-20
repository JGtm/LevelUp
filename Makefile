# =============================================================================
# LevelUp — Makefile (macOS / Linux)
#
# Usage:
#   make install       # Premier lancement : crée .venv + installe les dépendances
#   make run           # Lance le dashboard
#   make sync          # Synchronise tous les joueurs
#   make add-player    # Ajoute un nouveau joueur (guidé)
#   make doctor        # Vérifie l'environnement
#   make update        # Met à jour les dépendances
#   make check         # Lint + format + taille (avant chaque commit)
#   make test          # Lance la suite de tests rapide
#   make test-all      # Lance tous les tests (incl. intégration)
#   make clean         # Supprime le venv
# =============================================================================

PYTHON   := $(shell command -v python3 2>/dev/null || command -v python 2>/dev/null)
VENV_PY  := .venv/bin/python
LAUNCHER := $(VENV_PY) launcher.py

.PHONY: install run sync add-player doctor update check test test-all clean help \
        api web dev test-api test-parity test-web test-e2e test-e2e-ui check-types \
        generate-types install-web _check_venv

## Installe le venv et les dépendances (idempotent)
install:
	@if [ -f "$(VENV_PY)" ]; then \
		echo "✓ .venv déjà présent — utilisez 'make update' pour mettre à jour"; \
	else \
		$(PYTHON) launcher.py setup; \
	fi

## Lance le dashboard (mode interactif si aucun joueur)
run: _check_venv
	$(LAUNCHER)

## Synchronise tous les joueurs configurés
sync: _check_venv
	$(LAUNCHER) sync

## Ajoute / synchronise un joueur (guidé si --gamertag absent)
## Usage : make add-player  ou  make add-player GAMERTAG=JGtm
add-player: _check_venv
ifdef GAMERTAG
	$(LAUNCHER) add-player --gamertag $(GAMERTAG)
else
	$(LAUNCHER) add-player
endif

## Vérifie l'environnement (packages, données, configuration)
doctor: _check_venv
	$(LAUNCHER) doctor

## Met à jour les dépendances du venv
update: _check_venv
	$(LAUNCHER) setup --update

## Lint + format + vérification taille (identique au pre-commit, à lancer avant git commit)
check: _check_venv
	$(VENV_PY) -m ruff check src/ scripts/ --fix
	$(VENV_PY) -m ruff format src/ scripts/ tests/
	$(VENV_PY) scripts/check_code_size.py

## Lance la suite de tests rapide (hors intégration)
test: _check_venv
	$(VENV_PY) -m pytest --ignore=tests/integration -q

## Lance tous les tests (intégration incluse)
test-all: _check_venv
	$(VENV_PY) -m pytest -q

## Supprime le venv (les données ne sont PAS affectées)
clean:
	@echo "Suppression de .venv…"
	rm -rf .venv
	@echo "✓ Terminé. Relanez 'make install' pour recréer l'environnement."

## Affiche cette aide
help:
	@grep -E '^##' Makefile | sed 's/^## //'

# =============================================================================
# Migration FastAPI + React — cibles Slice 0a
# =============================================================================

## Lance l'API FastAPI en mode dev (port 8000, hot-reload)
api: _check_venv
	$(VENV_PY) -m uvicorn apps.api.app.main:app --host 127.0.0.1 --port 8000 --reload

## Lance le frontend React/Vite en mode dev (port 5173)
web:
	cd apps/web && npm run dev

## Lance API + frontend en parallèle (nécessite un terminal avec support job control)
dev: _check_venv
	@echo "▶ Démarrage API (port 8000) + Web (port 5173)…"
	$(VENV_PY) -m uvicorn apps.api.app.main:app --host 127.0.0.1 --port 8000 --reload & \
	cd apps/web && npm run dev

## Lance uniquement les tests API (tests/api/)
test-api: _check_venv
	$(VENV_PY) -m pytest tests/api/ -v

## Lance les tests de parité backend (tests/parity/)
test-parity: _check_venv
	$(VENV_PY) -m pytest tests/parity/ -v

## Génère le fichier de types TypeScript depuis le schéma OpenAPI FastAPI
## Prérequis : l'API doit être démarrée (make api ou make dev)
## Usage : make generate-types
generate-types:
	@command -v npx >/dev/null 2>&1 || (echo "npx requis" && exit 1)
	@curl -s http://127.0.0.1:8000/api/v1/health > /dev/null 2>&1 || \
		(echo "❌ API non joignable — lancez 'make api' d'abord." && exit 1)
	cd apps/web && npm run generate-types
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

# ── Cible interne ─────────────────────────────────────────────────────────────
_check_venv:
	@if [ ! -f "$(VENV_PY)" ]; then \
		echo "❌ .venv introuvable — lancez 'make install' d'abord."; \
		exit 1; \
	fi

# =============================================================================
# Go API — cibles build/run/test (Sprint 34)
# =============================================================================

GO_API_DIR := apps/go-api
API_PORT ?= 8000
# Injection de la version depuis pyproject.toml ou tag Git (fallback "dev")
GO_VERSION := $(shell python3 -c "import re,pathlib; m=re.search(r'version\s*=\s*\"([^\"]+)\"', pathlib.Path('pyproject.toml').read_text()); print(m.group(1) if m else 'dev')" 2>/dev/null || echo "dev")
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

## Go API: démarre le serveur Go seul avec hot-reload (air)
go-api-run:
	@if curl -fsS "http://127.0.0.1:$(API_PORT)/health" >/dev/null 2>&1; then \
		echo "  ℹ️ LevelUp API déjà disponible sur http://127.0.0.1:$(API_PORT) — réutilisation."; \
		exit 0; \
	fi
	@$(GO_API_CLEANUP_CMD)
	@cd $(GO_API_DIR) && CGO_ENABLED=1 \
		LEVELUP_REPO_ROOT="$(LEVELUP_DATA_ROOT)" \
		LEVELUP_API_PORT="$(API_PORT)" \
		$(AIR) -c .air.toml

## Go API: démarre le serveur Go (air hot-reload) + frontend Vite en parallèle
## Sur Windows/MSYS2, ouvrir deux terminaux : make go-api-run | make web-dev
go-api-dev:
	@$(MAKE) -j2 --no-print-directory go-api-run web-dev

## Frontend Vite uniquement
web-dev:
	@cd apps/web && VITE_API_PROXY_TARGET="$(VITE_API_PROXY_TARGET)" npm run dev

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
