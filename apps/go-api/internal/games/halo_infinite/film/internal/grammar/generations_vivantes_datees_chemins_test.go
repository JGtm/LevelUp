package grammar

// generations_vivantes_datees_chemins_test.go — LES LECTEURS DE RECORDS DELTA BIPEDES NE LISENT PAS UN
// CORPS AVANT SA CREATION (lot R2-bis, 2026-09-29), sur la mini-bobine versionnee : le gate de CI de
// la garde datee pour les huit canaux (marcheur `walkDeltaBipedRecords`), la visee seule
// ([ScanBipedAimOnly]) et la recuperation d equipement ([scanEquipRecoveryPacket]).
//
// LA FABRICATION. La mini-bobine n a aucun slot recycle : aucun de ses en-tetes n est anterieur a la
// creation de son corps. Le test pose donc, dans le contexte du film (en memoire, les octets ne sont
// pas touches), des DATES de corps fabriquees pour un slot : un premier corps de generation 2 a
// l instant 0, puis le corps de generation 1 — celui que ses records designent — cree a un instant
// `t` pris au milieu de ses records. Ses records anterieurs a `t` designent alors un corps qui n existe
// pas encore, la forme exacte des en-tetes que le lot ecarte (32 sur `084a804d`) ; les posterieurs
// restent lus.

import (
	"slices"
	"testing"
)

// avecCorpsCreeA rend un contexte du film dont le filtre de generation date le corps (slot, 1) a
// l instant `t`, precede d un corps (slot, 2) cree a l instant 0 (cf. l en-tete).
func avecCorpsCreeA(t *testing.T, fc *FilmContext, slot uint32, tUS uint64) *FilmContext {
	t.Helper()
	g := *fc.GenerationsVivantes()
	if len(g.creations) == 0 {
		t.Fatal("la mini-bobine n a aucune creation datee : le filtre n a pas de dates a porter")
	}
	g.masques = slices.Clone(g.masques)
	g.masques[slot] |= 1<<1 | 1<<2
	g.creations = slices.Clone(g.creations)
	g.creations[slot] = []creationDatee{{tUS: 0, gen: 2}, {tUS: tUS, gen: 1}}
	fc.vies.vivantes = &g
	return fc
}

// instantsDuSlot rend les instants des records du slot que le marcheur rend sur un contexte intact.
func instantsDuSlot(t *testing.T, fc *FilmContext, slot uint32) []uint64 {
	t.Helper()
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("decoupage i0 : %v", err)
	}
	var ts []uint64
	walkDeltaBipedRecords(fc, fc.ChunkNumbers(), fc.BipedSlots(), lay, func(r deltaBipedRecord) {
		if r.Slot == slot {
			ts = append(ts, r.Packet.TimestampUS)
		}
	})
	if len(ts) < 2 {
		t.Fatalf("slot %d : %d record(s) : le test ne prouverait rien", slot, len(ts))
	}
	return ts
}

// avantApres compte les instants strictement anterieurs a `t`, et les autres.
func avantApres(ts []uint64, t uint64) (avant, apres int) {
	for _, x := range ts {
		if x < t {
			avant++
		} else {
			apres++
		}
	}
	return avant, apres
}

// TestMarcheurDelta_CorpsAvantSaCreationRefuse : le marcheur des huit canaux ne rend aucun record
// d un corps anterieur a sa creation, et rend tous les posterieurs.
func TestMarcheurDelta_CorpsAvantSaCreationRefuse(t *testing.T) {
	film := chargerBobineFamilles(t)
	intacts := instantsDuSlot(t, NewFilmContext(film), slotVisee)
	creation := intacts[len(intacts)/2]
	avantIntact, apresIntact := avantApres(intacts, creation)
	if avantIntact == 0 {
		t.Fatalf("aucun record du slot %d avant %d us : le test ne prouverait rien", slotVisee, creation)
	}
	lus := instantsDuSlot(t, avecCorpsCreeA(t, NewFilmContext(film), slotVisee, creation), slotVisee)
	if avant, apres := avantApres(lus, creation); avant != 0 || apres != apresIntact {
		t.Errorf("slot %d, corps cree a %d us : %d record(s) anterieur(s) rendu(s) aux canaux (attendu 0 sur %d), "+
			"%d posterieur(s) (attendu %d)", slotVisee, creation, avant, avantIntact, apres, apresIntact)
	}
}

// TestVisee_CorpsAvantSaCreationRefuse : meme garde pour la visee seule ([ScanBipedAimOnly]).
func TestVisee_CorpsAvantSaCreationRefuse(t *testing.T) {
	film := chargerBobineFamilles(t)
	instants := func(fc *FilmContext) []uint64 {
		l, err := ScanBipedAimOnly(fc)
		if err != nil {
			t.Fatalf("visees : %v", err)
		}
		var ts []uint64
		for _, a := range l {
			if a.Slot == slotVisee {
				ts = append(ts, a.TimestampUS)
			}
		}
		return ts
	}
	intacts := instants(NewFilmContext(film))
	if len(intacts) < 2 {
		t.Fatalf("slot %d : %d visee(s) seule(s) : le test ne prouverait rien", slotVisee, len(intacts))
	}
	creation := intacts[len(intacts)/2]
	avantIntact, apresIntact := avantApres(intacts, creation)
	lues := instants(avecCorpsCreeA(t, NewFilmContext(film), slotVisee, creation))
	if avant, apres := avantApres(lues, creation); avant != 0 || apres != apresIntact {
		t.Errorf("slot %d, corps cree a %d us : %d visee(s) anterieure(s) lue(s) (attendu 0 sur %d), "+
			"%d posterieure(s) (attendu %d)", slotVisee, creation, avant, avantIntact, apres, apresIntact)
	}
}

// TestRecuperationI48_CorpsAvantSaCreationRefuse : la recuperation gatee refuse un record lu dans un
// paquet ANTERIEUR a la creation du corps qu il designe, et le retrouve dans un paquet posterieur.
func TestRecuperationI48_CorpsAvantSaCreationRefuse(t *testing.T) {
	film := chargerBobineFamilles(t)
	s, err := resolveAbilityScan(NewFilmContext(film))
	if err != nil {
		t.Fatalf("balayage i48 : %v", err)
	}
	r := recordI48DuSlot(t, s)
	creation := r.Packet.TimestampUS
	s.fc = avecCorpsCreeA(t, NewFilmContext(film), slotChaineI48, creation)
	for _, cas := range []struct {
		tUS  uint64
		want int
	}{{creation, 1}, {creation - 1, 0}} {
		w := equipRecoveryWindow{slot: slotChaineI48, gen: 1, fromC: 5, toC: 0, miss: 2,
			tsMax: ^uint64(0), chunkMin: r.Chunk, chunkMax: r.Chunk}
		last, obs := equipRecoveryHook()
		s.gram.obs = obs
		pk := FilmPacket{Index: r.Packet.Index, TimestampUS: cas.tUS}
		scanEquipRecoveryPacket(s, enFormeDense(r, 1), []*equipRecoveryWindow{&w}, r.Chunk, pk, last)
		if len(w.cands) != cas.want {
			t.Errorf("record du corps (%d, 1) lu dans un paquet de %d us (corps cree a %d us) : %d candidat(s), "+
				"attendu %d", slotChaineI48, cas.tUS, creation, len(w.cands), cas.want)
		}
	}
}

// recordI48DuSlot rend le premier record avec i0 du slot slotChaineI48 qui porte i48.
func recordI48DuSlot(t *testing.T, s abilityScanSetup) deltaBipedRecord {
	t.Helper()
	var trouve *deltaBipedRecord
	walkDeltaBipedRecords(s.fc, s.chunks, s.slots, s.gram.lay, func(r deltaBipedRecord) {
		if trouve == nil && r.Slot == slotChaineI48 && maskHas(r.Mask, i48Index) {
			trouve = &r
		}
	})
	if trouve == nil {
		t.Fatalf("aucun record i48 du slot %d", slotChaineI48)
	}
	return *trouve
}
