package weaponv3

import "testing"

// TestUSEstimator_NoFrames — un buffer sans paquet FRAME renvoie toujours
// float64(startMS).
func TestUSEstimator_NoFrames(t *testing.T) {
	est := USEstimator([]byte{1, 2, 3}, 5000) // trop court pour un header
	if got := est(0); got != 5000 {
		t.Fatalf("buffer sans FRAME attendu 5000, obtenu %v", got)
	}
	if got := est(999); got != 5000 {
		t.Fatalf("buffer sans FRAME attendu 5000, obtenu %v", got)
	}
}

// TestUSEstimator_Synthetic — sur deux FRAMES synthétiques, l'estimateur ancre
// le 1er FRAME sur startMS et convertit l'écart µs en ms pour le 2e.
func TestUSEstimator_Synthetic(t *testing.T) {
	// FRAME 1 : us=1_000_000, size=4 ; FRAME 2 : us=1_500_000, size=4.
	// Ancre = startMS=2000 ms ; 2e FRAME = 2000 + (1_500_000-1_000_000)/1000 = 2500 ms.
	buf := append(framePacket(1_000_000, 4), framePacket(1_500_000, 4)...)
	est := USEstimator(buf, 2000)

	// FRAME 1 payload commence à offset 16.
	if got := est(16); got != 2000 {
		t.Fatalf("1er FRAME attendu 2000 ms, obtenu %v", got)
	}
	// FRAME 2 payload commence après [header1(16)+payload1(4)] + header2(16) = 36.
	if got := est(36); got != 2500 {
		t.Fatalf("2e FRAME attendu 2500 ms, obtenu %v", got)
	}
}

// framePacket forge un paquet FRAME (Type==0) : [Type u16=0][b2][b3][Size u32][µs u64][payload].
func framePacket(us uint64, size int) []byte {
	p := make([]byte, packetHeaderSize+size)
	// Type=0 (frameType) en LE @0 -> octets nuls, déjà OK.
	p[4] = byte(size)
	p[5] = byte(size >> 8)
	p[6] = byte(size >> 16)
	p[7] = byte(size >> 24)
	for i := 0; i < 8; i++ {
		p[8+i] = byte(us >> (8 * i))
	}
	return p
}

// TestUSEstimator_CacheMonotone — sur 000d5950/chunk_02 (skip si absent),
// l'estimateur ne panique pas, renvoie des ms >= startMS et croissants avec
// bytePos.
func TestUSEstimator_CacheMonotone(t *testing.T) {
	chunk := loadCachedChunk(t, "000d5950", "chunk_02.bin")
	if chunk == nil {
		t.Skip("cache film 000d5950/chunk_02.bin absent — skip estimateur réel")
	}
	const startMS = 1000
	est := USEstimator(chunk, startMS)

	prev := est(0)
	if prev < float64(startMS) {
		t.Fatalf("ms initiale %v < startMS %d", prev, startMS)
	}
	// Parcours par pas réguliers : les ms doivent rester >= startMS et croître.
	step := len(chunk) / 50
	if step < 1 {
		step = 1
	}
	for pos := 0; pos < len(chunk); pos += step {
		ms := est(pos)
		if ms < float64(startMS) {
			t.Fatalf("ms %v < startMS %d à pos %d", ms, startMS, pos)
		}
		if ms < prev {
			t.Fatalf("ms non croissante : %v < %v à pos %d", ms, prev, pos)
		}
		prev = ms
	}
}
