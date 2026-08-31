package filmdec

import "testing"

// fire_aim_modal_test.go — CONSTRUCTEUR de records type-36 synthétiques à la grammaire Ghidra
// (fire_aim_modal.go) et vérification du décodeur forward de visée modale.
//
// Ce constructeur écrit un record À LA GRAMMAIRE RÉELLE (préfixe + type 36, trois références
// d'entête, champs a..k, comptes cibles/composantes), au contraire de l'ancre à offsets fixes de
// TestDecodeFireEventLayout. Il sert deux fins opposées : prouver que le forward pose la visée à
// post-comptes+2 sur un vrai record modal (0 cible, 0 composante), et fabriquer des records
// réellement NON modaux que le forward doit refuser (TestDecodeFireEventAimGatedOff, récrit).

// modalHeaderOpts configure l'en-tête modèle-M d'un record type-36 synthétique, jusqu'aux comptes
// exclus. Chaque champ optionnel présent allonge l'en-tête, ce qui déplace la position
// post-comptes — et donc la visée : c'est précisément ce que le forward doit savoir traverser.
type modalHeaderOpts struct {
	ref0, ref1, ref2 bool // références d'entête présentes (ref0 dom1, ref1 dom8, ref2 dom7)
	ref0Wide         bool // ref0 : largeur 9 (true) au lieu de 13
	dExtra, eExtra   bool // champs d / e portent leur charge conditionnelle (R(5) / R(2))
	fWeapon          bool // champ f porte son R(32) (famille d'arme)
}

// writeModalHeader écrit l'en-tête jusqu'à i,j inclus, en miroir EXACT de modalPostCountsBit.
func writeModalHeader(w *bitWriter, o modalHeaderOpts) {
	w.bits(0, 2)               // préfixe config / continuation
	w.bits(modalRecordType, 7) // type 36
	if o.ref0 {                // ref0 : gate + sélecteur de largeur + corps
		w.bit(1)
		if o.ref0Wide {
			w.bit(1) // -> largeur 9
			w.bits(0x2AB, 9+2)
		} else {
			w.bit(0) // -> largeur 13
			w.bits(0x1234, 13+2)
		}
	} else {
		w.bit(0)
	}
	for _, present := range []bool{o.ref1, o.ref2} { // ref1, ref2 : gate + 15 bits si présent
		if present {
			w.bit(1)
			w.bits(0x5A5A, 15)
		} else {
			w.bit(0)
		}
	}
	w.bit(0)        // a : estCourt = 0 (record long)
	w.bit(0)        // b : estBloc = 0
	w.bits(0x2B, 8) // c : attaquant R(7)+R(1)
	if o.dExtra {   // d : polarité Ghidra — gate==0 porte le R(5)
		w.bit(0)
		w.bits(0x1F, 5)
	} else {
		w.bit(1)
	}
	if o.eExtra { // e : gate==0 porte le R(2)
		w.bit(0)
		w.bits(0x3, 2)
	} else {
		w.bit(1)
	}
	if o.fWeapon { // f : gate==1 porte le R(32)
		w.bit(1)
		w.bits(0xDEADBEEF, 32)
	} else {
		w.bit(0)
	}
	w.bits(0x0C0FFEE1, 32) // g : arme variante R(32)
	w.bits(0, 2)           // i, j
}

// writeModalCounts écrit le bloc des comptes en miroir de modalPostCountsBit : 0 cible ET
// 0 composante => modal (une seule porte à 1).
func writeModalCounts(w *bitWriter, targets, comps int) {
	if targets == 0 && comps == 0 {
		w.bit(1) // gate1 = 1 : bloc entier sauté -> record modal
		return
	}
	w.bit(0) // gate1 = 0 : le bloc est lu
	if targets == 1 {
		w.bit(1)
	} else {
		w.bit(0)
		w.bits(uint64(targets), 4)
	}
	if comps == 0 {
		w.bit(1) // gate3 = 1 : composantes non lues
	} else {
		w.bit(0)
		if comps == 1 {
			w.bit(1)
		} else {
			w.bit(0)
			w.bits(uint64(comps), 4)
		}
	}
}

// padModalHead complète le payload jusqu'à porter la tête entière du record (bit 112), pour
// passer la garde de longueur de decodeFireEvent.
func padModalHead(w *bitWriter) {
	for len(w.buf) < (FireHeadBits+7)/8 {
		w.buf = append(w.buf, 0)
	}
}

// buildModalFire écrit un record type-36 MODAL avec sa visée à post-comptes+2, et rend le payload
// et la position de bit attendue de la visée.
func buildModalFire(o modalHeaderOpts, aimCode uint32) (pay []byte, aimBit int) {
	w := &bitWriter{}
	writeModalHeader(w, o)
	writeModalCounts(w, 0, 0)
	w.bits(0, modalAimGap) // les deux drapeaux qui précèdent la visée
	aimBit = w.n
	w.bits(uint64(aimCode), int(FireAimBits))
	padModalHead(w)
	return w.buf, aimBit
}

// buildNonModalFire écrit un record type-36 réellement NON modal (au moins une cible ou une
// composante de dégât) : le forward doit refuser d'y localiser une visée.
func buildNonModalFire(targets, comps int) []byte {
	w := &bitWriter{}
	writeModalHeader(w, modalHeaderOpts{})
	writeModalCounts(w, targets, comps)
	padModalHead(w)
	return w.buf
}

// TestModalAimBitWalksRealisticHeader : le forward atteint post-comptes+2 sur un en-tête RÉALISTE.
//
// Le gabarit porte les trois références d'entête et les champs d/e/f — pas un en-tête à zéros. Le
// point mesuré est que le forward, en traversant cette grammaire, tombe EXACTEMENT sur la position
// où la visée a été écrite (post-comptes + 2), et que la valeur relue est bien celle écrite.
func TestModalAimBitWalksRealisticHeader(t *testing.T) {
	aimCode, ok := EncodeAimVector([3]float32{0.6, 0, 0.8}, FireAimBits)
	if !ok {
		t.Fatal("EncodeAimVector a refusé la largeur 30")
	}
	opts := modalHeaderOpts{
		ref0: true, ref0Wide: true, ref1: true, ref2: true,
		dExtra: true, eExtra: true, fWeapon: true,
	}
	pay, aimBit := buildModalFire(opts, aimCode)

	got, ok := modalAimBit(pay)
	if !ok {
		t.Fatal("modalAimBit refuse un record modal réaliste")
	}
	if got != aimBit {
		t.Fatalf("visée localisée au bit %d, attendue à post-comptes+2 = %d", got, aimBit)
	}
	// Le gabarit ne doit pas coïncider avec les drapeaux du chemin fixe (bit 110/111/112 =
	// 1/0/0), sans quoi l'assertion end-to-end ne dirait pas lequel des deux chemins a posé la
	// visée. On le vérifie pour que ce test prouve bien le CHEMIN MODAL.
	if readBitsAt(pay, 110, 1) == 1 && readBitsAt(pay, 111, 1) == 0 && readBitsAt(pay, 112, 1) == 0 {
		t.Fatal("le gabarit modal déclenche le chemin fixe : choisir un autre en-tête")
	}
	e, dok := decodeFireEvent(pay)
	if !dok || !e.HasAim {
		t.Fatalf("decodeFireEvent : ok=%v HasAim=%v — la visée modale n'est pas posée", dok, e.HasAim)
	}
	if e.Aim[0] < 0.55 || e.Aim[0] > 0.65 || e.Aim[2] < 0.75 || e.Aim[2] > 0.85 {
		t.Errorf("visée modale décodée = %v, attendu ~(0.6, 0, 0.8)", e.Aim)
	}
}

// TestModalAimBitOnMinimalRecord : sur l'en-tête MINIMAL (aucune option), la visée tombe bien
// avant le bit 108 — les drapeaux du chemin fixe sont dans le rembourrage, donc c'est bien le
// forward qui la pose, de bout en bout par decodeFireEvent.
func TestModalAimBitOnMinimalRecord(t *testing.T) {
	aimCode, ok := EncodeAimVector([3]float32{0, 0.8, 0.6}, FireAimBits)
	if !ok {
		t.Fatal("EncodeAimVector a refusé la largeur 30")
	}
	pay, aimBit := buildModalFire(modalHeaderOpts{}, aimCode)
	if got, ok := modalAimBit(pay); !ok || got != aimBit {
		t.Fatalf("modalAimBit = (%d, %v), attendu (%d, true)", got, ok, aimBit)
	}
	if readBitsAt(pay, 110, 1) == 1 && readBitsAt(pay, 111, 1) == 0 && readBitsAt(pay, 112, 1) == 0 {
		t.Fatal("l'en-tête minimal ne devrait pas déclencher le chemin fixe")
	}
	e, dok := decodeFireEvent(pay)
	if !dok || !e.HasAim {
		t.Fatalf("decodeFireEvent : ok=%v HasAim=%v", dok, e.HasAim)
	}
	if e.Aim[1] < 0.75 || e.Aim[1] > 0.85 || e.Aim[2] < 0.55 || e.Aim[2] > 0.65 {
		t.Errorf("visée modale décodée = %v, attendu ~(0, 0.8, 0.6)", e.Aim)
	}
}
