#!/bin/bash
# fixcmp.sh <rev_git_avant> <sed_substitution> : compare chaque fixture de contrat de l arbre a celle
# de <rev_git_avant>, apres substitution sed appliquee a l arbre ; imprime le nombre d octets differents.
cd /c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-campagne-grammaire
F=apps/web/src/features/match-replay/test/fixtures/go
for f in $F/replay_schema_77_*.json.gz; do
  a=$(git show "$1:$f" | gzip -dc | jq -S . | md5sum)
  n=$(gzip -dc $f | sed "$2" | jq -S . | md5sum)
  c=$(gzip -dc $f | grep -o "$3" | wc -l)
  [ "$a" = "$n" ] && echo "$(basename $f) IDENTIQUE hors substitution ($c occurrences)" || echo "$(basename $f) DIFFERENT"
done
