#!/bin/bash
# mutations.sh : retire chaque regle de l ecrivain, l une apres l autre, et joue les tests qui la
# gardent. Attendu : ROUGE a chaque mutation. Le fichier mute est restaure apres chaque essai.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L0/env.sh
G=$API/internal/games/halo_infinite/film/internal/grammar
RUN='TestOrdreDeLaVueB|TestMasque|TestVueCEcrite|TestPremiereRegleEtEnsemble|TestVerdictDeVueC|TestPaquetFermeAuBit|TestDebutParFermeture|TestFrameClosureDetaillee_SortieParRejetHorsDatum'
muter() { # nom fichier perl
  cp $G/$2 $S/L0/mut_sauve.go
  perl -0pi -e "$3" $G/$2
  if cmp -s $G/$2 $S/L0/mut_sauve.go; then echo "$1 : MUTATION NON APPLIQUEE"; return; fi
  out=$(cd $API && go test -count=1 -run "$RUN" ./internal/games/halo_infinite/film/internal/grammar/ 2>&1)
  cp $S/L0/mut_sauve.go $G/$2
  rouges=$(echo "$out" | grep -E -- '^--- FAIL' | sed 's/--- FAIL: //; s/ (.*//' | tr '\n' ' ')
  if echo "$out" | grep -q -E "build failed|setup failed"; then echo "$1 : COMPILATION EN ECHEC (mutation a reecrire)"; return; fi
  if echo "$out" | grep -q "^ok"; then echo "$1 : VERT (la mutation n est pas vue)"; else echo "$1 : ROUGE ($rouges)"; fi
}
muter "M1 sortie par rejet" ecrivain_invariants.go 's/\tcase rejet:\n\t\tl\.Invariant = InvariantSortieParRejet\n/\tcase false:\n\t\tl.Invariant = InvariantSortieParRejet\n/'
muter "M2 ordre de la vue B" ecrivain_invariants.go 's/&& !j\.noter\(InvariantOrdreVueB\)/\&\& false/'
muter "M3 masque au-dela de l archetype" traverse.go 's/\t\tt\.MasqueNonEcrit = InvariantMasqueHorsArchetype/\t\t_ = InvariantMasqueHorsArchetype/'
muter "M4 masque dense court" ecrivain_invariants.go 's/\t\t\treturn m, InvariantMasqueDenseCourt/\t\t\treturn m, InvariantAucun/'
muter "M5 masque epars non croissant" ecrivain_invariants.go 's/\t\t\tviole = InvariantMasqueEparsNonCroissant/\t\t\tviole = InvariantAucun/'
muter "M7 vue C plus de 32 entrees" ecrivain_invariants.go 's/len\(c\.Entrees\) > entreesVueCMax && !j\.noter\(InvariantVueCTropDEntrees\)/false/'
muter "M9 vue C en-tete cdc04" ecrivain_invariants.go 's/e\.Champs\.Cdc04 != ChampDeControleAbsent && !j\.noter\(InvariantVueCEnTete\)/false/'
muter "M10 vue C code analogique 63" ecrivain_invariants.go 's/e\.Bloc && analogique63 && !j\.noter\(InvariantVueCCodeAnalogique\)/false \&\& analogique63/'
muter "M11 definition retiree (ferme = ferme au bit)" ecrivain_invariants.go 's/\tl\.Fermee = l\.FermeeAuBit && l\.Invariant == InvariantAucun/\tl.Fermee = l.FermeeAuBit/'
muter "M12 debut par fermeture au bit (ancienne regle)" debut_de_liste.go 's/\t\tif l\.Fermee \{\n\t\t\treturn p - extra, true\n\t\t\}\n\t\tif l\.FermeeAuBit && auBit < 0 \{/\t\tif l.FermeeAuBit {\n\t\t\treturn p - extra, true\n\t\t}\n\t\tif l.FermeeAuBit \&\& auBit < 0 {/'
muter "M6 vue C kind" ecrivain_invariants.go 's/k != kindVueCControle && !j\.noter\(InvariantVueCKind\)/k < 0 \&\& !j.noter(InvariantVueCKind)/'
muter "M8 vue C index" ecrivain_invariants.go 's/i > 0 && e\.Index <= c\.Entrees\[i-1\]\.Index && !j\.noter\(InvariantVueCIndex\)/i > 0 \&\& e.Index < 0 \&\& !j.noter(InvariantVueCIndex)/'
