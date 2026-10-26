#!/bin/bash
# ks_jouer.sh <variante> : killsource json (surcouche unique) sur les 19 temoins de carte connue,
# sous les bascules de la variante. Sortie : ks/<variante>/<film>.json (+ .err).
SP=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad
V=$1
CAT=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/titles/halo_infinite/reference/map_quant_bounds.json
CACHE=C:/Users/Guillaume/Downloads/Scripts/LevelUp/data/cache
BORNES="0:0.266705,51.033180,-9.330649,55.209633,106.171257,78.573174;2:27.834414,23.467054,-9.330649,101.837700,106.171265,78.573174;3:-3500.449951,-3412.616455,-9.330649,4389.524902,4362.876953,880.165344"
mkdir -p $SP/rcomb2/ks/$V
BIN=$SP/rcomb2/ks_su.exe
LEV=""; LS=""; L7=""
case "$V" in
  prod) BIN=$SP/rcomb2/ks_prod.exe ;;
  inerte) ;;
  *) for x in ${V//+/ }; do
       case $x in
         LS) LS=1 ;;
         L7) L7=1 ;;
         *) LEV="$LEV,$x" ;;
       esac
     done ;;
esac
LEV=${LEV#,}
while IFS=$'\t' read -r id carte; do
  B=""
  case $carte in "Live Fire"*) B=$BORNES ;; esac
  env -u CAMPAGNE_RLOC_LS -u CAMPAGNE_RCOMB2_KS -u CAMPAGNE_RCOMB2_L7 \
    ${LEV:+CAMPAGNE_RCOMB2_KS=$LEV} ${LS:+CAMPAGNE_RLOC_LS=1} ${L7:+CAMPAGNE_RCOMB2_L7=1} \
    CAMPAGNE_RCOMB2_KS_BORNES="$B" \
    $BIN json $id -carte "$carte" -cache $CACHE -catalogue $CAT > $SP/rcomb2/ks/$V/$id.json 2> $SP/rcomb2/ks/$V/$id.err
  echo "$id rc=$?" >> $SP/rcomb2/ks/$V/rc.txt
done < $SP/rcomb2/cartes.tsv
