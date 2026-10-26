#!/bin/bash
# mut.sh : mutations du controle, appliquees par -overlay sur la source du lot (git archive 3031a2b25) ;
# survivante = TOUTE la suite du paquet grammar (hors research) reste verte.
C=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LN
source $C/env.sh; export GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg2-ln
API=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-cg2-ln/apps/go-api
G=$API/internal/games/halo_infinite/film/internal/grammar
M=$C/mutc
cd $API
base_suite() {
  if go test ./internal/games/halo_infinite/film/internal/grammar/ -count=1 > $M/base.out 2>&1; then echo "SUITE DE BASE : VERTE"; else echo "SUITE DE BASE : ROUGE"; fi
}
muter() {
  local nom=$1 fic=$2 expr=$3
  local src=$G/$fic dst=$M/$nom.go
  perl -0pe "$expr" $src > $dst
  if cmp -s $src $dst; then echo "$nom : MUTATION NON APPLIQUEE"; return; fi
  printf '{"Replace":{"%s":"%s"}}' "$(cygpath -m $src)" "$(cygpath -m $dst)" > $M/$nom.json
  if go test -overlay=$(cygpath -m $M/$nom.json) ./internal/games/halo_infinite/film/internal/grammar/ -count=1 > $M/$nom.out 2>&1; then
    echo "$nom : VERT (survivante)"
  else
    echo "$nom : ROUGE ($(grep -m3 -o -- '--- FAIL: [A-Za-z0-9_]*' $M/$nom.out | tr '\n' ' '))"
  fi
}
base_suite
muter c1_domaine_genre0 vue_a_genres.go 's/"117\.187\.187\.087v/"217.187.187.087v/'
muter c2_impact_objet_queue vue_a_charges.go 's/br.Skip\(9 \+ 16 \+ 1 \+ 1\)/br.Skip(9 + 16 + 1)/'
muter c3_effet_ia_positions vue_a_charges.go 's/\{1, 3, 0, 0\}/{1, 3, 1, 0}/'
muter c4_chaine_sans_nul vue_a_charges_sacs.go 's/(\t\t\treturn true\n\t\t\}\n\t\}\n)\treturn false/${1}\treturn true/'
muter c5_cycle_ia vue_a_charges.go 's/(consumeGateR\(br, 32\)\n\tconsumeGateR\(br, 32\)\n\t)br.Skip\(32\)/${1}br.Skip(31)/'
muter c6_bit_configuration vue_a_lecture.go 's#if !br\.ReadBit\(\) \{ // FUN_142987460#if br.ReadBit(); false { // FUN_142987460#'
muter c7_degats_drapeaux vue_a_charges.go 's/const drapeauxDeDegats = 15/const drapeauxDeDegats = 14/'
muter c8_queue_equipe vue_a_charges_sacs.go 's/const largeurQueueDEquipe = 9/const largeurQueueDEquipe = 8/'
muter c9_poussee vue_a_charges_armes.go 's/largeurMagnitudePoussee   = 10/largeurMagnitudePoussee   = 9/'
muter c10_ramassage vue_a_charges.go 's/(func chargeRamassage\(br \*Lecteur\) bool \{\n\t)br.Skip\(3\)/${1}br.Skip(2)/'
