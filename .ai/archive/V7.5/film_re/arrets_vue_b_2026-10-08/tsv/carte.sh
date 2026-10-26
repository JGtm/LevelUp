#!/bin/bash
# usage: carte.sh <nom> : construit fermeture.exe depuis l arbre du worktree et joue la carte v2 des 20 films dans $S/carte_<nom>
S=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/bfd187a4-ee4b-4656-9ab4-0654cbc1dfb8/scratchpad
W=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-grammaire-arrets-vue-b-2; P=/c/Users/Guillaume/Downloads/Scripts/LevelUp; K=$W/.ai/V7.5/film_re/campagne_grammaire_2026-10-01/kit_gates
export GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-vueb CGO_ENABLED=1 CC=gcc
FILMS=bcb6d393,fb1a1a72,d9781168,c75f33b8,bf15f7ab,51ebbc0f,084a804d,0797ce72,111fa685,e5adf7b2,60ae07c4,a349fea8,a521164d,11de8353,50247b26,bfecd02b,4f77afc1,396cfc92,f75e7053,1c4c63c2
mkdir -p $S/bin_$1 $S/carte_$1
cd $W/apps/go-api && go build -tags=research -o $S/bin_$1/fermeture.exe ./internal/games/halo_infinite/film/research/cmd_fermeture || exit 1
$S/bin_$1/fermeture.exe -racine $P/data/cache/film_chunks -films $FILMS -sortie $S/carte_$1 -table internal/games/halo_infinite/film/internal/grammar/testdata/ecs_table.tsv -mode v2 -denominateur-fixe $K/denominateurs.tsv -paquets -plafond-gib 4 -mpp-declare > $S/carte_$1.log 2>&1
echo "carte rc=$?"; tail -1 $S/carte_$1.log
