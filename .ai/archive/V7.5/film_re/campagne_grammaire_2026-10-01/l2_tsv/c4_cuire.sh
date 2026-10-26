#!/bin/bash
# cuire.sh : les quatre films, base puis tete, binaires instrumentes (sonde C4), un a la fois.
S=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad
C4=$S/L2/c4; R=$C4/root; mkdir -p $C4/dump $C4/art
export LEVELUP_REPO_ROOT=$(cygpath -m $R)
for spec in "bcb6d393|Cliffhanger" "4f77afc1|Flood Gulch" "084a804d|Fortitude Heavies" "1c4c63c2|Refuge"; do
  f=${spec%%|*}; carte=${spec#*|}
  mid=$(jq -r .matchId $R/facts/facts_$f.json)
  for v in base head; do
    debut=$(date +%s)
    C4_DUMP=$(cygpath -m $C4/dump/$v.$f.tsv) $C4/rb_$v.exe --map "$carte" --title halo_infinite --facts $(cygpath -m $R/facts/facts_$f.json) --mem-gib 3 $mid > $C4/dump/$v.$f.log 2>&1
    rc=$?
    mv $R/data/cache/replays/halo_infinite/$f.json $C4/art/$v.$f.json 2>/dev/null
    echo "$f $v rc=$rc $(( $(date +%s) - debut ))s"
  done
done
