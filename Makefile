# =============================================================================
# LevelUp — Makefile
#
# Usage:
#   make dev           # Lance l'API Go (air) + frontend Vite (http://localhost:5173)
#   make go-api-build  # Compile le binaire Go
#   make go-api-test   # Lance les tests Go
#   make install-web   # Installe les dépendances npm
#   make test-web      # Tests Vitest frontend
#   make openapi-gen   # Régénère api/openapi.yaml (Huma + fragment manuel)
#   make generate-types # Génère les types TypeScript depuis openapi.yaml
#   make check-types   # Vérifie les types TypeScript
#   make gate-push     # Filet local avant merge vers main (~25 min, cf. cible)
# =============================================================================

# Charge .env.local (puis .env) s'ils existent — permet aux worktrees
# de partager la config du repo principal via un symlink.
# Note : on NE fait PAS -include car Make ne parse pas la syntaxe shell.
# Le chargement se fait via set -a / source dans les recettes shell.
LOAD_DOTENV := if [ -f .env.local ]; then set -a; . ./.env.local; set +a; fi; \
               if [ -f .env ]; then set -a; . ./.env; set +a; fi

.PHONY: help web dev stop restart test-web test-e2e test-e2e-ui check-types \
        generate-types install-web \
        go-api-build go-api-test go-api-dev _go-api-run \
        go-api-test-shared-social-gate install-git-hooks gate-push \
        go-api-test-coverage-ratchet openapi-gen openapi-check

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
## Délègue au script npm generate-types (openapi-typescript
## apps/go-api/api/openapi.yaml -> apps/web/src/lib/api/generated.ts).
generate-types:
	cd apps/web && npm run generate-types

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
# air hot-reload — which air donne le chemin complet depuis le PATH (plus robuste
# que go env GOPATH qui retourne vide quand USERPROFILE n'est pas transmis au shell make)
AIR := $(or $(shell which air 2>/dev/null),$(shell cygpath -u "$$(go env GOPATH)")/bin/air)

ifeq ($(OS),Windows_NT)
GO_API_CLEANUP_CMD := cmd //C "taskkill /F /IM server.exe /T >NUL 2>&1 || exit /B 0"
# Arret ROBUSTE des serveurs dev : on tue par PORT (API + Vite 5173), insensible au
# nom du binaire (server.exe, server_redeploy.exe, enfant de air, etc.) + air par nom.
# Single-quotes obligatoires : empechent sh d'expanser $_ avant PowerShell.
STOP_SERVERS_CMD := powershell -NoProfile -Command '@($(API_PORT),5173) | ForEach-Object { Get-NetTCPConnection -LocalPort $$_ -State Listen -ErrorAction SilentlyContinue } | ForEach-Object { Stop-Process -Id $$_.OwningProcess -Force -ErrorAction SilentlyContinue }; Stop-Process -Name air,server,server_redeploy -Force -ErrorAction SilentlyContinue; exit 0'
else
GO_API_CLEANUP_CMD := true
STOP_SERVERS_CMD := bash -c 'for p in $(API_PORT) 5173; do pid=$$(lsof -ti tcp:$$p 2>/dev/null); [ -n "$$pid" ] && kill -9 $$pid 2>/dev/null; done; pkill -f bin/air 2>/dev/null; true'
endif

## Régénère apps/go-api/api/openapi.yaml (document Huma + fragment manuel)
## Usage : make openapi-gen   puis   make generate-types
## Le contrat est GÉNÉRÉ : ne jamais éditer openapi.yaml à la main — éditer le
## handler/DTO Go, ou api/openapi_manual_fragment.yaml pour le non-dérivable.
## Verrouillé par TestOpenAPIYAMLIsUpToDate (go test ./internal/api/).
openapi-gen:
	cd $(GO_API_DIR) && CGO_ENABLED=1 go run ./cmd/openapi-gen

## Vérifie que openapi.yaml est à jour sans l'écrire (sortie 1 si drift), PUIS que
## generated.ts en dérive bien (sinon le front resterait typé sur l'ancien contrat
## avec un tsc vert). Le second maillon est aussi joué par la CI (job Frontend) via
## src/lib/api/generated-types-fresh.guard.test.ts — même script, une seule logique.
openapi-check:
	cd $(GO_API_DIR) && CGO_ENABLED=1 go run ./cmd/openapi-gen -check
	node tools/check-generated-types-fresh.mjs

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

## Arrete les serveurs dev (API + front Vite). Kill par port, donc insensible au
## nom du binaire (server.exe / server_redeploy.exe / air).
stop:
	@echo "  [..] Arret des serveurs dev (API:$(API_PORT) + Web:5173)..."
	-@$(STOP_SERVERS_CMD)
	@echo "  [OK] Serveurs arretes."

## Redemarre tout : arret propre (stop) puis demarrage (dev, foreground — Ctrl+C arrete).
## stop + dev en PREREQUIS (executes dans l'ordre, sans -j) : evite un appel recursif
## $(MAKE) dont le chemin GnuWin32 "C:/Program Files (x86)/..." casse sh (parentheses).
restart: stop dev

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

## Go API: lint (2026-07-22, F8 — aligné sur la CI)
##
## Le lint qui FAIT FOI est le job CI `go-lint` (.github/workflows/ci.yml) :
## golangci-lint v2.12.2 sur TOUT apps/go-api, config apps/go-api/.golangci.yml,
## avec RATCHET `--new-from-merge-base=origin/main` — seules les issues AJOUTÉES
## rougissent ; la dette baseline (~479 issues gelées) reste invisible.
##
## Ce target reproduit CE lint QUAND golangci-lint est installé (même ratchet →
## n'échoue jamais sur la dette gelée). Sinon (binaire absent — cas de certains
## environnements de dev), REPLI VOLONTAIRE sur `go vet` domain+analysis (miroir
## exact du step vet du job CI `go-test`, CGO-free) + message renvoyant au job qui
## fait foi. Scope réduit du repli assumé : `go vet` ne couvre pas gocyclo/funlen/
## lll ; ceux-ci sont vérifiés en CI. Retrait du repli : quand golangci-lint entre
## dans l'image de dev standard.
go-api-lint:
	cd $(GO_API_DIR) && \
	if command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint présent — lint complet (ratchet CI, dette gelée exclue)."; \
		golangci-lint run --timeout 5m --new-from-merge-base=origin/main; \
	else \
		echo "golangci-lint absent — REPLI go vet (domain+analysis). Le lint complet FAIT FOI en CI (job go-lint, .github/workflows/ci.yml)."; \
		go vet ./internal/domain/... ./internal/analysis/...; \
	fi

## Installe les hooks git du projet (lefthook — seul système de hooks).
##
## HISTORIQUE (2026-07-26) : cette cible copiait scripts/git-hooks/pre-commit
## dans .git/hooks/pre-commit, donc PAR-DESSUS le shim lefthook — elle cassait
## l'installation (plus aucun hook lefthook joué, et le hook copié était lui-même
## remplacé au prochain `lefthook install`). Les deux systèmes ont été fusionnés :
## lefthook.yml porte désormais tous les hooks, gate ADR 0021 inclus (stage
## pre-push, cf. scripts/git-hooks/lefthook/shared-social-gate.sh).
## Bypass exceptionnel : LEFTHOOK=0 git commit / git push (à documenter).
install-git-hooks:
	@command -v lefthook >/dev/null 2>&1 || { \
		echo "[install-git-hooks] lefthook absent du PATH."; \
		echo "  Installation : go install github.com/evilmartians/lefthook@latest"; \
		exit 1; \
	}
	lefthook install

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

## Filet local AVANT tout merge/push vers main (~25 min). La CI reste le gate
## d'AUTORITÉ : cette cible ne fait que rejouer localement les 3 gates les plus
## bruyants, qui n'avaient AUCUN équivalent local — d'où les « arrivées rouges »
## sur main (149 des 188 jobs CI rouges sur 14 jours viennent de ces 3 gates).
##
## Contenu, dans l'ordre du plus rapide au plus lent (feedback utile d'abord) :
##   1. ratchet golangci-lint — MÊME commande que le job CI go-lint
##      (--new-from-merge-base=origin/main : la dette gelée reste invisible) ;
##   2. typecheck + lint web — mêmes scripts npm que le job CI Frontend ;
##   3. baseline de tests Go — scripts/check_test_baseline.sh tests (le plus long,
##      ~20 min : suite complète avec -tags=integration -p 1).
## NON reproduit : le ratchet de COUVERTURE (il rejouerait toute la suite une 2e
## fois avec -coverprofile, doublant la durée) — il reste au job CI go-coverage.
##
## Prérequis : golangci-lint dans le PATH (échec explicite sinon — le but est de
## reproduire la CI, pas de la contourner par un repli silencieux) + gcc/CGO pour
## la baseline (le script bascule seul sur msys64 sous Windows).
gate-push:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "[gate-push] golangci-lint absent du PATH — gate impossible à reproduire."; \
		echo "  Installation : https://golangci-lint.run/usage/install/ (version CI : v2.12.2)"; \
		exit 1; \
	}
	cd $(GO_API_DIR) && golangci-lint run --timeout 5m --new-from-merge-base=origin/main
	cd apps/web && npm run typecheck
	cd apps/web && npm run lint
	bash scripts/check_test_baseline.sh tests

## Coverage ratchet (Gap 4 / Phase 5.1 strict).
go-api-test-coverage-ratchet:
	./scripts/check_coverage_ratchet.sh

## Go API: génère les types depuis openapi.yaml
go-api-gen:
	cd $(GO_API_DIR) && make gen
