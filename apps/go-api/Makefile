# Makefile — apps/go-api
# Toolchain CGo requise : MinGW ucrt64 via MSYS2
# Installation : pacman -S mingw-w64-ucrt-x86_64-gcc
#
# Usage :
#   make build       — compile le binaire server
#   make run         — démarre le serveur en mode dev (pointe sur LEVELUP_REPO_ROOT)
#   make test        — lance les tests Go
#   make msal-poc    — lance le POC MSAL Device Code Flow
#   make vet         — go vet sur tous les packages
#   make tidy        — go mod tidy

.PHONY: build run test msal-poc vet tidy

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

test:
	$(CGO_ENV) go test ./... -v -timeout 30s

msal-poc:
	$(CGO_ENV) go run ./cmd/msal-poc/

vet:
	$(CGO_ENV) go vet ./...

tidy:
	go mod tidy
