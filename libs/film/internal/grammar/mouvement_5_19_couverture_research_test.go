//go:build research

package grammar

// mouvement_5_19_couverture_research_test.go — LA COUVERTURE DE L IMAGE-CLE ET L ORIGINE DE LA
// FAUTE (lot 5.19.2).
//
// Deplacement PUR depuis `mouvement_5_19_temoin_research_test.go`, qui depassait le seuil de
// 500 lignes du ratchet de taille (`archlint/film_file_size_test.go`). Trois mesures, toutes
// sur la meme question : d ou vient le slot que la vue B rejette ?
//
//	TestCouverture519    combien de la table d image-cle le balayeur d ancres lit reellement
//	TestEcartes519       le slot rejete est-il un candidat d ancre ECARTE par la croissance ?
//	TestPremierRejet519  la premiere faute d un chunk a-t-elle une cause en amont ?

import (
	"sort"
	"testing"
)

// TestCouverture519 mesure LA COUVERTURE DE L IMAGE-CLE sur un film dense, chunk par chunk :
// combien de slots le balayeur d ancres rend, jusqu a quel bit il va, ce que la table de datums
// trouve a position libre sur le MEME payload, et combien de slots les deltas du chunk
// referencent sans qu aucune des deux sources ne les ait declares.
//
// C est la mesure de D2 (5.16) — « la chaine de l image-cle se coupe, la fenetre de 120 000 bits
// en est la cause » — portee du film a un joueur au film dense.
func TestCouverture519(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		declares := map[uint32]bool{}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			recs := WalkKeyframeWorld(pay)
			dernier, maxSlot := 0, uint32(0)
			for _, r := range recs {
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
				declares[uint32(r.Slot)] = true //nolint:gosec // borne par le walker
				dernier = r.Bit
				if uint32(r.Slot) > maxSlot { //nolint:gosec // borne par le walker
					maxSlot = uint32(r.Slot) //nolint:gosec // borne par le walker
				}
			}
			table, amb := TableDeDatums(pay)
			var maxDatum uint32
			for s := range table {
				declares[s] = true
				if s > maxDatum {
					maxDatum = s
				}
			}
			t.Logf("chunk %2d image-cle : %d bits · balayeur %d ancres, dernier bit %d "+
				"(%.1f %% du payload), slot max %d · datums %d slots (max %d, %d ambigus)",
				c, len(pay)*8, len(recs), dernier,
				m533bPart(dernier, len(pay)*8), maxSlot, len(table), maxDatum, amb)
		}
		LierTableDeDatums(w, data, pks)
		rejetes, vus := map[uint32]bool{}, 0
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
				rejetes[s] = true
				vus++
			}
		}
		manquants := 0
		for s := range rejetes {
			if !declares[s] {
				manquants++
			}
		}
		t.Logf("  deltas du chunk : %d rejets · %d slots rejetes distincts, dont %d que NI le "+
			"balayeur NI la table de datums de CE chunk ne declarent", vus, len(rejetes),
			manquants)
	}
}

// TestEcartes519 ferme la chaine : les slots que les deltas rejettent sont-ils parmi les
// CANDIDATS D ANCRE que `TableDeDatums` a trouves dans le payload de l image-cle du chunk et que
// la contrainte de CROISSANCE (`plusLongueSuiteCroissante`) a ECARTES ?
//
// `TestCouverture519` a mesure que sur `bfecd02b` chaque image-cle rend 424 a 483 ancres retenues
// pour 1 374 a 1 540 candidats ecartes, et que le balayeur s arrete a 45-56 % du payload. Si les
// slots rejetes sont dans les ecartes, alors la cause du residu n est pas une largeur de
// composant : c est l INCOMPLETUDE de la lecture de la table d image-cle sur un film dense.
func TestEcartes519(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	var dansRetenus, dansEcartes, nullePart, total int
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		retenus, ecartes := map[uint32]bool{}, map[uint32]bool{}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			for _, r := range WalkKeyframeWorld(pay) {
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
			}
			table, _ := TableDeDatums(pay)
			for s := range table {
				retenus[s] = true
			}
			for _, cand := range candidatsDeDatum(pay) {
				if !retenus[cand.slot] {
					ecartes[cand.slot] = true
				}
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
			s, ok := t519SlotRejete(pay, tc.cfg, mar)
			if !ok {
				continue
			}
			total++
			switch {
			case retenus[s]:
				dansRetenus++
			case ecartes[s]:
				dansEcartes++
			default:
				nullePart++
			}
		}
	}
	t.Logf("SLOT REJETE CONTRE LES CANDIDATS D ANCRE DE L IMAGE-CLE DU MEME CHUNK — %d rejets :",
		total)
	t.Logf("  dans les candidats RETENUS par la croissance : %6d (%5.1f %%)",
		dansRetenus, m533bPart(dansRetenus, total))
	t.Logf("  dans les candidats ECARTES par la croissance : %6d (%5.1f %%)",
		dansEcartes, m533bPart(dansEcartes, total))
	t.Logf("  dans AUCUN candidat du payload                : %6d (%5.1f %%)",
		nullePart, m533bPart(nullePart, total))
}

// TestPremierRejet519 dit si la premiere faute d un chunk a une CAUSE EN AMONT dans le meme
// chunk. Pour chaque chunk : le rang du premier paquet fautif parmi les deltas du chunk, le slot
// rejete, et combien de records `NEW` la marche a lus AVANT lui. Si le premier paquet fautif est
// le PREMIER paquet du chunk et qu aucun `NEW` ne l a precede, alors le slot rejete n a ete
// declare par RIEN de lisible : ni l image-cle du chunk, ni un `NEW`.
func TestPremierRejet519(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
			}
		}
		LierTableDeDatums(w, data, pks)
		rang, news, fermes, premier := 0, 0, 0, -1
		var slot uint32
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
			rang++
			mar := t519Marcher(pay, w, tc.cfg, debut)
			s, fautif := t519SlotRejete(pay, tc.cfg, mar)
			if fautif && premier < 0 {
				premier, slot = rang, s
			}
			if !fautif {
				fermes++
			}
			for _, r := range mar.recs {
				if r.Type == recNew {
					news++
				}
			}
		}
		t.Logf("chunk %2d : premier paquet fautif au rang %d sur %d deltas · slot rejete %d · "+
			"%d records NEW lus dans tout le chunk · %d paquets non fautifs",
			c, premier, rang, slot, news, fermes)
	}
}

// TestType9519 recense les paquets par TYPE et dump la tete des paquets de type 9 — le bloc que
// le repartiteur du jeu (`FUN_1428e22c0`, branche `sVar2 == 9`) SAUTE en avancant le curseur
// d octets de la session (`*(session+0xf8) += *(int *)(paquet+2)`) sans en donner un octet a
// personne. Il ne cherche pas une grammaire : il dit ce que le bloc PESE et a quoi il ressemble.
func TestType9519(t *testing.T) {
	tc := t516Cadre(t)
	parType := map[uint16][2]int{}
	tetes, tailles := map[string]int{}, map[int]int{}
	dumps := 0
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, present := tc.fc.ChunkAt(c)
		if !present {
			continue
		}
		for _, pk := range pks {
			e := parType[pk.Type]
			e[0]++
			e[1] += pk.Size
			parType[pk.Type] = e
			if pk.Type != 9 {
				continue
			}
			pay := pk.Payload(data)
			tailles[pk.Size]++
			tetes[m511Bits(pay, 0, 64)]++
			if dumps < 4 {
				dumps++
				t.Logf("  type 9 · chunk %d · paquet %d · %d octets · t=%d us · tete : %s",
					c, pk.Index, pk.Size, pk.TimestampUS, m511Bits(pay, 0, 256))
			}
		}
	}
	types := make([]int, 0, len(parType))
	for k := range parType {
		types = append(types, int(k))
	}
	sort.Ints(types)
	t.Logf("POPULATION DE TYPES :")
	for _, k := range types {
		e := parType[uint16(k)] //nolint:gosec // k vient des cles de la carte, un u16
		t.Logf("  type %2d : %7d paquets · %10d octets", k, e[0], e[1])
	}
	t.Logf("TAILLES DES PAQUETS DE TYPE 9 (les plus frequentes) :")
	t519Restes(t, tailles)
	t.Logf("LES 64 PREMIERS BITS D UN PAQUET DE TYPE 9 :")
	t519Top(t, tetes, 8)
}
