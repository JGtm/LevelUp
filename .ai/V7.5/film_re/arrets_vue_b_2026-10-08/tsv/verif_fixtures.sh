#!/bin/bash
# usage: verif_fixtures.sh <rev ancienne> <rev nouvelle> : compare les fixtures de contrat de HEAD et de l arbre, hors chaine de revision
W=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-grammaire-arrets-vue-b-2
S=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/bfd187a4-ee4b-4656-9ab4-0654cbc1dfb8/scratchpad
cd $W
for f in apps/web/src/features/match-replay/test/fixtures/go/replay_schema_*.json.gz; do
  git show HEAD:$f | gzip -dc > $S/fx_a.json
  gzip -dc $f | sed "s/$2/$1/g" > $S/fx_b.json
  if cmp -s $S/fx_a.json $S/fx_b.json; then echo "identique $(basename $f)"; else echo "DIFFERENT $(basename $f)"; fi
done
