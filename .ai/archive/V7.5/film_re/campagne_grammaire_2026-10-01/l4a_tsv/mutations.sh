#!/bin/bash
# mutations.sh : chaque regle neuve du lot L4a retiree ou faussee une a une (copie + go test -overlay) ;
# attendu : ROUGE. Sortie : mutations.txt (une ligne par mutation).
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L4a/env.sh
G=$API/internal/games/halo_infinite/film/internal/grammar
V=$G/composants_vehicule_ti40.go; K=$G/keyframe_fullstate_loop.go; W=$G/unit_weaponstate.go
OUT=$L/mutations.txt; : > $OUT
muter() { # id fichier expression-perl description
  local id=$1 f=$2 e=$3 d=$4
  local c=$L/mut/$id.go
  perl -0pe "$e" $f > $c
  if cmp -s $f $c; then echo -e "$id\tNON APPLIQUEE\t$d" >> $OUT; return; fi
  printf '{"Replace":{"%s":"%s"}}' "$(cygpath -m $f)" "$(cygpath -m $c)" > $L/mut/$id.json
  cd $API
  if go test -overlay $(cygpath -m $L/mut/$id.json) ./internal/games/halo_infinite/film/internal/grammar/ -run 'Ti40|EtatComplet|TestG4|TestG1' -count=1 > $L/mut/$id.log 2>&1; then
    echo -e "$id\tVERTE (non vue)\t$d" >> $OUT
  else
    echo -e "$id\tROUGE\t$d\t$(grep -m1 -- '--- FAIL' $L/mut/$id.log)" >> $OUT
  fi
}
muter M1 $K 's/\tbr\.etatComplet = true\n//' "la marche d etat complet ne pose plus etatComplet"
muter M2 $V 's/(compVehicleTypePhysics: \/\/ i33, i34 : porte de la loi du masque\n)\t\tif br\.etatComplet \{\n\t\t\treturn variant, nil, false\n\t\t\}\n/$1/' "i33/i34 lus en etat complet (porte supposee)"
muter M3 $V 's/(\/\/ largeur du flux\n)\t\tif br\.etatComplet \{\n\t\t\treturn variant, nil, false\n\t\t\}\n/$1/' "composants a largeur du flux lus en etat complet"
muter M4 $V 's/v == 1 \|\| v == 3/v == 1/' "i33 : complement R(6) seulement pour v=1"
muter M5 $V 's/(case compVehicleAutoTurretTriggers: [^\n]*\n\t\tbr\.ReadBit\(\)\n)\t\tbr\.ReadBit\(\)\n/$1/' "i30 : deux bits au lieu de trois"
muter M6 $V 's/largeurVecteurViseeTourelle = 19/largeurVecteurViseeTourelle = 18/' "i31 : R(18)"
muter M7 $V 's/largeurEtatOuverture = 8/largeurEtatOuverture = 7/' "i32 : R(1)+R(7)"
muter M8 $V 's/categorieCibleTourelle = 1/categorieCibleTourelle = 0/' "i35 : categorie 0 (sans sonde)"
muter M9 $V 's/(br\.ReadBits\(largeurEtatSentinelle\)\n)\t\tbr\.ReadBit\(\)\n/$1/' "i36 : sans le R(1)"
muter M10 $V 's/largeurTourelleAuto = 2/largeurTourelleAuto = 3/' "i39 : R(3)"
muter M11 $V 's/categorieParentTourelle = 0/categorieParentTourelle = 1/' "i40 : categorie 1 (sonde)"
muter M12 $V 's/(\t\tbr\.ReadBits\(largeurAngleDeSiege\)\n)\t\tbr\.ReadBits\(largeurAngleDeSiege\)\n/$1/' "i41/i42 : un seul R(8)"
muter M13 $V 's/largeurTempsLargage, largeurHauteurLargage = 2, 14, 8/largeurTempsLargage, largeurHauteurLargage = 2, 13, 8/' "i45 : R(13) au lieu de R(14)"
muter M14 $V 's/	if b {/	if b && false {/' "i46 : second corps ignore"
muter M15 $V 's/\t\tconsumeOpt5\(br\)\n/\t\tbr.ReadBits(5)\n/' "i47 : R(5) sans porte"
muter M16 $W 's/(\tconsumeID2\(br\)                \/\/ FUN_1406d00ec\n)\tconsumeID2\(br\)                \/\/ FUN_1406d00ec\n/$1/' "i38 (FUN_1406d01fc) : un seul FUN_1406d00ec"
muter M17 $V 's/(case compVehicleEmpTimer: [^\n]*\n)/$1\t\tif br.etatComplet {\n\t\t\treturn variant, nil, false\n\t\t}\n/' "i37 refuse en etat complet"
muter M18 $V 's/largeurComplementEtatDeType = 2, 6/largeurComplementEtatDeType = 2, 5/' "i33 : complement R(5)"
