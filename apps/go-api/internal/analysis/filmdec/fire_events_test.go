package filmdec

import "testing"

// putBits écrit n bits MSB-first à une position ABSOLUE (le record type 105 se décrit par
// offsets de bits, pas par écriture séquentielle). Complète le bitWriter séquentiel de
// frame_chain_infer_test.go.
func (w *bitWriter) putBits(pos int, n int, v uint64) {
	for i := 0; i < n; i++ {
		p := pos + i
		for len(w.buf) <= p/8 {
			w.buf = append(w.buf, 0)
		}
		if (v>>uint(n-1-i))&1 == 1 {
			w.buf[p/8] |= 1 << uint(7-p%8)
		}
	}
	if pos+n > w.n {
		w.n = pos + n
	}
}

// TestDecodeFireEventLayout ancre les OFFSETS DE BITS du record type 105 (spec Ghidra).
// Un décalage d'un seul bit sur l'un des champs fait échouer le test — c'est le but : ces
// offsets sont la seule chose qui sépare une arme réelle d'un mot de bruit.
func TestDecodeFireEventLayout(t *testing.T) {
	const (
		wantPlayer = 5
		wantWeapon = uint64(0x6ACDC44D42C9679F)
	)
	aimCode, ok := EncodeAimVector([3]float32{0.6, 0.8, 0}, FireAimBits)
	if !ok {
		t.Fatal("EncodeAimVector a refusé la largeur 30")
	}
	w := &bitWriter{}
	w.putBits(0, 7, uint64(FireEventType))
	w.putBits(fireVariantBit, 1, 0) // record long
	w.putBits(fireAttackerBit, fireAttackerW, uint64(wantPlayer)<<1)
	w.putBits(fireWeaponHiBit, fireWeaponW, wantWeapon>>32)
	w.putBits(fireWeaponLoBit, fireWeaponW, wantWeapon&0xFFFFFFFF)
	w.putBits(fireFlagsBit+2, 1, 1) // compteurs nuls -> la visée suit à fireAimBit
	w.putBits(fireAimBit, int(FireAimBits), uint64(aimCode))

	if got := int(w.buf[0] >> 1); got != FireEventType {
		t.Fatalf("type d'event dans payload[0] = %d, attendu %d", got, FireEventType)
	}
	e := decodeFireEvent(w.buf)
	if e.FilmIndex != wantPlayer {
		t.Errorf("FilmIndex = %d, attendu %d", e.FilmIndex, wantPlayer)
	}
	if e.WeaponID != wantWeapon {
		t.Errorf("WeaponID = %#016x, attendu %#016x", e.WeaponID, wantWeapon)
	}
	if !e.HasAim {
		t.Fatal("visée non décodée alors que les trois drapeaux sont sur le chemin sûr")
	}
	if e.Aim[0] < 0.55 || e.Aim[0] > 0.65 || e.Aim[1] < 0.75 || e.Aim[1] > 0.85 {
		t.Errorf("visée décodée = %v, attendu ~(0.6, 0.8, 0)", e.Aim)
	}
	h, _ := e.AimHeadingDeg()
	if h < 50 || h > 55 { // atan2(0.8, 0.6) = 53,1 deg
		t.Errorf("cap = %.1f deg, attendu ~53,1", h)
	}
}

// TestDecodeFireEventAimGatedOff : hors du chemin « record vide », la visée n'est PAS lue.
// Le champ existe toujours dans le flux mais après des boucles de longueur variable : le
// décodeur doit refuser plutôt que lire des bits qui ne sont pas les bons.
func TestDecodeFireEventAimGatedOff(t *testing.T) {
	for _, tc := range []struct {
		name  string
		flags [3]uint64
	}{
		{"compteurs non nuls", [3]uint64{0, 0, 0}},
		{"porte 111 ouverte", [3]uint64{1, 1, 0}},
		{"porte 112 ouverte", [3]uint64{1, 0, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := &bitWriter{}
			w.putBits(0, 7, uint64(FireEventType))
			w.putBits(fireFlagsBit+2, 1, tc.flags[0])
			w.putBits(fireFlagsBit+3, 1, tc.flags[1])
			w.putBits(fireFlagsBit+4, 1, tc.flags[2])
			w.putBits(fireAimBit, int(FireAimBits), 12345)
			if e := decodeFireEvent(w.buf); e.HasAim {
				t.Errorf("visée décodée hors du chemin sûr (drapeaux %v)", tc.flags)
			}
		})
	}
}
