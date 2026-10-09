#!/bin/bash
# jouer.sh : chaque mutation en surcouche (-overlay), sur le test du lot ; ROUGE attendu.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LT/env.sh
cd $API; : > $L/mut/resultats.txt
for m in m1 m2 m3 m4 m5; do
  out=$(go test -overlay $(cygpath -m $L/mut/$m.json) ./internal/games/halo_infinite/film/internal/grammar/ -run 'TestUneChaineQuiTraverseUnMasqueNonEcritNeProuveRien' -count=1 2>&1)
  if echo "$out" | grep -q '^ok'; then v=VERT; else v=ROUGE; fi
  echo "$m $v" >> $L/mut/resultats.txt; echo "$out" | grep -E 'debut_de_liste_masque_test.go' | head -4 | sed "s/^/  $m /" >> $L/mut/resultats.txt
done
