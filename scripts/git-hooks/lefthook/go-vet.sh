#!/usr/bin/env bash
# go vet BLOQUANT sur les cibles que la CI vete (ci.yml, étape « go vet (domain + analysis — sans CGo) ») :
# domain, analysis, games/weapons, CGO desactive (rapide). Une faute de vet dans ces paquets fait
# echouer le commit ; le code sans erreur de vet sort en 0.
# LIMITE STRUCTURELLE (documentee le 2026-08-25) : CGO=0 exclut du build tout
# fichier tagge //go:build cgo (d'ou les « build constraints exclude all Go
# files » dans sa sortie) — une casse dans ces fichiers est INVISIBLE ici.
# Le filet bloquant est le hook pre-push go-vet-cgo.sh ; l'autorite est la CI.
cd apps/go-api || exit 1
CGO_ENABLED=0 go vet ./internal/domain/... ./internal/analysis/... ./internal/games/weapons/...
