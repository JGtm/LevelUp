#!/bin/bash
# construire.sh <nom> [overlay.json] : binaires de mesure. nom=base : git archive 6fa631df0 ; sinon l arbre du worktree (+ surcouche).
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-integ/rev/env.sh
mkdir -p $R/bin/$1
OV=""; [ -n "$2" ] && OV="-overlay=$(cygpath -m $2)"
if [ $1 = base ]; then
  if [ ! -d $R/src_base/apps/go-api ]; then mkdir -p $R/src_base; (cd $WT && git archive $BASE apps/go-api | tar -x -C $R/src_base); fi
  cd $R/src_base/apps/go-api
else
  cd $API
fi
cp internal/games/halo_infinite/film/internal/grammar/testdata/ecs_table.tsv $R/ecs_table_$1.tsv
go build $OV -tags=research -o $R/bin/$1/fermeture.exe ./internal/games/halo_infinite/film/research/cmd_fermeture || echo ECHEC fermeture
if [ -z "$3" ]; then
go build $OV -o $R/bin/$1/killsource.exe ./cmd/killsource || echo ECHEC ks
go build $OV -o $R/bin/$1/replay-equiv.exe ./cmd/replay-equiv || echo ECHEC re
go build $OV -o $R/bin/$1/gate.exe ./cmd/replay-corpus-gate || echo ECHEC gate
fi
echo CONSTRUIT $1
