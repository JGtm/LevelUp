package filmdec

// event_list_exit_grammar_test.go — GARDE-RAIL de la grammaire de la SORTIE, et en particulier de
// sa RÉFÉRENCE 1 (le véhicule). Sur payload synthétique, SANS environnement ni film : il tourne
// dans la suite ordinaire.
//
// CE QU'IL FIGE : `unit_exit_vehicle` porte DEUX références de domaine 1 (occupant puis véhicule,
// chacune avec sa sonde) puis une de domaine 7, puis R(6) siège. Il tombe si quelqu'un réordonne
// les domaines, retire une sonde, change une largeur, ou re-jette la référence 1 comme le
// décodeur le faisait avant le lot V8.

import "testing"

// TestExitEventVehicleRef fige la lecture de la référence 1 (le véhicule) d'une SORTIE.
//
// TROIS TÉMOINS INTÉGRÉS, parce qu'un test qui n'affirme qu'un slot attendu ne prouve pas grand
// chose : (1) l'occupant et le véhicule doivent sortir DIFFÉRENTS du même payload — sinon la
// seconde référence n'est qu'une relecture de la première ; (2) la même charge lue comme un
// EMBARQUEMENT ne doit PAS publier de véhicule (ses domaines sont 2/3/7) ; (3) une référence 1
// dont le bit de garde est à ZÉRO ne publie aucun véhicule, et le siège se décale d'autant.
func TestExitEventVehicleRef(t *testing.T) {
	const (
		base   = 512
		idxOcc = 49  // occupant, sonde = 1 : slot attendu = 512 + 49 = 561
		idxVeh = 258 // véhicule, sonde = 1 : slot attendu = 512 + 258 = 770
		genVeh = 2
		seat   = 0
	)
	champs := [][2]uint32{
		{1, 1},                              // bit 0 : drapeau de configuration
		{1, 1},                              // bit 1 : continuation
		{EventUnitExitVehicle, 7},           // R(7) : type de tête
		{1, 1}, {1, 1}, {idxOcc, 9}, {1, 2}, // réf 0, domaine 1 : garde, sonde=1, R(9), gén.
		{1, 1}, {1, 1}, {idxVeh, 9}, {genVeh, 2}, // réf 1, domaine 1 : LE VÉHICULE
		{0, 1},                  // réf 2, domaine 7 : garde à 0 (cas mesuré)
		{seat, vehicleSeatBits}, // R(6) : siège
		{0, 8},                  // rembourrage
	}
	band := map[uint32]bool{base + idxOcc: true}
	ev, ok := decodeVehicleEvent(evbEcritBits(champs), base, band)
	if !ok || ev.Kind != EventUnitExitVehicle {
		t.Fatalf("sortie non décodée : ok=%v kind=%d", ok, ev.Kind)
	}
	if !ev.OccupantPresent || ev.OccupantSlot != base+idxOcc || !ev.OccupantInBand {
		t.Errorf("occupant : présent=%v slot=%d (attendu %d) enBande=%v",
			ev.OccupantPresent, ev.OccupantSlot, base+idxOcc, ev.OccupantInBand)
	}
	if !ev.VehicleSlotValid || ev.VehicleSlot != base+idxVeh || ev.VehicleGen != genVeh {
		t.Errorf("véhicule : valide=%v slot=%d (attendu %d) gén=%d (attendue %d)",
			ev.VehicleSlotValid, ev.VehicleSlot, base+idxVeh, ev.VehicleGen, genVeh)
	}
	if !ev.SeatValid || ev.Seat != seat {
		t.Errorf("siège : valide=%v valeur=%d (attendu %d)", ev.SeatValid, ev.Seat, seat)
	}
	// TÉMOIN 1 : les deux références de domaine 1 ne rendent pas la même unité.
	if ev.VehicleSlot == ev.OccupantSlot {
		t.Errorf("témoin : occupant et véhicule au même slot %d — la réf 1 n'est qu'une relecture",
			ev.VehicleSlot)
	}
	// TÉMOIN 2 : LES MÊMES BITS de charge, lus par la grammaire de l'EMBARQUEMENT (domaines
	// 2/3/7, sans sonde), ne doivent publier AUCUN véhicule.
	board := evbEcritBits(champs)
	evbForceType(board, EventBipedBoardVehicle)
	evb, ok := decodeVehicleEvent(board, base, band)
	if !ok || evb.Kind != EventBipedBoardVehicle {
		t.Fatalf("témoin embarquement non décodé : ok=%v kind=%d", ok, evb.Kind)
	}
	if evb.VehicleSlotValid {
		t.Errorf("témoin : l'embarquement publie un véhicule (slot %d) — ses domaines sont 2/3/7",
			evb.VehicleSlot)
	}
	// TÉMOIN 3 : référence 1 ABSENTE (garde à 0). Aucun véhicule, et le siège se lit 12 bits plus
	// tôt — le décalage prouve que la référence était bien LUE, pas sautée à largeur fixe.
	sans := append([][2]uint32(nil), champs[:7]...)
	sans = append(sans, [2]uint32{0, 1}, [2]uint32{0, 1}, [2]uint32{seat + 1, vehicleSeatBits},
		[2]uint32{0, 8})
	evs, ok := decodeVehicleEvent(evbEcritBits(sans), base, band)
	if !ok {
		t.Fatal("témoin réf 1 absente : sortie non décodée")
	}
	if evs.VehicleSlotValid {
		t.Errorf("témoin : réf 1 gardée-absente et pourtant un véhicule publié (slot %d)",
			evs.VehicleSlot)
	}
	if !evs.SeatValid || evs.Seat != seat+1 {
		t.Errorf("témoin : siège = %d (valide=%v), attendu %d — la réf absente n'a pas décalé la"+
			" lecture", evs.Seat, evs.SeatValid, seat+1)
	}
}

// evbForceType réécrit EN PLACE le champ R(7) de type de tête d'un payload déjà sérialisé.
func evbForceType(pay []byte, typ int) {
	for i := 0; i < eventTypeBits; i++ {
		bit := 2 + i
		masque := byte(1) << uint(7-bit%8)
		pay[bit/8] &^= masque
		if typ>>uint(eventTypeBits-1-i)&1 == 1 {
			pay[bit/8] |= masque
		}
	}
}
