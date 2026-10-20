#!/usr/bin/env bash
# go vet INFORMATIF sur les packages hors cmd/server — CGO desactive (rapide) et
# non bloquant (|| true), doctrine pre-commit : jamais bloquer un commit sur la
# dette situee ailleurs dans l'arbre.
# LIMITE STRUCTURELLE (documentee le 2026-08-25) : CGO=0 exclut du build tout
# fichier tagge //go:build cgo (d'ou les « build constraints exclude all Go
# files » dans sa sortie) — une casse dans ces fichiers est INVISIBLE ici.
# Le filet bloquant est le hook pre-push go-vet-cgo.sh ; l'autorite est la CI.
cd apps/go-api || exit 0
pkgs=$(go list ./... 2>/dev/null | grep -v "cmd/server" | tr "\n" " ")
CGO_ENABLED=0 go vet $pkgs 2>&1 || true
