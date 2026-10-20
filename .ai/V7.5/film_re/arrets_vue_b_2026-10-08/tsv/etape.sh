#!/bin/bash
# usage: etape.sh <rev ancienne> <rev nouvelle> <carte precedente> <carte> : rang, gofmt, tests film, carte v2, gate 2
S=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/bfd187a4-ee4b-4656-9ab4-0654cbc1dfb8/scratchpad
W=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-grammaire-arrets-vue-b-2
export GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-vueb CGO_ENABLED=1 CC=gcc
$S/rang.sh $1 $2 | grep -v "^identique\|^ok"
cd $W/apps/go-api
gofmt -l internal/games/halo_infinite/film/
go test ./internal/games/halo_infinite/film/... -count=1 2>&1 | grep -v "^ok\|no test files" | head -30
$S/carte.sh $4 && $S/gate2.sh $3 $4 && tail -1 $S/gate2_$4_contre_$3.tsv && grep -c BAISSE $S/gate2_$4_contre_$3.tsv
