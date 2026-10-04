#!/bin/bash
# re.sh <base|neuf> : replay-equiv sur les 20 films, racine factice du lot (faits et artefacts vides au depart).
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LT/env.sh
R=$L/repo
rm -rf $R/data/cache/film_facts $R/data/cache/replays $R/data/cache/film_decode.lock
mkdir -p $L/re_$1_tsv
cd $R
$L/bin/$1/replay-equiv.exe -repo-root $(cygpath -m $R) -out-dir $(cygpath -m $L/re_$1_tsv) > $L/re_$1.log 2>&1
echo "rc=$?" >> $L/re_$1.log
