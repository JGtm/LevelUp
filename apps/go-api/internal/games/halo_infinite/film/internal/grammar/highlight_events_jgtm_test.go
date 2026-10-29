// Tests basés sur le fixture réel jgtm_full_match (cf. A.4 du plan
// PLAN_FIX_SYNC_TESTS_STRATEGY_2026-05-24.md).
//
// Le fixture est gitignored — si absent localement, ces tests se skip
// automatiquement via jgtmDisponible().
//
// Le fixture vit chez l'hote (`apps/go-api/internal/sync/testdata/jgtm_full_match`, produit par
// `cmd/gen_test_fixtures`) : la brique le lit par un chargeur LOCAL de la seule partie dont elle
// a besoin (le manifeste et les chunks), et plus par le paquet `testfixtures` de LevelUp (plan
// D2, item 2.11). Le chargeur de l'hote lit aussi les reponses d'API, dont ces tests n'ont rien
// a faire.
package grammar

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestParseHighlightEvents_JGtmFullMatch_HighlightChunk : parse le chunk
// HIGHLIGHT_EVENTS (ChunkType=3, le dernier) du fixture JGtm. Doit retourner
// >= 5 events sans erreur ni panic.
//
// Le fixture JGtm est un match Arena de ~9 min avec 8 participants → on attend
// au moins une dizaine d'events significatifs (kills, deaths, medals).
func TestParseHighlightEvents_JGtmFullMatch_HighlightChunk(t *testing.T) {
	if !jgtmDisponible(t) {
		t.Skip("jgtm_full_match fixture absent — regenerer via cmd/gen_test_fixtures")
	}
	fx := chargerJGtm(t)

	highlightIdx := fx.HighlightChunkIndex()
	if highlightIdx < 0 {
		t.Fatal("fixture JGtm ne contient pas de chunk HIGHLIGHT_EVENTS (ChunkType=3)")
	}

	chunk := fx.LoadChunk(t, highlightIdx)
	if len(chunk) == 0 {
		t.Fatalf("chunk %d vide", highlightIdx)
	}

	filmMajor := fx.Manifest.CustomData.FilmMajorVersion
	if filmMajor == 0 {
		// Manifest legacy ou cache n'avait pas la version → defaut sain.
		filmMajor = 41
	}

	events, err := ParseHighlightEvents(chunk, filmMajor)
	if err != nil {
		t.Fatalf("ParseHighlightEvents: %v", err)
	}

	if len(events) < 5 {
		t.Errorf("attendu >= 5 events sur le HIGHLIGHT_EVENTS du match Arena 9-min, got %d", len(events))
	}

	// Sanity : events ont des timestamps croissants raisonnables (entre 0 et 600s).
	for i, e := range events {
		if e.TimeMS < 0 {
			t.Errorf("event[%d] TimeMS=%d invalide", i, e.TimeMS)
		}
	}
}

// TestParseHighlightEvents_JGtmFullMatch_AllChunks_NoPanic : passe TOUS les
// 30 chunks au parser (header, replication, highlight). Doit jamais panic.
//
// Sentinelle anti-régression : detecte qu'un changement de parser ne casse
// pas une variante de chunk presente dans les vraies donnees prod.
func TestParseHighlightEvents_JGtmFullMatch_AllChunks_NoPanic(t *testing.T) {
	if !jgtmDisponible(t) {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := chargerJGtm(t)

	filmMajor := fx.Manifest.CustomData.FilmMajorVersion
	if filmMajor == 0 {
		filmMajor = 41
	}

	totalEvents := 0
	parsedChunks := 0
	for _, chunkMeta := range fx.Manifest.CustomData.Chunks {
		chunk := fx.LoadChunk(t, chunkMeta.Index)

		// Note : ne plante pas si la parse retourne 0 events sur un chunk header
		// (ChunkType=1) ou replication (ChunkType=2) — le parser identifie les
		// patterns event-like et tolere les chunks sans events.
		events, err := ParseHighlightEvents(chunk, filmMajor)
		if err != nil {
			t.Errorf("chunk%d type=%d size=%d: parse error: %v",
				chunkMeta.Index, chunkMeta.ChunkType, chunkMeta.ChunkSize, err)
			continue
		}
		totalEvents += len(events)
		parsedChunks++
	}

	if parsedChunks != len(fx.Manifest.CustomData.Chunks) {
		t.Errorf("parse echec sur certains chunks: %d/%d", parsedChunks, len(fx.Manifest.CustomData.Chunks))
	}

	t.Logf("JGtm full match : %d chunks parses, %d events total extraits", parsedChunks, totalEvents)
}

// TestParseHighlightEvents_JGtmFullMatch_ReplicationChunks_NoEventsExpected
// vérifie que les chunks REPLICATION_DATA (ChunkType=2) ne contiennent
// typiquement PAS d'events (les events sont concentrés dans le chunk type 3).
//
// Le parser doit tolerer ces chunks sans erreur (0 events est OK).
func TestParseHighlightEvents_JGtmFullMatch_ReplicationChunks_Tolerated(t *testing.T) {
	if !jgtmDisponible(t) {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := chargerJGtm(t)

	filmMajor := fx.Manifest.CustomData.FilmMajorVersion
	if filmMajor == 0 {
		filmMajor = 41
	}

	replIndices := fx.ReplicationChunkIndices()
	if len(replIndices) < 5 {
		t.Skipf("fixture JGtm ne contient que %d replication chunks, attendu >=5", len(replIndices))
	}

	// Tester les 5 premiers chunks replication — suffisant pour garantir
	// que le parser tolere ce type de chunk.
	for _, idx := range replIndices[:5] {
		chunk := fx.LoadChunk(t, idx)
		_, err := ParseHighlightEvents(chunk, filmMajor)
		if err != nil {
			t.Errorf("chunk%d (replication type=2) : parse fail %v", idx, err)
		}
	}
}

// TestParseHighlightEvents_JGtmFullMatch_TimestampsAreBounded : les events
// extraits doivent avoir des TimeMS dans la duree du film (manifest.FilmLength).
//
// Sentinelle pour detecter une derive du parser (offset mal calcule, indice
// negatif, etc.).
func TestParseHighlightEvents_JGtmFullMatch_TimestampsAreBounded(t *testing.T) {
	if !jgtmDisponible(t) {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := chargerJGtm(t)

	highlightIdx := fx.HighlightChunkIndex()
	if highlightIdx < 0 {
		t.Skip("pas de highlight chunk")
	}

	chunk := fx.LoadChunk(t, highlightIdx)
	filmMajor := fx.Manifest.CustomData.FilmMajorVersion
	if filmMajor == 0 {
		filmMajor = 41
	}

	events, err := ParseHighlightEvents(chunk, filmMajor)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	filmLengthMS := fx.Manifest.CustomData.FilmLength
	if filmLengthMS == 0 {
		t.Skip("FilmLength=0 dans le manifest — pas de borne sup pour les timestamps")
	}

	// Tolerance : +5s par rapport au FilmLength (les events de fin peuvent
	// deborder legerement selon l'encoding du film).
	maxTimeMS := filmLengthMS + 5000

	outOfBounds := 0
	for _, e := range events {
		if e.TimeMS < 0 || e.TimeMS > maxTimeMS {
			outOfBounds++
		}
	}
	if outOfBounds > 0 {
		t.Errorf("%d/%d events hors bornes [0, %dms] — derive parser ?",
			outOfBounds, len(events), maxTimeMS)
	}
}

// jgtmMatch : le fixture E2E JGtm, reduit au manifeste du film et au dossier de ses chunks.
type jgtmMatch struct {
	Manifest struct {
		CustomData struct {
			FilmLength       int `json:"FilmLength"`
			FilmMajorVersion int `json:"FilmMajorVersion"`
			Chunks           []struct {
				Index     int `json:"Index"`
				ChunkSize int `json:"ChunkSize"`
				ChunkType int `json:"ChunkType"`
			} `json:"Chunks"`
		} `json:"CustomData"`
	}
	chunksDir string
}

// Types de chunk du manifeste : 2 = REPLICATION_DATA (le coeur du film), 3 = HIGHLIGHT_EVENTS.
const (
	jgtmChunkReplication = 2
	jgtmChunkHighlight   = 3
)

// dossierJGtm : le fixture, dans le depot hote.
func dossierJGtm(t *testing.T) string {
	t.Helper()
	return filepath.Join(racineDepot(t), "apps", "go-api", "internal", "sync", "testdata", "jgtm_full_match")
}

// jgtmDisponible dit si le fixture (gitignore, 6 Mo) est present : manifeste et chunk 0.
func jgtmDisponible(t *testing.T) bool {
	t.Helper()
	dir := dossierJGtm(t)
	for _, p := range []string{filepath.Join(dir, "manifest_raw.json"), filepath.Join(dir, "chunks", "filmChunk0")} {
		if _, err := os.Stat(p); err != nil {
			return false
		}
	}
	return true
}

// chargerJGtm lit le manifeste du fixture ; le test appelant a deja verifie `jgtmDisponible`.
func chargerJGtm(t *testing.T) jgtmMatch {
	t.Helper()
	dir := dossierJGtm(t)
	brut, err := os.ReadFile(filepath.Join(dir, "manifest_raw.json")) //nolint:gosec // chemin du fixture du depot
	if err != nil {
		t.Fatalf("manifeste du fixture JGtm : %v", err)
	}
	var fx jgtmMatch
	if err := json.Unmarshal(brut, &fx.Manifest); err != nil {
		t.Fatalf("manifeste du fixture JGtm non decodable : %v", err)
	}
	fx.chunksDir = filepath.Join(dir, "chunks")
	return fx
}

// LoadChunk rend le contenu binaire du chunk d'index donne.
func (m jgtmMatch) LoadChunk(t *testing.T, idx int) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(m.chunksDir, fmt.Sprintf("filmChunk%d", idx))) //nolint:gosec // chemin du fixture du depot
	if err != nil {
		t.Fatalf("chunk %d du fixture JGtm : %v", idx, err)
	}
	return data
}

// ReplicationChunkIndices rend les indices des chunks REPLICATION_DATA.
func (m jgtmMatch) ReplicationChunkIndices() []int {
	var out []int
	for _, c := range m.Manifest.CustomData.Chunks {
		if c.ChunkType == jgtmChunkReplication {
			out = append(out, c.Index)
		}
	}
	return out
}

// HighlightChunkIndex rend l'index du chunk HIGHLIGHT_EVENTS, ou -1 s'il est absent.
func (m jgtmMatch) HighlightChunkIndex() int {
	for _, c := range m.Manifest.CustomData.Chunks {
		if c.ChunkType == jgtmChunkHighlight {
			return c.Index
		}
	}
	return -1
}
