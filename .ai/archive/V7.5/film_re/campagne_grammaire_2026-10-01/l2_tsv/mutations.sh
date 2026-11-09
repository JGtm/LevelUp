#!/bin/bash
# mutations.sh : chaque mutation remplace UN fichier de production par une copie mutee (go test -overlay),
# et joue les tests qui doivent la voir. Attendu : ROUGE (rc != 0) pour chacune ; base : VERT.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L2/env.sh
G=$API/internal/games/halo_infinite/film/internal/grammar; M=$L2/mut
TESTS='TestLesDispositifsLisentCeQueLEcrivainEcrit|TestUneChaineQuiTraverseUnMasqueNonEcritNeProuveRien|TestFUN140d580d0AUnSeulLecteur|TestG4LargeursEntieresSuiventLeCode|TestCaptureConsumesSameBitsAsDispatch'
muter() { # muter <id> <fichier> <perl-subst> <description>
  local id=$1 f=$2 s=$3 d=$4
  perl -0pe "$s" $G/$f > $M/$id.go
  if cmp -s $G/$f $M/$id.go; then echo -e "$id\tNON APPLIQUEE\t$d"; return; fi
  echo "{\"Replace\":{\"$(cygpath -m $G/$f)\":\"$(cygpath -m $M/$id.go)\"}}" > $M/$id.json
  (cd $API && go test ./internal/games/halo_infinite/film/internal/grammar/ -overlay=$(cygpath -m $M/$id.json) -run "$TESTS" -count=1 > $M/$id.out 2>&1)
  local rc=$?; local v=VERTE; [ $rc -ne 0 ] && v=ROUGE
  echo -e "$id\t$v\t$d\t$(grep -m1 -E '^\s+[a-z_0-9]+_test.go:[0-9]+' $M/$id.out | sed 's/^\s*//' | cut -c1-140)"
}
(cd $API && go test ./internal/games/halo_infinite/film/internal/grammar/ -run "$TESTS" -count=1 > $M/base.out 2>&1); echo -e "base\trc=$?"
muter M1 debut_de_liste.go 's/tr.DesyncAt == -1 && tr.MasqueNonEcrit == InvariantAucun/tr.DesyncAt == -1/' "pas de chaine NEW : regle du masque retiree"
muter M2 debut_de_liste.go 's/return fin, ok && rec.Trace.MasqueNonEcrit == InvariantAucun/_ = rec
		return fin, ok/' "pas de chaine delta : regle du masque retiree"
muter M3 components_device_ti43.go 's/if br.ReadBit\(\) && br.ReadBits\(largeurPoidsCouche\) != 0 \{/if br.ReadBit() { br.ReadBits(largeurPoidsCouche)/' "i35 : condition du poids ignoree"
muter M4 components_device_ti43.go 's/if n > capaciteMoniteurs \{\n\t\treturn false\n\t\}/if n > capaciteMoniteurs {\n\t\tn = capaciteMoniteurs\n\t}/' "i31 : N borne a 8 au lieu d echouer"
muter M5 components_device_ti43.go 's/consume1408f0ac4\(br, categorieReferenceSansSonde\)/consume1408f0ac4(br, categorieReferenceSonde)/' "i36 : categorie 1 (sonde) au lieu de 0"
muter M6 components_device_ti43.go 's/\tconsume140d580d0\(br, n\)\n\tbr.Skip\(int\(n\)\)/\tconsume140d580d0(br, n)/' "i37 : troisieme Q(n) de FUN_142ba78dc omis"
muter M7 components_device_ti43.go 's/br.Skip\(largeurMotBrut \+ 3\*largeurReglageCouche14/br.Skip(largeurMotBrut + 2*largeurReglageCouche14/' "i34 : un Q(14) de moins par couche"
muter M8 components_device_ti43.go 's/br.Skip\(largeurMotBrut\)\n\t\tconsumeGateR\(br, largeurIndexDeGroupe\)/br.Skip(largeurMotBrut)/' "i21 : porte du groupe omise"
muter M9 components_device_ti43.go 's/consume1408f0ac4\(br, categorieReferenceSonde\)\n\tcase compDeviceInteractionHoldTime/consume1408f0ac4(br, categorieReferenceSansSonde)\n\tcase compDeviceInteractionHoldTime/' "i24, i29 : categorie 0 au lieu de 1"
muter M10 components_walk_batch9.go 's/consume140d580d0\(br, largeurMinuteurCampagne\)/br.Skip(37)/' "minuteur de campagne recopie hors du lecteur unique"
muter M11 components_device_ti43.go 's/br.Skip\(largeurEtatEmplacement\)/_ = 0/' "i36 : queue R(3) omise"
muter M12 dispatch_biped.go 's/return consumeComposantsDispositif\(br, name\)/return consumeComposantsVueBM4b(br, name)/' "maillon des dispositifs debranche de la chaine"
# M10 bis : le garde-rail du lecteur unique LIT LES SOURCES SUR DISQUE (go/parser), pas la compilation :
# un overlay ne le voit pas. Mutation EN PLACE sur une copie de sauvegarde, rendue a l octet ensuite.
cp $G/components_walk_batch9.go $M/sauve_batch9.go
perl -0pi -e 's/consume140d580d0\(br, largeurMinuteurCampagne\)/br.Skip(37)/' $G/components_walk_batch9.go
(cd $API && go test ./internal/games/halo_infinite/film/internal/grammar/ -run 'TestFUN140d580d0AUnSeulLecteur' -count=1 > $M/M10bis.out 2>&1); rc=$?
cp $M/sauve_batch9.go $G/components_walk_batch9.go
cmp -s $M/sauve_batch9.go $G/components_walk_batch9.go && rendu=rendu || rendu=NON_RENDU
v=VERTE; [ $rc -ne 0 ] && v=ROUGE
echo -e "M10bis\t$v\tminuteur de campagne recopie hors du lecteur unique (en place, fichier $rendu)\t$(grep -m1 -E '_test.go:[0-9]+' $M/M10bis.out | sed 's/^\s*//' | cut -c1-140)"
