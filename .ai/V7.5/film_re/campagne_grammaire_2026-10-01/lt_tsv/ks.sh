#!/bin/bash
# ks.sh <base|neuf> : killsource json sur les 19 temoins de carte connue (lecture seule du parc).
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LT/env.sh
CAT=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/titles/halo_infinite/reference/map_quant_bounds.json
CACHE=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache
D=$L/ks_$1; mkdir -p $D; : > $D/rc.txt
while IFS=$'\t' read -r id carte; do
  $L/bin/$1/killsource.exe json $id -carte "$carte" -cache $CACHE -catalogue $CAT > $D/$id.json 2> $D/$id.err
  echo "$id rc=$?" >> $D/rc.txt
done < $L/cartes_ls.tsv
