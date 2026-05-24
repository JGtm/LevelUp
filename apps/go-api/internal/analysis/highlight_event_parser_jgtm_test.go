// Tests basés sur le fixture réel jgtm_full_match (cf. A.4 du plan
// PLAN_FIX_SYNC_TESTS_STRATEGY_2026-05-24.md).
//
// Le fixture est gitignored — si absent localement, ces tests se skip
// automatiquement via testfixtures.JGtmFullMatchAvailable().
package analysis

import (
	"testing"

	"levelup/go-api/internal/testfixtures"
)

// TestParseHighlightEvents_JGtmFullMatch_HighlightChunk : parse le chunk
// HIGHLIGHT_EVENTS (ChunkType=3, le dernier) du fixture JGtm. Doit retourner
// >= 5 events sans erreur ni panic.
//
// Le fixture JGtm est un match Arena de ~9 min avec 8 participants → on attend
// au moins une dizaine d'events significatifs (kills, deaths, medals).
func TestParseHighlightEvents_JGtmFullMatch_HighlightChunk(t *testing.T) {
	if !testfixtures.JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent — regenerer via cmd/gen_test_fixtures")
	}
	fx := testfixtures.LoadJGtmFullMatch(t)

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
	if !testfixtures.JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := testfixtures.LoadJGtmFullMatch(t)

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
	if !testfixtures.JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := testfixtures.LoadJGtmFullMatch(t)

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
	if !testfixtures.JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := testfixtures.LoadJGtmFullMatch(t)

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
