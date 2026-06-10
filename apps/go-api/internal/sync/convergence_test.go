//go:build integration

// Package sync — convergence_test.go : garde-fou de la propriété centrale de la
// convergence — la sélection est pilotée par le LEDGER (events_loaded / bits
// weapon), donc un match COMPLET n'est JAMAIS resélectionné. C'est ce qui
// distingue la convergence d'un heal à fenêtre floue (qui re-traitait du
// complet) : le work-set rétrécit prouvablement vers zéro.
//
// Tag `integration` car le driver DuckDB (CGO) est requis.

package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func seedConvergenceMatch(t *testing.T, shared *sql.DB, id, xuid string, eventsLoaded bool, backfillCompleted int) {
	t.Helper()
	if _, err := shared.Exec(
		`INSERT INTO match_registry (match_id, events_loaded, backfill_completed, start_time)
		 VALUES (?, ?, ?, now())`, id, eventsLoaded, backfillCompleted); err != nil {
		t.Fatalf("seed registry %s: %v", id, err)
	}
	if _, err := shared.Exec(
		`INSERT INTO match_participants (match_id, xuid) VALUES (?, ?)`, id, xuid); err != nil {
		t.Fatalf("seed participant %s: %v", id, err)
	}
}

// TestConvergence_SelectsOnlyIncompleteEvents : seul le match events_loaded=false
// est sélectionné ; le complet est exclu (idempotence / convergence vers zéro).
func TestConvergence_SelectsOnlyIncompleteEvents(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"

	seedConvergenceMatch(t, shared, "ev-incomplete", xuid, false, 0)
	seedConvergenceMatch(t, shared, "ev-complete", xuid, true, 0)

	got := selectMatchesMissingEvents(context.Background(), player, shared, xuid)
	if len(got) != 1 || got[0] != "ev-incomplete" {
		t.Fatalf("convergence events doit sélectionner UNIQUEMENT l'incomplet, got %v", got)
	}
}

// TestConvergence_SelectsOnlyIncompleteWeapons : un match dont le bit
// MBitWeaponKills est posé n'est pas resélectionné ; celui sans bit l'est.
func TestConvergence_SelectsOnlyIncompleteWeapons(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"

	// bit weapon posé → complet ; bit absent → incomplet.
	seedConvergenceMatch(t, shared, "wk-complete", xuid, true, MBitWeaponKills)
	seedConvergenceMatch(t, shared, "wk-incomplete", xuid, true, 0)

	got := selectMatchesMissingWeapons(context.Background(), player, shared, xuid)
	if len(got) != 1 || got[0] != "wk-incomplete" {
		t.Fatalf("convergence weapons doit sélectionner UNIQUEMENT l'incomplet, got %v", got)
	}
}

// TestConvergence_NothingWhenAllComplete : work-set vide quand tout est complet
// (la "roue de secours" reste dans le coffre — aucun retraitement récurrent).
// « Complet » inclut depuis 2026-06-10 la présence des rows enrichment du
// joueur (countSharedMatchesMissingEnrichment) — un match shared sans row
// player_match_enrichment EST un backlog (contrat delta-skip).
func TestConvergence_NothingWhenAllComplete(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"

	seedConvergenceMatch(t, shared, "done-1", xuid, true, MBitWeaponKills)
	seedConvergenceMatch(t, shared, "done-2", xuid, true, MBitWeaponKills)
	for _, id := range []string{"done-1", "done-2"} {
		// psa_checked_at non-NULL : la convergence PSA est terminale pour ces matchs.
		if _, err := player.Exec(
			`INSERT INTO player_match_enrichment (match_id, psa_checked_at) VALUES (?, now())`, id); err != nil {
			t.Fatalf("seed enrichment %s: %v", id, err)
		}
	}

	if backlog := hasConvergenceBacklog(context.Background(), player, shared, xuid); backlog {
		t.Fatal("hasConvergenceBacklog doit être false quand tout est complet")
	}
}

// TestConvergence_BacklogWhenPSANeverChecked : un match enrichi sans PSA et
// jamais tenté (psa_checked_at NULL) déclenche le backlog ; une fois stampé,
// il devient terminal même sans aucune row PSA.
func TestConvergence_BacklogWhenPSANeverChecked(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"

	seedConvergenceMatch(t, shared, "psa-1", xuid, true, MBitWeaponKills)
	if _, err := player.Exec(
		`INSERT INTO player_match_enrichment (match_id) VALUES ('psa-1')`); err != nil {
		t.Fatalf("seed enrichment: %v", err)
	}

	if got := selectMatchesMissingPSA(context.Background(), player); len(got) != 1 || got[0] != "psa-1" {
		t.Fatalf("selectMatchesMissingPSA = %v (attendu [psa-1])", got)
	}

	// Stamp terminal → plus jamais sélectionné, même sans row PSA.
	if _, err := player.Exec(
		`UPDATE player_match_enrichment SET psa_checked_at = now() WHERE match_id = 'psa-1'`); err != nil {
		t.Fatalf("stamp: %v", err)
	}
	if got := selectMatchesMissingPSA(context.Background(), player); len(got) != 0 {
		t.Fatalf("selectMatchesMissingPSA post-stamp = %v (attendu vide — état terminal)", got)
	}
}

// TestConvergence_BacklogWhenEnrichmentMissing : un match présent en shared
// pour ce xuid mais SANS row player_match_enrichment déclenche le backlog —
// même si events et weapons sont complets (cycle « pur skip », gate 2026-06-10).
func TestConvergence_BacklogWhenEnrichmentMissing(t *testing.T) {
	shared := openBatchPathTestDB(t, migration.TargetShared)
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "x1"

	seedConvergenceMatch(t, shared, "skipped-1", xuid, true, MBitWeaponKills)

	if missing := countSharedMatchesMissingEnrichment(context.Background(), player, shared, xuid); missing != 1 {
		t.Fatalf("countSharedMatchesMissingEnrichment = %d (attendu 1)", missing)
	}
	if !hasConvergenceBacklog(context.Background(), player, shared, xuid) {
		t.Fatal("hasConvergenceBacklog doit être true quand l'enrichment manque")
	}
}
