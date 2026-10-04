#!/bin/bash
# gates.sh <suffixe> : gofmt, vet, vet research, archlint, G-film (une commande go a la fois).
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LT/env.sh
S=$1; cd $API
echo "== gofmt -l" > $L/gates_$S.txt; gofmt -l ./internal/ ./cmd/ >> $L/gates_$S.txt 2>&1; echo "rc=$?" >> $L/gates_$S.txt
echo "== go vet ./..." >> $L/gates_$S.txt; go vet ./... >> $L/gates_$S.txt 2>&1; echo "rc=$?" >> $L/gates_$S.txt
echo "== go vet -tags=research film" >> $L/gates_$S.txt; go vet -tags=research ./internal/games/halo_infinite/film/... >> $L/gates_$S.txt 2>&1; echo "rc=$?" >> $L/gates_$S.txt
echo "== archlint" >> $L/gates_$S.txt; go test ./internal/archlint/ -count=1 >> $L/gates_$S.txt 2>&1; echo "rc=$?" >> $L/gates_$S.txt
echo "== G-film" >> $L/gates_$S.txt; go test ./internal/games/halo_infinite/film/... ./internal/replaybuild/... ./internal/sync/killcollector/... -count=1 -timeout 30m > $L/gfilm_$S.txt 2>&1; echo "rc=$?" >> $L/gates_$S.txt
echo FINI >> $L/gates_$S.txt
