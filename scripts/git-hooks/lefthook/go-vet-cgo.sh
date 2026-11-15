#!/usr/bin/env bash
# go vet BLOQUANT — CGO actif + tag integration, module entier (_test.go compris).
# GARDE-RAIL pose le 2026-08-25. Classe fermee : une rupture de compilation dans
# un fichier tagge //go:build cgo ou integration traversait tout le filet local —
# le hook pre-commit go-vet.sh est structurellement aveugle (CGO=0 : ces fichiers
# sortent du build ; et || true) et le job CI go-build ne couvre que
# domain/analysis sans CGO. Cas vecu : build_queue_e2e_cgo_test.go appelait
# NewAdminMonitoringHandler avec 11 args au lieu de 10 (auto-merge wt/csrf-ouvrier
# b687c2c39, posterieur au retrait d'ErrorStats c42624dd5), pousse sans bruit ;
# la CI de branche etait deja rouge pour une autre cause, le nouveau rouge etait
# indistinguable. Repare en de615564f. Ce hook aurait bloque le push avec l'erreur
# exacte. L'autorite reste la CI (go-coverage : go test -tags=integration CGO=1).
# Cout mesure le 2026-08-25 (Windows, cache chaud) : ~54 s vert, ~31 s en echec.
# Cache froid (worktree neuf, montee de version Go) : plusieurs minutes (compile
# DuckDB) — assume : on est en pre-push, pas en pre-commit.
# Tags niche HORS filet, ici comme en CI (fichiers a lancement manuel voulu) :
# dev, art_repro, bug_repro, ignore. Ne pas les ajouter sans decision : le hook
# resterait plus strict que la CI et fabriquerait des rouges locaux sans verdict
# CI correspondant.
set -euo pipefail
cd apps/go-api
CGO_ENABLED=1 go vet -tags=integration ./...
