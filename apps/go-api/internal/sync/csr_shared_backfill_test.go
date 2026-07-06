// Package sync â€” csr_shared_backfill_test.go : tests intÃ©gration de
// BackfillSharedCSRsFromAPI.
//
// Couvre :
//   - dry-run sans appel API
//   - idempotence (skip si shared.match_csrs dÃ©jÃ  rempli)
//   - --force re-fetch
//   - capture per-participant (4 joueurs sur 1 match â†’ 4 rows)
//   - mock Halo client (no network)
package sync

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// mockSkillClient implÃ©mente l'interface HaloClient. Seul GetMatchSkill est
// effectivement exercÃ© par BackfillSharedCSRsFromAPI ; les autres mÃ©thodes
// retournent zÃ©ro pour satisfaire le contrat.
type mockSkillClient struct {
	skillByMatch map[string]map[string]*MatchSkillData
	calls        int
	failNext     bool
}

func (m *mockSkillClient) GetMatchSkill(_ context.Context, matchID string, xuids []string) (map[string]*MatchSkillData, error) {
	m.calls++
	if m.failNext {
		m.failNext = false
		return nil, errors.New("simulated network error")
	}
	data, ok := m.skillByMatch[matchID]
	if !ok {
		return map[string]*MatchSkillData{}, nil
	}
	out := make(map[string]*MatchSkillData, len(xuids))
	for _, x := range xuids {
		if v, ok := data[x]; ok {
			out[x] = v
		}
	}
	return out, nil
}

func (m *mockSkillClient) GetMatchHistory(_ context.Context, _, _ string, _, _ int) ([]MatchHistoryEntry, error) {
	return nil, nil
}
func (m *mockSkillClient) GetMatchStats(_ context.Context, _ string) (map[string]any, error) {
	return nil, nil
}
func (m *mockSkillClient) GetMatchFilm(_ context.Context, _ string) (map[int]FilmChunkData, bool, error) {
	return nil, false, nil
}
func (m *mockSkillClient) GetHighlightEventsChunk(_ context.Context, _ string) ([]byte, int, bool, error) {
	return nil, 0, false, nil
}
func (m *mockSkillClient) GetCareerRank(_ context.Context, _ string) (*CareerRankData, error) {
	return nil, nil
}
func (m *mockSkillClient) GetPlayerCSRs(_ context.Context, _, _ string) ([]PlayerPlaylistCSR, error) {
	return nil, nil
}

func (m *mockSkillClient) GetPlaylistCsr(_ context.Context, _, _, _ string) (*PlayerPlaylistCSR, error) {
	return nil, nil
}

func openSharedForCSRBackfill(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := EnsureSharedSchema(context.Background(), db); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	return db
}

// seedBackfillScenario : 1 match ranked avec 4 participants, et 1 match social
// (ne doit jamais Ãªtre touchÃ© par le backfill).
func seedBackfillScenario(t *testing.T, db *sql.DB, player string) {
	t.Helper()
	// 1 match ranked
	if _, err := db.Exec(`
		INSERT INTO match_registry (match_id, start_time, playlist_id, playlist_name, pair_name, is_ranked, season_id)
		VALUES ('m-ranked', TIMESTAMP '2026-04-15 12:00:00', 'pl-arena', 'Ranked Arena', 'Ranked:Slayer', TRUE, 'CsrSeason13-1')
	`); err != nil {
		t.Fatalf("insert match_registry ranked: %v", err)
	}
	for _, x := range []string{player, "xuid-B", "xuid-C", "xuid-D"} {
		if _, err := db.Exec(`
			INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
			VALUES ('m-ranked', ?, ?, 0, 2)
		`, x, x); err != nil {
			t.Fatalf("insert match_participants %s: %v", x, err)
		}
	}
	// 1 match social (control â€” backfill ne doit PAS le voir)
	if _, err := db.Exec(`
		INSERT INTO match_registry (match_id, start_time, playlist_id, playlist_name, pair_name, is_ranked)
		VALUES ('m-social', TIMESTAMP '2026-04-14 12:00:00', 'pl-qp', 'Quick Play', 'Slayer', FALSE)
	`); err != nil {
		t.Fatalf("insert match_registry social: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
		VALUES ('m-social', ?, 'gt', 0, 2)
	`, player); err != nil {
		t.Fatalf("insert participants social: %v", err)
	}
}

func mockSkillForRankedMatch() map[string]map[string]*MatchSkillData {
	return map[string]map[string]*MatchSkillData{
		"m-ranked": {
			"xuid-A": mkSkill("Gold", 1100, 4, 0, 1085),
			"xuid-B": mkSkill("Diamond", 1500, 3, 0, 1495),
			"xuid-C": mkSkill("Onyx", 1850, 0, 0, 1830),
			"xuid-D": mkSkill("", 0, 0, 3, -1), // placement
		},
	}
}

// â”€â”€â”€ DRY-RUN â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestBackfillSharedCSRs_DryRun_CountsWithoutAPICall(t *testing.T) {
	db := openSharedForCSRBackfill(t)
	seedBackfillScenario(t, db, "xuid-A")
	client := &mockSkillClient{skillByMatch: mockSkillForRankedMatch()}

	res, err := BackfillSharedCSRsFromAPI(context.Background(), client, db, "xuid-A",
		SharedCSRBackfillOpts{DryRun: true})
	if err != nil {
		t.Fatalf("BackfillSharedCSRsFromAPI: %v", err)
	}
	if !res.DryRun {
		t.Error("Result.DryRun: want true")
	}
	if res.RankedMatches != 1 {
		t.Errorf("RankedMatches: want 1 (1 ranked + 1 social ignored), got %d", res.RankedMatches)
	}
	if res.NeedBackfill != 1 {
		t.Errorf("NeedBackfill: want 1, got %d", res.NeedBackfill)
	}
	if res.Fetched != 0 {
		t.Errorf("Fetched: want 0 (dry-run skips API), got %d", res.Fetched)
	}
	if res.Inserted != 0 {
		t.Errorf("Inserted: want 0 (dry-run skips writes), got %d", res.Inserted)
	}
	if client.calls != 0 {
		t.Errorf("mock client called %d times in dry-run, expected 0", client.calls)
	}
	// VÃ©rifier que rien n'a Ã©tÃ© Ã©crit dans match_csrs.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_csrs`).Scan(&n); err != nil {
		t.Fatalf("count match_csrs: %v", err)
	}
	if n != 0 {
		t.Errorf("match_csrs should be empty after dry-run, got %d rows", n)
	}
}

// â”€â”€â”€ EXECUTION RÃ‰ELLE â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestBackfillSharedCSRs_RealRun_InsertsAllParticipants(t *testing.T) {
	db := openSharedForCSRBackfill(t)
	seedBackfillScenario(t, db, "xuid-A")
	client := &mockSkillClient{skillByMatch: mockSkillForRankedMatch()}

	res, err := BackfillSharedCSRsFromAPI(context.Background(), client, db, "xuid-A",
		SharedCSRBackfillOpts{DryRun: false})
	if err != nil {
		t.Fatalf("BackfillSharedCSRsFromAPI: %v", err)
	}
	if res.Inserted != 4 {
		t.Errorf("Inserted: want 4 (4 participants), got %d", res.Inserted)
	}
	if res.Fetched != 1 {
		t.Errorf("Fetched: want 1 (1 match), got %d", res.Fetched)
	}
	if client.calls != 1 {
		t.Errorf("mock client calls: want 1, got %d", client.calls)
	}
	// VÃ©rifier les 4 rows prÃ©sentes.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_csrs WHERE match_id='m-ranked'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 4 {
		t.Errorf("want 4 rows in match_csrs, got %d", n)
	}
	// VÃ©rifier que le match social n'a PAS Ã©tÃ© touchÃ©.
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_csrs WHERE match_id='m-social'`).Scan(&n); err != nil {
		t.Fatalf("count social: %v", err)
	}
	if n != 0 {
		t.Errorf("social match should not be in match_csrs, got %d rows", n)
	}
}

// â”€â”€â”€ IDEMPOTENCE â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestBackfillSharedCSRs_Idempotent_SkipsAlreadyComplete(t *testing.T) {
	db := openSharedForCSRBackfill(t)
	seedBackfillScenario(t, db, "xuid-A")
	client := &mockSkillClient{skillByMatch: mockSkillForRankedMatch()}

	// 1er run : insÃ¨re 4 rows.
	if _, err := BackfillSharedCSRsFromAPI(context.Background(), client, db, "xuid-A",
		SharedCSRBackfillOpts{}); err != nil {
		t.Fatalf("1er run: %v", err)
	}

	// 2e run : doit skip (already complete).
	client.calls = 0
	res, err := BackfillSharedCSRsFromAPI(context.Background(), client, db, "xuid-A",
		SharedCSRBackfillOpts{})
	if err != nil {
		t.Fatalf("2e run: %v", err)
	}
	if res.AlreadyComplete != 1 {
		t.Errorf("AlreadyComplete: want 1, got %d", res.AlreadyComplete)
	}
	if res.NeedBackfill != 0 {
		t.Errorf("NeedBackfill: want 0 (idempotent), got %d", res.NeedBackfill)
	}
	if client.calls != 0 {
		t.Errorf("mock client called %d times on 2e run, expected 0 (idempotent)", client.calls)
	}
}

// â”€â”€â”€ FORCE â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestBackfillSharedCSRs_Force_RefetchesEvenIfComplete(t *testing.T) {
	db := openSharedForCSRBackfill(t)
	seedBackfillScenario(t, db, "xuid-A")
	client := &mockSkillClient{skillByMatch: mockSkillForRankedMatch()}

	// 1er run normal
	if _, err := BackfillSharedCSRsFromAPI(context.Background(), client, db, "xuid-A",
		SharedCSRBackfillOpts{}); err != nil {
		t.Fatalf("1er run: %v", err)
	}
	// 2e run avec --force : doit re-fetch mÃªme si dÃ©jÃ  complet.
	client.calls = 0
	res, err := BackfillSharedCSRsFromAPI(context.Background(), client, db, "xuid-A",
		SharedCSRBackfillOpts{Force: true})
	if err != nil {
		t.Fatalf("force run: %v", err)
	}
	if res.NeedBackfill != 1 {
		t.Errorf("Force NeedBackfill: want 1, got %d", res.NeedBackfill)
	}
	if res.Fetched != 1 {
		t.Errorf("Force Fetched: want 1, got %d", res.Fetched)
	}
	if client.calls != 1 {
		t.Errorf("Force should call API once, got %d", client.calls)
	}
}

// â”€â”€â”€ NETWORK ERROR TOLERANCE â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestBackfillSharedCSRs_SkillErrorContinues(t *testing.T) {
	db := openSharedForCSRBackfill(t)
	seedBackfillScenario(t, db, "xuid-A")
	// 2e match ranked pour exercer la boucle aprÃ¨s une erreur.
	if _, err := db.Exec(`
		INSERT INTO match_registry (match_id, start_time, playlist_id, playlist_name, pair_name, is_ranked, season_id)
		VALUES ('m-ranked-2', TIMESTAMP '2026-04-16 12:00:00', 'pl-arena', 'Ranked Arena', 'Ranked:CTF', TRUE, 'CsrSeason13-1')
	`); err != nil {
		t.Fatalf("seed m-ranked-2: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome) VALUES ('m-ranked-2', 'xuid-A', 'gt', 0, 2)`); err != nil {
		t.Fatalf("seed participants m-ranked-2: %v", err)
	}
	skill := mockSkillForRankedMatch()
	skill["m-ranked-2"] = map[string]*MatchSkillData{
		"xuid-A": mkSkill("Gold", 1200, 5, 0, 1180),
	}
	client := &mockSkillClient{skillByMatch: skill, failNext: true} // 1er appel Ã©choue

	res, err := BackfillSharedCSRsFromAPI(context.Background(), client, db, "xuid-A",
		SharedCSRBackfillOpts{})
	if err != nil {
		t.Fatalf("BackfillSharedCSRsFromAPI: %v", err)
	}
	if res.SkillErrors != 1 {
		t.Errorf("SkillErrors: want 1, got %d", res.SkillErrors)
	}
	// Le 2e match doit s'Ãªtre passÃ© OK.
	if res.Inserted < 1 {
		t.Errorf("Inserted: want â‰¥1 (m-ranked-2 should succeed), got %d", res.Inserted)
	}
}

// â”€â”€â”€ CONTEXT CANCEL â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestBackfillSharedCSRs_RespectContextCancel(t *testing.T) {
	db := openSharedForCSRBackfill(t)
	seedBackfillScenario(t, db, "xuid-A")
	client := &mockSkillClient{skillByMatch: mockSkillForRankedMatch()}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // annule immÃ©diatement
	_, err := BackfillSharedCSRsFromAPI(ctx, client, db, "xuid-A", SharedCSRBackfillOpts{})
	if err == nil {
		t.Error("expected context.Canceled error, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") && err != context.Canceled {
		t.Errorf("expected context canceled error, got %v", err)
	}
}

// â”€â”€â”€ PARTIAL COVERAGE (gap detected) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func TestBackfillSharedCSRs_PartialCoverage_BackfillsGap(t *testing.T) {
	db := openSharedForCSRBackfill(t)
	seedBackfillScenario(t, db, "xuid-A")
	// PrÃ©-remplir match_csrs avec seulement 2 rows sur 4 (gap = 2 vs 4 participants).
	now := time.Now()
	for _, x := range []string{"xuid-A", "xuid-B"} {
		if _, err := db.Exec(`
			INSERT INTO match_csrs (match_id, xuid, rating_type, rating_value, tier, sub_tier, tier_label, season_id, created_at, updated_at)
			VALUES ('m-ranked', ?, 'CSR', 1000, 'Gold', 1, 'Or 1', 'CsrSeason13-1', ?, ?)
		`, x, now, now); err != nil {
			t.Fatalf("pre-fill: %v", err)
		}
	}
	client := &mockSkillClient{skillByMatch: mockSkillForRankedMatch()}

	res, err := BackfillSharedCSRsFromAPI(context.Background(), client, db, "xuid-A",
		SharedCSRBackfillOpts{})
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	// 2 rows < 4 participants â†’ NeedBackfill=1, UPSERT remplace les 2 existantes + ajoute les 2 manquantes.
	if res.NeedBackfill != 1 {
		t.Errorf("NeedBackfill: want 1 (partial), got %d", res.NeedBackfill)
	}
	if res.Inserted != 4 {
		t.Errorf("Inserted: want 4 (all upserted), got %d", res.Inserted)
	}
	// Sémantique append-only : la 2e passe INSERT 4 nouvelles rows
	// physiques en plus des 2 préexistantes → 6 rows physiques.
	// La vue match_csrs_latest projette 1 row par (match_id, xuid) → 4 rows
	// fonctionnelles, soit le count attendu.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_csrs_latest WHERE match_id='m-ranked'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 4 {
		t.Errorf("want 4 rows latest (1 par xuid), got %d", n)
	}
}
