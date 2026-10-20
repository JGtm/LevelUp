//go:build integration

// Package sync — convergence_report_integration_test.go : le rapport de
// backlog exposé au dashboard monitoring reflète exactement les sélecteurs de
// convergence (même sémantique que hasConvergenceBacklog, mais en compteurs).
//
// Tag `integration` car le driver DuckDB (CGO) est requis.
package sync

import (
	"context"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// TestConvergenceBacklog_CountsMirrorSelectors : un match sans events, un
// match sans bit weapons, un match shared sans row enrichment et un match
// enrichi sans PSA tenté produisent chacun leur compteur — et un état
// entièrement complet produit un backlog Total() == 0.
func TestConvergenceBacklog_CountsMirrorSelectors(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"
	ctx := context.Background()

	// 1 match events incomplet (events_loaded=false) + 1 match complet niveau
	// shared mais sans row enrichment + 1 match enrichi sans PSA jamais tenté.
	seedConvergenceMatch(t, shared, "m-events", xuid, false, MBitWeaponKills)
	seedConvergenceMatch(t, shared, "m-enrich", xuid, true, MBitWeaponKills)
	seedConvergenceMatch(t, shared, "m-psa", xuid, true, MBitWeaponKills)
	for _, id := range []string{"m-events"} {
		if _, err := player.Exec(
			`INSERT INTO player_match_enrichment (match_id, psa_checked_at) VALUES (?, now())`, id); err != nil {
			t.Fatalf("seed enrichment %s: %v", id, err)
		}
	}
	if _, err := player.Exec(
		`INSERT INTO player_match_enrichment (match_id) VALUES ('m-psa')`); err != nil {
		t.Fatalf("seed enrichment m-psa: %v", err)
	}

	got := ConvergenceBacklog(ctx, player, shared, xuid)
	if got.MissingEvents != 1 {
		t.Errorf("MissingEvents = %d (attendu 1)", got.MissingEvents)
	}
	if got.MissingEnrichment != 1 {
		t.Errorf("MissingEnrichment = %d (attendu 1)", got.MissingEnrichment)
	}
	if got.MissingPSA != 1 {
		t.Errorf("MissingPSA = %d (attendu 1)", got.MissingPSA)
	}
	if got.Total() != 3 {
		t.Errorf("Total() = %d (attendu 3)", got.Total())
	}

	// Cohérence avec le déclencheur du post-sync : backlog non vide.
	if !hasConvergenceBacklog(ctx, player, shared, xuid) {
		t.Fatal("hasConvergenceBacklog doit être true quand le rapport compte du backlog")
	}
}

// TestConvergenceBacklog_ZeroWhenComplete : tout complet → tous les compteurs
// à zéro (le dashboard affiche un état vert, pas un faux backlog).
func TestConvergenceBacklog_ZeroWhenComplete(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"

	seedConvergenceMatch(t, shared, "ok-1", xuid, true, MBitWeaponKills)
	if _, err := player.Exec(
		`INSERT INTO player_match_enrichment (match_id, psa_checked_at) VALUES ('ok-1', now())`); err != nil {
		t.Fatalf("seed enrichment: %v", err)
	}

	got := ConvergenceBacklog(context.Background(), player, shared, xuid)
	if got.Total() != 0 {
		t.Fatalf("Total() = %d (attendu 0) — détail %+v", got.Total(), got)
	}
}

// TestConvergenceBacklog_NilSafe : DBs nil → compteurs à zéro, pas de panic
// (même sémantique défensive que countSharedMatchesMissingEnrichment).
func TestConvergenceBacklog_NilSafe(t *testing.T) {
	got := ConvergenceBacklog(context.Background(), nil, nil, "")
	if got.MissingEnrichment != 0 || got.Total() < 0 {
		t.Fatalf("backlog nil-safe attendu, got %+v", got)
	}
}
