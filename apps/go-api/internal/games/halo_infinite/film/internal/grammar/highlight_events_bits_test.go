package grammar

import (
	"bytes"
	"encoding/binary"
	"testing"

	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// highlight_events_bits_test.go — LES LECTURES AU BIT DU LECTEUR DES TEMPS FORTS, ET LE SCAN
// NON ALIGNE SUR L OCTET.
//
// SCINDE DE `highlight_events_test.go` AU LOT 2.5.e : une fois descendu dans `film/`, le
// fichier de tests tombait sous le ratchet `archlint/film_file_size_test.go` (667 lignes pour
// un seuil de 500, CLAUDE.md regle 5). La coupe suit la frontiere naturelle du fichier : d un
// cote ce que `ParseHighlightEvents` DECODE, de l autre COMMENT il atteint les bits. Aucun cas
// n a ete ajoute ni retire.

// ─── Tests des lectures au bit (offset non aligné sur l'octet) ────────────────
//
// `source.OctetAuBit` a remplacé la copie locale `readByteAtBit` au lot 2.5.e : ces cas sont
// la preuve, du côté de l'appelant, que la convention de bord n'a pas changé (ZÉRO quand
// l'octet ne tient pas entièrement dans le tampon, des deux côtés).

func TestOctetAuBitDuLecteurDeTempsForts_Aligned(t *testing.T) {
	data := []byte{0xAB, 0xCD, 0xEF}
	for i, want := range []byte{0xAB, 0xCD, 0xEF} {
		got := source.OctetAuBit(data, i*8)
		if got != want {
			t.Errorf("bit=%d: got %02x want %02x", i*8, got, want)
		}
	}
}

func TestOctetAuBitDuLecteurDeTempsForts_Shifted(t *testing.T) {
	// data = 1010_1011 1100_1101  → bit 4 = 1011_1100 = 0xBC
	data := []byte{0xAB, 0xCD}
	got := source.OctetAuBit(data, 4)
	if got != 0xBC {
		t.Errorf("bit=4: got %02x want %02x", got, 0xBC)
	}
	// bit 1 = 0101_0111 (top 7 bits of 0xAB shifted left + MSB of 0xCD = 0)
	// 0xAB = 10101011, shift left 1 = 01010110, lo = 0xCD>>7 = 1 → 01010111 = 0x57
	got = source.OctetAuBit(data, 1)
	if got != 0x57 {
		t.Errorf("bit=1: got %02x want %02x", got, 0x57)
	}
}

func TestOctetAuBitDuLecteurDeTempsForts_OutOfBounds(t *testing.T) {
	data := []byte{0xAB}
	if got := source.OctetAuBit(data, -1); got != 0 {
		t.Errorf("negative offset: got %02x want 0", got)
	}
	if got := source.OctetAuBit(data, 1); got != 0 {
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
			if ev.EventType != highlightevent.EventTypeKill {
				t.Errorf("shift=%d EventType: got %q want %q", shift, ev.EventType, highlightevent.EventTypeKill)
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
	if events[0].EventType != highlightevent.EventTypeKill {
		t.Errorf("EventType: got %q want %q", events[0].EventType, highlightevent.EventTypeKill)
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
	if typeCount[highlightevent.EventTypeKill] == 0 {
		t.Errorf("aucun event 'kill' parsé — parser cassé ou match anormal")
	}
	if typeCount[highlightevent.EventTypeDeath] == 0 {
		t.Errorf("aucun event 'death' parsé — parser cassé ou match anormal")
	}

	t.Logf("v41 fixture parsed: %d events, %d distinct XUIDs, kills=%d deaths=%d medals=%d mode=%d",
		len(events), len(xuids),
		typeCount[highlightevent.EventTypeKill], typeCount[highlightevent.EventTypeDeath],
		typeCount[highlightevent.EventTypeMedal], typeCount[highlightevent.EventTypeMode])
}
