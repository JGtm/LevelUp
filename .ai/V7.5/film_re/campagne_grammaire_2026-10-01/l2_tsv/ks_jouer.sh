#!/bin/bash
# ks_jouer.sh <bin> <dir> : killsource json sur les 19 temoins de carte connue (lecture seule du cache).
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L2/env.sh
CAT=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/titles/halo_infinite/reference/map_quant_bounds.json
CACHE=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache
mkdir -p $2
while IFS=$'\t' read -r id carte; do
  $1 json $id -carte "$carte" -cache $CACHE -catalogue $CAT > $2/$id.json 2> $2/$id.err
  echo "$id rc=$?" >> $2/rc.txt
done < $L2/cartes.tsv
