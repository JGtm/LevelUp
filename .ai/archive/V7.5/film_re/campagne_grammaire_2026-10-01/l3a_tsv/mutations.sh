#!/bin/bash
# mutations.sh : une mutation par regle neuve du lot, appliquee par -overlay sur une COPIE du
# fichier (le worktree n est jamais modifie). Attendu : ROUGE a chaque mutation, base verte.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L3a/env.sh
RUN='TestComposantsDuMoteurSuiventLEcrivain|TestCompteDeVolumesNonBorne|TestLecteurDeMinuteurRendLesQuanta|TestLecteurDeMinuteurUnique|TestG1TableSuitLeCode|TestG4LargeursEntieresSuiventLeCode|TestHooks|TestCapture|TestKeyframeClosureRatchet|TestFrameClosureRatchet'
jouer() { # nom overlay
  out=$(cd $API && go test -count=1 ${2:+-overlay=$2} -run "$RUN" ./internal/games/halo_infinite/film/internal/grammar/ 2>&1)
  rouges=$(echo "$out" | grep -E -- '^--- FAIL' | sed 's/--- FAIL: //; s/ (.*//' | sort -u | tr '\n' ' ')
  if echo "$out" | grep -q -E "build failed|setup failed|cannot|undefined"; then echo "$1 : COMPILATION EN ECHEC"; echo "$out" | head -5; return; fi
  if echo "$out" | grep -q "^ok"; then echo "$1 : VERT"; else echo "$1 : ROUGE ($rouges)"; fi
}
muter() { # nom fichier perl
  src=$G/$2; dst=$L/mut/$(echo "$1" | cut -d' ' -f1)_$2
  perl -0pe "$3" $src > $dst
  if cmp -s $src $dst; then echo "$1 : MUTATION NON APPLIQUEE"; return; fi
  printf '{"Replace":{"%s":"%s"}}' "$(cygpath -m $src)" "$(cygpath -m $dst)" > $dst.json
  jouer "$1" "$(cygpath -m $dst.json)"
}
jouer "BASE (sans mutation)" ""
muter "M1 i11 largeur 127" components_moteur_de_partie.go 's/br\.Skip\(largeurPlafondsDeMoteur\)/br.Skip(largeurPlafondsDeMoteur - 1)/'
muter "M2 i13 bits de volume non lus" components_moteur_de_partie.go 's/\tbr\.Skip\(int\(n\)\)/\t_ = n/'
muter "M3 i13 compte sur 12 bits" components_moteur_de_partie.go 's/(largeurCompteVolumes\s+uint) = 13/$1 = 12/'
muter "M4 i14 niveau ignore (forme longue toujours)" components_moteur_de_partie.go 's/if level < niveauLetterboxLong \{/if false \&\& level < niveauLetterboxLong {/'
muter "M5 i14 porte inversee lue droite" components_moteur_de_partie.go 's/\t\tif !br\.ReadBit\(\) \{\n\t\t\tbr\.ReadBits\(largeurLetterboxIndex\)/\t\tif br.ReadBit() {\n\t\t\tbr.ReadBits(largeurLetterboxIndex)/'
muter "M6 i15 etiquette 1 lue en forme courte" components_moteur_de_partie.go 's/lireMinuteur142ba78dc\(br, largeurMinuteurFente\)/lireMinuteur140d580d0(br, largeurMinuteurFente)/'
muter "M7 i15 fente eteinte lue comme une fente" components_moteur_de_partie.go 's/\t\tcase fenteEteinte:\n\t\tcase fenteQuatreChamps:/\t\tcase fenteQuatreChamps:/'
muter "M8 i16 sur 8 bits" components_moteur_de_partie.go 's/(largeurScenarioIntro\s+uint) = 7/$1 = 8/'
muter "M9 i17 sur 7 bits" components_moteur_de_partie.go 's/largeurDrapeauxMatchflow uint = 8/largeurDrapeauxMatchflow uint = 7/'
muter "M10 queue du minuteur sur 4 bits" lecteur_minuteur.go 's/const largeurQueueMinuteur uint = 5/const largeurQueueMinuteur uint = 4/'
muter "M13 maillon non chaine" dispatch_biped.go 's/return consumeMoteurDePartie\(br, name, level\)/return consumeComposantsVueBM4b(br, name)/'
# M11 et M12 : le garde-rail lit les SOURCES sur disque (os.ReadFile), que -overlay ne remplace pas.
# Mutation EN PLACE sur une copie de sauvegarde, restauree et verifiee a l octet apres l essai.
muter_en_place() { # nom fichier perl
  src=$G/$2; sauve=$L/mut/sauve_$2
  cp $src $sauve
  perl -0pi -e "$3" $src
  if cmp -s $src $sauve; then echo "$1 : MUTATION NON APPLIQUEE"; return; fi
  jouer "$1 (en place)" ""
  cp $sauve $src
  cmp -s $src $sauve && echo "  restaure a l octet : $2" || echo "  RESTAURATION EN ECHEC : $2"
}
muter_en_place "M11 copie en ligne revenue (ti=5 i2)" components_player.go 's/\tm := lireMinuteur140d580d0\(br, largeurMinuteurSoftKill\)\n/\ta0 := br.ReadBits(5)\n\ta1 := br.ReadBits(5)\n\ta2 := br.ReadBits(5)\n\tm := Minuteur{A: a0, B: a1, Queue: a2}\n/'
muter_en_place "M12 saut de 37 bits revenu (i12)" components_walk_batch9.go 's/\{ lireMinuteur140d580d0\(br, roundTimerBits\) \}/{ br.Skip(37) }/'
muter_en_place "M14 lecteur hote sans la sequence" lecteur_minuteur.go 's/\ta := br\.ReadBits\(n\)\n\tb := br\.ReadBits\(n\)\n\treturn Minuteur\{A: a, B: b, Queue: br\.ReadBits\(largeurQueueMinuteur\)\}/\tvar m Minuteur\n\tm.A = br.ReadBits(n)\n\tm.Queue = 0\n\tm.B = br.ReadBits(n)\n\tm.Queue = br.ReadBits(largeurQueueMinuteur)\n\treturn m/'
