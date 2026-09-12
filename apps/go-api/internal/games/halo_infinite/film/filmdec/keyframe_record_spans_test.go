package filmdec

// keyframe_record_spans_test.go — GARDE-RAIL SANS ENVIRONNEMENT de `KeyframeRecordSpans` et
// de la sonde `SetUnitRefHook`. Aucun film, aucune donnée réelle : des payloads fabriqués,
// dont on connaît la réponse d'avance.

import "testing"

// kfSpanEcrire pose `n` bits de `v` à la position `pos` dans `buf` (MSB d'abord — la même
// convention que kfReadBits).
func kfSpanEcrire(buf []byte, pos, n int, v uint64) {
	for i := 0; i < n; i++ {
		bit := (v >> uint(n-1-i)) & 1
		p := pos + i
		if idx := p >> 3; idx < len(buf) {
			mask := byte(1) << (7 - uint(p&7))
			if bit == 1 {
				buf[idx] |= mask
			} else {
				buf[idx] &^= mask
			}
		}
	}
}

// kfSpanRecord écrit un en-tête de record d'image-clé acceptable par `kfValidAnchor` :
// `[id:32][field26:26 == 0][ti:6]`, génération 1.
func kfSpanRecord(buf []byte, pos, slot, ti int) {
	kfSpanEcrire(buf, pos, 32, uint64(1)<<30|uint64(slot))
	kfSpanEcrire(buf, pos+32, 26, 0)
	kfSpanEcrire(buf, pos+58, 6, uint64(ti))
}

// TestKeyframeRecordSpans exige que l'emprise rendue soit exactement l'intervalle jusqu'au
// record suivant, et que `SlotGap` rapporte l'écart de slot — c'est LUI le garde-fou de la
// mesure V5 (une emprise dont le voisin a été sauté couvre plusieurs records).
func TestKeyframeRecordSpans(t *testing.T) {
	// Trois records à 1 (préfixe), 300 et 700 bits, slots 10, 11 puis 14 (deux slots sautés).
	buf := make([]byte, 200)
	kfSpanRecord(buf, 1, 10, 35)
	kfSpanRecord(buf, 300, 11, 40)
	kfSpanRecord(buf, 700, 14, 35)

	spans := KeyframeRecordSpans(buf)
	if len(spans) != 3 {
		t.Fatalf("emprises rendues : %d, attendu 3 (%+v)", len(spans), spans)
	}
	attendu := []KeyframeRecordSpan{
		{Slot: 10, TI: 35, BitStart: 1, BitEnd: 300, LengthBits: 299, SlotGap: 1},
		{Slot: 11, TI: 40, BitStart: 300, BitEnd: 700, LengthBits: 400, SlotGap: 3},
		{Slot: 14, TI: 35, BitStart: 700, BitEnd: len(buf) * 8, LengthBits: len(buf)*8 - 700, SlotGap: 0},
	}
	for i, a := range attendu {
		g := spans[i]
		if g.Slot != a.Slot || g.TI != a.TI || g.BitStart != a.BitStart ||
			g.BitEnd != a.BitEnd || g.LengthBits != a.LengthBits || g.SlotGap != a.SlotGap {
			t.Errorf("emprise %d : %+v, attendu %+v", i, g, a)
		}
	}
	// LE POINT QUI COMPTE : le deuxième record a un voisin SAUTÉ (SlotGap == 3). Sa longueur
	// n'est donc PAS celle d'un seul record, et un consommateur qui compare des longueurs
	// doit pouvoir l'écarter. Si SlotGap cessait de le dire, la mesure V5 redeviendrait
	// fausse sans que rien ne tombe.
	if spans[1].SlotGap == 1 {
		t.Error("le record à voisin sauté est rapporté comme ayant un voisin immédiat")
	}
}

// TestUnitRefProbePublieSansChangerLesBits exige DEUX choses de la sonde : qu'elle publie ce
// que le déserialiseur lit, et qu'elle ne change AUCUN bit consommé. La seconde est la
// condition de sa présence dans du code de production.
func TestUnitRefProbePublieSansChangerLesBits(t *testing.T) {
	// Charge : porte=1, puis R(13) = 0x0ABC, puis R(2) de queue = 0b10.
	buf := make([]byte, 8)
	kfSpanEcrire(buf, 0, 1, 1)
	kfSpanEcrire(buf, 1, 13, 0x0ABC)
	kfSpanEcrire(buf, 14, 2, 0b10)

	sans := NewBitReader(buf)
	consume1408f0ac4Probe(sans, false)
	posSans := sans.BitPos()

	var vues []UnitRefRead
	prev := unitRefHook
	SetUnitRefHook(func(r UnitRefRead) { vues = append(vues, r) })
	defer SetUnitRefHook(prev)

	avec := NewBitReader(buf)
	val, tail, present := consume1408f0ac4Probe(avec, false)

	if avec.BitPos() != posSans {
		t.Fatalf("la sonde a changé le curseur : %d avec, %d sans", avec.BitPos(), posSans)
	}
	if !present || val != 0x0ABC || tail != 0b10 {
		t.Fatalf("lecture : present=%v val=%#x tail=%#x, attendu true/0xabc/0x2", present, val, tail)
	}
	if len(vues) != 1 {
		t.Fatalf("publications : %d, attendu 1", len(vues))
	}
	v := vues[0]
	if v.Kind != UnitRefVarWidth || !v.Present || v.Val != 0x0ABC || v.Tail != 0b10 ||
		v.StartBit != 0 || v.EndBit != posSans {
		t.Errorf("publication : %+v (curseur attendu 0 -> %d)", v, posSans)
	}

	// PORTE FERMÉE : la sonde doit publier l'ABSENCE, sinon un canal muet serait
	// indiscernable d'un canal jamais lu — et la mesure V5 compte les deux séparément.
	vues = nil
	ferme := make([]byte, 8)
	if _, _, present := consume1408f0ac4Probe(NewBitReader(ferme), false); present {
		t.Fatal("porte fermée lue comme ouverte")
	}
	if len(vues) != 1 || vues[0].Present {
		t.Errorf("porte fermée : publications %+v, attendu une publication non présente", vues)
	}
}
