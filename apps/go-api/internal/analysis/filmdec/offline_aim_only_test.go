package filmdec

// offline_aim_only_test.go — GARDE-RAIL DE GRAMMAIRE de la visee SANS position.
//
// Il tourne dans la suite ORDINAIRE (aucune variable d'environnement, aucun film) : il fabrique
// un payload synthetique bit a bit et verifie que `ScanBipedAimRecords` y lit exactement la
// visee posee. Ses TEMOINS sont ce qui lui donne sa valeur — chacun casse UNE contrainte de
// l'en-tete et doit faire disparaitre la lecture.

import "testing"

// aimBits est un ecrivain de bits MSB-first : le miroir exact de `readBitsAt`.
type aimBits struct {
	b   []byte
	pos int
}

func (w *aimBits) put(v uint32, n int) {
	for i := n - 1; i >= 0; i-- {
		if w.pos>>3 >= len(w.b) {
			w.b = append(w.b, 0)
		}
		if v>>uint(i)&1 == 1 {
			w.b[w.pos>>3] |= 1 << (7 - uint(w.pos&7))
		}
		w.pos++
	}
}

// aimRecord ecrit un record delta bipede dont le masque est `idx` (sans i0) et dont `i21`
// porte (yaw, pitch). `tag` et `zeros` sont exposes pour que les temoins puissent les casser.
func aimRecord(slot uint32, tag, zeros uint32, idx []int, yaw, pitch uint32) []byte {
	w := &aimBits{b: make([]byte, 0, 64)}
	w.put(1, 1)                // prefixe delta
	w.put(slot, bipedSlotBits) // slot
	w.put(tag, 2)              // tag
	w.put(zeros, 2)            // 14e bit d'id + selecteur de baseline
	w.put(uint32(len(idx)), 3) // compteur de masque
	for _, i := range idx {    // index de composants, croissants
		w.put(uint32(i), bipedIndexBits)
	}
	w.put(0, 1)            // i21 : flag0
	w.put(yaw, aimYawBits) // i21 : cap
	w.put(pitch, aimPitchBits)
	w.put(0, 64) // queue : de quoi ne jamais tronquer la lecture
	return w.b
}

func TestScanBipedAimRecordsGrammaire(t *testing.T) {
	const slot = 517
	bande := map[uint32]bool{slot: true}
	got := ScanBipedAimRecords(aimRecord(slot, 1, 0, []int{21, 25}, 1234, 1100), bande)
	if len(got) != 1 {
		t.Fatalf("masque i21,i25 : %d lecture(s), attendu 1", len(got))
	}
	if got[0].Slot != slot || got[0].YawRaw != 1234 || got[0].PitchRaw != 1100 {
		t.Fatalf("lu slot=%d yaw=%d pitch=%d, attendu %d/1234/1100",
			got[0].Slot, got[0].YawRaw, got[0].PitchRaw, slot)
	}
	// Les deux angles doivent sortir avec la convention DEJA publiee : le meme quantum lu par
	// un record porteur de position donne le meme degre.
	ref := BipedPosition{}
	ref.HasYaw, ref.YawRaw, ref.PitchRaw = true, 1234, 1100
	h, _ := ref.AimHeadingDeg()
	p, _ := ref.AimPitchDeg()
	if got[0].AimHeadingDeg() != h || got[0].AimPitchDeg() != p {
		t.Fatalf("conventions divergentes : %.4f/%.4f contre %.4f/%.4f",
			got[0].AimHeadingDeg(), got[0].AimPitchDeg(), h, p)
	}
}

func TestScanBipedAimRecordsTemoins(t *testing.T) {
	const slot = 517
	bande := map[uint32]bool{slot: true}
	for _, cas := range []struct {
		nom  string
		pay  []byte
		band map[uint32]bool
	}{
		{"tag != 1 (le filtre bipede eprouve)", aimRecord(slot, 2, 0, []int{21, 25}, 1234, 1100), bande},
		{"couple de zeros casse", aimRecord(slot, 1, 1, []int{21, 25}, 1234, 1100), bande},
		{"slot hors bande", aimRecord(slot, 1, 0, []int{21, 25}, 1234, 1100), map[uint32]bool{999: true}},
		{"masque declarant i0 (releve de ScanBipedRecords)", aimRecord(slot, 1, 0, []int{0, 21}, 1234, 1100), bande},
		{"masque sans i21", aimRecord(slot, 1, 0, []int{22, 25}, 1234, 1100), bande},
		{"composant non modelise avant i21", aimRecord(slot, 1, 0, []int{9, 21}, 1234, 1100), bande},
	} {
		if got := ScanBipedAimRecords(cas.pay, cas.band); len(got) != 0 {
			t.Errorf("temoin %q : %d lecture(s), attendu 0 (%+v)", cas.nom, len(got), got)
		}
	}
}

// TestScanBipedAimRecordsPrecede verifie que les composants ANTERIEURS a i21 sont bien
// consommes par leurs detenteurs : le masque `i5,i21` place i21 apres un bouclier, et la visee
// doit rester lisible au bon bit.
func TestScanBipedAimRecordsPrecede(t *testing.T) {
	const slot = 517
	w := &aimBits{b: make([]byte, 0, 64)}
	w.put(1, 1)
	w.put(slot, bipedSlotBits)
	w.put(1, 2)
	w.put(0, 2)
	w.put(2, 3)
	w.put(5, bipedIndexBits)
	w.put(21, bipedIndexBits)
	// i5 object-shield-vitality, chemin le plus court : la largeur vient de son detenteur,
	// on la mesure ici plutot que de la recopier.
	deb := w.pos
	w.put(0, shieldVitalityMinBits+32)
	br := NewBitReader(w.b)
	br.SetBitPos(deb)
	decodeObjectShieldVitality(br)
	fin := br.BitPos()
	w2 := &aimBits{b: append([]byte(nil), w.b...), pos: fin}
	w2.put(0, 1)
	w2.put(777, aimYawBits)
	w2.put(1024, aimPitchBits)
	w2.put(0, 64)
	got := ScanBipedAimRecords(w2.b, map[uint32]bool{slot: true})
	if len(got) != 1 || got[0].YawRaw != 777 || got[0].PitchRaw != 1024 {
		t.Fatalf("masque i5,i21 : %+v, attendu une lecture yaw=777 pitch=1024", got)
	}
}
