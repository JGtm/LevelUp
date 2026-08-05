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
	e, ok := decodeFireEvent(w.buf)
	if !ok {
		t.Fatal("record complet refuse par la garde de longueur")
	}
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
			if e, _ := decodeFireEvent(w.buf); e.HasAim {
				t.Errorf("visée décodée hors du chemin sûr (drapeaux %v)", tc.flags)
			}
		})
	}
}

// TestDecodeFireEventRefuseRecordTronque : LA GARDE DE LONGUEUR, ET POURQUOI ELLE EXISTE.
//
// Le décodeur lit la tête à des offsets FIXES jusqu'au bit 112, via `readBitsAt` qui indexe le
// tableau SANS borne (contrairement à `PeekBits`, qui rend 0 au-delà). `ScanFilmFireEvents`
// n'exige que `p.Size >= 1` : avant la garde, un paquet delta tronqué dont le premier octet
// vaut 0xD2 — type 105, variante longue — faisait paniquer le décodeur. Un film tronqué par un
// téléchargement partiel suffit, et en J4 ce décodeur tourne dans un collecteur de fond du
// process de sync, où une panique coûte le process entier.
//
// Le test balaie TOUTES les longueurs sous le seuil, pas seulement zéro : c'est la longueur du
// dernier champ obligatoire qui fixe le seuil, et une régression le déplacerait d'un octet.
func TestDecodeFireEventRefuseRecordTronque(t *testing.T) {
	full := (FireHeadBits + 7) / 8
	for n := 0; n < full; n++ {
		pay := make([]byte, n)
		if n > 0 {
			pay[0] = FireEventType << 1 // 0xD2 : type 105, variante longue
		}
		e, ok := decodeFireEvent(pay)
		if ok {
			t.Errorf("payload de %d octet(s) accepté alors que la tête en exige %d", n, full)
		}
		if e != (FireEvent{}) {
			t.Errorf("payload de %d octet(s) : event non nul rendu avec ok=false", n)
		}
	}
	if _, ok := decodeFireEvent(make([]byte, full)); !ok {
		t.Errorf("payload de %d octets refusé alors qu'il porte la tête entière", full)
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
