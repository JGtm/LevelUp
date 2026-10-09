#!/bin/bash
# mutations.sh : chaque regle neuve du lot LN, mutee par -overlay ; le test nomme doit ROUGIR.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LN/env.sh
M=$L/mut
cd $API
muter() { # nom fichier test perl-expr
  local nom=$1 fic=$2 test=$3 expr=$4
  local src=$G/$fic dst=$M/$nom.go
  perl -0pe "$expr" $src > $dst
  if cmp -s $src $dst; then echo "$nom : MUTATION NON APPLIQUEE"; return; fi
  printf '{"Replace":{"%s":"%s"}}' "$(cygpath -m $src)" "$(cygpath -m $dst)" > $M/$nom.json
  if go test -overlay=$(cygpath -m $M/$nom.json) ./internal/games/halo_infinite/film/internal/grammar/ -run "^$test\$" -count=1 > $M/$nom.out 2>&1; then
    echo "$nom : VERT (mutation survivante) [$test]"
  else
    echo "$nom : ROUGE [$test]"
  fi
}
muter m1_genre_vide vue_a_genres.go TestLaTableDesGenresEstCelleDuJeu 's/"187.687.287.007v007/"187.687.287.007.007/'
muter m1b_version_native vue_a_versions.go TestLaTableDuFilmEstUnPrefixeDeLaTableNative 's/return uint32\(versionsNativesDesGenres\[genre\] - (.)0(.)\)/return uint32(versionsNativesDesGenres[0] - ${1}0${2})/'
muter m2_terminateur vue_a_lecture.go TestLaVueALueRendLeDebutDeLaVueB 's/return br.BitPos\(\), arretHorsGenre/return br.BitPos() - 1, arretHorsGenre/'
muter m3_corruption vue_a_lecture.go TestLeControleDeCorruptionSuitLaCharge 's/if cfg.Profil.Grammaire.ControleDeCorruption && br.ReadBit\(\)/if false \&\& br.ReadBit()/'
muter m4_versions vue_a_versions.go TestLaTableDuFilmEstUnPrefixeDeLaTableNative 's/if v != versionNative\(genre\) \{/if false \&\& v != versionNative(genre) {/'
muter m5_polarite_degats vue_a_charges.go TestLesDegatsLisentLaPorteInverseeDuJeu 's/(func chargeDegatsApres\(br \*Lecteur\) bool \{\n\tconsumeGateR\(br, 32\)\s*\/\/ FUN_14080d69c\n\t)consumeGate0R\(br, 5\)/$1consumeGateR(br, 5)/'
muter m6_tir_court vue_a_charges_tir.go TestLeTirCourtSArreteApresSaDirection 's/const largeurTirCourt = 10/const largeurTirCourt = 9/'
muter m7_crochet debut_de_liste.go TestLaVueADonneLeDebutQueLaSignatureNeTrouvePas 's/\t\tif d, lue := lireLaVueA\(pay, cfg\); lue \{\n\t\t\treturn d, lecture.DebutParVueA\n\t\t\}\n//'
muter m8_cardinal vue_a_lecture.go TestLaVueANeDevineRien 's/if genre >= genres \{/if genre > genres {/'
muter m9_polarite_rechargement vue_a_charges.go TestLaVueALueRendLeDebutDeLaVueB 's/(func chargeRechargement\(br \*Lecteur\) bool \{\n\tbr.Skip\(4\)\s*\/\/ FUN_1406cf008 x 4\n\t)consumeGate0R\(br, 5\)/$1consumeGateR(br, 5)/'
muter m10_index_de_plage vue_a_charges_armes.go TestUnIndexDePlageNeSeLitQuAuNiveauDObjet 's/\tt.indexLisible = false\n//'
muter m11_genre_non_porte vue_a_lecture.go TestLaVueANeDevineRien 's/if charge == nil \|\| !charge\(br\) \{/if charge != nil \&\& !charge(br) {/'
