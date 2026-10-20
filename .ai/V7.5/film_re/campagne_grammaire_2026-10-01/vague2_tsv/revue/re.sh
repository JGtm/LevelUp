#!/bin/bash
# re.sh <nom> : replay-equiv sur les 20 films du corpus, racine factice (faits et artefacts vides au depart).
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-integ/rev/env.sh
RP=$R/repo
rm -rf $RP/data/cache/film_facts $RP/data/cache/replays $RP/data/cache/film_decode.lock
mkdir -p $R/re_$1_tsv
cd $RP
$R/bin/$1/replay-equiv.exe -repo-root $(cygpath -m $RP) -out-dir $(cygpath -m $R/re_$1_tsv) > $R/re_$1.log 2>&1
echo "rc=$?" >> $R/re_$1.log
