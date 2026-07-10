// highlight_events_orchestration_test.go — Tests d'orchestration pour la
// détection d'anomalie de parsing highlight_events.
//
// Avant le fix bit-aligné de mai 2026, l'orchestration loggeait silencieusement
// en DEBUG quand le parser retournait 0 events sur un chunk non-vide. Ces tests
// gardent que ce cas remonte désormais en WARN + counter expvar + warning dans
// le SyncResult.
package sync

import (
	"bytes"
	"compress/zlib"
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// makeBenignZlibChunk produit un chunk zlib non-vide qui décompresse en
// un buffer de zéros — donc sans aucun marqueur XUID, donc 0 events parsés.
func makeBenignZlibChunk(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(make([]byte, 200)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessHighlightEvents_ZeroEventsFromNonEmptyChunk_FlagsAnomaly(t *testing.T) {
	observability.Reset()
	chunk := makeBenignZlibChunk(t)

	mock := &mockHaloClient{
		highlightChunkData:    chunk,
		highlightChunkVersion: 41,
		highlightChunkFound:   true,
	}
	result := &domain.SyncResult{}

	// processHighlightEvents marque events_loaded en DB en cas de succès — on
	// passe nil pour sharedDB et on s'attend à ce que la fonction return AVANT
	// d'y toucher (puisque events==0).
	err := ProcessHighlightEvents(context.Background(), mock, nil, "match-test-id", result)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}
	if !strings.Contains(result.Warnings[0], "parse_anomaly") {
		t.Errorf("warning should mention parse_anomaly, got %q", result.Warnings[0])
	}
	if got := observability.LoadCounter("highlight_events_parse_anomaly_total"); got != 1 {
		t.Errorf("counter should be 1, got %d", got)
	}
}
