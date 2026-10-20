#!/bin/bash
# re.sh <base|tete> : replay-equiv sur les 20 films du corpus, racine factice (faits et artefacts vides au depart).
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-integ/env.sh
R=$V/repo
rm -rf $R/data/cache/film_facts $R/data/cache/replays $R/data/cache/film_decode.lock
mkdir -p $V/re_$1_tsv
cd $R
$V/bin/$1/replay-equiv.exe -repo-root $(cygpath -m $R) -out-dir $(cygpath -m $V/re_$1_tsv) > $V/re_$1.log 2>&1
echo "rc=$?" >> $V/re_$1.log
