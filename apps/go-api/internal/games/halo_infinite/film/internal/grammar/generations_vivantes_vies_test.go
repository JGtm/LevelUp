package grammar

// generations_vivantes_vies_test.go — LA VIE (slot, generation) QUE LE MARCHEUR PORTE, ET LA
// RECUPERATION i48 D UN CORPS DE GENERATION 2 (correction de revue du jalon J5, constat GB-1).
//
// Meme fabrication que generations_vivantes_chemins_test.go : un slot de la mini-bobine 000d5950 est
// scinde a coupureGB1, ses records posterieurs et sa creation passent a la generation 2.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestMarcheurDelta_GenerationPorteeEstCelleDuHandle : chaque record que le MARCHEUR ancre porte la
// generation lue dans son handle (et non une constante), et les chaines par vie qui en dependent
// coupent le slot en deux corps : la chaine i48 du corps de generation 2 ouvre une vie a elle, et
// son emission d arme ne se lit pas contre celle de la generation 1.
func TestMarcheurDelta_GenerationPorteeEstCelleDuHandle(t *testing.T) {
	film := chargerBobineFamilles(t)
	avantFC := NewFilmContext(film)
	_, eqAvant, err := ScanEquipmentChanges(avantFC, nil)
	if err != nil {
		t.Fatalf("equipement de la mini-bobine : %v", err)
	}
	armeAvant := derniereEmissionI45(t, avantFC)
	if armeAvant.Previous == NoWeaponVariant {
		t.Fatalf("slot %d : l emission d arme d apres la coupure ne se lit contre aucune autre sur la "+
			"bobine intacte — le test ne prouverait rien", slotArmeI45)
	}
	scinds := map[uint32]uint64{slotChaineI48: coupureGB1, slotArmeI45: coupureGB1}
	scinderALaGenerationDeux(t, NewFilmContext(film), scinds)
	fc := NewFilmContext(film)
	for slot := range scinds {
		exigerGenerationDeuxVivante(t, fc, slot)
	}
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("decoupage i0 : %v", err)
	}
	faux, parVie := 0, map[types.LifeKey]int{}
	walkDeltaBipedRecords(fc, fc.ChunkNumbers(), fc.BipedSlots(), lay, func(r deltaBipedRecord) {
		attendu := uint32(1)
		if d, ok := scinds[r.Slot]; ok && r.Packet.TimestampUS >= d {
			attendu = 2
		}
		if r.Gen != attendu || r.Vie() != (types.LifeKey{Slot: r.Slot, Gen: attendu}) {
			if faux++; faux <= 5 {
				t.Errorf("record du slot %d (chunk %d, paquet %d) : generation %d portee, %d dans son handle",
					r.Slot, r.Chunk, r.Packet.Index, r.Gen, attendu)
			}
		}
		parVie[types.LifeKey{Slot: r.Slot, Gen: attendu}]++
	})
	if faux > 0 {
		t.Errorf("%d record(s) ancre(s) portent une generation qui n est pas celle de leur handle", faux)
	}
	for slot := range scinds {
		if parVie[types.LifeKey{Slot: slot, Gen: 1}] == 0 || parVie[types.LifeKey{Slot: slot, Gen: 2}] == 0 {
			t.Fatalf("slot %d : %v — les deux corps doivent porter des records", slot, parVie)
		}
	}
	_, eqApres, err := ScanEquipmentChanges(fc, nil)
	if err != nil {
		t.Fatalf("equipement apres scission : %v", err)
	}
	if eqApres.Lives != eqAvant.Lives+1 || eqApres.LivesFirstOffSpec != eqAvant.LivesFirstOffSpec+1 {
		t.Errorf("chaines i48 : %d vie(s), %d a premiere hors norme ; attendu %d et %d — le corps de "+
			"generation 2 du slot %d (c7) se rechaine sur la generation 1 (c5, c6)", eqApres.Lives,
			eqApres.LivesFirstOffSpec, eqAvant.Lives+1, eqAvant.LivesFirstOffSpec+1, slotChaineI48)
	}
	if arme := derniereEmissionI45(t, fc); arme.Previous != NoWeaponVariant {
		t.Errorf("slot %d : l emission d arme du corps de generation 2 se lit contre la famille %d du "+
			"corps de generation 1 (attendu : premiere de sa vie, aucune famille precedente)",
			slotArmeI45, arme.Previous)
	}
}

// derniereEmissionI45 rend l emission d arme du slot slotArmeI45 posterieure a la coupure.
func derniereEmissionI45(t *testing.T, fc *FilmContext) types.HeldWeaponChange {
	t.Helper()
	hw, _, err := ScanHeldWeaponChanges(fc, nil)
	if err != nil {
		t.Fatalf("armes en main : %v", err)
	}
	var out []types.HeldWeaponChange
	for _, h := range hw {
		if h.Slot == slotArmeI45 && h.TimestampUS >= coupureGB1 {
			out = append(out, h)
		}
	}
	if len(out) != 1 {
		t.Fatalf("slot %d : %d emission(s) d arme apres la coupure, attendu 1", slotArmeI45, len(out))
	}
	return out[0]
}

// recordI48ApresLaCoupure rend le premier record avec i0 du slot slotChaineI48 qui porte i48 apres la
// coupure, et le compteur que la marche de production y lit.
func recordI48ApresLaCoupure(t *testing.T, s abilityScanSetup) (deltaBipedRecord, uint32) {
	t.Helper()
	var trouve *deltaBipedRecord
	walkDeltaBipedRecords(s.fc, s.chunks, s.slots, s.gram.lay, func(r deltaBipedRecord) {
		if trouve == nil && r.Slot == slotChaineI48 && r.Packet.TimestampUS >= coupureGB1 && maskHas(r.Mask, i48Index) {
			trouve = &r
		}
	})
	if trouve == nil {
		t.Fatalf("aucun record i48 du slot %d apres la coupure", slotChaineI48)
	}
	last, obs := equipRecoveryHook()
	s.gram.obs = obs
	if !walkRecordTo(trouve.Payload, trouve.I0, trouve.Total, trouve.Mask, s.gram, i48Index) || !last.got {
		t.Fatalf("i48 illisible sur le record retenu du slot %d", slotChaineI48)
	}
	return *trouve, last.counter
}

// enFormeDense recopie un record avec i0 sous la FORME DENSE R(64) que la recuperation gatee lit
// (porte du masque levee, bit k = composant 63-k, i0 juste apres), avec la generation `tag`.
func enFormeDense(r deltaBipedRecord, tag uint32) []byte {
	reste := r.Total - r.I0
	pay := make([]byte, (18+64+reste+7)/8)
	ecrisBits(pay, 0, 1, 1)
	ecrisBits(pay, prefixeDeltaBits, handleSlotBits, r.Slot)
	ecrisBits(pay, prefixeDeltaBits+handleSlotBits, handleGenBits, tag)
	ecrisBits(pay, 17, 1, 1) // porte du masque : forme dense ; le bit 16 reste nul
	for _, comp := range r.Mask {
		ecrisBits(pay, 18+63-comp, 1, 1)
	}
	for k := 0; k < reste; k++ {
		ecrisBits(pay, 18+64+k, 1, uint32(r.Payload[(r.I0+k)/8]>>(7-uint((r.I0+k)%8))&1))
	}
	return pay
}

// TestRecuperationI48_CorpsDeGenerationDeuxComble : la recuperation gatee des emissions i48 d une vie
// retrouve un record de la generation 2 VIVANTE du slot — le filtre qu elle applique est celui des
// generations vivantes du film —, et en refuse un d une generation qu aucune lecture ne designe.
//
// Le niveau exerce est le balayage d UN paquet ([scanEquipRecoveryPacket], le site du filtre), sur le
// contexte du film scinde : la mini-bobine n a aucun saut de compteur a combler, le record candidat
// est donc un record i48 reel du corps de generation 2, recopie sous la forme dense.
func TestRecuperationI48_CorpsDeGenerationDeuxComble(t *testing.T) {
	film := chargerBobineFamilles(t)
	scinderALaGenerationDeux(t, NewFilmContext(film), map[uint32]uint64{slotChaineI48: coupureGB1})
	fc := NewFilmContext(film)
	exigerGenerationDeuxVivante(t, fc, slotChaineI48)
	s, err := resolveAbilityScan(fc)
	if err != nil {
		t.Fatalf("balayage i48 : %v", err)
	}
	r, compteur := recordI48ApresLaCoupure(t, s)
	for _, cas := range []struct {
		gen  uint32
		want int
	}{{2, 1}, {3, 0}} {
		w := equipRecoveryWindow{slot: slotChaineI48, gen: cas.gen, fromC: 5, toC: 0, miss: 2,
			tsMax: ^uint64(0), chunkMin: r.Chunk, chunkMax: r.Chunk}
		last, obs := equipRecoveryHook()
		s.gram.obs = obs
		pk := FilmPacket{Index: r.Packet.Index, TimestampUS: r.Packet.TimestampUS}
		scanEquipRecoveryPacket(s, enFormeDense(r, cas.gen), []*equipRecoveryWindow{&w}, r.Chunk, pk, last)
		if len(w.cands) != cas.want {
			t.Fatalf("generation %d du slot %d (vivantes : 1 et 2) : %d candidat(s) recupere(s), attendu %d",
				cas.gen, slotChaineI48, len(w.cands), cas.want)
		}
		if cas.want == 0 {
			continue
		}
		if c := w.cands[0]; c.Gen != cas.gen || c.Slot != slotChaineI48 || c.Counter != compteur || !c.dense || c.off != 0 {
			t.Errorf("candidat recupere %+v : attendu la vie (%d, %d), compteur c%d, forme dense a l offset 0",
				c, slotChaineI48, cas.gen, compteur)
		}
	}
}
