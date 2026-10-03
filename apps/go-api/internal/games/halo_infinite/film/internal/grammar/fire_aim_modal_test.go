package grammar

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
	// Valeurs écrites (lot M4b) : zéro garde celles du gabarit historique.
	ref0Index   uint64 // index de la ref0 (défaut 0x2AB en largeur 9, 0x1234 en largeur 13)
	tireur      uint64 // champ d quand dExtra (défaut 0x1F)
	numero      uint64 // champ c, huit bits bruts R(7)+R(1) (défaut 0x2B)
	armeHaute   uint64 // champ f quand fWeapon (défaut 0xDEADBEEF)
	armeBasse   uint64 // champ g (défaut 0x0C0FFEE1)
	court, bloc bool   // drapeaux a / b
}

// refBrute rend les bits [index][génération] d'une référence : l'index demandé, génération 0 ;
// ou, sans index demandé, la valeur brute du gabarit historique.
func refBrute(index, brute uint64) uint64 {
	if index == 0 {
		return brute
	}
	return index << 2
}

// ouDefaut rend v, ou d quand v vaut zéro.
func ouDefaut(v, d uint64) uint64 {
	if v == 0 {
		return d
	}
	return v
}

// writeModalHeader écrit l'en-tête jusqu'à i,j inclus, en miroir EXACT de modalPostCountsBit.
func writeModalHeader(w *bitWriter, o modalHeaderOpts) {
	w.bits(0b11, 2)        // préfixe : configuration et continuation, posés comme dans le film
	w.bits(TypeTirArme, 7) // type 36
	if o.ref0 {            // ref0 : gate + sélecteur de largeur + corps
		w.bit(1)
		if o.ref0Wide {
			w.bit(1) // -> largeur 9
			w.bits(refBrute(o.ref0Index, 0x2AB), 9+2)
		} else {
			w.bit(0) // -> largeur 13
			w.bits(refBrute(o.ref0Index, 0x1234), 13+2)
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
	w.bit(b2u(o.court))                 // a : estCourt
	w.bit(b2u(o.bloc))                  // b : estBloc
	w.bits(ouDefaut(o.numero, 0x2B), 8) // c : numéro de tir R(7)+R(1)
	if o.dExtra {                       // d : polarité Ghidra — gate==0 porte le R(5)
		w.bit(0)
		w.bits(ouDefaut(o.tireur, 0x1F), 5)
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
		w.bits(ouDefaut(o.armeHaute, 0xDEADBEEF), 32)
	} else {
		w.bit(0)
	}
	w.bits(ouDefaut(o.armeBasse, 0x0C0FFEE1), 32) // g : arme variante R(32)
	w.bits(0, 2)                                  // i, j
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

// padModalHead complète le payload de seize octets nuls : la tête et les comptes y tiennent
// toujours, et la garde de longueur de decodeFireEvent ne dépend plus d'un offset fixe.
func padModalHead(w *bitWriter) {
	w.buf = append(w.buf, make([]byte, 16)...)
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
	e, dok := decodeFireEvent(pay)
	if !dok || !e.HasAim {
		t.Fatalf("decodeFireEvent : ok=%v HasAim=%v — la visée modale n'est pas posée", dok, e.HasAim)
	}
	if e.Aim[0] < 0.55 || e.Aim[0] > 0.65 || e.Aim[2] < 0.75 || e.Aim[2] > 0.85 {
		t.Errorf("visée modale décodée = %v, attendu ~(0.6, 0, 0.8)", e.Aim)
	}
}

// TestModalAimBitOnMinimalRecord : sur l'en-tête MINIMAL (aucune option), la visée tombe à
// post-comptes + 2, de bout en bout par decodeFireEvent (il n'y a plus de chemin à offsets fixes
// depuis le lot M4b : la tête est lue par la grammaire).
func TestModalAimBitOnMinimalRecord(t *testing.T) {
	aimCode, ok := EncodeAimVector([3]float32{0, 0.8, 0.6}, FireAimBits)
	if !ok {
		t.Fatal("EncodeAimVector a refusé la largeur 30")
	}
	pay, aimBit := buildModalFire(modalHeaderOpts{}, aimCode)
	if got, ok := modalAimBit(pay); !ok || got != aimBit {
		t.Fatalf("modalAimBit = (%d, %v), attendu (%d, true)", got, ok, aimBit)
	}
	e, dok := decodeFireEvent(pay)
	if !dok || !e.HasAim {
		t.Fatalf("decodeFireEvent : ok=%v HasAim=%v", dok, e.HasAim)
	}
	if e.Aim[1] < 0.75 || e.Aim[1] > 0.85 || e.Aim[2] < 0.55 || e.Aim[2] > 0.65 {
		t.Errorf("visée modale décodée = %v, attendu ~(0, 0.8, 0.6)", e.Aim)
	}
}
