#!/bin/bash
# cmp_equiv.sh : etapes dont le digest differe entre la base (re_avant_tsv) et le lot (re_apres_tsv).
L=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L3a
for f in $L/re_apres_tsv/*.tsv; do
  n=$(basename $f .tsv)
  d=$(join -t$'\t' <(grep -v '^#' $L/re_avant_tsv/$n.tsv | sort) <(grep -v '^#' $f | sort) -a1 -a2 -e MANQUE -o 0,1.2,1.3,2.2,2.3 | awk -F'\t' '$3!=$5{printf "%s(%s->%s) ", $1, $2, $4}')
  echo -e "$n\t${d:-aucune}"
done
