package analysis

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"testing"
)

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
	if ev.EventType != "kill" {
		t.Errorf("EventType: got %q, want %q", ev.EventType, "kill")
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
	if events[0].EventType != "death" {
		t.Errorf("EventType: got %q, want %q", events[0].EventType, "death")
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
	if !types["kill"] || !types["death"] {
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
	if ev.EventType != "kill" {
		t.Errorf("EventType: got %q, want %q", ev.EventType, "kill")
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
	if err != nil || ev != "kill" {
		t.Errorf("inferEventType kill: got %q, %v", ev, err)
	}
}

func TestInferEventType_Death(t *testing.T) {
	ev, err := inferEventType(typeHintDeath, false)
	if err != nil || ev != "death" {
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
	if err != nil || ev != "kill" {
		t.Errorf("inferEventType type50 not medal: got %q, %v", ev, err)
	}
}

func TestInferEventType_Unknown(t *testing.T) {
	_, err := inferEventType(99, false)
	if err == nil {
		t.Error("expected error for unknown type_hint=99, got nil")
	}
}
