package filmdec

// event_list_board_grammar_test.go — GARDE-RAIL de la grammaire de l'EMBARQUEMENT, sans
// environnement ni film : il tourne dans la suite ordinaire. Separe de
// event_list_board_test.go (la mesure sur corpus) pour tenir le seuil de 500 lignes par fichier.

import "testing"

// --- CAS BOARD : garde-rail de grammaire, SANS environnement ni film ---------------------------

// evbEcritBits sérialise une suite de champs (valeur, largeur) en MSB-first, la convention du
// BitReader. C'est l'inverse exact de readBitsAt : il fabrique un payload synthétique.
func evbEcritBits(champs [][2]uint32) []byte {
	total := 0
	for _, c := range champs {
		total += int(c[1])
	}
	out := make([]byte, (total+7)/8+1)
	bit := 0
	for _, c := range champs {
		v, w := c[0], int(c[1])
		for i := w - 1; i >= 0; i-- {
			if v>>uint(i)&1 == 1 {
				out[bit/8] |= 1 << uint(7-bit%8)
			}
			bit++
		}
	}
	return out
}

// TestBoardEventGrammar fige la grammaire de l'EMBARQUEMENT lue dans l'exécutable le 2026-09-02
// (réfs en domaines 2, 3, 7 — sans sonde — puis R(6) siège) sur un payload fabriqué. Il tombe si
// quelqu'un remet une sonde sur la réf 0, réordonne les domaines ou change une largeur.
//
// TÉMOIN INTÉGRÉ : le même payload lu comme une SORTIE doit rendre un occupant DIFFÉRENT — les
// deux grammaires ne peuvent pas coïncider, sinon le test ne prouverait rien.
func TestBoardEventGrammar(t *testing.T) {
	const (
		base   = 512
		idxOcc = 49 // occupant : slot attendu = base + 49 = 561
		seat   = 3
	)
	pay := evbEcritBits([][2]uint32{
		{1, 1},                                 // bit 0 : drapeau de configuration
		{1, 1},                                 // bit 1 : continuation (la liste porte un événement)
		{EventBipedBoardVehicle, 7},            // R(7) : type de tête
		{1, 1}, {idxOcc, dom2RefWidth}, {2, 2}, // réf 0, domaine 2 : garde, index, génération
		{1, 1}, {5, dom3RefWidth}, {1, 2}, // réf 1, domaine 3
		{0, 1},                  // réf 2, domaine 7 : garde à 0 (absente, cas dominant mesuré)
		{seat, vehicleSeatBits}, // R(6) : siège
		{0, 8},                  // rembourrage
	})
	band := map[uint32]bool{base + idxOcc: true}
	ev, ok := decodeVehicleEvent(pay, base, band)
	if !ok || ev.Kind != EventBipedBoardVehicle {
		t.Fatalf("embarquement non décodé : ok=%v kind=%d", ok, ev.Kind)
	}
	if !ev.OccupantPresent || ev.OccupantSlot != base+idxOcc || !ev.OccupantInBand {
		t.Errorf("occupant : présent=%v slot=%d (attendu %d) enBande=%v",
			ev.OccupantPresent, ev.OccupantSlot, base+idxOcc, ev.OccupantInBand)
	}
	if ev.OccupantSonde != 0 {
		t.Errorf("sonde = %d : l'embarquement n'en porte pas (domaines 2/3/7)", ev.OccupantSonde)
	}
	if !ev.SeatValid || ev.Seat != seat {
		t.Errorf("siège : valide=%v valeur=%d (attendu %d)", ev.SeatValid, ev.Seat, seat)
	}
	// Témoin : LES MÊMES BITS de charge, mais lus par la grammaire de la SORTIE (domaines
	// 1/1/7, avec sonde). Seul le champ de type change ; tout ce qui suit est identique.
	sortie := append([]byte(nil), pay...)
	for i := 0; i < eventTypeBits; i++ {
		bit := 2 + i
		masque := byte(1) << uint(7-bit%8)
		sortie[bit/8] &^= masque
		if EventUnitExitVehicle>>uint(eventTypeBits-1-i)&1 == 1 {
			sortie[bit/8] |= masque
		}
	}
	evx, ok := decodeVehicleEvent(sortie, base, band)
	if !ok || evx.Kind != EventUnitExitVehicle {
		t.Fatalf("témoin sortie non décodé : ok=%v kind=%d", ok, evx.Kind)
	}
	if evx.OccupantSlot == ev.OccupantSlot {
		t.Errorf("témoin : les deux grammaires rendent le même occupant (%d) — le test ne prouve rien",
			evx.OccupantSlot)
	}
}
