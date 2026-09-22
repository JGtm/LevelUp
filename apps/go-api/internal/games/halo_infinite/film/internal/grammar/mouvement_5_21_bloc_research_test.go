//go:build research

package grammar

// mouvement_5_21_bloc_research_test.go — LE BLOC DE TYPE 1, LU SUR LES FILMS (lot 5.21).
//
// GATE (i) DU LOT : chaque bloc de type 1 de chaque chunk se referme a l octet — 8 191
// entrees de 79 bits, 8 191 masques de composants de 256 bits, cinq mots de queue, au plus
// sept bits de bourrage.
//
// ET LES TROIS MESURES QUI DISENT CE QUE LE BLOC PORTE :
//
//	TestBloc521         la ventilation par chunk : classes d entree, queue, confrontation
//	                    du masque de composants aux bornes de l archetype de l image-cle ;
//	TestBloc521Masques  le masque de 256 bits determine-t-il l archetype ?
//	TestBloc521Rejets   LA MESURE QUI DECIDE : les slots REJETES par la marche sont-ils
//	                    declares vivants par le bloc de leur chunk ?
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestBloc521' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"testing"
)

// b521Bloc rend le bloc de type 1 d un chunk (le dernier s il y en avait plusieurs).
func b521Bloc(t *testing.T, c int, data []byte, pks []FilmPacket) (BlocDeDatums, bool) {
	t.Helper()
	var out BlocDeDatums
	ok := false
	for _, pk := range pks {
		if pk.Type != PacketTypeDatums {
			continue
		}
		b, err := LireBlocDeDatums(pk.Payload(data))
		if err != nil {
			t.Errorf("chunk %d, paquet %d (%d o) : %v", c, pk.Index, pk.Size, err)
			continue
		}
		out, ok = b, true
	}
	return out, ok
}

// b521ImageCle rend le slot -> ti que le balayeur d ancres de l image-cle du chunk declare.
func b521ImageCle(data []byte, pks []FilmPacket) map[uint32]uint32 {
	kf := map[uint32]uint32{}
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			kf[uint32(r.Slot)] = uint32(r.TI) //nolint:gosec // bornes par le walker
		}
	}
	return kf
}

// TestBloc521 joue le GATE (i) et publie la ventilation par chunk.
func TestBloc521(t *testing.T) {
	tc := t516Cadre(t)
	classes := map[string]int{}
	var blocs, entrees, bourrageMax int
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		b, present := b521Bloc(t, c, data, pks)
		if !present {
			t.Logf("chunk %2d : AUCUN bloc de type 1", c)
			continue
		}
		blocs++
		entrees += len(b.Entrees)
		if b.Bourrage > bourrageMax {
			bourrageMax = b.Bourrage
		}
		if b.Bourrage < 0 || b.Bourrage > datumBourrageMax {
			t.Errorf("chunk %d : bourrage %d hors [0 ; %d]", c, b.Bourrage, datumBourrageMax)
		}
		vivantes, horsInvariant := 0, 0
		for _, e := range b.Entrees {
			// L INVARIANT MESURE DU LOT : `+0x04` est la GENERATION, et `+0x01` n en garde
			// que les deux bits de tete de l eid. S il tombe, `+0x04` n est pas ce qu on dit.
			if e.Generation != uint32(e.Gen)+1 {
				horsInvariant++
			}
			classes[fmt.Sprintf("drapeaux %#x · gen %d · generation %d", e.Drapeaux, e.Gen, e.Generation)]++
			if e.Vivante() {
				vivantes++
			}
		}
		kf := b521ImageCle(data, pks)
		dans, hors := b521BornesDuMasque(b, kf, tc.reg)
		if horsInvariant > 0 {
			t.Errorf("chunk %d : %d entrees ou `+0x04` != generation + 1", c, horsInvariant)
		}
		t.Logf("chunk %2d : %d entrees, %d bits lus, bourrage %d · VIVANTES %d "+
			"(image-cle : %d slots) · masque : %d bits dans les bornes de l archetype, %d hors "+
			"· queue %v", c, len(b.Entrees), b.BitsLus, b.Bourrage, vivantes, len(kf),
			dans, hors, b.Queue)
	}
	t.Logf("GATE (i) : %d blocs, %d entrees, bourrage max %d bit(s)", blocs, entrees, bourrageMax)
	t.Logf("CLASSES D ENTREE : %s", b521Classes(classes))
}

// b521BornesDuMasque compte les bits de masque qui tombent DANS et HORS de la liste de
// composants de l archetype que l image-cle donne au slot. C est le test de la semantique :
// un bit hors bornes dirait que le masque n est pas indexe par le composant de l archetype.
func b521BornesDuMasque(b BlocDeDatums, kf map[uint32]uint32, reg *Registry) (dans, hors int) {
	for slot, e := range b.Entrees {
		ti, connu := kf[uint32(slot)] //nolint:gosec // slot < 8191
		if !e.Vivante() || !connu {
			continue
		}
		a, ok := reg.Archetype(int(ti))
		if !ok {
			continue
		}
		for k := 0; k < datumBitmapBits; k++ {
			if !e.Composant(k) {
				continue
			}
			if k < len(a.Components) {
				dans++
			} else {
				hors++
			}
		}
	}
	return dans, hors
}

// b521Classes rend les classes d entree triees par volume.
func b521Classes(h map[string]int) string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return h[keys[i]] > h[keys[j]] })
	s := ""
	for i, k := range keys {
		if i == 12 {
			s += fmt.Sprintf(" (+%d classes)", len(keys)-12)
			break
		}
		s += fmt.Sprintf(" [%s] x%d", k, h[k])
	}
	return s
}

// TestBloc521Masques mesure si le masque de 256 bits DETERMINE l archetype.
func TestBloc521Masques(t *testing.T) {
	tc := t516Cadre(t)
	dico := map[[4]uint64]map[uint32]int{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		b, present := b521Bloc(t, c, data, pks)
		if !present {
			continue
		}
		kf := b521ImageCle(data, pks)
		for slot, e := range b.Entrees {
			ti, connu := kf[uint32(slot)] //nolint:gosec // slot < 8191
			if !e.Vivante() || !connu {
				continue
			}
			if dico[e.Composants] == nil {
				dico[e.Composants] = map[uint32]int{}
			}
			dico[e.Composants][ti]++
		}
	}
	ambigus := 0
	for _, tis := range dico {
		if len(tis) > 1 {
			ambigus++
		}
	}
	t.Logf("MASQUE DE COMPOSANTS -> ARCHETYPE : %d masques distincts, %d ambigus",
		len(dico), ambigus)
}

// TestBloc521Rejets — LA MESURE QUI DECIDE SI LE BLOC PEUT FERMER DES PAQUETS.
//
// Pour chaque paquet que la marche clot sur un REJET de slot (`t519SlotRejete`, l instrument
// du 5.19), elle demande au bloc de type 1 du MEME chunk ce qu il sait de ce slot.
func TestBloc521Rejets(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	var total, vivant, libere, vide, hors int
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		bloc, present := b521Bloc(t, c, data, pks)
		for slot, ti := range b521ImageCle(data, pks) {
			w.BindImageCle(0, slot, ti)
		}
		LierTableDeDatums(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, ev := PacketHeadEventType(pay); ev {
				if debut = marchLocateStrict(pay, w, tc.cfg); debut < 0 {
					continue
				}
			}
			s, rejet := t519SlotRejete(pay, tc.cfg, t519Marcher(pay, w, tc.cfg, debut))
			if !rejet {
				continue
			}
			total++
			switch e, dedans := b521Entree(bloc, present, s); {
			case !dedans:
				hors++
			case e.Vivante():
				vivant++
			case e.Gen != 0 || e.Drapeaux != 0:
				libere++
			default:
				vide++
			}
		}
	}
	t.Logf("SLOT REJETE CONTRE LE BLOC DE TYPE 1 DU MEME CHUNK — %d rejets :", total)
	t.Logf("  VIVANT dans le bloc              : %6d", vivant)
	t.Logf("  trace (generation ou drapeau)    : %6d", libere)
	t.Logf("  entree VIDE                      : %6d", vide)
	t.Logf("  hors bloc                        : %6d", hors)
}

// b521Entree rend l entree d un slot, et dit si le bloc la porte.
func b521Entree(b BlocDeDatums, present bool, slot uint32) (DatumEntry, bool) {
	if !present || int(slot) >= len(b.Entrees) {
		return DatumEntry{}, false
	}
	return b.Entrees[slot], true
}

// ————————————————————————————————————————————————————————————————————————————————————————
// LA LIAISON (lot 5.21.2) — ELLE VIT ICI, ET LA MESURE DIT POURQUOI.
//
// L ordre du jeu est : conteneur vierge (`FUN_142e2aab4`), bloc de TYPE 1, puis IMAGE-CLE qui
// ecrase, puis `FUN_142f22be8` qui applique au monde. Hors ligne l ordre s inverse, parce que
// le bloc ne porte PAS l archetype (5.21.1) : l image-cle NOMME les archetypes, le bloc dit
// quels slots vivent et avec quel MASQUE DE COMPOSANTS, et le masque fait le pont.
//
// CE QU ELLE POSE, MESURE : 0 liaison sur `bfecd02b` (12 685 entrees vivantes, TOUTES deja
// liees par l image-cle) et 1 sur `dad793c7`. Le gate (ii) ne bouge donc d AUCUN paquet —
// `TestBloc521Rejets` l avait annonce, 0 des 23 325 slots rejetes n etant vivant dans le bloc
// de son chunk. Elle n est PAS cablee en production : un chemin de cuisson qui lirait
// 343 019 octets par chunk pour poser zero liaison serait un cout sans contrepartie.
// ————————————————————————————————————————————————————————————————————————————————————————

// b521Liaison ventile ce qu une passe de liaison a fait d un chunk.
type b521Liaison struct {
	vivantes, posees, dejaLiees, ambigus, masqueInconnu int
}

// b521Lier pose dans `w` les liaisons que le bloc de type 1 porte, sans jamais ecraser.
func b521Lier(w *World, bloc BlocDeDatums, kf map[uint32]uint32) b521Liaison {
	var l b521Liaison
	dico := map[[4]uint64]struct {
		ti     uint32
		ambigu bool
	}{}
	for slot, e := range bloc.Entrees {
		ti, nomme := kf[uint32(slot)] //nolint:gosec // slot < 8 191
		if !e.Vivante() || !nomme {
			continue
		}
		if vu, deja := dico[e.Composants]; deja && vu.ti != ti {
			vu.ambigu = true
			dico[e.Composants] = vu
			continue
		} else if deja {
			continue
		}
		dico[e.Composants] = struct {
			ti     uint32
			ambigu bool
		}{ti: ti}
	}
	for slot, e := range bloc.Entrees {
		if !e.Vivante() {
			continue
		}
		l.vivantes++
		s := uint32(slot) //nolint:gosec // slot < 8 191
		if _, lie := w.ArchetypeForSlot(s); lie {
			l.dejaLiees++
			continue
		}
		m, connu := dico[e.Composants]
		switch {
		case !connu:
			l.masqueInconnu++
		case m.ambigu:
			l.ambigus++
		default:
			w.BindDatum(s, m.ti)
			l.posees++
		}
	}
	return l
}

// TestBloc521Liaison joue la liaison sur le film et publie son bilan.
func TestBloc521Liaison(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	var tot b521Liaison
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		kf := b521ImageCle(data, pks)
		for slot, ti := range kf {
			w.BindImageCle(0, slot, ti)
		}
		bloc, present := b521Bloc(t, c, data, pks)
		if !present {
			continue
		}
		l := b521Lier(w, bloc, kf)
		tot.vivantes += l.vivantes
		tot.posees += l.posees
		tot.dejaLiees += l.dejaLiees
		tot.ambigus += l.ambigus
		tot.masqueInconnu += l.masqueInconnu
	}
	t.Logf("LIAISON DU BLOC DE TYPE 1 : %d entrees vivantes · %d POSEES · %d deja liees par "+
		"l image-cle · %d masques ambigus · %d masques inconnus", tot.vivantes, tot.posees,
		tot.dejaLiees, tot.ambigus, tot.masqueInconnu)
}
