#!/bin/bash
# usage: carte.sh <exe> <table> <sortie>
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L2/env.sh
cd $API
"$1" -racine $RACINE -films $FILMS,81c02726 -sortie "$3" -table "$2" -mode v2 -denominateur-fixe $NOTES/r_comb2_tsv/r_comb2_denominateurs.tsv -paquets -top 0
