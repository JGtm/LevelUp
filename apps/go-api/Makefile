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

.PHONY: build gen run test msal-poc vet tidy lint

# Toolchain CGo (Windows MinGW ucrt64)
CC      := /c/msys64/ucrt64/bin/gcc.exe
CGO_ENV := PATH="/c/msys64/ucrt64/bin:$(PATH)" CC=$(CC) CGO_ENABLED=1

# Paramètres serveur locaux (override via env)
REPO_ROOT   ?= $(shell cd ../.. && pwd)
API_PORT    ?= 8000
OAPI_CODEGEN := oapi-codegen

build:
	$(CGO_ENV) go build -o bin/server.exe ./cmd/server/

# Régénère internal/api/gen/types.gen.go depuis api/openapi.yaml
gen:
	$(OAPI_CODEGEN) --config api/oapi-codegen-types.yaml api/openapi.yaml

run:
	$(CGO_ENV) LEVELUP_REPO_ROOT="$(REPO_ROOT)" LEVELUP_API_PORT=$(API_PORT) \
		go run ./cmd/server/

run-demo:
	$(CGO_ENV) LEVELUP_REPO_ROOT="$(REPO_ROOT)" LEVELUP_API_PORT=$(API_PORT) \
		LEVELUP_DEMO_MODE=true go run ./cmd/server/

test:
	$(CGO_ENV) go test ./... -v -timeout 30s

msal-poc:
	$(CGO_ENV) go run ./cmd/msal-poc/

vet:
	$(CGO_ENV) go vet ./...

lint: vet build
	@echo "lint OK"

tidy:
	go mod tidy
