//go:build research

package grammar

// m3_marche_signature_research_test.go — LOT M3.2, UNE REGRESSION A INSTRUIRE : sur 0797ce72, la
// marche des etats de mouvement localise 203 paquets a evenements de MOINS apres la reparation de
// l etat par defaut du bipede. Hypothese : la marche, qui traversait mal le record NEW d un
// bipede et s y arretait, va desormais PLUS LOIN dans le paquet, et lie au monde des records
// NEW mal lus plus loin — dont le slot de la signature (123), que le localisateur ne reconnait
// plus ensuite.
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> M3_SIG=1 \
//	  go test -tags=research -count=1 -v -run '^TestM3SignatureDuMonde$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"testing"
)

// TestM3SignatureDuMonde rejoue la marche de `ScanMovementStates` et publie chaque changement de
// la liaison du slot de signature, avec le paquet qui l a produit.
func TestM3SignatureDuMonde(t *testing.T) {
	if os.Getenv("M3_SIG") == "" {
		t.Skip("M3_SIG absent")
	}
	tc := t516Cadre(t)
	cres, _, _ := ScanBipedCreations(tc.fc)
	naissances := map[[2]int]bool{}
	for _, c := range cres {
		naissances[[2]int{c.Chunk, c.PacketIndex}] = true
	}
	w := NewWorld(tc.reg)
	w.PoserTableAnticipee(ConstruireTableAnticipee(tc.fc))
	cfg := tc.fc.CadreDeBalayage()
	var localises, nonLocalises, changements, changementsNaissance int
	dumps := 0
	for _, ch := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(ch)
		if !ok {
			continue
		}
		w.PoserChunkCourant(ch)
		t525Lier(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := movementStateSkipLeadBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					nonLocalises++
					continue
				}
				localises++
			}
			avant, lieAvant := w.ArchetypeForSlot(marchSignatureSlot)
			recs, _ := DecodeFrameViews(pay, w, cfg, MovementStateViews, debut)
			apres, lieApres := w.ArchetypeForSlot(marchSignatureSlot)
			cible := os.Getenv("M3_SIG_PAQUET")
			if cible != "" && cible == fmt.Sprintf("%d/%d", ch, pk.Index) {
				for _, r := range recs {
					t.Logf("      rec type %d slot %d ti %d desync %d bit %d", r.Type, r.Slot, r.TypeIndex,
						r.DesyncAt, r.HeaderBit)
				}
			}
			if avant != apres || lieAvant != lieApres {
				changements++
				nee := naissances[[2]int{ch, pk.Index}]
				if nee {
					changementsNaissance++
				}
				if dumps < 12 {
					dumps++
					t.Logf("   slot %d : %d(%v) -> %d(%v) au chunk %d paquet %d (naissance dans le paquet : %v)",
						marchSignatureSlot, avant, lieAvant, apres, lieApres, ch, pk.Index, nee)
				}
			}
		}
	}
	t.Logf("== paquets a evenements localises %d, non localises %d ; changements de liaison du slot %d : %d "+
		"(dont %d dans un paquet portant une naissance)", localises, nonLocalises, marchSignatureSlot,
		changements, changementsNaissance)
}
