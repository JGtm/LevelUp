//go:build integration

// Package sync — convergence_psa_flush_test.go : awards et marqueur terminal psa_checked_at ne sont
// écrits qu'une fois les alias du match écrits dans shared.xuid_aliases. Un flush en échec ne laisse
// AUCUN award : le match reste sélectionné et le cycle suivant retente tout, sans empiler une
// génération d'awards de plus. Les tests passent par le chemin de production (runPSAConvergence).
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

const psaTestXUID = "9999"

// psaBody : JSON de match dont l'unique participant (le joueur) porte un alias à écrire et, si
// withAwards, un PersonalScores extractible.
func psaBody(matchID, gamertag string, withAwards bool) map[string]any {
	player := map[string]any{"PlayerId": "xuid(" + psaTestXUID + ")", "Gamertag": gamertag}
	if withAwards {
		player["PlayerTeamStats"] = []any{
			map[string]any{"Stats": map[string]any{"CoreStats": map[string]any{
				"PersonalScores": []any{
					map[string]any{"NameId": float64(1024030246), "Count": float64(3),
						"TotalPersonalScoreAwarded": float64(300)},
				},
			}}},
		}
	}
	return map[string]any{"MatchId": matchID, "Players": []any{player}}
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

func psaCount(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s : %v", query, err)
	}
	return n
}

// TestRunPSAConvergence_FlushAliasEnEchecAvecAwards : un match qui a des awards ET dont l'alias ne
// peut pas être écrit ne reçoit ni award ni marqueur, reste sélectionné, est compté ; le cycle
// suivant (shared disponible) écrit l'alias, UNE génération d'awards et le marqueur.
func TestRunPSAConvergence_FlushAliasEnEchecAvecAwards(t *testing.T) {
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	shared := openBatchPathTestDB(t, migration.TargetShared)
	ctx := context.Background()
	seedPSAMatch(t, player, "psa-full")

	client := &mockHaloClient{statsBody: map[string]map[string]any{
		"psa-full": psaBody("psa-full", "AliasGT", true),
	}}
	e := &SyncEngine{xuid: psaTestXUID, gamertag: "gt"}
	var r domain.PostSyncResult

	failedBefore := observability.LoadCounter("convergence_alias_upsert_failed_total")
	deferredBefore := observability.LoadCounter("convergence_psa_deferred_total")
	e.runPSAConvergence(ctx, player, NewPinnedSharedAccess(closedSharedDB(t)), client, &r)
	if r.ConvergedPSA != 0 {
		t.Fatalf("flush KO : ConvergedPSA = %d, want 0", r.ConvergedPSA)
	}
	if got := selectMatchesMissingPSA(ctx, player); len(got) != 1 || got[0] != "psa-full" {
		t.Fatalf("flush KO : sélection au cycle suivant = %v, want [psa-full] (alias jamais retentés sinon)", got)
	}
	if n := psaCount(t, player, `SELECT COUNT(*) FROM personal_score_awards WHERE match_id = 'psa-full'`); n != 0 {
		t.Fatalf("flush KO : %d row(s) d'awards écrite(s), want 0", n)
	}
	if d := observability.LoadCounter("convergence_alias_upsert_failed_total") - failedBefore; d != 1 {
		t.Fatalf("flush KO : compteur d'échecs d'alias +%d, want +1", d)
	}
	if d := observability.LoadCounter("convergence_psa_deferred_total") - deferredBefore; d != 1 {
		t.Fatalf("flush KO : compteur de matchs différés +%d, want +1", d)
	}

	e.runPSAConvergence(ctx, player, NewPinnedSharedAccess(shared), client, &r)
	if r.ConvergedPSA != 1 {
		t.Fatalf("cycle suivant : ConvergedPSA = %d, want 1", r.ConvergedPSA)
	}
	if got := selectMatchesMissingPSA(ctx, player); len(got) != 0 {
		t.Fatalf("cycle suivant : sélection = %v, want vide", got)
	}
	if n := psaCount(t, player, `SELECT COUNT(*) FROM player_match_enrichment_latest
		WHERE match_id = 'psa-full' AND psa_checked_at IS NOT NULL`); n != 1 {
		t.Fatalf("cycle suivant : marqueur posé sur %d match(s), want 1", n)
	}
	if n := psaCount(t, player, `SELECT COUNT(*) FROM personal_score_awards WHERE match_id = 'psa-full'`); n != 1 {
		t.Fatalf("cycle suivant : %d row(s) d'awards (toutes générations), want 1 (jamais de réinsertion)", n)
	}
	var gt string
	if err := shared.QueryRow(`SELECT gamertag FROM xuid_aliases WHERE xuid = ?`, psaTestXUID).Scan(&gt); err != nil || gt != "AliasGT" {
		t.Fatalf("alias après le cycle suivant = %q (%v), want AliasGT", gt, err)
	}
}

// TestRunPSAConvergence_BurstIndisponible : la fenêtre d'écriture shared refusée ne pose ni award
// ni marqueur pour les matchs qui ont des alias à écrire ; un match sans alias est marqué quand même.
// Une fois la fenêtre rendue, le match retenté est marqué.
func TestRunPSAConvergence_BurstIndisponible(t *testing.T) {
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	shared := openBatchPathTestDB(t, migration.TargetShared)
	ctx := context.Background()
	seedPSAMatch(t, player, "psa-win", "psa-bare")

	client := &mockHaloClient{statsBody: map[string]map[string]any{
		"psa-win":  psaBody("psa-win", "WinGT", true),
		"psa-bare": {"MatchId": "psa-bare", "Players": []any{}},
	}}
	e := &SyncEngine{xuid: psaTestXUID, gamertag: "gt"}
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
	if n := psaCount(t, player, `SELECT COUNT(*) FROM personal_score_awards WHERE match_id = 'psa-win'`); n != 0 {
		t.Fatalf("fenêtre refusée : %d row(s) d'awards écrite(s), want 0", n)
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

// TestRunPSAConvergence_MatchDuCheminPerMatchNonRepris : un match qui a déjà des awards sans marqueur
// (écrits par le chemin per-match, qui ne pose jamais le marqueur) n'est ni sélectionné ni refetché.
func TestRunPSAConvergence_MatchDuCheminPerMatchNonRepris(t *testing.T) {
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	shared := openBatchPathTestDB(t, migration.TargetShared)
	ctx := context.Background()
	seedPSAMatch(t, player, "psa-live")
	if err := InsertPersonalScoreAwards(ctx, player, "psa-live", psaTestXUID,
		ExtractPersonalScoreAwards(psaBody("psa-live", "LiveGT", true), "psa-live", psaTestXUID)); err != nil {
		t.Fatalf("seed awards: %v", err)
	}

	client := &mockHaloClient{statsBody: map[string]map[string]any{}}
	e := &SyncEngine{xuid: psaTestXUID, gamertag: "gt"}
	var r domain.PostSyncResult
	e.runPSAConvergence(ctx, player, NewPinnedSharedAccess(shared), client, &r)
	if r.ConvergedPSA != 0 {
		t.Fatalf("ConvergedPSA = %d, want 0", r.ConvergedPSA)
	}
	if n := psaCount(t, player, `SELECT COUNT(*) FROM personal_score_awards WHERE match_id = 'psa-live'`); n != 1 {
		t.Fatalf("%d row(s) d'awards, want 1 (aucune réinsertion)", n)
	}
}

// TestConvergePSA_SharedNilAvecAlias : enrobage de test. shared absent alors qu'un alias est à
// écrire est traité comme une indisponibilité transitoire : ni award ni marqueur, le match est repris.
// Le chemin de production ne rend jamais de handle nil (SharedAccess.Write refuse : erreur, donc
// même issue, voir TestRunPSAConvergence_BurstIndisponible).
func TestConvergePSA_SharedNilAvecAlias(t *testing.T) {
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	ctx := context.Background()
	seedPSAMatch(t, player, "psa-nil")

	client := &mockHaloClient{statsBody: map[string]map[string]any{
		"psa-nil": psaBody("psa-nil", "NilGT", true),
	}}
	if done := convergePSA(ctx, player, nil, client, psaTestXUID, []string{"psa-nil"}); done != 0 {
		t.Fatalf("shared nil avec alias : done = %d, want 0", done)
	}
	if got := selectMatchesMissingPSA(ctx, player); len(got) != 1 {
		t.Fatalf("shared nil avec alias : sélection = %v, want [psa-nil]", got)
	}
}
