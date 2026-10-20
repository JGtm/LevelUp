#!/bin/bash
# construire.sh <base|tete> : binaires fermeture (research), killsource, replay-equiv, gate.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-integ/env.sh
if [ "$1" = base ]; then cd $V/src_base/apps/go-api; else cd $API; fi
go build -tags=research -o $V/bin/$1/fermeture.exe ./internal/games/halo_infinite/film/research/cmd_fermeture || echo ECHEC fermeture $1
go build -o $V/bin/$1/killsource.exe ./cmd/killsource || echo ECHEC ks $1
go build -o $V/bin/$1/replay-equiv.exe ./cmd/replay-equiv || echo ECHEC re $1
go build -o $V/bin/$1/gate.exe ./cmd/replay-corpus-gate || echo ECHEC gate $1
echo CONSTRUIT $1
