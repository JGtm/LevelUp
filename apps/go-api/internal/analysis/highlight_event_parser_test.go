package analysis

import (
	"bytes"
	"compress/zlib"
	_ "embed"
	"encoding/binary"
	"testing"
)

//go:embed testdata/v41_chunk_he.bin
var realV41Chunk []byte

// ─── Helpers de construction de fixtures binaires ────────────────────────────

// buildEventBytes construit les 60 octets d'event data (layout A, version ≤38/≥41).
// Structure :
//
//	[0:32]  gamertag UTF-16LE (padded avec 0x00)
//	[32:47] padding
//	[47]    type_hint
//	[48:52] time_ms (big-endian uint32)
//	[52:55] padding
//	[55]    is_medal (0 ou 1)
//	[56:59] padding
//	[59]    medal_type
func buildEventBytes(gamertag string, typeHint int, timeMS int, isMedal bool, medalType int) []byte {
	b := make([]byte, eventDataBytes)

	// Encoder le gamertag en UTF-16LE dans [0:32].
	runes := []rune(gamertag)
	for i := 0; i < 16 && i < len(runes); i++ {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(runes[i]))
	}

	// type_hint à l'offset 47.
	b[47] = byte(typeHint)

	// time_ms big-endian à [48:52].
	binary.BigEndian.PutUint32(b[48:52], uint32(timeMS))

	// is_medal à l'offset 55.
	if isMedal {
		b[55] = 1
	}

	// medal_type à l'offset 59.
	b[59] = byte(medalType)
	return b
}

// buildRawChunk construit le flux binaire décompressé pour un événement.
// Structure minimale :
//
//	[...8 bytes XUID little-endian...][0x2d][0xc0] → marqueur de présence du XUID
//	[...padding jusqu'à endMarker...]
//	[60 bytes event data][0x00, 0x00, 0x2e, 0xe0]   → marqueur de fin
func buildRawChunk(xuid uint64, gamertag string, typeHint, timeMS int, isMedal bool, medalType int) []byte {
	var buf bytes.Buffer

	// Préfixe de rembourrage pour que i >= 9 soit vérifié.
	buf.Write(make([]byte, 20))

	// Écrire XUID (8 bytes LE) suivi de 0x2d + 0xc0.
	xuidBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(xuidBytes, xuid)
	buf.Write(xuidBytes) // bytes à position 20..27
	buf.WriteByte(0x2d)  // position 28 (i-1)
	buf.WriteByte(0xc0)  // position 29 (i) → scanner le détecte ici

	// Rembourrage jusqu'au bloc event + marqueur de fin.
	buf.Write(make([]byte, 30))

	// 60 bytes d'event data.
	buf.Write(buildEventBytes(gamertag, typeHint, timeMS, isMedal, medalType))

	// Marqueur de fin.
	buf.Write(endMarker)

	// Rembourrage final.
	buf.Write(make([]byte, 10))

	return buf.Bytes()
}

// zlibCompress compresse les données avec zlib.
func zlibCompress(data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, _ = w.Write(data)
	_ = w.Close()
	return buf.Bytes()
}

// ─── Tests ParseHighlightEvents ──────────────────────────────────────────────

func TestParseHighlightEvents_Nil(t *testing.T) {
	events, err := ParseHighlightEvents(nil, 40)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestParseHighlightEvents_EmptySlice(t *testing.T) {
	events, err := ParseHighlightEvents([]byte{}, 40)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestParseHighlightEvents_InvalidZlib(t *testing.T) {
	_, err := ParseHighlightEvents([]byte{0x01, 0x02, 0x03}, 40)
	if err == nil {
		t.Fatal("expected error for invalid zlib data, got nil")
	}
}

func TestParseHighlightEvents_KillEvent(t *testing.T) {
	const (
		xuid     = uint64(2_500_000_000_000_001) // dans la plage [2e15..3e15]
		gamertag = "TestPlayer"
		timeMS   = 5000
	)
	raw := buildRawChunk(xuid, gamertag, typeHintKill, timeMS, false, 0)
	compressed := zlibCompress(raw)

	events, err := ParseHighlightEvents(compressed, 42) // version ≥ 41 → layout A
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least 1 event, got 0")
	}

	ev := events[0]
	if ev.XUID != xuid {
		t.Errorf("XUID: got %d, want %d", ev.XUID, xuid)
	}
	if ev.EventType != EventTypeKill {
		t.Errorf("EventType: got %q, want %q", ev.EventType, EventTypeKill)
	}
	if ev.TimeMS != timeMS {
		t.Errorf("TimeMS: got %d, want %d", ev.TimeMS, timeMS)
	}
	if ev.IsMedal {
		t.Error("IsMedal: got true, want false")
	}
	if ev.Gamertag != gamertag {
		t.Errorf("Gamertag: got %q, want %q", ev.Gamertag, gamertag)
	}
}

func TestParseHighlightEvents_DeathEvent(t *testing.T) {
	const xuid = uint64(2_600_000_000_000_002)
	raw := buildRawChunk(xuid, "Victim", typeHintDeath, 8000, false, 0)
	events, err := ParseHighlightEvents(zlibCompress(raw), 35)
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least 1 event, got 0")
	}
	if events[0].EventType != EventTypeDeath {
		t.Errorf("EventType: got %q, want %q", events[0].EventType, EventTypeDeath)
	}
}

func TestParseHighlightEvents_MedalEvent(t *testing.T) {
	const xuid = uint64(2_700_000_000_000_003)
	// type_hint=50 + is_medal=true → "medal"
	// Éviter timeMS=12000 (0x00002EE0) qui est identique à endMarker et génère une fausse détection.
	const timeMS = 15000
	raw := buildRawChunk(xuid, "Medallist", 50, timeMS, true, 200)
	events, err := ParseHighlightEvents(zlibCompress(raw), 42)
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least 1 event, got 0")
	}
	ev := events[0]
	if ev.EventType != "medal" {
		t.Errorf("EventType: got %q, want %q", ev.EventType, "medal")
	}
	if !ev.IsMedal {
		t.Error("IsMedal: got false, want true")
	}
	if ev.MedalType != 200 {
		t.Errorf("MedalType: got %d, want 200", ev.MedalType)
	}
}

func TestParseHighlightEvents_XUIDOutOfRange_Ignored(t *testing.T) {
	// XUID hors plage [2e15..3e15] → ignoré
	const xuid = uint64(1_000_000_000_000_000) // < minXUID
	raw := buildRawChunk(xuid, "Ghost", typeHintKill, 1000, false, 0)
	events, err := ParseHighlightEvents(zlibCompress(raw), 42)
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for out-of-range XUID, got %d", len(events))
	}
}

func TestParseHighlightEvents_ModeEvent(t *testing.T) {
	const xuid = uint64(2_500_000_000_000_010)
	// type_hint=10 + isMedal=false → "mode"
	raw := buildRawChunk(xuid, "ModePlayer", typeHintMode, 6000, false, 0)
	events, err := ParseHighlightEvents(zlibCompress(raw), 42)
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least 1 event, got 0")
	}
	if events[0].EventType != "mode" {
		t.Errorf("EventType: got %q, want %q", events[0].EventType, "mode")
	}
}

func TestParseHighlightEvents_MultipleEvents(t *testing.T) {
	// Deux XUIDs distincts dans le même flux binaire → 2 events retournés.
	const (
		xuid1 = uint64(2_500_000_000_000_020)
		xuid2 = uint64(2_600_000_000_000_021)
	)
	chunk1 := buildRawChunk(xuid1, "Alpha", typeHintKill, 1000, false, 0)
	chunk2 := buildRawChunk(xuid2, "Beta", typeHintDeath, 2000, false, 0)
	// Concaténer les deux chunks dans un seul flux.
	combined := append(chunk1, chunk2...)
	events, err := ParseHighlightEvents(zlibCompress(combined), 42)
	if err != nil {
		t.Fatalf("ParseHighlightEvents multi: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
	types := map[string]bool{}
	for _, ev := range events {
		types[ev.EventType] = true
	}
	if !types[EventTypeKill] || !types[EventTypeDeath] {
		t.Errorf("expected both kill and death events, got types=%v", types)
	}
}

func TestParseHighlightEvents_VersionLayout39(t *testing.T) {
	// Version 39-40 : gamertag à l'offset 12 dans le bloc d'event (layout B).
	const xuid = uint64(2_800_000_000_000_004)
	b := make([]byte, eventDataBytes)
	gamertag := "LayoutB"
	runes := []rune(gamertag)
	for i := 0; i < 16 && i < len(runes); i++ {
		binary.LittleEndian.PutUint16(b[12+i*2:], uint16(runes[i]))
	}
	b[47] = typeHintKill
	binary.BigEndian.PutUint32(b[48:52], 3000)

	raw := buildRawChunkWithEventBytes(xuid, b)
	events, err := ParseHighlightEvents(zlibCompress(raw), 39)
	if err != nil {
		t.Fatalf("ParseHighlightEvents v39: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected at least 1 event, got 0")
	}
	ev := events[0]
	if ev.EventType != EventTypeKill {
		t.Errorf("EventType: got %q, want %q", ev.EventType, EventTypeKill)
	}
	if ev.Gamertag != gamertag {
		t.Errorf("Gamertag v39: got %q, want %q", ev.Gamertag, gamertag)
	}
}

// buildRawChunkWithEventBytes construit le flux brut en fournissant directement
// les 60 octets d'event (permet de tester les layouts manuellement).
func buildRawChunkWithEventBytes(xuid uint64, eventBytes []byte) []byte {
	var buf bytes.Buffer
	buf.Write(make([]byte, 20))
	xuidB := make([]byte, 8)
	binary.LittleEndian.PutUint64(xuidB, xuid)
	buf.Write(xuidB)
	buf.WriteByte(0x2d)
	buf.WriteByte(0xc0)
	buf.Write(make([]byte, 30))
	buf.Write(eventBytes)
	buf.Write(endMarker)
	buf.Write(make([]byte, 10))
	return buf.Bytes()
}

// ─── Tests decodeUTF16LE ──────────────────────────────────────────────────────

func TestDecodeUTF16LE_Basic(t *testing.T) {
	// "AB" en UTF-16LE = [0x41, 0x00, 0x42, 0x00]
	b := []byte{0x41, 0x00, 0x42, 0x00}
	got := decodeUTF16LE(b)
	if got != "AB" {
		t.Errorf("decodeUTF16LE: got %q, want %q", got, "AB")
	}
}

func TestDecodeUTF16LE_WithNullTerminator(t *testing.T) {
	// "Hi\x00\x00padding" → doit retourner "Hi"
	b := []byte{0x48, 0x00, 0x69, 0x00, 0x00, 0x00, 0x58, 0x00}
	got := decodeUTF16LE(b)
	if got != "Hi" {
		t.Errorf("decodeUTF16LE null-terminated: got %q, want %q", got, "Hi")
	}
}

func TestDecodeUTF16LE_Empty(t *testing.T) {
	got := decodeUTF16LE([]byte{})
	if got != "" {
		t.Errorf("decodeUTF16LE empty: got %q, want %q", got, "")
	}
}

// ─── Tests inferEventType ─────────────────────────────────────────────────────

func TestInferEventType_Kill(t *testing.T) {
	ev, err := inferEventType(typeHintKill, false)
	if err != nil || ev != EventTypeKill {
		t.Errorf("inferEventType kill: got %q, %v", ev, err)
	}
}

func TestInferEventType_Death(t *testing.T) {
	ev, err := inferEventType(typeHintDeath, false)
	if err != nil || ev != EventTypeDeath {
		t.Errorf("inferEventType death: got %q, %v", ev, err)
	}
}

func TestInferEventType_Mode(t *testing.T) {
	ev, err := inferEventType(typeHintMode, false)
	if err != nil || ev != "mode" {
		t.Errorf("inferEventType mode: got %q, %v", ev, err)
	}
}

func TestInferEventType_Medal(t *testing.T) {
	// typeHint dans medalSortingWeights + isMedal=true → "medal"
	ev, err := inferEventType(50, true)
	if err != nil || ev != "medal" {
		t.Errorf("inferEventType medal: got %q, %v", ev, err)
	}
}

func TestInferEventType_MedalNotInWeights_Kill(t *testing.T) {
	// typeHint=50 mais isMedal=false → kill (50 == typeHintKill)
	ev, err := inferEventType(50, false)
	if err != nil || ev != EventTypeKill {
		t.Errorf("inferEventType type50 not medal: got %q, %v", ev, err)
	}
}

func TestInferEventType_Unknown(t *testing.T) {
	_, err := inferEventType(99, false)
	if err == nil {
		t.Error("expected error for unknown type_hint=99, got nil")
	}
}

// ─── Tests bit-reader (lecture à offset non byte-aligné) ─────────────────────

func TestReadByteAtBit_Aligned(t *testing.T) {
	data := []byte{0xAB, 0xCD, 0xEF}
	for i, want := range []byte{0xAB, 0xCD, 0xEF} {
		got := readByteAtBit(data, i*8)
		if got != want {
			t.Errorf("bit=%d: got %02x want %02x", i*8, got, want)
		}
	}
}

func TestReadByteAtBit_Shifted(t *testing.T) {
	// data = 1010_1011 1100_1101  → bit 4 = 1011_1100 = 0xBC
	data := []byte{0xAB, 0xCD}
	got := readByteAtBit(data, 4)
	if got != 0xBC {
		t.Errorf("bit=4: got %02x want %02x", got, 0xBC)
	}
	// bit 1 = 0101_0111 (top 7 bits of 0xAB shifted left + MSB of 0xCD = 0)
	// 0xAB = 10101011, shift left 1 = 01010110, lo = 0xCD>>7 = 1 → 01010111 = 0x57
	got = readByteAtBit(data, 1)
	if got != 0x57 {
		t.Errorf("bit=1: got %02x want %02x", got, 0x57)
	}
}

func TestReadByteAtBit_OutOfBounds(t *testing.T) {
	data := []byte{0xAB}
	if got := readByteAtBit(data, -1); got != 0 {
		t.Errorf("negative offset: got %02x want 0", got)
	}
	if got := readByteAtBit(data, 1); got != 0 {
		t.Errorf("offset+8 > len*8: got %02x want 0", got)
	}
}

func TestReadUint64LEAtBit_Aligned(t *testing.T) {
	xuid := uint64(2_500_000_000_000_010)
	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data, xuid)
	got := readUint64LEAtBit(data, 0)
	if got != xuid {
		t.Errorf("aligned: got %d want %d", got, xuid)
	}
}

func TestReadUint64LEAtBit_Shifted(t *testing.T) {
	// On préfixe 3 bits de zéros, puis l'XUID byte-aligné.
	xuid := uint64(2_700_000_000_000_001)
	xb := make([]byte, 8)
	binary.LittleEndian.PutUint64(xb, xuid)
	shifted := shiftBitsRight(xb, 3) // décale tout de 3 bits à droite
	got := readUint64LEAtBit(shifted, 3)
	if got != xuid {
		t.Errorf("shifted: got %d want %d", got, xuid)
	}
}

func TestFindBitMarker_FindsByteAligned(t *testing.T) {
	pat := []byte{0x00, 0x00, 0x2e, 0xe0}
	data := append([]byte{0xAB, 0xCD}, pat...)
	got := findBitMarker(data, 0, len(data)*8, pat)
	if got != 16 {
		t.Errorf("got bit %d want 16", got)
	}
}

func TestFindBitMarker_FindsBitShifted(t *testing.T) {
	pat := []byte{0x00, 0x00, 0x2e, 0xe0}
	// Stream 37 bits : 5 bits de tête à 1 (pour exclure une fausse détection à
	// bit 0 sur des zéros) + 32 bits du pattern.
	//   bits  0..4  = 11111
	//   bits  5..12 = 00000000  (pat byte 0)
	//   bits 13..20 = 00000000  (pat byte 1)
	//   bits 21..28 = 00101110  (pat byte 2 = 0x2e)
	//   bits 29..36 = 11100000  (pat byte 3 = 0xe0)
	// Repacké en bytes (MSB-first, padding zéros sur la fin) :
	merged := []byte{0xF8, 0x00, 0x01, 0x77, 0x00}

	got := findBitMarker(merged, 0, len(merged)*8, pat)
	if got != 5 {
		t.Errorf("got bit %d want 5", got)
	}
}

func TestFindBitMarker_NotFound(t *testing.T) {
	pat := []byte{0x00, 0x00, 0x2e, 0xe0}
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	got := findBitMarker(data, 0, len(data)*8, pat)
	if got != -1 {
		t.Errorf("got %d want -1", got)
	}
}

// ─── Tests scan bit-aligné (XUID à offset non multiple de 8) ────────────────

// shiftBitsRight décale `data` de `n` bits vers la droite (n entre 0 et 7),
// retournant un nouveau slice de longueur len(data)+1 (le dernier octet contient
// les bits décalés au-delà). Utilisé pour construire des fixtures non
// byte-alignées.
func shiftBitsRight(data []byte, n int) []byte {
	if n == 0 {
		return append([]byte{}, data...)
	}
	if n < 0 || n > 7 {
		panic("shiftBitsRight: n must be in [0..7]")
	}
	out := make([]byte, len(data)+1)
	for i := 0; i < len(data); i++ {
		out[i] |= data[i] >> uint(n)
		out[i+1] = data[i] << uint(8-n)
	}
	return out
}

func TestParseHighlightEvents_BitOffset_AllAlignments(t *testing.T) {
	const (
		xuid     = uint64(2_500_000_000_000_077)
		gamertag = "BitShifted"
		timeMS   = 7000
	)
	// Construire un raw byte-aligned, puis le décaler de N bits (N = 1..7) et
	// vérifier que le parser trouve toujours l'event.
	rawAligned := buildRawChunk(xuid, gamertag, typeHintKill, timeMS, false, 0)

	for shift := 0; shift <= 7; shift++ {
		t.Run("shift="+string(rune('0'+shift)), func(t *testing.T) {
			rawShifted := shiftBitsRight(rawAligned, shift)
			compressed := zlibCompress(rawShifted)

			events, err := ParseHighlightEvents(compressed, 42)
			if err != nil {
				t.Fatalf("ParseHighlightEvents: %v", err)
			}
			if len(events) == 0 {
				t.Fatalf("shift=%d: expected at least 1 event, got 0", shift)
			}
			ev := events[0]
			if ev.XUID != xuid {
				t.Errorf("shift=%d XUID: got %d want %d", shift, ev.XUID, xuid)
			}
			if ev.EventType != EventTypeKill {
				t.Errorf("shift=%d EventType: got %q want %q", shift, ev.EventType, EventTypeKill)
			}
			if ev.TimeMS != timeMS {
				t.Errorf("shift=%d TimeMS: got %d want %d", shift, ev.TimeMS, timeMS)
			}
			if ev.Gamertag != gamertag {
				t.Errorf("shift=%d Gamertag: got %q want %q", shift, ev.Gamertag, gamertag)
			}
		})
	}
}

func TestParseHighlightEvents_NoEndMarker_ReturnsNoEvent(t *testing.T) {
	// XUID valide + marqueur 0x2d/0xc0 mais aucun end-marker dans la fenêtre :
	// l'event ne doit pas être ajouté (et pas d'erreur fatale).
	const xuid = uint64(2_500_000_000_000_001)
	var buf bytes.Buffer
	buf.Write(make([]byte, 20))
	xb := make([]byte, 8)
	binary.LittleEndian.PutUint64(xb, xuid)
	buf.Write(xb)
	buf.WriteByte(0x2d)
	buf.WriteByte(0xc0)
	// Aucun end-marker, juste du padding.
	buf.Write(make([]byte, 100))

	events, err := ParseHighlightEvents(zlibCompress(buf.Bytes()), 42)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events when end-marker absent, got %d", len(events))
	}
}

func TestParseHighlightEvents_FalsePositiveEndMarker_FallsThrough(t *testing.T) {
	// Si la fenêtre contient un end-marker bit-shifté qui produit un type_hint
	// inconnu, le parser doit continuer jusqu'au vrai end-marker et retourner
	// l'event correct. C'est exactement le scénario qui faisait échouer la
	// première version du parser bit-aligné.
	const xuid = uint64(2_500_000_000_000_002)
	raw := buildRawChunk(xuid, "RobustGT", typeHintKill, 4000, false, 0)
	events, err := ParseHighlightEvents(zlibCompress(raw), 42)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected the parser to skip false-positive end-markers and find the real one")
	}
	if events[0].EventType != EventTypeKill {
		t.Errorf("EventType: got %q want %q", events[0].EventType, EventTypeKill)
	}
}

// ─── Test fixture v41 réel (capturé depuis l'API Halo) ──────────────────────

// TestParseHighlightEvents_RealV41Fixture parse un chunk highlight events réel
// téléchargé depuis l'API Halo Infinite (FilmMajorVersion=41) — c'est le test
// qui échouait avant le fix bit-aligné. Le fixture est commité dans testdata/.
func TestParseHighlightEvents_RealV41Fixture(t *testing.T) {
	if len(realV41Chunk) == 0 {
		t.Fatal("fixture testdata/v41_chunk_he.bin manquant")
	}

	events, err := ParseHighlightEvents(realV41Chunk, 41)
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}

	// Halo 4v4 ⇒ 8 humains. Avant le fix : 0 events. Après : ~270 events typiquement.
	if len(events) < 100 {
		t.Fatalf("expected at least 100 events from real v41 chunk, got %d (regression vers le bug byte-aligné ?)", len(events))
	}

	xuids := map[uint64]struct{}{}
	typeCount := map[string]int{}
	for _, ev := range events {
		xuids[ev.XUID] = struct{}{}
		typeCount[ev.EventType]++

		if ev.XUID <= minXUID || ev.XUID >= maxXUID {
			t.Errorf("XUID hors plage Xbox Live: %d", ev.XUID)
		}
		if ev.TypeHint < 0 || ev.TypeHint > 255 {
			t.Errorf("type_hint hors [0..255]: %d", ev.TypeHint)
		}
	}

	// 4v4 = 8 humains attendus
	if len(xuids) < 4 || len(xuids) > 16 {
		t.Errorf("nombre de joueurs distincts inattendu: %d (attendu 4..16)", len(xuids))
	}

	// Le match doit avoir au moins un kill et une death (matchmaking standard).
	if typeCount[EventTypeKill] == 0 {
		t.Errorf("aucun event 'kill' parsé — parser cassé ou match anormal")
	}
	if typeCount[EventTypeDeath] == 0 {
		t.Errorf("aucun event 'death' parsé — parser cassé ou match anormal")
	}

	t.Logf("v41 fixture parsed: %d events, %d distinct XUIDs, kills=%d deaths=%d medals=%d mode=%d",
		len(events), len(xuids),
		typeCount[EventTypeKill], typeCount[EventTypeDeath],
		typeCount[EventTypeMedal], typeCount[EventTypeMode])
}
