//go:build research

package grammar

// mouvement_5_16_source_research_test.go — LA SOURCE DE L ARCHETYPE : L IMAGE-CLE, ET NOTRE
// LECTURE LA TRONQUE (lot 5.16.2).
//
// # CE QUE L ECRIVAIN DIT, PAR ADRESSE
//
// L ecrivain de la liste d entites d une vue est `FUN_142f2e174` (slot `0x10` de la vtable de
// vue) : il parcourt la table de SA vue (`vue+0x38` a `vue+0x40`) sous le bitmap `vue+0x58` et
// rend un mot de 32 bits par entite VIVANTE — `vue+8 << 0x1e | slot & 0x1fff | genre`, genres
// `0x800000` / `0x1000000` / `0x1800000` (donc `e >> 0x17 & 0x7f` vaut 1, 2 ou 3). Chaque mot est
// ensuite SERIALISE par `FUN_142f2c658`, qui aiguille sur le genre : `FUN_142f303bc` (1),
// `FUN_142f304a8` (2), `FUN_142f30610` (3, l etat complet). Et `FUN_142f30610` ecrit, DANS CET
// ORDRE :
//
//	FUN_142f2c754(writer, 3, eid, archetype)   l en-tete : [magic 0xf0c3a57e si HasExtraFields]
//	                                           [1 bit de prefixe][2 bits de type si type != 3]
//	                                           [drapeau + R(8) d archetype si HasExtraFields]
//	                                           l identifiant par FUN_1406d5110 (classe 7)
//	FUN_140769e08(writer, i)                   le SELECTEUR DE BASELINE : 1 bit, + 7 si i != 0xff
//	FUN_142e35e60(...)                         le corps (masque + composants)
//
// et l archetype qu il serialise est `*(int *)(*(vue[0x20] + 0x120) + 4 + slot * 0x18)`.
// **L IMAGE-CLE EST DONC LE DUMP DE LA TABLE DE DATUMS**, et c est la source de l archetype d un
// slot qu aucun NEW n a declare.
//
// # CE QUE LA MESURE AJOUTE, ET C EST LE DEFAUT
//
// `WalkKeyframeWorld` (`keyframe_world.go`) n est pas un parseur : c est un BALAYEUR d ancres,
// borne par une fenetre de recherche de `maxWin` bits. Sur `dad793c7` :
//
//	chunk 1 : 123 records, dernier slot 122, arret au bit 139 754 sur 1 028 032
//	chunks 2..5 : 186 a 187 records, jusqu au slot 1345
//
// et l identifiant du slot rejete par le temoin de 96 bits (1298, `0x40000512`) EST dans
// l image-cle du chunk 1, au bit **279 659**, avec `field26 = 0` et `ti = 47` — soit
// **139 841 bits apres** le point d arret, c est-a-dire AU-DELA de la fenetre de 120 000 bits.
// La table est dans le film ; la fenetre l a coupee.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> //	  go test -tags=research -count=1 -v -timeout 30m //	  -run '^TestTemoin516(ImageCle|Population|Ou|ArretImageCle|Chercher)$' //	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"testing"
)

// TestTemoin516ImageCle compare, sur tout le film, ce que les DEUX lectures de la table
// d image-cle declarent — et croise le manque avec les slots que la vue B REJETTE.
//
// C EST LA MESURE DE VERIFICATION DU MAILLON LU : si le filtre `Field26 == 0` du balayeur cache
// des records, les slots rejetes par la vue B doivent se retrouver dans le marcheur deterministe
// et pas dans le balayeur.
func TestTemoin516ImageCle(t *testing.T) {
	tc := t516Cadre(t)
	ctx := tc.fc.ContexteDeLecture()
	w := NewWorld(tc.reg)
	var totBalaye, totMarche, totSurplus int
	rejetes := map[uint32]int{}
	rejetesDansMarche := map[uint32]bool{}
	rejetesDansBalaye := map[uint32]bool{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		balaye, marche, stops := t516SlotsImageCle(data, pks, tc.reg, ctx)
		surplus := 0
		for s := range marche {
			if _, vu := balaye[s]; !vu {
				surplus++
			}
		}
		totBalaye += len(balaye)
		totMarche += len(marche)
		totSurplus += surplus
		if len(stops) > 0 {
			t.Logf("CHUNK %2d : balayeur %4d slots · marcheur %4d slots · SURPLUS %4d · %v",
				c, len(balaye), len(marche), surplus, stops)
		}
		m533bLierMonde(w, data, pks)
		t516Rejets(tc, data, pks, w, rejetes, balaye, marche, rejetesDansBalaye,
			rejetesDansMarche)
	}
	t.Logf("TOTAL IMAGE-CLE : balayeur %d · marcheur deterministe %d · SURPLUS %d",
		totBalaye, totMarche, totSurplus)
	t.Logf("SLOTS REJETES PAR LA VUE B : %d distincts · declares par le balayeur %d · "+
		"declares par le marcheur deterministe %d", len(rejetes), len(rejetesDansBalaye),
		len(rejetesDansMarche))
	t.Logf("  etendue des slots rejetes : %s", t516Etendue(rejetes))
}

// t516Rejets rejoue les paquets delta d un chunk et recense les slots sur lesquels la vue B
// sort par REJET, puis dit si les deux lectures de l image-cle les declarent.
func t516Rejets(tc t516Temoin, data []byte, pks []FilmPacket, w *World, rejetes map[uint32]int,
	balaye, marche map[uint32]uint32, dansBalaye, dansMarche map[uint32]bool) {
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
		m := t515Marcher(pay, w, tc.cfg, debut)
		if !m.HitEndB || t515SortieVueB(pay, tc.cfg, m) != "rejet de table de vue" {
			continue
		}
		enTete := m.FinVueB - (1 + tc.cfg.IDLowBits + 2)
		if enTete < 0 {
			continue
		}
		br := LecteurSur(pay)
		br.poserCadre(tc.cfg)
		br.Skip(enTete)
		if readRecordType(br) != recDelta {
			continue
		}
		slot := readRecordID(br, tc.cfg.IDLowBits, tc.cfg.IDBase) & 0x3fffffff
		rejetes[slot]++
		if _, ok := balaye[slot]; ok {
			dansBalaye[slot] = true
		}
		if _, ok := marche[slot]; ok {
			dansMarche[slot] = true
		}
	}
}

// t516Etendue rend « min · max · mediane » d un ensemble de slots compte.
func t516Etendue(m map[uint32]int) string {
	if len(m) == 0 {
		return "vide"
	}
	l := make([]int, 0, len(m))
	for k := range m {
		l = append(l, int(k))
	}
	sort.Ints(l)
	return fmt.Sprintf("min %d · max %d · mediane %d", l[0], l[len(l)-1], l[len(l)/2])
}

// TestTemoin516Archetype est L ORACLE D ARCHETYPE DU TEMOIN, et il ne balaie AUCUNE largeur :
// il essaie les archetypes DU REGISTRE (un artefact LU du film, chunk_00) sur le corps du record
// que la marche rejette, et retient ceux pour lesquels le PAQUET FERME — terminateur de la vue B,
// vue C portee, reste a bourrage NUL.
//
// C est la mesure de verification du maillon lu au lot 5.16.1 : la branche vive de
// `FUN_1406cd128` lit le corps d un delta depuis la table de datums (`*(vue+0x20)+0x20`, eid
// ENTIER, archetype a `+0x04`) ; le monde hors ligne n a pas cette table, mais le REGISTRE porte
// la liste complete des archetypes et de leurs composants. Si le modele est le bon, un archetype
// maillon manquant est une source, et elle a une adresse.
func TestTemoin516Population(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	parType := map[int]int{}
	tailleParType := map[int]int{}
	kfSlots := map[uint32]uint32{}
	newSlots := map[uint32]uint32{}
	rejetes := map[uint32]int{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			parType[int(pk.Type)]++
			tailleParType[int(pk.Type)] += pk.Size
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				kfSlots[uint32(r.Slot)] = uint32(r.TI) //nolint:gosec // bornes du walker
			}
		}
		m533bLierMonde(w, data, pks)
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
			recs, _, _ := t515Records(pay, w, tc.cfg, debut)
			for _, r := range recs {
				if r.Type == recNew {
					newSlots[r.Slot] = r.TypeIndex
				}
			}
			m := t515Marcher(pay, w, tc.cfg, debut)
			if !m.HitEndB || t515SortieVueB(pay, tc.cfg, m) != "rejet de table de vue" {
				continue
			}
			enTete := m.FinVueB - (1 + tc.cfg.IDLowBits + 2)
			if enTete < 0 {
				continue
			}
			br := LecteurSur(pay)
			br.poserCadre(tc.cfg)
			br.Skip(enTete)
			if readRecordType(br) != recDelta {
				continue
			}
			rejetes[readRecordID(br, tc.cfg.IDLowBits, tc.cfg.IDBase)&0x3fffffff]++
		}
	}
	t.Logf("TYPES DE PAQUET : %s", c514Hist(parType))
	t.Logf("  octets par type : %s", c514Hist(tailleParType))
	t.Logf("IMAGE-CLE : %d slots · NEW lus : %d slots · REJETES : %d slots",
		len(kfSlots), len(newSlots), len(rejetes))
	inconnus := 0
	for s := range rejetes {
		_, a := kfSlots[s]
		_, b := newSlots[s]
		if !a && !b {
			inconnus++
		}
	}
	t.Logf("  slots rejetes qu AUCUNE source lue ne declare : %d", inconnus)
	t.Logf("ARCHETYPES DU REGISTRE (nom du premier composant) :")
	for _, ti := range []int{4, 30, 35, 40, 47} {
		if ti >= len(tc.reg.Archetypes) {
			continue
		}
		a := tc.reg.Archetypes[ti]
		n := len(a.Components)
		tete := a.Components
		if n > 3 {
			tete = a.Components[:3]
		}
		t.Logf("  ti=%2d : %d composants · %v", ti, n, tete)
	}
}

// TestTemoin516Ou dit, POUR CHAQUE SLOT REJETE, quel paquet d image-cle le declare et avec quel
// archetype — chunk par chunk. C est la mesure qui tranche l hypothese (a) du brief : si un slot
// absent de l image-cle du chunk COURANT figure dans celle d un AUTRE chunk, alors la table que
// l image-cle dump est la MEME (le jeu la porte de bout en bout) et c est notre LECTURE de cette
// table qui est partielle, pas le film qui est muet.
func TestTemoin516Ou(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	type decl struct {
		chunk int
		ti    uint32
	}
	ou := map[uint32][]decl{}
	cible := map[uint32]bool{}
	parChunk := map[int]int{}
	maxSlot := map[int]uint32{}
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
				s := uint32(r.Slot)                                     //nolint:gosec // borne du walker
				ou[s] = append(ou[s], decl{chunk: c, ti: uint32(r.TI)}) //nolint:gosec // idem
				parChunk[c]++
				if s > maxSlot[c] {
					maxSlot[c] = s
				}
			}
		}
		m533bLierMonde(w, data, pks)
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
			m := t515Marcher(pay, w, tc.cfg, debut)
			if !m.HitEndB || t515SortieVueB(pay, tc.cfg, m) != "rejet de table de vue" {
				continue
			}
			enTete := m.FinVueB - (1 + tc.cfg.IDLowBits + 2)
			if enTete < 0 {
				continue
			}
			br := LecteurSur(pay)
			br.poserCadre(tc.cfg)
			br.Skip(enTete)
			if readRecordType(br) != recDelta {
				continue
			}
			cible[readRecordID(br, tc.cfg.IDLowBits, tc.cfg.IDBase)&0x3fffffff] = true
		}
	}
	t.Logf("IMAGE-CLE PAR CHUNK : %s", c514Hist(parChunk))
	for c, s := range maxSlot {
		t.Logf("  chunk %d : slot maximal declare %d", c, s)
	}
	for s := range cible {
		t.Logf("SLOT REJETE %d : declare par %v", s, ou[s])
	}
}

// TestTemoin516ArretImageCle dit OU s arrete le balayeur d image-cle et POURQUOI : pour chaque
// paquet d image-cle, le nombre de records rendus, le dernier slot, le bit atteint, la taille du
// payload, et les bits bruts a la position d arret.
//
// C est l instrument de l item 5.16.2 : le balayeur `WalkKeyframeWorld` s arrete AVANT la fin de
// la table sur certains paquets (chunk 1 de `dad793c7` : 123 records, slot maximal 122, alors que
// les chunks suivants en declarent 187 jusqu au slot 1345). La table de datums que la branche
// vive de `FUN_1406cd128` interroge est donc BIEN dans le film — c est notre lecture qui la
// tronque.
func TestTemoin516ArretImageCle(t *testing.T) {
	tc := t516Cadre(t)
	ctx := tc.fc.ContexteDeLecture()
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			recs := WalkKeyframeWorld(pay)
			dern, bit, ti := -1, -1, -1
			if n := len(recs); n > 0 {
				dern, bit, ti = recs[n-1].Slot, recs[n-1].Bit, recs[n-1].TI
			}
			det, stop := WalkKeyframeRecords(pay, tc.reg, ctx)
			t.Logf("CHUNK %d · image-cle de %d octets (%d bits)", c, len(pay), len(pay)*8)
			t.Logf("  BALAYEUR      : %d records · dernier slot %d (ti %d) au bit %d · "+
				"il reste %d bits", len(recs), dern, ti, bit, len(pay)*8-bit)
			t.Logf("  DETERMINISTE  : %d records · arret %s", len(det), stop)
			if bit >= 0 {
				t.Logf("  BITS a l arret : %s", m511Bits(pay, bit, 256))
			}
		}
	}
}

// TestTemoin516Chercher CHERCHE l identifiant d un slot rejete dans TOUT le payload de chaque
// paquet d image-cle, a toutes les positions de bit — et rend, a chaque occurrence, le mot
// suivant (donc `field26` et `ti`).
//
// C est la mesure qui tranche : si l identifiant du slot 1298 figure dans l image-cle du chunk 1
// au-dela du point ou le balayeur s arrete, la table de datums que la branche vive interroge est
// BIEN dans le film et notre lecture la TRONQUE (hypothese (a) du brief). Si elle n y figure pas,
// la source est ailleurs, et il faut la nommer.
func TestTemoin516Chercher(t *testing.T) {
	tc := t516Cadre(t)
	cibles := []uint32{0x40000512, 0x40000534} // slots 1298 et 1332, generation 1
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			recs := WalkKeyframeWorld(pay)
			arret := 0
			if n := len(recs); n > 0 {
				arret = recs[n-1].Bit
			}
			for _, cible := range cibles {
				var hits []string
				for q := 0; q+64 <= total; q++ {
					if uint32(kfReadBits(pay, q, 32)) != cible {
						continue
					}
					mot := kfReadBits(pay, q+32, 32)
					hits = append(hits, fmt.Sprintf("bit %d (%s l arret) mot1 %#x ti %d",
						q, t516Avant(q, arret), mot, kfReadBits(pay, q+58, 6)))
					if len(hits) >= 6 {
						break
					}
				}
				t.Logf("CHUNK %d · id %#x : %d occurrence(s) listee(s) · %v", c, cible,
					len(hits), hits)
			}
		}
	}
}

// t516Avant dit si une position est avant ou apres l arret du balayeur.
func t516Avant(q, arret int) string {
	if q < arret {
		return "avant"
	}
	return "APRES"
}

// t516TableExhaustive construit la TABLE DE DATUMS d un paquet d image-cle par un balayage
// EXHAUSTIF des positions de bit : a chaque position, les gardes de `kfAnchorFromID` (generation
// non nulle, slot borne, `field26` nul, `ti` sous le cap objet de 50) — mais SANS la contrainte
// de croissance des slots, qui est celle d une MARCHE de records et non d une table.
//
// C est la forme minimale du modele que la branche vive de `FUN_1406cbaa0` interroge : slot ->
// archetype, rien d autre. Rend aussi les conflits (un slot vu avec deux archetypes).
func t516TableExhaustive(pay []byte) (map[uint32]uint32, int) {
	total := len(pay) * 8
	out := map[uint32]uint32{}
	conflits := 0
	for q := 0; q+64 <= total; q++ {
		slot, ti, _, ok := kfAnchorFromID(pay, q, kfReadBits(pay, q, 32), -1, total)
		if !ok {
			continue
		}
		s, t := uint32(slot), uint32(ti) //nolint:gosec // bornes des gardes
		if vu, deja := out[s]; deja && vu != t {
			conflits++
			continue
		}
		out[s] = t
	}
	return out, conflits
}

// TestTemoin516Datums MESURE la table de datums exhaustive : combien de slots, combien de
// conflits, et ce qu elle fait au gate de fermeture des paquets quand la marche s en sert pour
// l archetype d un slot que ni l image-cle balayee ni un NEW n a lie.
func TestTemoin516Datums(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	var paquets, fermes, nul, hors int
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			table, conflits := t516TableExhaustive(pk.Payload(data))
			t.Logf("CHUNK %d · TABLE DE DATUMS : %d slots · %d conflits · slot 1298 -> %d · "+
				"slot 1332 -> %d", c, len(table), conflits, table[1298], table[1332])
			for s, ti := range table {
				if _, lie := w.ArchetypeForSlot(s); !lie {
					w.BindSoft(s, ti)
				}
			}
		}
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
			paquets++
			_, _, curseur := DecodeFrameViewsCurseur(pay, w, tc.cfg, 3, debut)
			reste := len(pay)*8 - curseur
			if reste < 0 || reste > m5116GateOctet {
				hors++
				continue
			}
			fermes++
			if c514ResteNul(pay, curseur) {
				nul++
			}
		}
	}
	t.Logf("GATE AVEC LA TABLE DE DATUMS : %d paquets · fermes %d · a reste NUL %d · "+
		"hors de [0 ; 7] %d", paquets, fermes, nul, hors)
}
