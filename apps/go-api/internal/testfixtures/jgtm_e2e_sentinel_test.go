// jgtm_e2e_sentinel_test.go — sentinelle E2E pour le fixture JGtm full match.
//
// Reference G.4 du plan de tests. Ce test verifie la presence et l'integrite
// du fixture local jgtm_full_match (gitignore). Skip auto si absent.
//
// Ce fichier est COMPLEMENTAIRE de G.3 (FakeHaloServer) et G.4 (TestSyncEngine
// E2E run complet). Le test E2E complet requiert :
//  1. FakeHaloServer pour servir les manifests + chunks (G.3 a implementer
//     en PR infrastructure dedie — typiquement co-livre avec Phase 1 du plan
//     principal qui necessite cette infrastructure pour les tests integration).
//  2. SyncEngine constructible sans dependances DuckDB hard (refactor partiel
//     des helpers Connection — egalement PR infrastructure).
//
// Pour l'instant, la sentinelle valide la presence du fixture + son schema
// minimal — sufisant pour que les tests parsing (A.4) + persist (C.x) marchent.
package testfixtures

import (
	"testing"
)

// TestG4_JGtmFullMatchFixtureAvailable verifie que le fixture est complet.
// Skip auto si absent (gitignore + cmd/gen_test_fixtures regenere).
func TestG4_JGtmFullMatchFixtureAvailable(t *testing.T) {
	if !JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent — regenerer via `go run ./cmd/gen_test_fixtures download-full-match` (G.1)")
	}
	fx := LoadJGtmFullMatch(t)

	// Schema minimum attendu : manifest non-vide, chunks accessibles, API responses present.
	if fx.Manifest.BlobStoragePathPrefix == "" {
		t.Error("manifest sans BlobStoragePathPrefix — fixture incomplet")
	}
	if len(fx.Manifest.CustomData.Chunks) < 10 {
		t.Errorf("attendu >= 10 chunks dans le manifest, got %d", len(fx.Manifest.CustomData.Chunks))
	}
	if fx.ChunksDir == "" {
		t.Error("ChunksDir vide")
	}
	if len(fx.MatchStatsRaw) == 0 {
		t.Log("api_match_stats.json absent — fixture partiel (acceptable, mais E2E sera limite)")
	}
	if len(fx.SkillRaw) == 0 {
		t.Log("api_skill.json absent — fixture partiel")
	}

	// Sentinelle : chunk types attendus (1=header, 2=replication, 3=highlight).
	hasHeader := false
	hasReplication := false
	hasHighlight := false
	for _, c := range fx.Manifest.CustomData.Chunks {
		switch c.ChunkType {
		case 1:
			hasHeader = true
		case 2:
			hasReplication = true
		case 3:
			hasHighlight = true
		}
	}
	if !hasHeader {
		t.Error("manifest sans chunk_type=1 (header)")
	}
	if !hasReplication {
		t.Error("manifest sans chunk_type=2 (replication)")
	}
	if !hasHighlight {
		t.Error("manifest sans chunk_type=3 (highlight events)")
	}
}

// TestG4_JGtmFullMatchChunksAccessible verifie que le chunk 0 (header) est
// lisible. Sentinelle anti-regression : le fixture doit etre integre.
func TestG4_JGtmFullMatchChunksAccessible(t *testing.T) {
	if !JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent")
	}
	fx := LoadJGtmFullMatch(t)
	chunk0 := fx.LoadChunk(t, 0)
	if len(chunk0) == 0 {
		t.Error("chunk0 vide — header film corrompu")
	}
}
