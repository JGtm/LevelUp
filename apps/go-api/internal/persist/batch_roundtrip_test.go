// Tests round-trip JSON pour MatchBatch (C.1 du plan de tests).
//
// Couvre :
//  1. Tous les WAL reels presents dans data/wal/ (si non vide) — round-trip
//     stable + structure minimum validee.
//  2. Batches synthetiques avec edge cases NaN/Inf — sentinelle anti-regression
//     pour le bug `json: unsupported value: NaN`.
package persist

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/testfixtures"
)

// TestMatchBatch_RoundTrip_AllWAL : pour chaque WAL present dans data/wal/,
// verifie qu'il s'unmarshal vers MatchBatch et que le re-marshal est stable
// (champ-par-champ via deep-equal sur les fields critiques).
//
// Skip si data/wal/ est vide (cas usuel sur dev fresh — les WAL sont consumes
// par le worker async au fil du sync).
func TestMatchBatch_RoundTrip_AllWAL(t *testing.T) {
	entries := testfixtures.LoadAllWAL(t)
	t.Logf("loaded %d WAL entries from %s", len(entries), testfixtures.WALDir())

	for _, entry := range entries {
		t.Run(entry.BatchID, func(t *testing.T) {
			testfixtures.AssertWALStructure(t, entry)

			var batch MatchBatch
			if err := json.Unmarshal(entry.RawJSON, &batch); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			// Verifs identite : champs critiques preserves.
			if batch.BatchID != entry.BatchID {
				t.Errorf("batch_id mismatch: got %q, want %q", batch.BatchID, entry.BatchID)
			}
			if batch.TitleSlug == "" {
				t.Errorf("title_slug vide")
			}
			if batch.Player == "" {
				t.Errorf("player vide")
			}

			// Re-marshal : doit jamais echouer (sentinelle bug NaN).
			out, err := json.Marshal(&batch)
			if err != nil {
				t.Fatalf("re-marshal echec — possiblement NaN dans le batch: %v", err)
			}
			if len(out) < 100 {
				t.Errorf("re-marshal produit %d bytes — probablement vide", len(out))
			}

			// Round-trip : 2e unmarshal preserve le BatchID.
			var batch2 MatchBatch
			if err := json.Unmarshal(out, &batch2); err != nil {
				t.Fatalf("2nd unmarshal: %v", err)
			}
			if batch2.BatchID != batch.BatchID {
				t.Errorf("round-trip casse batch_id")
			}
		})
	}
}

// TestMatchBatch_RoundTrip_JGtmPlayer : si data/wal/ contient des batches JGtm,
// les verifie en sous-test dedie (plus de visibilite que le test global).
func TestMatchBatch_RoundTrip_JGtmPlayer(t *testing.T) {
	entries := testfixtures.LoadWALByPlayer(t, "JGtm")
	t.Logf("found %d JGtm WAL entries", len(entries))

	for _, entry := range entries {
		t.Run(entry.BatchID, func(t *testing.T) {
			var batch MatchBatch
			if err := json.Unmarshal(entry.RawJSON, &batch); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if batch.Player != "JGtm" {
				t.Errorf("player = %q, want JGtm", batch.Player)
			}
			if batch.XUID == "" {
				t.Errorf("xuid vide pour JGtm")
			}
			// re-marshal sentinelle NaN
			if _, err := json.Marshal(&batch); err != nil {
				t.Fatalf("re-marshal: %v", err)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests synthetiques — toujours actifs, ne dependent pas de WAL.
// ─────────────────────────────────────────────────────────────────────────────

// makeMinimalBatch fabrique un MatchBatch valide minimal.
func makeMinimalBatch() *MatchBatch {
	return &MatchBatch{
		BatchID:   "test-batch-001",
		TitleSlug: "halo_infinite",
		Player:    "TestPlayer",
		XUID:      "2533274823110022",
		CreatedAt: time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC),
		Source:    "test_synthetic",
		Shared: SharedBatch{
			Match: &domain.MatchRegistryRow{
				MatchID:   "11111111-2222-3333-4444-555555555555",
				StartTime: time.Date(2026, 5, 24, 11, 0, 0, 0, time.UTC),
			},
		},
	}
}

// TestMatchBatch_RoundTrip_Minimal : batch minimum valide doit round-tripper.
func TestMatchBatch_RoundTrip_Minimal(t *testing.T) {
	batch := makeMinimalBatch()

	raw, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var back MatchBatch
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if back.BatchID != batch.BatchID {
		t.Errorf("BatchID round-trip casse: got %q want %q", back.BatchID, batch.BatchID)
	}
	if back.Player != batch.Player {
		t.Errorf("Player round-trip casse: got %q want %q", back.Player, batch.Player)
	}
	if back.Shared.Match == nil {
		t.Fatal("Shared.Match nil apres round-trip")
	}
	if back.Shared.Match.MatchID != batch.Shared.Match.MatchID {
		t.Errorf("Shared.Match.MatchID round-trip casse")
	}
}

// TestMatchBatch_Marshal_FailsOnNaN reproduit le bug observe en prod :
// un champ float64 = NaN dans le batch fait echouer json.Marshal avec
// "json: unsupported value: NaN".
//
// Ce test documente le bug (sentinelle anti-regression — il DOIT echouer
// AVANT la mise en place de SanitizeFloat dans le BatchBuilder).
func TestMatchBatch_Marshal_FailsOnNaN(t *testing.T) {
	batch := makeMinimalBatch()

	// Inject NaN dans Enrichment.PerformanceScore
	nan := math.NaN()
	batch.PlayerData.Enrichment = &EnrichmentRow{
		MatchID:          batch.Shared.Match.MatchID,
		PerformanceScore: &nan,
	}

	_, err := json.Marshal(batch)
	if err == nil {
		t.Skip("Go a corrige json.Marshal(NaN) — sentinelle obsolete (revise SanitizeFloat ?)")
	}
	t.Logf("attendu : marshal echec sur NaN — %v", err)
}

// TestMatchBatch_Marshal_SafeAfterSanitize : apres SanitizeNullableFloat,
// le batch contenant un NaN devient marshal-able. Le champ devient nil
// (omit ou null selon json tag).
func TestMatchBatch_Marshal_SafeAfterSanitize(t *testing.T) {
	batch := makeMinimalBatch()
	nan := math.NaN()
	batch.PlayerData.Enrichment = &EnrichmentRow{
		MatchID:          batch.Shared.Match.MatchID,
		PerformanceScore: analysis.SanitizeNullableFloat(&nan),
	}

	raw, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("apres sanitize, marshal doit reussir: %v", err)
	}

	// Verif : PerformanceScore omitempty → absent du JSON.
	var back MatchBatch
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if back.PlayerData.Enrichment == nil {
		t.Fatal("Enrichment nil apres round-trip — perdu")
	}
	if back.PlayerData.Enrichment.PerformanceScore != nil {
		t.Errorf("PerformanceScore = %v, want nil (sanitize doit produire nil pour NaN)",
			*back.PlayerData.Enrichment.PerformanceScore)
	}
}

// TestMatchBatch_Marshal_NormalFloatsPreserved : les floats normaux sont
// preserves sans modification.
func TestMatchBatch_Marshal_NormalFloatsPreserved(t *testing.T) {
	batch := makeMinimalBatch()
	v := 72.5
	batch.PlayerData.Enrichment = &EnrichmentRow{
		MatchID:          batch.Shared.Match.MatchID,
		PerformanceScore: &v,
	}

	raw, err := json.Marshal(batch)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back MatchBatch
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.PlayerData.Enrichment == nil || back.PlayerData.Enrichment.PerformanceScore == nil {
		t.Fatal("PerformanceScore perdu round-trip")
	}
	if *back.PlayerData.Enrichment.PerformanceScore != 72.5 {
		t.Errorf("PerformanceScore = %v, want 72.5", *back.PlayerData.Enrichment.PerformanceScore)
	}
}

// TestMatchBatch_Targets_AllConstants : sanity check sur les constantes
// DBTarget (pas de regression silencieuse si quelqu'un renomme).
func TestMatchBatch_Targets_AllConstants(t *testing.T) {
	if TargetShared != "shared" {
		t.Errorf("TargetShared = %q, want shared", TargetShared)
	}
	if TargetPlayer != "player" {
		t.Errorf("TargetPlayer = %q, want player", TargetPlayer)
	}
	if TargetPVE != "pve" {
		t.Errorf("TargetPVE = %q, want pve", TargetPVE)
	}
	if TargetMetadata != "metadata" {
		t.Errorf("TargetMetadata = %q, want metadata", TargetMetadata)
	}
}
