package grammar

import "testing"

// fire_events_test.go — LA TÊTE DU RECORD `action_weapon_fire` (type 36), LUE PAR SA GRAMMAIRE
// (lot M4b). Les records sont écrits par le constructeur à la grammaire de `fire_aim_modal_test.go`
// ([writeModalHeader]) : chaque champ optionnel présent déplace tout ce qui suit, et c'est ce que
// les anciens offsets fixes ne savaient pas traverser.

// tirSynthetique écrit un record type 36 MODAL (visée comprise) avec l'en-tête `o`.
func tirSynthetique(t *testing.T, o modalHeaderOpts) []byte {
	t.Helper()
	code, ok := EncodeAimVector([3]float32{0.6, 0.8, 0}, FireAimBits)
	if !ok {
		t.Fatal("EncodeAimVector a refusé la largeur 30")
	}
	pay, _ := buildModalFire(o, code)
	return pay
}

// TestDecodeFireEventGrammaireCanonique : la disposition CANONIQUE (ref0 avec sonde, tireur et
// arme haute présents) se lit champ par champ — l'indice de tireur sur CINQ bits (19 : un joueur de
// BTB que l'ancien champ à quatre bits confondait avec le 3), le numéro de tir, l'unité tireuse et
// l'arme entière.
func TestDecodeFireEventGrammaireCanonique(t *testing.T) {
	const arme = uint64(0x6ACDC44D42C9679F)
	pay := tirSynthetique(t, modalHeaderOpts{ref0: true, ref0Wide: true, ref0Index: 0x0F1,
		dExtra: true, tireur: 19, fWeapon: true, armeHaute: arme >> 32, armeBasse: arme & 0xFFFFFFFF,
		numero: 0x61}) // R(7) = 0x30, R(1) = 1 -> numéro 0x30 | 0x80
	e, ok := decodeFireEvent(pay)
	if !ok {
		t.Fatal("record canonique refusé")
	}
	if !e.HasShooter || e.FilmIndex != 19 {
		t.Errorf("tireur = (%v, %d), attendu (true, 19)", e.HasShooter, e.FilmIndex)
	}
	if e.WeaponID != arme {
		t.Errorf("arme = %#016x, attendu %#016x", e.WeaponID, arme)
	}
	if e.FireNumber != 0xB0 {
		t.Errorf("numéro de tir = %#x, attendu 0xb0", e.FireNumber)
	}
	if !e.Unit.Present || !e.Unit.Probe || e.Unit.Slot != parentHandleBase+0x0F1 {
		t.Errorf("unité = %+v, attendu slot %d avec sonde", e.Unit, parentHandleBase+0x0F1)
	}
	if !e.HasAim || e.Aim[0] < 0.55 || e.Aim[0] > 0.65 || e.Aim[1] < 0.75 || e.Aim[1] > 0.85 {
		t.Errorf("visée = (%v, %v), attendu ~(0.6, 0.8, 0)", e.HasAim, e.Aim)
	}
}

// TestDecodeFireEventTireurAbsent : la garde du tireur FERMÉE retire cinq bits à tout ce qui suit.
// Les anciens offsets fixes y lisaient une arme DÉCALÉE (1 205 tirs publiés au parc, rapport
// `RAPPORT_tirs_vehicules.md` §6) ; la grammaire rend l'arme juste et « pas de tireur ».
func TestDecodeFireEventTireurAbsent(t *testing.T) {
	const arme = uint64(0x121B400942C9679F)
	pay := tirSynthetique(t, modalHeaderOpts{ref0: true, ref0Wide: true, fWeapon: true,
		armeHaute: arme >> 32, armeBasse: arme & 0xFFFFFFFF})
	e, ok := decodeFireEvent(pay)
	if !ok {
		t.Fatal("record sans tireur refusé")
	}
	if e.HasShooter || e.FilmIndex != -1 {
		t.Errorf("tireur = (%v, %d), attendu (false, -1)", e.HasShooter, e.FilmIndex)
	}
	if e.WeaponID != arme {
		t.Errorf("arme = %#016x, attendu %#016x (décalée : l'offset fixe est revenu)", e.WeaponID, arme)
	}
}

// TestDecodeFireEventReferencesNonCanoniques : une ref0 SANS sonde (treize bits) et des refs 1 et 2
// PRÉSENTES déplacent la tête de 4 + 15 + 15 bits ; l'arme et le tireur restent justes.
func TestDecodeFireEventReferencesNonCanoniques(t *testing.T) {
	const arme = uint64(0x49E40D1742C9679F)
	pay := tirSynthetique(t, modalHeaderOpts{ref0: true, ref1: true, ref2: true, ref0Index: 0x333,
		dExtra: true, tireur: 7, eExtra: true, fWeapon: true, armeHaute: arme >> 32,
		armeBasse: arme & 0xFFFFFFFF})
	e, ok := decodeFireEvent(pay)
	if !ok {
		t.Fatal("record non canonique refusé")
	}
	if e.FilmIndex != 7 || e.WeaponID != arme {
		t.Errorf("tireur %d arme %#016x, attendu 7 et %#016x", e.FilmIndex, e.WeaponID, arme)
	}
	if e.Unit.Probe || e.Unit.Slot != parentHandleBase+0x333 {
		t.Errorf("unité = %+v, attendu slot %d sans sonde", e.Unit, parentHandleBase+0x333)
	}
}

// TestDecodeFireEventEcarteLeType37 : `weapon_overheat` (type 37) partage l'octet de tête 0xD2 du
// type 36 ; l'ancien filtre sur cet octet le laissait passer. La grammaire lit le type ENTIER.
func TestDecodeFireEventEcarteLeType37(t *testing.T) {
	pay := tirSynthetique(t, modalHeaderOpts{ref0: true, ref0Wide: true, dExtra: true, fWeapon: true})
	if pay[0] != 0xD2 {
		t.Fatalf("octet de tête du type 36 = %#x, attendu 0xd2", pay[0])
	}
	pay[1] |= 0x80 // le bit de poids faible du type (bit 8 du payload) : 36 -> 37
	if _, ok := decodeFireEvent(pay); ok {
		t.Error("record de type 37 accepté comme un tir")
	}
}

// TestDecodeFireEventAimGatedOff : sur un record réellement NON MODAL (au moins une cible ou une
// composante de dégât), la visée n'est PAS lue : les boucles ont une largeur venant d'une table
// peuplée au runtime, non localisable hors ligne. Le tir, lui, est rendu.
func TestDecodeFireEventAimGatedOff(t *testing.T) {
	for _, tc := range []struct {
		name           string
		targets, comps int
	}{
		{"une cible", 1, 0},
		{"trois cibles", 3, 0},
		{"une composante de degat", 0, 1},
		{"cible et composante", 2, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pay := buildNonModalFire(tc.targets, tc.comps)
			if _, ok := modalAimBit(pay); ok {
				t.Fatalf("record non modal (%d cible(s), %d composante(s)) accepté comme modal : "+
					"le forward localiserait une visée parasite", tc.targets, tc.comps)
			}
			e, ok := decodeFireEvent(pay)
			if !ok {
				t.Fatal("record non modal refusé : le tir existe, seule sa visée est illisible")
			}
			if e.HasAim {
				t.Errorf("visée décodée sur un record non modal (%d cible(s), %d composante(s))",
					tc.targets, tc.comps)
			}
		})
	}
}

// TestDecodeFireEventRefuseRecordTronque : LA GARDE DE LONGUEUR. Un film tronqué par un
// téléchargement partiel porte des paquets coupés, et ce décodeur tourne aussi dans un collecteur
// de fond du process de sync, où une panique coûte le process entier. Toute longueur SOUS la fin
// de la tête est refusée sans paniquer ; la tête entière est acceptée.
func TestDecodeFireEventRefuseRecordTronque(t *testing.T) {
	w := &bitWriter{}
	writeModalHeader(w, modalHeaderOpts{ref0: true, ref0Wide: true, dExtra: true, fWeapon: true})
	finTete := w.n
	complet := append([]byte(nil), w.buf...)
	for n := 0; n*8 < finTete; n++ {
		e, ok := decodeFireEvent(complet[:n])
		if ok {
			t.Errorf("payload de %d octet(s) accepté alors que la tête finit au bit %d", n, finTete)
		}
		if e != (FireEvent{}) {
			t.Errorf("payload de %d octet(s) : event non nul rendu avec ok=false", n)
		}
	}
	if _, ok := decodeFireEvent(complet[:(finTete+7)/8]); !ok {
		t.Errorf("payload portant la tête entière (%d bits) refusé", finTete)
	}
}

// TestPeekBitsToleranceDesDeuxCotes : la tolérance de PeekBits vaut aux DEUX bouts.
//
// Sa documentation a toujours annoncé « ne jamais paniquer sur un payload tronqué », mais
// jusqu'au 2026-08-01 une position NÉGATIVE paniquait (`index out of range [-1]`) — une
// primitive dont c'est la seule raison d'être ne tenait sa promesse que d'un côté.
func TestPeekBitsToleranceDesDeuxCotes(t *testing.T) {
	d := []byte{0xFF, 0xFF}
	if got := PeekBits(d, -8, 8); got != 0 {
		t.Errorf("lecture entièrement avant le début = %#x, attendu 0", got)
	}
	if got := PeekBits(d, len(d)*8, 8); got != 0 {
		t.Errorf("lecture entièrement après la fin = %#x, attendu 0", got)
	}
	// À cheval sur le début : 4 bits hors buffer (0) puis 4 bits à 1 -> 0b00001111.
	if got := PeekBits(d, -4, 8); got != 0x0F {
		t.Errorf("lecture à cheval sur le début = %#x, attendu 0x0F", got)
	}
	if got := PeekBits(nil, -4, 8); got != 0 {
		t.Errorf("buffer vide = %#x, attendu 0", got)
	}
}
