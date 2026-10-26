#!/bin/bash
# cmp_equiv.sh : compare, film par film, les digests de replay-equiv de la base (re_avant_tsv) et du lot (re_apres_tsv).
L=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L4a
A=$L/re_avant_tsv; B=$L/re_apres_tsv
for f in $B/*.tsv; do
  n=$(basename $f .tsv)
  d=$(diff <(grep -v '^#' $A/$n.tsv) <(grep -v '^#' $f) | grep '^>' | awk '{print $2}' | sort -u | tr '\n' ' ')
  echo -e "$n\t${d:-aucune}"
done
