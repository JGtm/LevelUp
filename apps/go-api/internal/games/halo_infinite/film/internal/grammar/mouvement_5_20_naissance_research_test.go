//go:build research

package grammar

// mouvement_5_20_naissance_research_test.go — LE SLOT REJETE EST-IL DANS L IMAGE-CLE DE SON
// CHUNK ? (lot 5.20.2 / 5.20.3, le discriminant).
//
// Deux causes possibles au rejet, et une seule mesure les departage :
//
//	(2) L IMAGE-CLE LE PORTE MAIS NOTRE LECTURE LE MANQUE — le balayeur d ancres s arrete a
//	    45-56 % du payload. Alors une lecture COMPLETE de la table le trouverait.
//	(3) L ENTITE NAIT ENTRE DEUX IMAGES-CLES — l image-cle du chunk ne le porte pas du tout,
//	    et seule celle du chunk SUIVANT le declare.
//
// L instrument cherche donc, pour chaque slot rejete, un en-tete de record portant ce slot
// A TOUTES LES POSITIONS DE BIT du payload d image-cle du MEME chunk — sans la contrainte de
// croissance, sans fenetre, et en acceptant l entree SANS ARCHETYPE (`keyframeArchetypeNone`,
// invisible au filtre fort du balayeur). Il publie ensuite si le chunk SUIVANT, lui, le
// declare.
//
// Rejouable (un film a la fois) :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestNaissance520$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"sort"
	"testing"
)

// n520SlotsDuPayload rend TOUS les slots qu un payload d image-cle porte a une position de bit
// quelconque, avec leur archetype (`-1` = entree sans archetype). Aucune contrainte de
// croissance, aucune fenetre : c est la borne HAUTE de ce que l image-cle peut declarer.
func n520SlotsDuPayload(pay []byte) map[uint32]int {
	total := len(pay) * 8
	out := map[uint32]int{}
	for q := 0; q+keyframeHeaderBits <= total; q++ {
		h, ok := readKeyframeHeader(pay, q, total)
		if !ok {
			continue
		}
		//nolint:gosec // le slot est borne par kfTableCap
		s := uint32(h.Slot)
		if prev, vu := out[s]; !vu || (prev < 0 && h.TI >= 0) {
			out[s] = h.TI
		}
	}
	return out
}

// n520Chunk porte ce qu un chunk declare et ce qu il rejette.
type n520Chunk struct {
	Num      int
	Candidat map[uint32]int // slot -> archetype, toutes positions libres du payload
	Ancres   map[uint32]int // slot -> archetype, ce que le balayeur rend
	Rejets   map[uint32]int // slot -> nombre de rejets
}

// TestNaissance520 departage les deux causes, chunk par chunk.
func TestNaissance520(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	var chunks []n520Chunk
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		ch := n520Chunk{Num: c, Candidat: map[uint32]int{}, Ancres: map[uint32]int{},
			Rejets: map[uint32]int{}}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for s, ti := range n520SlotsDuPayload(pay) {
				ch.Candidat[s] = ti
			}
			for _, r := range WalkKeyframeWorld(pay) {
				ch.Ancres[uint32(r.Slot)] = r.TI //nolint:gosec // borne par le walker
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
			}
		}
		LierTableDeDatums(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, tc.cfg); debut < 0 {
					continue
				}
			}
			mar := t519Marcher(pay, w, tc.cfg, debut)
			if s, ok := t519SlotRejete(pay, tc.cfg, mar); ok {
				ch.Rejets[s]++
			}
		}
		chunks = append(chunks, ch)
	}
	n520Publier(t, chunks)
}

// n520Publier croise, chunk par chunk, les slots rejetes avec ce que l image-cle du MEME chunk
// porte (ancres du balayeur, puis candidats a position libre) et ce que le chunk SUIVANT
// declare.
func n520Publier(t *testing.T, chunks []n520Chunk) {
	t.Helper()
	var totRejets, totAncre, totCandidat, totSuivant, totNullePart int
	for i, ch := range chunks {
		var rejets, dansAncre, dansCandidat, dansSuivant, nullePart int
		slots := make([]uint32, 0, len(ch.Rejets))
		for s := range ch.Rejets {
			slots = append(slots, s)
		}
		sort.Slice(slots, func(a, b int) bool { return ch.Rejets[slots[a]] > ch.Rejets[slots[b]] })
		for _, s := range slots {
			n := ch.Rejets[s]
			rejets += n
			switch {
			case n520Present(ch.Ancres, s):
				dansAncre += n
			case n520Present(ch.Candidat, s):
				dansCandidat += n
			case i+1 < len(chunks) && n520Present(chunks[i+1].Ancres, s):
				dansSuivant += n
			default:
				nullePart += n
			}
		}
		if rejets == 0 {
			continue
		}
		t.Logf("chunk %2d · %5d rejets · balayeur %4d slots, candidats a position libre %5d |"+
			" deja ancre %5d · candidat NON ancre %5d · declare par le chunk SUIVANT %5d ·"+
			" NULLE PART %5d", ch.Num, rejets, len(ch.Ancres), len(ch.Candidat),
			dansAncre, dansCandidat, dansSuivant, nullePart)
		totRejets += rejets
		totAncre += dansAncre
		totCandidat += dansCandidat
		totSuivant += dansSuivant
		totNullePart += nullePart
	}
	t.Logf("TOTAL %d rejets : ancre %d (%.1f %%) · candidat non ancre %d (%.1f %%) ·"+
		" chunk suivant %d (%.1f %%) · nulle part %d (%.1f %%)",
		totRejets, totAncre, m533bPart(totAncre, totRejets),
		totCandidat, m533bPart(totCandidat, totRejets),
		totSuivant, m533bPart(totSuivant, totRejets),
		totNullePart, m533bPart(totNullePart, totRejets))
}

// n520Present dit si la table declare ce slot.
func n520Present(m map[uint32]int, s uint32) bool {
	_, ok := m[s]
	return ok
}

// TestBlocAvantImageCle520 nomme LE BLOC QUI PRECEDE L IMAGE-CLE dans chaque chunk.
//
// POURQUOI CETTE MESURE. Le chemin de chargement d etat du jeu est
// `FUN_1428e2a04` -> `FUN_1428e2a9c`, et il lit DEUX blocs : d abord `FUN_1429883ec`, ensuite
// l image-cle (`FUN_142e2bfd0`). Or `FUN_1429883ec` remplit un tableau de pas **0x18** borne
// par `DAT_144706100` — c est-a-dire LA TABLE DE DATUMS elle-meme (`monde+0x120`, dont
// `FUN_142f30610` lit l archetype en `+4 + slot*0x18` et que `FUN_1408f1618` ecrit). Le depot
// ne lit ce bloc dans aucune de ses marches. Nommer son TYPE, c est nommer la source.
func TestBlocAvantImageCle520(t *testing.T) {
	tc := t516Cadre(t)
	avant := map[int]int{}
	tailles := map[int]int{}
	for _, c := range tc.fc.ChunkNumbers() {
		_, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for i, pk := range pks {
			tailles[int(pk.Type)] += int(pk.Size)
			if pk.Type != PacketTypeKeyframe || i == 0 {
				continue
			}
			avant[int(pks[i-1].Type)]++
			t.Logf("chunk %2d : image-cle (%d o) precedee du type %d (%d o), suivie du type %d",
				c, pk.Size, pks[i-1].Type, pks[i-1].Size, n520TypeSuivant(pks, i))
		}
	}
	t.Logf("TYPES QUI PRECEDENT L IMAGE-CLE : %v", avant)
	t.Logf("OCTETS PAR TYPE DE BLOC : %v", tailles)
}

// n520TypeSuivant rend le type du bloc qui suit l indice i, ou -1 en fin de chunk.
func n520TypeSuivant(pks []FilmPacket, i int) int {
	if i+1 >= len(pks) {
		return -1
	}
	return int(pks[i+1].Type)
}
