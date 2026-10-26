#!/bin/bash
# construire.sh <base|neuf> : binaires de la base 80d2acd20 (git archive, sans worktree) ou de l arbre du lot.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LT/env.sh
set -e
mkdir -p $L/bin/$1
if [ $1 = base ]; then
  rm -rf $L/src_base; mkdir -p $L/src_base
  cd $WT && git archive $BASE apps/go-api | tar -x -C $L/src_base
  cd $L/src_base/apps/go-api
  cp internal/games/halo_infinite/film/internal/grammar/testdata/ecs_table.tsv $L/ecs_table_base.tsv
else
  cd $API
  cp internal/games/halo_infinite/film/internal/grammar/testdata/ecs_table.tsv $L/ecs_table_neuf.tsv
fi
go build -tags=research -o $L/bin/$1/fermeture.exe ./internal/games/halo_infinite/film/research/cmd_fermeture
go build -o $L/bin/$1/killsource.exe ./cmd/killsource
go build -o $L/bin/$1/replay-equiv.exe ./cmd/replay-equiv
echo CONSTRUIT $1
