# Makefile — apps/go-api
# Toolchain CGo requise : MinGW ucrt64 via MSYS2
# Installation : pacman -S mingw-w64-ucrt-x86_64-gcc
#
# Usage :
#   make build       — compile le binaire server (Windows)
#   make gen         — regénère les types Go depuis l'OpenAPI spec (Sprint 4)
#   make run         — démarre le serveur en mode dev
#   make test        — lance les tests Go
#   make msal-poc    — lance le POC MSAL Device Code Flow
#   make vet         — go vet sur tous les packages
#   make tidy        — go mod tidy
#   make lint        — go vet + build

.PHONY: build gen run test test-unit test-all coverage coverage-report msal-poc vet tidy lint lint-go

# Toolchain CGo (Windows MinGW ucrt64)
CC      := /c/msys64/ucrt64/bin/gcc.exe
CGO_ENV := PATH="/c/msys64/ucrt64/bin:$(PATH)" CC=$(CC) CGO_ENABLED=1

# Paramètres serveur locaux (override via env)
REPO_ROOT   ?= $(shell cd ../.. && pwd)
API_PORT    ?= 8000
build:
	$(CGO_ENV) go build -o bin/server.exe ./cmd/server/

run:
	$(CGO_ENV) LEVELUP_REPO_ROOT="$(REPO_ROOT)" LEVELUP_API_PORT=$(API_PORT) \
		go run ./cmd/server/

run-demo:
	$(CGO_ENV) LEVELUP_REPO_ROOT="$(REPO_ROOT)" LEVELUP_API_PORT=$(API_PORT) \
		LEVELUP_DEMO_MODE=true go run ./cmd/server/

test:
	$(CGO_ENV) go test -tags=integration -count=1 -timeout=120s -p 1 ./...

# Tests unitaires uniquement (pas de CGO/DuckDB)
test-unit:
	go test -count=1 -timeout=60s $$(go list ./... | grep -v -E 'cmd/|sync/testutil|platform/duckdb|/migration$$')

# Tests rapides (pas de -count=1, cache activé)
test-fast:
	$(CGO_ENV) go test -tags=integration -timeout=120s -p 1 ./...

# Couverture locale complète — résultat affiché + coverage.html généré
coverage:
	$(CGO_ENV) go test -tags=integration -count=1 -timeout=120s -p 1 \
		-coverprofile=coverage.raw -covermode=atomic -coverpkg=./... ./...
	bash scripts/coverage_filter.sh coverage.raw > coverage.out
	@go tool cover -func=coverage.out | grep "^total:"
	@echo ""
	@echo "Détail par package :"
	@$(CGO_ENV) go test -tags=integration -count=1 -timeout=120s -p 1 -cover ./... 2>&1 \
		| grep -E "coverage:" | sort -t% -k1 -rn

# Ouvrir le rapport HTML dans le navigateur
coverage-report: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Rapport : coverage.html"

msal-poc:
	$(CGO_ENV) go run ./cmd/msal-poc/

vet:
	$(CGO_ENV) go vet ./...

# lint-go : golangci-lint (équivalent ruff + enforce_size_limits + McCabe pour Go)
# Installation : https://golangci-lint.run/usage/install/
lint-go:
	golangci-lint run ./...

lint: vet build lint-go
	@echo "lint OK"

tidy:
	go mod tidy
