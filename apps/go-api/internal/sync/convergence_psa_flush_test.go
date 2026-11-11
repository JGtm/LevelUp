//go:build integration

// Package sync — convergence_psa_flush_test.go : le marqueur terminal psa_checked_at n'est
// posé qu'une fois les alias du match écrits dans shared.xuid_aliases. Un flush en échec laisse
// le match sélectionnable : il est repris au cycle suivant, comme le dit le journal du post-sync.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
)

// psaBodyWithParticipant : JSON de match dont l'unique participant porte un alias à écrire.
func psaBodyWithParticipant(matchID, xuid, gamertag string) map[string]any {
	return map[string]any{
		"MatchId": matchID,
		"Players": []any{
			map[string]any{"PlayerId": "xuid(" + xuid + ")", "Gamertag": gamertag},
		},
	}
}

// closedSharedDB rend une base dont toute écriture échoue (connexion fermée).
func closedSharedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close duckdb: %v", err)
	}
	return db
}

func seedPSAMatch(t *testing.T, player *sql.DB, ids ...string) {
	t.Helper()
	for _, mid := range ids {
		if _, err := player.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES (?)`, mid); err != nil {
			t.Fatalf("seed %s: %v", mid, err)
		}
	}
}

// TestConvergePSA_FlushAliasEnEchec : un flush d'alias en échec ne pose pas le marqueur ; le match
// reste dans selectMatchesMissingPSA, l'échec est compté, et le cycle suivant (shared disponible)
// écrit l'alias puis pose le marqueur.
func TestConvergePSA_FlushAliasEnEchec(t *testing.T) {
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	shared := openBatchPathTestDB(t, migration.TargetShared)
	const xuid = "9999"
	ctx := context.Background()
	seedPSAMatch(t, player, "psa-alias")

	client := &mockHaloClient{statsBody: map[string]map[string]any{
		"psa-alias": psaBodyWithParticipant("psa-alias", xuid, "AliasGT"),
	}}

	failedBefore := observability.LoadCounter("convergence_alias_upsert_failed_total")
	deferredBefore := observability.LoadCounter("convergence_psa_deferred_total")
	if done := convergePSA(ctx, player, closedSharedDB(t), client, xuid, []string{"psa-alias"}); done != 0 {
		t.Fatalf("flush KO : done = %d, want 0 (marqueur non posé)", done)
	}
	if got := selectMatchesMissingPSA(ctx, player); len(got) != 1 || got[0] != "psa-alias" {
		t.Fatalf("flush KO : sélection = %v, want [psa-alias] (le match doit être repris au cycle suivant)", got)
	}
	if d := observability.LoadCounter("convergence_alias_upsert_failed_total") - failedBefore; d != 1 {
		t.Fatalf("flush KO : compteur d'échecs d'alias +%d, want +1", d)
	}
	if d := observability.LoadCounter("convergence_psa_deferred_total") - deferredBefore; d != 1 {
		t.Fatalf("flush KO : compteur de matchs différés +%d, want +1", d)
	}

	// Cycle suivant : shared disponible.
	if done := convergePSA(ctx, player, shared, client, xuid, []string{"psa-alias"}); done != 1 {
		t.Fatalf("cycle suivant : done = %d, want 1", done)
	}
	if got := selectMatchesMissingPSA(ctx, player); len(got) != 0 {
		t.Fatalf("cycle suivant : sélection = %v, want vide", got)
	}
	var gt string
	if err := shared.QueryRow(`SELECT gamertag FROM xuid_aliases WHERE xuid = ?`, xuid).Scan(&gt); err != nil {
		t.Fatalf("alias absent après le cycle suivant : %v", err)
	}
	if gt != "AliasGT" {
		t.Fatalf("alias = %q, want AliasGT", gt)
	}
}

// TestConvergePSA_SansAliasAEcrire : un match sans alias à écrire n'a pas besoin de shared ; son
// marqueur est posé même si shared est indisponible (nil ou fermé).
func TestConvergePSA_SansAliasAEcrire(t *testing.T) {
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "9999"
	ctx := context.Background()
	seedPSAMatch(t, player, "psa-none")

	client := &mockHaloClient{statsBody: map[string]map[string]any{
		"psa-none": {"MatchId": "psa-none", "Players": []any{}},
	}}
	if done := convergePSA(ctx, player, nil, client, xuid, []string{"psa-none"}); done != 1 {
		t.Fatalf("sans alias, shared nil : done = %d, want 1", done)
	}
	if got := selectMatchesMissingPSA(ctx, player); len(got) != 0 {
		t.Fatalf("sans alias : sélection = %v, want vide", got)
	}
}

// TestConvergePSA_SharedNilAvecAlias : shared absent alors qu'un alias est à écrire : pas de
// marqueur, le match est repris.
func TestConvergePSA_SharedNilAvecAlias(t *testing.T) {
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	const xuid = "9999"
	ctx := context.Background()
	seedPSAMatch(t, player, "psa-nil")

	client := &mockHaloClient{statsBody: map[string]map[string]any{
		"psa-nil": psaBodyWithParticipant("psa-nil", xuid, "NilGT"),
	}}
	if done := convergePSA(ctx, player, nil, client, xuid, []string{"psa-nil"}); done != 0 {
		t.Fatalf("shared nil avec alias : done = %d, want 0", done)
	}
	if got := selectMatchesMissingPSA(ctx, player); len(got) != 1 {
		t.Fatalf("shared nil avec alias : sélection = %v, want [psa-nil]", got)
	}
}

// TestRunPSAConvergence_BurstIndisponible : la fenêtre d'écriture shared refusée ne pose aucun
// marqueur pour les matchs qui ont des alias à écrire ; un match sans alias est marqué quand même.
// Une fois la fenêtre rendue, le match retenté est marqué.
func TestRunPSAConvergence_BurstIndisponible(t *testing.T) {
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	shared := openBatchPathTestDB(t, migration.TargetShared)
	const xuid = "9999"
	ctx := context.Background()
	seedPSAMatch(t, player, "psa-win", "psa-bare")

	client := &mockHaloClient{statsBody: map[string]map[string]any{
		"psa-win":  psaBodyWithParticipant("psa-win", xuid, "WinGT"),
		"psa-bare": {"MatchId": "psa-bare", "Players": []any{}},
	}}
	e := &SyncEngine{xuid: xuid, gamertag: "gt"}
	var r domain.PostSyncResult

	refus := NewBurstSharedAccess(func(context.Context) (*sql.DB, func(), error) {
		return nil, nil, errors.New("fenêtre refusée")
	}, nil, "test")
	deferredBefore := observability.LoadCounter("convergence_psa_deferred_total")
	e.runPSAConvergence(ctx, player, refus, client, &r)
	if r.ConvergedPSA != 1 {
		t.Fatalf("fenêtre refusée : ConvergedPSA = %d, want 1 (le match sans alias)", r.ConvergedPSA)
	}
	if got := selectMatchesMissingPSA(ctx, player); len(got) != 1 || got[0] != "psa-win" {
		t.Fatalf("fenêtre refusée : sélection = %v, want [psa-win]", got)
	}
	if d := observability.LoadCounter("convergence_psa_deferred_total") - deferredBefore; d != 1 {
		t.Fatalf("fenêtre refusée : compteur de matchs différés +%d, want +1", d)
	}

	e.runPSAConvergence(ctx, player, NewPinnedSharedAccess(shared), client, &r)
	if r.ConvergedPSA != 1 {
		t.Fatalf("fenêtre rendue : ConvergedPSA = %d, want 1", r.ConvergedPSA)
	}
	if got := selectMatchesMissingPSA(ctx, player); len(got) != 0 {
		t.Fatalf("fenêtre rendue : sélection = %v, want vide", got)
	}
}
