export GOCACHE=C:/Users/Guillaume/AppData/Local/go-build-cg-l2 PATH=/c/msys64/ucrt64/bin:$PATH CGO_ENABLED=1; API=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-cg-l2/apps/go-api; C=C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L2
G=$API/internal/games/halo_infinite/film/internal/grammar; M=$C/mut2
TESTS="${TESTS:-.}"
muter() {
  local id=$1 f=$2 s=$3 d=$4
  perl -0pe "$s" $G/$f > $M/$id.go
  if cmp -s $G/$f $M/$id.go; then echo -e "$id\tNON APPLIQUEE\t$d"; return; fi
  echo "{\"Replace\":{\"$(cygpath -m $G/$f)\":\"$(cygpath -m $M/$id.go)\"}}" > $M/$id.json
  (cd $API && go test ./internal/games/halo_infinite/film/internal/grammar/ -overlay=$(cygpath -m $M/$id.json) -run "$TESTS" -count=1 > $M/$id.out 2>&1)
  local rc=$?; local v=VERTE; [ $rc -ne 0 ] && v=ROUGE
  echo -e "$id\t$v\t$d\t$(grep -m1 -E '_test.go:[0-9]+' $M/$id.out | sed 's/^\s*//' | cut -c1-160)"
}
muter K1 debut_de_liste.go 's/tr.DesyncAt == -1 && tr.MasqueNonEcrit == InvariantAucun/tr.DesyncAt == -1/' "M1 rejoue : regle du masque retiree (NEW)"
muter K2 debut_de_liste.go 's/return fin, ok && rec.Trace.MasqueNonEcrit == InvariantAucun/_ = rec\n\t\treturn fin, ok/' "M2 rejoue : regle du masque retiree (delta)"
muter K3 components_device_ti43.go 's/if n > capaciteMoniteurs \{/if n >= capaciteMoniteurs {/' "i31 : N = 8 traite comme echec (borne CMP 8 ; JLE decalee d un)"
muter K4 debut_de_liste.go 's/tr.DesyncAt == -1 && tr.MasqueNonEcrit == InvariantAucun/tr.DesyncAt == -1 \&\& tr.MasqueNonEcrit != InvariantMasqueHorsArchetype/; s/ok && rec.Trace.MasqueNonEcrit == InvariantAucun/ok \&\& rec.Trace.MasqueNonEcrit != InvariantMasqueHorsArchetype/' "pasDEssai : seule la regle hors archetype (dense<=7 et epars non croissant acceptes)"
muter K5 components_device_ti43.go 's/largeurCharges              = 6 /largeurCharges              = 5 /' "i27 : R(5) au lieu de R(6)"
muter K6 components_device_ti43.go 's/case compDeviceInteractionHoldTime, compDeviceInteractionStartTime:/case compDeviceInteractionHoldTime:/' "i40 retire du maillon"
muter K7 components_device_ti43.go 's/largeurControleAnimation    = 256/largeurControleAnimation    = 255/' "i20 : 255 bits"
muter K8 bit_leaf_readers.go 's/const largeurQueue1407f0354 = 5/const largeurQueue1407f0354 = 6/' "queue de FUN_1407f0354 a 6 bits"
muter K9 components_device_ti43.go 's/if br.ReadBit\(\) && br.ReadBits\(largeurPoidsCouche\) != 0 \{/if br.ReadBit() \&\& br.ReadBits(largeurPoidsCouche) > 1 {/' "i35 : code 1 traite comme poids nul"
muter K10 components_device_ti43.go 's/largeurTempsInteraction     = 8 /largeurTempsInteraction     = 9 /' "i25, i40 : Q(9)"
muter K11 components_device_ti43.go 's/largeurVitesseTransition    = 18/largeurVitesseTransition    = 17/' "i38 : Q(17)"
muter K12 components_device_ti43.go 's/br.Skip\(2 \* largeurMotBrut\)/br.Skip(largeurMotBrut)/' "i26 : un seul R(32)"
muter K4n debut_de_liste.go 's/tr.DesyncAt == -1 && tr.MasqueNonEcrit == InvariantAucun/tr.DesyncAt == -1 \&\& tr.MasqueNonEcrit != InvariantMasqueHorsArchetype/' "pasDEssai NEW seul : seule la regle hors archetype"
muter K4d debut_de_liste.go 's/ok && rec.Trace.MasqueNonEcrit == InvariantAucun/ok \&\& rec.Trace.MasqueNonEcrit != InvariantMasqueHorsArchetype/' "pasDEssai delta seul : seule la regle hors archetype"
