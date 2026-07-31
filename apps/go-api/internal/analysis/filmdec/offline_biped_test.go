package filmdec

import "testing"

// writeBipedRecord écrit un record biped conforme à la grammaire décodée (bitWriter est
// l'écrivain MSB-first partagé des tests du package, cf. frame_chain_infer_test.go).
func writeBipedRecord(w *bitWriter, slot uint32, tag, maskCount uint64, qx, qy, qz uint64) {
	w.bits(1, 1)                        // préfixe
	w.bits(uint64(slot), bipedSlotBits) // idLow = slot
	w.bits(tag, 2)                      // tag
	w.bits(0, 2)                        // gate + maskSel
	w.bits(maskCount, 3)                // maskCount
	for k := uint64(0); k < maskCount; k++ {
		w.bits(k, bipedIndexBits) // indices croissants depuis 0
	}
	w.bits(0, bipedI0GateBits) // i0 absolu
	w.bits(qx, 13)
	w.bits(qy, 13)
	w.bits(qz, 14)
}

// TestScanBipedRecords_RoundTrip : un record synthétique conforme est retrouvé et
// déquantifié aux mêmes valeurs que DequantBipedAxis.
func TestScanBipedRecords_RoundTrip(t *testing.T) {
	const slot uint32 = 517
	w := &bitWriter{}
	w.bits(0, 11) // bruit de tête (le scan est bit à bit, pas aligné octet)
	writeBipedRecord(w, slot, 1, 3, 4096, 5000, 8192)
	w.bits(0, 64) // queue

	got := ScanBipedRecords(w.buf, map[uint32]bool{slot: true}, DefaultScanFilmOptions())
	if len(got) != 1 {
		t.Fatalf("attendu 1 record, obtenu %d (%+v)", len(got), got)
	}
	if got[0].Slot != slot {
		t.Errorf("slot = %d, attendu %d", got[0].Slot, slot)
	}
	wantX, wantY, wantZ := DequantBipedAxis(4096, 0), DequantBipedAxis(5000, 1), DequantBipedAxis(8192, 2)
	if got[0].X != wantX || got[0].Y != wantY || got[0].Z != wantZ {
		t.Errorf("position = (%f,%f,%f), attendu (%f,%f,%f)", got[0].X, got[0].Y, got[0].Z, wantX, wantY, wantZ)
	}
}

// TestScanBipedRecords_Rejects : slot hors bande, tag != 1 et bucket saturé sont rejetés.
func TestScanBipedRecords_Rejects(t *testing.T) {
	const slot uint32 = 517
	opt := DefaultScanFilmOptions()

	w := &bitWriter{}
	writeBipedRecord(w, slot, 1, 3, 4096, 5000, 8192)
	if got := ScanBipedRecords(w.buf, map[uint32]bool{999: true}, opt); len(got) != 0 {
		t.Errorf("slot hors bande accepté: %+v", got)
	}

	w2 := &bitWriter{}
	writeBipedRecord(w2, slot, 2, 3, 4096, 5000, 8192) // tag != 1
	if got := ScanBipedRecords(w2.buf, map[uint32]bool{slot: true}, opt); len(got) != 0 {
		t.Errorf("tag != 1 accepté: %+v", got)
	}

	w3 := &bitWriter{}
	writeBipedRecord(w3, slot, 1, 3, 0, 5000, 8192) // qx == 0 -> écrêté
	if got := ScanBipedRecords(w3.buf, map[uint32]bool{slot: true}, opt); len(got) != 0 {
		t.Errorf("quantum saturé accepté: %+v", got)
	}
	noDrop := opt
	noDrop.DropSaturated = false
	if got := ScanBipedRecords(w3.buf, map[uint32]bool{slot: true}, noDrop); len(got) != 1 {
		t.Errorf("DropSaturated=false devrait conserver le record, obtenu %d", len(got))
	}
}

// TestDequantBipedAxis vérifie la formule mi-bucket sur les bornes du range biped.
func TestDequantBipedAxis(t *testing.T) {
	rng := QuantRangeCEBiped[0]
	step := (float64(rng.Max) - float64(rng.Min)) / 8192
	want := float32(float64(rng.Min) + step*0.5)
	if got := DequantBipedAxis(0, 0); got != want {
		t.Errorf("DequantBipedAxis(0,0) = %f, attendu %f", got, want)
	}
	if got := DequantBipedAxis(8191, 0); got <= float32(float64(rng.Max)-step) || got >= rng.Max {
		t.Errorf("DequantBipedAxis(8191,0) = %f, attendu juste sous %f", got, rng.Max)
	}
}

// TestWalkPackets valide l'en-tête de paquet (type, taille, timestamp) et l'arrêt propre
// sur un en-tête incohérent.
func TestWalkPackets(t *testing.T) {
	chunk := make([]byte, 0, 64)
	appendPacket := func(typ uint16, size uint32, ts uint64, withPayload bool) {
		h := make([]byte, packetHeaderSize)
		h[0], h[1] = byte(typ), byte(typ>>8)
		h[4], h[5], h[6], h[7] = byte(size), byte(size>>8), byte(size>>16), byte(size>>24)
		for i := 0; i < 8; i++ {
			h[8+i] = byte(ts >> (8 * uint(i)))
		}
		chunk = append(chunk, h...)
		if withPayload {
			chunk = append(chunk, make([]byte, size)...)
		}
	}
	appendPacket(PacketTypeDelta, 4, 1_000_000, true)
	appendPacket(PacketTypeKeyframe, 8, 2_000_000, true)
	appendPacket(PacketTypeDelta, 1<<20, 3_000_000, false) // taille > reste du chunk -> arrêt

	pk := WalkPackets(chunk)
	if len(pk) != 2 {
		t.Fatalf("attendu 2 paquets, obtenu %d", len(pk))
	}
	if pk[0].Type != PacketTypeDelta || pk[0].Size != 4 || pk[0].TimestampUS != 1_000_000 {
		t.Errorf("paquet 0 = %+v", pk[0])
	}
	if pk[1].Type != PacketTypeKeyframe || pk[1].Index != 1 || pk[1].TimestampUS != 2_000_000 {
		t.Errorf("paquet 1 = %+v", pk[1])
	}
	if len(pk[1].Payload(chunk)) != 8 {
		t.Errorf("payload paquet 1 = %d octets, attendu 8", len(pk[1].Payload(chunk)))
	}
}
