package filmdec

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// cliffLayout est le découpage d'i0 de la carte de 000d5950 (Cliffhanger), établi par
// DetectI0Layout sur le film et recoupé avec la table de largeurs live (13/13/14).
var cliffLayout = I0Layout{GateBits: DefaultI0GateBits, AxisW: [3]uint{13, 13, 14}}

// cliffRange est l'AABB monde du BSP de Cliffhanger (module `ridgeline`), lue avec
// internal/himap. Les bornes sont PROPRES À LA CARTE : le décodeur les exige désormais.
var cliffRange = QuantRangeCEBiped

// scanOptWorld : réglages par défaut + bornes de carte (le décodeur refuse d'émettre des
// coordonnées monde sans elles).
func scanOptWorld() ScanFilmOptions {
	o := DefaultScanFilmOptions()
	o.WorldRange = &cliffRange
	return o
}

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
	w.bits(0, cliffLayout.GateBits) // i0 absolu
	w.bits(qx, int(cliffLayout.AxisW[0]))
	w.bits(qy, int(cliffLayout.AxisW[1]))
	w.bits(qz, int(cliffLayout.AxisW[2]))
}

// TestScanBipedRecords_RoundTrip : un record synthétique conforme est retrouvé et
// déquantifié aux mêmes valeurs que DequantBipedAxis.
func TestScanBipedRecords_RoundTrip(t *testing.T) {
	const slot uint32 = 517
	w := &bitWriter{}
	w.bits(0, 11) // bruit de tête (le scan est bit à bit, pas aligné octet)
	writeBipedRecord(w, slot, 1, 3, 4096, 5000, 8192)
	w.bits(0, 64) // queue

	got := ScanBipedRecords(w.buf, NewSlotBand(map[uint32]bool{slot: true}), cliffLayout, scanOptWorld())
	if len(got) != 1 {
		t.Fatalf("attendu 1 record, obtenu %d (%+v)", len(got), got)
	}
	if got[0].Slot != slot {
		t.Errorf("slot = %d, attendu %d", got[0].Slot, slot)
	}
	wantX, wantY, wantZ := DequantBipedAxis(4096, 0, cliffLayout, cliffRange), DequantBipedAxis(5000, 1, cliffLayout, cliffRange), DequantBipedAxis(8192, 2, cliffLayout, cliffRange)
	if got[0].X != wantX || got[0].Y != wantY || got[0].Z != wantZ {
		t.Errorf("position = (%f,%f,%f), attendu (%f,%f,%f)", got[0].X, got[0].Y, got[0].Z, wantX, wantY, wantZ)
	}
}

// TestScanBipedRecords_Rejects : slot hors bande, tag != 1 et bucket saturé sont rejetés.
func TestScanBipedRecords_Rejects(t *testing.T) {
	const slot uint32 = 517
	opt := scanOptWorld()

	w := &bitWriter{}
	writeBipedRecord(w, slot, 1, 3, 4096, 5000, 8192)
	if got := ScanBipedRecords(w.buf, NewSlotBand(map[uint32]bool{999: true}), cliffLayout, opt); len(got) != 0 {
		t.Errorf("slot hors bande accepté: %+v", got)
	}

	w2 := &bitWriter{}
	writeBipedRecord(w2, slot, 2, 3, 4096, 5000, 8192) // tag != 1
	if got := ScanBipedRecords(w2.buf, NewSlotBand(map[uint32]bool{slot: true}), cliffLayout, opt); len(got) != 0 {
		t.Errorf("tag != 1 accepté: %+v", got)
	}

	w3 := &bitWriter{}
	writeBipedRecord(w3, slot, 1, 3, 0, 5000, 8192) // qx == 0 -> écrêté
	if got := ScanBipedRecords(w3.buf, NewSlotBand(map[uint32]bool{slot: true}), cliffLayout, opt); len(got) != 0 {
		t.Errorf("quantum saturé accepté: %+v", got)
	}
	noDrop := opt
	noDrop.DropSaturated = false
	if got := ScanBipedRecords(w3.buf, NewSlotBand(map[uint32]bool{slot: true}), cliffLayout, noDrop); len(got) != 1 {
		t.Errorf("DropSaturated=false devrait conserver le record, obtenu %d", len(got))
	}
}

// TestDequantBipedAxis vérifie la formule mi-bucket sur les bornes du range biped.
func TestDequantBipedAxis(t *testing.T) {
	rng := cliffRange[0]
	step := (float64(rng.Max) - float64(rng.Min)) / 8192
	want := float32(float64(rng.Min) + step*0.5)
	if got := DequantBipedAxis(0, 0, cliffLayout, cliffRange); got != want {
		t.Errorf("DequantBipedAxis(0,0) = %f, attendu %f", got, want)
	}
	if got := DequantBipedAxis(8191, 0, cliffLayout, cliffRange); got <= float32(float64(rng.Max)-step) || got >= rng.Max {
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

// TestScanFilmBipedPositionsDelegue : les DEUX entrées de balayage de film partagent le même
// pré-contrôle, donc la même erreur, mot pour mot.
//
// CE TEST N'A BESOIN D'AUCUN FILM, et c'est exactement le point. Le cache film n'existe pas en
// CI : un point d'entrée dont la délégation ne serait éprouvée que par des instruments sous
// garde d'environnement ne serait éprouvé par personne. ScanFilmBipedPositions déléguant à
// ScanFilmBipedPositionsForBand, toute divergence de pré-contrôle entre les deux se voit ici,
// sur un répertoire vide.
func TestScanFilmBipedPositionsDelegue(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "film-absent")
	band := NewSlotBand(map[uint32]bool{7: true})
	quanta := DefaultScanFilmOptions()
	quanta.QuantaOnly = true
	for _, c := range []struct {
		nom string
		opt ScanFilmOptions
	}{
		{"ni bornes de carte ni QuantaOnly", DefaultScanFilmOptions()},
		{"QuantaOnly sur un film absent", quanta},
	} {
		t.Run(c.nom, func(t *testing.T) {
			_, errBipede := ScanFilmBipedPositions(dir, c.opt)
			_, errBande := ScanFilmBipedPositionsForBand(dir, band, c.opt)
			if errBipede == nil || errBande == nil {
				t.Fatalf("les deux entrées doivent échouer : bipède = %v, bande = %v",
					errBipede, errBande)
			}
			if errBipede.Error() != errBande.Error() {
				t.Fatalf("erreurs divergentes :\n  bipède = %v\n  bande  = %v", errBipede, errBande)
			}
		})
	}
	if _, err := ScanFilmBipedPositions(dir, DefaultScanFilmOptions()); !errors.Is(err, ErrUnknownMapBounds) {
		t.Fatalf("sans bornes de carte, l'erreur doit envelopper ErrUnknownMapBounds : %v", err)
	}
}

// TestScanFilmBipedPositionsForBandRefuseBandeVide : une bande vide est une erreur NOMMÉE, pas
// un balayage silencieusement vide — l'équivalent, pour l'entrée générique, du « aucun slot
// biped » que l'entrée bipède rend quand les images-clés n'en portent pas.
func TestScanFilmBipedPositionsForBandRefuseBandeVide(t *testing.T) {
	opt := DefaultScanFilmOptions()
	opt.QuantaOnly, opt.Chunks = true, []int{1} // la liste de chunks passe le pré-contrôle
	_, err := ScanFilmBipedPositionsForBand(t.TempDir(), SlotBand{}, opt)
	if err == nil || !strings.Contains(err.Error(), "bande de slots vide") {
		t.Fatalf("bande vide : erreur « bande de slots vide » attendue, obtenu %v", err)
	}
}
