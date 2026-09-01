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
// même si les events sont complets (cycle « pur skip », gate 2026-06-10).
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

// TestConvergePSA_ErrorPaths : le contrat d'idempotence par cycles.
//   - fetch KO  → PAS de stamp → le match reste sélectionnable (retry au
//     cycle suivant) ; une régression qui stamperait avant le fetch passerait
//     verte sans ce test.
//   - fetch OK sans PersonalScores extractibles → stamp quand même (état
//     terminal, pas de re-fetch infini).
//   - fetch OK avec PersonalScores → rows insérées + stamp.
func TestConvergePSA_ErrorPaths(t *testing.T) {
	player := openBatchPathTestDB(t, migration.TargetPlayer)
	shared := openBatchPathTestDB(t, migration.TargetShared)
	const xuid = "9999"
	ctx := context.Background()

	for _, mid := range []string{"psa-err", "psa-empty", "psa-full"} {
		if _, err := player.Exec(
			`INSERT INTO player_match_enrichment (match_id) VALUES (?)`, mid); err != nil {
			t.Fatalf("seed %s: %v", mid, err)
		}
	}

	// 1. Fetch KO sur tout : aucun stamp, tout reste sélectionnable.
	failing := &mockHaloClient{getStatsErr: context.DeadlineExceeded}
	if done := convergePSA(ctx, player, shared, failing, xuid, []string{"psa-err"}); done != 0 {
		t.Fatalf("convergePSA avec fetch KO : done = %d, want 0", done)
	}
	if got := selectMatchesMissingPSA(ctx, player); len(got) != 3 {
		t.Fatalf("après fetch KO, sélection = %d matchs, want 3 (psa-err doit rester sélectionnable)", len(got))
	}

	// 2. Fetch OK mais JSON sans PersonalScores pour ce xuid : stamp terminal
	// malgré 0 award (sinon re-fetch infini).
	emptyBody := &mockHaloClient{statsBody: map[string]map[string]any{
		"psa-empty": {"MatchId": "psa-empty", "Players": []any{}},
	}}
	if done := convergePSA(ctx, player, shared, emptyBody, xuid, []string{"psa-empty"}); done != 1 {
		t.Fatalf("convergePSA sans PSA extractibles : done = %d, want 1 (stamp terminal)", done)
	}
	var awards int
	if err := player.QueryRow(
		`SELECT COUNT(*) FROM personal_score_awards_latest WHERE match_id = 'psa-empty'`).Scan(&awards); err != nil {
		t.Fatalf("count awards: %v", err)
	}
	if awards != 0 {
		t.Fatalf("psa-empty : %d awards, want 0", awards)
	}

	// 3. Fetch OK avec PersonalScores (NameId connu killed_player) : insert + stamp.
	fullBody := &mockHaloClient{statsBody: map[string]map[string]any{
		"psa-full": {
			"MatchId": "psa-full",
			"Players": []any{
				map[string]any{
					"PlayerId":   "xuid(" + xuid + ")",
					"PlayerName": "OpportunistGT",
					"PlayerTeamStats": []any{
						map[string]any{"Stats": map[string]any{"CoreStats": map[string]any{
							"PersonalScores": []any{
								map[string]any{"NameId": float64(1024030246), "Count": float64(3),
									"TotalPersonalScoreAwarded": float64(300)},
							},
						}}},
					},
				},
			},
		},
	}}
	if done := convergePSA(ctx, player, shared, fullBody, xuid, []string{"psa-full"}); done != 1 {
		t.Fatalf("convergePSA avec PSA : done = %d, want 1", done)
	}
	if err := player.QueryRow(
		`SELECT COUNT(*) FROM personal_score_awards_latest WHERE match_id = 'psa-full' AND xuid = ?`, xuid).Scan(&awards); err != nil {
		t.Fatalf("count awards full: %v", err)
	}
	if awards != 1 {
		t.Fatalf("psa-full : %d awards, want 1", awards)
	}

	// Convergence opportuniste des alias : le participant du JSON psa-full
	// doit avoir son alias upserté dans shared.xuid_aliases.
	var aliasGT string
	if err := shared.QueryRow(
		`SELECT gamertag FROM xuid_aliases WHERE xuid = ?`, xuid).Scan(&aliasGT); err != nil {
		t.Fatalf("alias opportuniste absent: %v", err)
	}
	if aliasGT != "OpportunistGT" {
		t.Fatalf("alias = %q, want OpportunistGT", aliasGT)
	}

	// État final : seul psa-err reste sélectionnable.
	got := selectMatchesMissingPSA(ctx, player)
	if len(got) != 1 || got[0] != "psa-err" {
		t.Fatalf("sélection finale = %v, want [psa-err]", got)
	}
}
