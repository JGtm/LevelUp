#!/bin/bash
# Chaque mutation remplace UN fichier de production par une copie mutee (go test -overlay) et joue
# les tests qui doivent la voir. Attendu : ROUGE pour chacune ; base : VERTE.
export GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-vueb CGO_ENABLED=1 CC=gcc
API=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-grammaire-arrets-vue-b/apps/go-api
G=$API/internal/games/halo_infinite/film/internal/grammar
M=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/bfd187a4-ee4b-4656-9ab4-0654cbc1dfb8/scratchpad/mut
TESTS='TestLesDispositifsLisentCeQueLEcrivainEcrit|TestLesArretsDeLaVueBLisentCeQueLEcrivainEcrit|TestG[0-9]|TestLecteurDeMinuteur|TestCaptureConsumesSameBitsAsDispatch|Ratchet|Site'
muter() {
  local id=$1 f=$2 s=$3 d=$4
  perl -0pe "$s" $G/$f > $M/$id.go
  if cmp -s $G/$f $M/$id.go; then echo -e "$id\tNON APPLIQUEE\t$d"; return; fi
  echo "{\"Replace\":{\"$(cygpath -m $G/$f)\":\"$(cygpath -m $M/$id.go)\"}}" > $M/$id.json
  (cd $API && go test ./internal/games/halo_infinite/film/internal/grammar/ -overlay=$(cygpath -m $M/$id.json) -run "$TESTS" -count=1 > $M/$id.out 2>&1)
  local rc=$?; local v=VERTE; [ $rc -ne 0 ] && v=ROUGE
  echo -e "$id\t$v\t$d\t$(grep -m1 -E '_test.go:[0-9]+' $M/$id.out | sed 's/^\s*//' | cut -c1-150)"
}
if [ "$1" = "base" ]; then (cd $API && go test ./internal/games/halo_infinite/film/internal/grammar/ -run "$TESTS" -count=1 > $M/base.out 2>&1); echo -e "base\trc=$?"; fi
shift
for g in "$@"; do case $g in
C1)
muter K3 components_device_ti43.go 's/if n > capaciteMoniteurs \{/if n >= capaciteMoniteurs {/' "i31 : N = 8 traite comme echec"
muter K5 components_device_ti43.go 's/largeurCharges              = 6 /largeurCharges              = 5 /' "i27 : R(5) au lieu de R(6)"
muter K7 components_device_ti43.go 's/largeurControleAnimation    = 256/largeurControleAnimation    = 255/' "i20 : 255 bits"
muter K9 components_device_ti43.go 's/if br.ReadBit\(\) && br.ReadBits\(largeurPoidsCouche\) != 0 \{/if br.ReadBit() \&\& br.ReadBits(largeurPoidsCouche) > 1 {/' "i35 : code 1 traite comme poids nul"
muter K10 components_device_ti43.go 's/largeurTempsInteraction     = 8 /largeurTempsInteraction     = 9 /' "i25, i40 : Q(9)"
muter K11 components_device_ti43.go 's/largeurVitesseTransition    = 18/largeurVitesseTransition    = 17/' "i38 : Q(17)"
muter K12 components_device_ti43.go 's/br.Skip\(2 \* largeurMotBrut\)/br.Skip(largeurMotBrut)/' "i26 : un seul R(32)"
muter M3 components_device_ti43.go 's/if br.ReadBit\(\) && br.ReadBits\(largeurPoidsCouche\) != 0 \{/if br.ReadBit() { br.ReadBits(largeurPoidsCouche)/' "i35 : condition du poids ignoree"
muter M4 components_device_ti43.go 's/if n > capaciteMoniteurs \{\n\t\treturn false\n\t\}/if n > capaciteMoniteurs {\n\t\tn = capaciteMoniteurs\n\t}/' "i31 : N borne a 8 au lieu d echouer"
muter M5 components_device_ti43.go 's/consume1408f0ac4\(br, categorieReferenceSansSonde\)/consume1408f0ac4(br, categorieReferenceSonde)/' "i36 : categorie 1 au lieu de 0"
muter M6 lecteur_minuteur.go 's/\tlireMinuteur140d580d0\(br, n\)\n\tbr.ReadBits\(n\)/\tlireMinuteur140d580d0(br, n)/' "i37 : troisieme Q(n) de FUN_142ba78dc omis"
muter M7 components_device_ti43.go 's/br.Skip\(largeurMotBrut \+ 3\*largeurReglageCouche14/br.Skip(largeurMotBrut + 2*largeurReglageCouche14/' "i34 : un Q(14) de moins par couche"
muter M8 components_device_ti43.go 's/br.Skip\(largeurMotBrut\)\n\t\tconsumeGateR\(br, largeurIndexDeGroupe\)/br.Skip(largeurMotBrut)/' "i21 : porte du groupe omise"
muter M9 components_device_ti43.go 's/consume1408f0ac4\(br, categorieReferenceSonde\)\n\tcase compDeviceInteractionHoldTime/consume1408f0ac4(br, categorieReferenceSansSonde)\n\tcase compDeviceInteractionHoldTime/' "i24, i29 : categorie 0 au lieu de 1"
muter M11 components_device_ti43.go 's/br.Skip\(largeurEtatEmplacement\)/_ = 0/' "i36 : queue R(3) omise"
muter M12 composants_vue_b_m4b.go 's/return consumeComposantsDispositif\(br, name\)/return consumeComposantsVehiculeTi40(br, name)/' "maillon des dispositifs debranche"
muter M13 components_device_ti43.go 's/largeurPositionAnimation    = 10 /largeurPositionAnimation    = 11 /' "i19 : Q(11)"
muter M14 components_device_ti43.go 's/largeurDrapeauxMachine      = 9 /largeurDrapeauxMachine      = 8 /' "i39 : R(8)"
;;
C2)
muter N1 components_navpoint_suite.go 's/navpointOverrideFlagsBits = 5/navpointOverrideFlagsBits = 4/' "ti=12 i16 : R(4)"
muter N2 dispatch_biped.go 's/\tcase compNavpointOverrideFlags: [^\n]*\n\t\tconsumeNavpointOverrideFlags\(br\)\n//' "ti=12 i16 debranche"
;;
C3)
muter S1 components_matchflow_ti45.go 's/largeurIndexDeSequence uint = 4 /largeurIndexDeSequence uint = 3 /' "ti=45 i0 : index R(3)"
muter S2 components_matchflow_ti45.go 's/motsDeSequence              = 4 /motsDeSequence              = 3 /' "ti=45 i0 : trois mots"
muter S3 components_moteur_de_partie.go 's/\tcase compMatchflowSequenceData: [^\n]*\n\t\tconsumeMatchflowSequenceData\(br\)\n//' "ti=45 i0 debranche"
;;
C4)
muter O1 components_managed_object.go 's/const largeurNavpointDObjetGere = 32/const largeurNavpointDObjetGere = 31/' "ti=10 i2-i17 : R(31)"
muter O2 composants_vue_b_m4b.go 's/\tcase compManagedObjectNavpoint: [^\n]*\n\t\tconsumeManagedObjectNavpoint\(br\)\n//' "ti=10 i2-i17 debranche"
;;
esac; done
