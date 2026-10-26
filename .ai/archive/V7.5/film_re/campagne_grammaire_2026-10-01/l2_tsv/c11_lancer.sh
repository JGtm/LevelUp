#!/bin/bash
# lancer.sh <binaire head|base> <groupe> <films> : passe 2 de TestRComb2 (C11 = sans L7).
S=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad
C=$S/L2/c11; V=$1; G=$2; FILMS=$3
OUT=$C/run/$V/$G; mkdir -p $OUT
export PATH=/c/msys64/ucrt64/bin:$PATH
cd /c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-cg-l2/apps/go-api/internal/games/halo_infinite/film/internal/grammar
unset CAMPAGNE_RLOC_LS CAMPAGNE_RCOMB2_KS CAMPAGNE_RCOMB2_L7
export CAMPAGNE_RACINE=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache/film_chunks
export CAMPAGNE_FILMS=$FILMS
export CAMPAGNE_RCOMB2_RETIRES=L7
export CAMPAGNE_SORTIE=$(cygpath -m $OUT)
export CAMPAGNE_CATALOGUE=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/titles/halo_infinite/reference/map_quant_bounds.json
export CAMPAGNE_CARTES="0797ce72=Live Fire;60ae07c4=Live Fire - Ranked"
export CAMPAGNE_BORNES_sgh_interlock="0:0.266705,51.033180,-9.330649,55.209633,106.171257,78.573174;2:27.834414,23.467054,-9.330649,101.837700,106.171265,78.573174;3:-3500.449951,-3412.616455,-9.330649,4389.524902,4362.876953,880.165344"
date '+debut %F %T' > $OUT/journal.txt
$C/rc2_$V.test.exe -test.run '^TestRComb2$' -test.count=1 -test.timeout 600m -test.v >> $OUT/journal.txt 2>&1
echo "rc=$?" >> $OUT/journal.txt
date '+fin %F %T' >> $OUT/journal.txt
