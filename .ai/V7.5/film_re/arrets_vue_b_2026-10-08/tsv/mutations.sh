#!/bin/bash
# Mutations du lot (overlay : un fichier de production remplace par une copie mutee). Attendu : ROUGE.
export GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-vueb CGO_ENABLED=1 CC=gcc
API=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-grammaire-arrets-vue-b-2/apps/go-api
G=$API/internal/games/halo_infinite/film/internal/grammar
M=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/bfd187a4-ee4b-4656-9ab4-0654cbc1dfb8/scratchpad/mut
mkdir -p $M
TESTS='TestLesArretsDeLaVueBLisentCeQueLEcrivainEcrit|TestI59|TestChaqueSiteDePositionLitCeQueLeJeuEcrit|Neuf|MarchesDEssai|TestG[0-9]'
muter() {
  local id=$1 f=$2 s=$3 d=$4
  perl -0pe "$s" $G/$f > $M/$id.go
  if cmp -s $G/$f $M/$id.go; then echo -e "$id\tNON APPLIQUEE\t$d"; return; fi
  echo "{\"Replace\":{\"$(cygpath -m $G/$f)\":\"$(cygpath -m $M/$id.go)\"}}" > $M/$id.json
  (cd $API && go test ./internal/games/halo_infinite/film/internal/grammar/ -overlay=$(cygpath -m $M/$id.json) -run "$TESTS" -count=1 > $M/$id.out 2>&1)
  local rc=$?; local v=VERTE; [ $rc -ne 0 ] && v=ROUGE
  echo -e "$id\t$v\t$d\t$(grep -m1 -E '_test.go:[0-9]+' $M/$id.out | sed 's/^\s*//' | cut -c1-140)"
}
(cd $API && go test ./internal/games/halo_infinite/film/internal/grammar/ -run "$TESTS" -count=1 > $M/base.out 2>&1); echo -e "base\trc=$?"
muter V1 components_navpoint_suite.go 's/groupeEtatsVisuelsMotBits = 32/groupeEtatsVisuelsMotBits = 31/' "E1 : mots R(31)"
muter V2 components_navpoint_suite.go 's/groupeEtatsVisuelsVersion = true/groupeEtatsVisuelsVersion = false/' "E1 : v = 0"
muter V3 components_navpoint_suite.go 's/\tfor range k \{\n\t\tbr.ReadBits\(groupeEtatsVisuelsMotBits\)\n\t\}\n//' "E1 : mots par filtre omis"
muter I1 components_managed_object.go 's/return consumeNavpointFilterOnly\(br, level > 1\)/return consumeNavpointFilterOnly(br, level > 2)/' "E2 : v = 2 < param_4"
muter A1 components_biped_anchor.go 's/anchorDrapeauxBits = 6/anchorDrapeauxBits = 7/' "E3 : drapeaux R(7)"
muter A2 components_biped_anchor.go 's/categorieAncreSource    = 5/categorieAncreSource    = 0/' "E3 : categorie 0 au lieu de 5"
muter A3 components_biped_anchor.go 's/\tcase anchorEtiquette4, anchorEtiquette5:/\tcase anchorEtiquette4:/' "E3 : etiquette 5 sans charge"
muter O4 components_managed_objective.go 's/return consumeNavpointFilterOnly\(br, level > 1\)/return consumeNavpointFilterOnly(br, level > 2)/' "E4 : v = 2 < param_4"
muter F1 components_managed_object.go 's/largeurDrapeauxDObjetGere = 2/largeurDrapeauxDObjetGere = 3/' "E5a : R(3)"
muter K1 components_navpoint_suite.go 's/navpointObjectMarkerBits = 32/navpointObjectMarkerBits = 31/' "E5b : R(31)"
muter N1 components_managed_object.go 's/largeurProprieteReseauDObjetGere = 32/largeurProprieteReseauDObjetGere = 31/' "E5c : R(31)"
muter S1 components_matchflow_ti45.go 's/largeurFocusB uint = 4 /largeurFocusB uint = 5 /' "E5d : R(5)"
muter B1 components_navpoint_suite.go 's/navpointBarreBits = 8/navpointBarreBits = 7/' "E5e/f : R(7)"
muter W1 lecteur_position_exceptions.go 's/axes = profile.LargeursAxeParDefautDuBuild\(niveauPosition\)/axes = br.worldObjectPrecision().AxisW; _ = profile.NiveauPositionDObjet/' "P2 : porte posee aux largeurs de la carte"
muter Q1 neufs_prouves.go 's/uint64\(n.bit\) < uint64\(prouveeDes\)/uint64(n.bit) > uint64(prouveeDes)/' "P3 : condition de preuve inversee"
muter Q2 neufs_prouves.go 's/\t\tw.BindFull\(n.id, n.ti\) \/\/ liaison lecture.LiaisonLueNeuf\n//' "P3 : NEW prouve non lie"
