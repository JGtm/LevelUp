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
        go-api-build go-api-test go-api-dev _go-api-run \
        go-api-test-shared-social-gate install-git-hooks \
        go-api-test-coverage-ratchet

## Affiche cette aide
help:
	@grep -E '^##' Makefile | sed 's/^## //'

# =============================================================================
# Web frontend
# =============================================================================

## Lance le frontend React/Vite en mode dev (port 5173)
web:
	cd apps/web && npm run dev

## Lance l'API Go seule en mode dev (air)
go-api-dev: _go-api-run

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
# Version depuis le dernier tag Git (fallback "dev" si pas de tag).
# Le fichier VERSION a été supprimé 2026-05-22 : numéro de version plus affiché
# côté UI (FeedbackDrawer.appVersion=null), seul le ldflags injection sert
# pour les logs boot + notif Discord nouvelle version.
GO_VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
# Racine du repo de données (contient db_profiles.json + data/players/).
# Laisser vide par défaut pour la résoudre au runtime depuis le repo courant.
# Surchargeable via env si on veut pointer ailleurs.
LEVELUP_DATA_ROOT ?=
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
	@REPO_ROOT="$(LEVELUP_DATA_ROOT)"; \
	if [ -z "$$REPO_ROOT" ]; then REPO_ROOT="$$(pwd -W 2>/dev/null || pwd)"; fi; \
	cd $(GO_API_DIR) && CGO_ENABLED=1 \
		LEVELUP_REPO_ROOT="$$REPO_ROOT" \
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
	REPO_ROOT="$(LEVELUP_DATA_ROOT)"; \
	if [ -z "$$REPO_ROOT" ]; then REPO_ROOT="$$(pwd -W 2>/dev/null || pwd)"; fi; \
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
		LEVELUP_REPO_ROOT="$$REPO_ROOT" \
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

## Installe le hook pre-commit ADR 0021 (Phase 3.5).
##
## Le hook lance go-api-test-shared-social-gate si le commit modifie des fichiers
## critiques (pool/persist/media/ops/migration shared_social). Bypass via
## --no-verify (documenter dans le commit msg).
install-git-hooks:
	@HOOKS_DIR=$$(git rev-parse --git-path hooks 2>/dev/null); \
	if [ -z "$$HOOKS_DIR" ]; then \
		echo "[install-git-hooks] git rev-parse échoué — repo non-initialisé ?"; \
		exit 1; \
	fi; \
	mkdir -p "$$HOOKS_DIR"; \
	cp scripts/git-hooks/pre-commit "$$HOOKS_DIR/pre-commit"; \
	chmod +x "$$HOOKS_DIR/pre-commit" 2>/dev/null || true; \
	echo "[install-git-hooks] hook pre-commit installé : $$HOOKS_DIR/pre-commit"; \
	echo "  Pour bypass exceptionnel : git commit --no-verify (à documenter)."

## Go API: gate ADR 0021 — tests invariants shared_social (récupération auto WAL,
## CHECKPOINT systématique, sentinelle AST anti-ATTACH). À lancer en CI sur PR
## qui touche pool/media/persist/shared_social ; race-clean exigé.
##
## Couvre :
##   - openSharedSocialWithWALRecovery + quarantineOrphanWAL + isWALReplayFailure
##   - CheckpointSharedSocial (helper Phase 3.2)
##   - sentinelle AST NoATTACHOnSocialDB + socialReceiverLabel
##   - tests intégration SetMediaMatchAssociation/SetMediaLike/ToggleSharedLike
##   - E2E kill brutal + restart cycle (sub-process)
go-api-test-shared-social-gate:
	cd $(GO_API_DIR) && CGO_ENABLED=1 \
		go test -race -count=1 -timeout 90s \
		-run 'TestOpenSharedSocial|TestIsWALReplayFailure|TestQuarantineOrphanWAL|TestCheckpointSharedSocial|TestSet.*PersistsAfter|TestToggle.*PersistsAfter|TestNoATTACHOnSocialDB|TestSocialReceiverLabel|TestSentinel|TestRequireSocialPersister|TestErrSocialPersisterNotWired|TestWALOrphanRepro' \
		./internal/platform/duckdb/
	cd $(GO_API_DIR) && CGO_ENABLED=1 \
		go test -race -count=1 -timeout 120s \
		-run 'TestE2E_KillBrutal|TestAssociateMediaWithMatches_RestartCycle|TestE2E_AssociateMedia' \
		./internal/ops/
	cd $(GO_API_DIR) && CGO_ENABLED=1 \
		go test -race -count=1 -timeout 60s \
		-run 'TestMediaE2E_RealDB' \
		./internal/api/handlers/

## Coverage ratchet (Gap 4 / Phase 5.1 strict).
go-api-test-coverage-ratchet:
	./scripts/check_coverage_ratchet.sh

## Go API: génère les types depuis openapi.yaml
go-api-gen:
	cd $(GO_API_DIR) && make gen
