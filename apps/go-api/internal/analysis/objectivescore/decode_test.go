package objectivescore

import (
	"encoding/binary"
	"testing"
)

// --- Constructeur de fixtures bit (token + varints synthétiques) ---

// bitWriter écrit des bits MSB-first dans un buffer croissant.
type bitWriter struct {
	bits []int
}

func (w *bitWriter) writeBits(v, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bits = append(w.bits, (v>>uint(i))&1)
	}
}

// writeVarAtCont écrit un varint à continuation (miroir de varAtBit) : si v<0x40
// → 1 octet (bit 0x40 à 0) ; sinon 2 octets avec bit 0x40 du 1er à 1, 6 bits
// hauts dans le 1er octet, 8 bits bas dans le 2e.
func (w *bitWriter) writeVarAtCont(v int) {
	if v < 0x40 {
		w.writeBits(v&0x3f, 8) // bit 0x40 reste 0
		return
	}
	hi := (v >> 8) & 0x3f
	lo := v & 0xff
	w.writeBits(0x40|hi, 8)
	w.writeBits(lo, 8)
}

func (w *bitWriter) bytes() []byte {
	n := (len(w.bits) + 7) / 8
	out := make([]byte, n)
	for i, b := range w.bits {
		if b == 1 {
			out[i/8] |= 1 << uint(7-(i%8))
		}
	}
	return out
}

// wrapType2 emballe un payload TYPE_2 dans un conteneur de paquet film valide
// (header 16o : type=2 LE@0, size LE@4) + un CHUNK_END derrière.
func wrapType2(payload []byte) []byte {
	hdr := make([]byte, packetHdr)
	binary.LittleEndian.PutUint16(hdr[0:], packetType2)
	binary.LittleEndian.PutUint32(hdr[4:], uint32(len(payload)))
	out := append(hdr, payload...)
	end := make([]byte, packetHdr)
	binary.LittleEndian.PutUint16(end[0:], packetEnd)
	return append(out, end...)
}

// buildStrongholdsChunk : payload avec du padding jusqu'à amener le token 0x7B6
// au byte tokenWinLo (aligné), suivi immédiatement de team1(+23) chevauchant
// team0(+24). Pour rester simple et déterministe, on encode SÉPARÉMENT : on
// place le token, puis on positionne les deux varints aux offsets bit exacts en
// écrivant un flux dont on contrôle chaque bit.
//
// Layout (depuis le 1er bit du token) :
//
//	bit 0..11   : token 0x7B6
//	bit 12..22  : padding (zéros) — non lu
//	bit 23..    : team1 varint (lu à +23)
//	bit 24..    : team0 varint (lu à +24)  ← chevauche team1 d'1 bit
//
// Le chevauchement +23/+24 reflète la réalité du film ; pour un test
// DÉTERMINISTE on choisit team0 tel que ses 8 premiers bits (à +24) donnent
// `team0`, et on n'asserte sur team1 que la lecture brute à +23 (cohérence
// décodeur, pas vérité métier).
func buildStrongholdsChunk(team0Vals []int, startMS int) []ChunkInput {
	var chunks []ChunkInput
	for _, t0 := range team0Vals {
		w := &bitWriter{}
		// Padding initial pour amener le token à tokenWinLo*8 bits.
		for len(w.bits) < tokenWinLo*8 {
			w.writeBits(0, 1)
		}
		w.writeBits(scoreToken, scoreTokenW)   // token
		w.writeBits(0, shOffTeam0-scoreTokenW) // padding jusqu'à token+24
		w.writeVarAtCont(t0)                   // team0 à +24
		// padding de fin pour dépasser la fenêtre + marge varint
		w.writeBits(0, 64)
		chunks = append(chunks, ChunkInput{
			Data: wrapType2(w.bytes()), StartMS: startMS, ChunkType: packetType2,
		})
		startMS += 20000
	}
	return chunks
}

// --- Tests primitives ---

func TestTokenBitAndVarAtBit_Synthetic(t *testing.T) {
	w := &bitWriter{}
	for len(w.bits) < tokenWinLo*8 {
		w.writeBits(0, 1)
	}
	tokenStart := len(w.bits)
	w.writeBits(scoreToken, scoreTokenW)
	w.writeBits(0, shOffTeam0-scoreTokenW)
	w.writeVarAtCont(300) // > 0x40 → varint 2 octets
	w.writeBits(0, 64)
	p := w.bytes()

	tb := tokenBit(p, tokenWinLo, tokenWinHi)
	if tb != tokenStart {
		t.Fatalf("tokenBit = %d, want %d", tb, tokenStart)
	}
	if got := varAtBit(p, tb+shOffTeam0); got != 300 {
		t.Fatalf("varAtBit(token+24) = %d, want 300", got)
	}
}

func TestType2Payload_Extract(t *testing.T) {
	payload := []byte{0xde, 0xad, 0xbe, 0xef, 0x10, 0x20}
	chunk := wrapType2(payload)
	got := type2Payload(chunk)
	if len(got) != len(payload) {
		t.Fatalf("len(payload) = %d, want %d", len(got), len(payload))
	}
	for i := range payload {
		if got[i] != payload[i] {
			t.Fatalf("payload[%d] = %#x, want %#x", i, got[i], payload[i])
		}
	}
}

// --- Tests Strongholds (calibration) ---

func TestDecodeStrongholds_MonotoneCalibratedToFinal(t *testing.T) {
	raw0 := []int{0, 5, 10, 25, 50} // monotone, last=50
	chunks := buildStrongholdsChunk(raw0, 0)
	const finalT0 = 193

	frames := DecodeStrongholds(chunks, finalT0, 0)
	if len(frames) != len(raw0) {
		t.Fatalf("len(frames) = %d, want %d", len(frames), len(raw0))
	}
	// Dernière frame == final exact.
	if last := frames[len(frames)-1].Team0; last != finalT0 {
		t.Fatalf("dernière frame Team0 = %d, want %d (final exact)", last, finalT0)
	}
	// Monotone croissant.
	for i := 1; i < len(frames); i++ {
		if frames[i].Team0 < frames[i-1].Team0 {
			t.Fatalf("Team0 non monotone à i=%d : %d < %d", i, frames[i].Team0, frames[i-1].Team0)
		}
	}
	// TimeMS = start_ms du chunk.
	if frames[1].TimeMS != 20000 {
		t.Errorf("frames[1].TimeMS = %d, want 20000", frames[1].TimeMS)
	}
	if frames[0].Source != sourceStrongholds || frames[0].Confidence != confidenceApprox {
		t.Errorf("source/conf = %q/%q", frames[0].Source, frames[0].Confidence)
	}
}

func TestCalibrateByFinal(t *testing.T) {
	// last=50, final=193 → out[last]=193, monotone, out[0]=0.
	out := calibrateByFinal([]int{0, 25, 50}, 193)
	if out[0] != 0 || out[2] != 193 {
		t.Fatalf("calibrate = %v, want [0 ~96 193]", out)
	}
	// round(25*193/50) = round(96.5) = 97.
	if out[1] != 97 {
		t.Errorf("out[1] = %d, want 97", out[1])
	}
	// Séquence plate → tout 0 (pas de division par 0).
	if flat := calibrateByFinal([]int{0, 0, 0}, 100); flat[0] != 0 || flat[2] != 0 {
		t.Errorf("flat calibrate = %v, want [0 0 0]", flat)
	}
}

// --- Tests KOTH (variantes byte-aligned) ---

// buildKOTHChunkByteAligned construit un chunk dont le token est byte-aligné à
// tokenWinLo, puis place des octets connus aux offsets +12/+13/+14/+16 relatifs
// au byte du token.
func buildKOTHChunkByteAligned(at12, at13, at14, at16 byte) ChunkInput {
	p := make([]byte, tokenWinLo+40)
	// token 0x7B6 sur 12 bits MSB-first à partir du byte tokenWinLo (aligné) :
	// 0x7B6 = 0111 1011 0110 → byte[lo]=0x7B, byte[lo+1] bits hauts = 0110....
	p[tokenWinLo] = 0x7B
	p[tokenWinLo+1] = 0x60 // 0110 0000
	ab := tokenWinLo
	p[ab+kothBOffT0] = at12
	p[ab+kothAOffT1] = at13
	p[ab+kothAOffTot] = at14
	p[ab+kothBOffT1] = at16
	return ChunkInput{Data: wrapType2(p), StartMS: 0, ChunkType: packetType2}
}

func TestDecodeKOTH_VariantB_MetersDiv5(t *testing.T) {
	// team0 meter=20 → 20/5=4 ; team1 meter=10 → 10/5=2.
	c := buildKOTHChunkByteAligned(20, 0, 0, 10)
	frames := DecodeKOTH([]ChunkInput{c}, VariantB)
	if len(frames) != 1 {
		t.Fatalf("len(frames) = %d, want 1", len(frames))
	}
	if frames[0].Team0 != 4 || frames[0].Team1 != 2 {
		t.Fatalf("KOTH B = %d-%d, want 4-2", frames[0].Team0, frames[0].Team1)
	}
	if frames[0].Source != sourceKOTHVariantB {
		t.Errorf("source = %q, want %q", frames[0].Source, sourceKOTHVariantB)
	}
}

func TestDecodeKOTH_VariantA_Points(t *testing.T) {
	// total=t2[+14]=104 ; team1=t2[+13]>>4 = 0x80>>4 = 8 ; team0=104-8=96.
	c := buildKOTHChunkByteAligned(0, 0x80, 104, 0)
	frames := DecodeKOTH([]ChunkInput{c}, VariantA)
	if len(frames) != 1 {
		t.Fatalf("len(frames) = %d, want 1", len(frames))
	}
	if frames[0].Team0 != 96 || frames[0].Team1 != 8 {
		t.Fatalf("KOTH A = %d-%d, want 96-8", frames[0].Team0, frames[0].Team1)
	}
}

// --- Tests dispatch ---

func TestClassifyMode(t *testing.T) {
	cases := map[string]Mode{
		"Arena:Strongholds":       ModeStrongholds,
		"KOTH:Arena":              ModeKOTH,
		"Ranked:King of the Hill": ModeKOTH,
		"Arena:Slayer":            ModeUnsupported,
		"CTF:Arena":               ModeUnsupported,
		"":                        ModeUnsupported,
	}
	for name, want := range cases {
		if got := ClassifyMode(name); got != want {
			t.Errorf("ClassifyMode(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestDecodeScoreTimeline_DispatchesAndAutoVariant(t *testing.T) {
	// Strongholds → calibre sur final.
	sh := DecodeScoreTimeline("Arena:Strongholds", buildStrongholdsChunk([]int{0, 50}, 0), 200, 0)
	if len(sh) != 2 || sh[1].Team0 != 200 {
		t.Fatalf("strongholds dispatch = %+v", sh)
	}
	// KOTH final bas → variante B.
	cB := buildKOTHChunkByteAligned(20, 0, 0, 10)
	kb := DecodeScoreTimeline("Ranked:King of the Hill", []ChunkInput{cB}, 4, 2)
	if len(kb) != 1 || kb[0].Source != sourceKOTHVariantB {
		t.Fatalf("KOTH B auto = %+v", kb)
	}
	// KOTH final haut → variante A.
	cA := buildKOTHChunkByteAligned(0, 0x80, 104, 0)
	ka := DecodeScoreTimeline("KOTH:Arena", []ChunkInput{cA}, 105, 8)
	if len(ka) != 1 || ka[0].Source != sourceKOTHVariantA {
		t.Fatalf("KOTH A auto = %+v", ka)
	}
	// Mode non supporté → nil.
	if got := DecodeScoreTimeline("Arena:Slayer", nil, 50, 49); got != nil {
		t.Fatalf("unsupported = %+v, want nil", got)
	}
}
