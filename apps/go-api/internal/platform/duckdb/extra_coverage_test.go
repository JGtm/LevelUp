//go:build integration

// Package duckdb — extra_coverage_test.go : tests complémentaires pour atteindre ≥ 70%.
//
// Couvre : CareerRepo.GetLUSRHistory, CitationsRepo.LoadMedalCitationMappings,
// DB.SQLDb/Path, MatchViewRepo (medals/events/weapons/kv), PoolKey, CloseAll,
// SquadRepo (LoadTeammateMatches, LoadImpactEvents, LoadSynthesisHeatmap, LoadSynthesisMatches).
package duckdb

import (
	"context"
	"testing"
)

// ---------------------------------------------------------------------------
// CareerRepo — GetLUSRHistory (Q8)
// ---------------------------------------------------------------------------

func TestCareerRepo_GetLUSRHistory_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCareerRepo(pdb)
	history, err := repo.GetLUSRHistory(context.Background())
	if err != nil {
		t.Fatalf("GetLUSRHistory: %v", err)
	}
	// Le seed insere 2 rows dans match_skill_rank (m1 CSR ranked, m2 LUSR social).
	// Q8LUSRHistory ne filtre pas par playlist_group : retourne l'historique
	// complet (CSR + LUSR), le filtrage etant la responsabilite du service
	// qui projette les checkpoints en cards par playlist_group.
	if len(history) != 2 {
		t.Errorf("attendu 2 (CSR + LUSR), obtenu %d", len(history))
	}
}

// TestCareerRepo_GetLUSRHistory_ExcludesLUSRV2 : signalement #8 — la row 'LUSR_V2'
// (étiquette d'audit interne) NE DOIT PAS remonter (sinon série fantôme dupliquée dans
// le graphe « Évolution LUSR / CSR »). On ajoute une row LUSR_V2 au seed (m1 CSR + m2
// LUSR) et on vérifie qu'elle est filtrée par Q8.
func TestCareerRepo_GetLUSRHistory_ExcludesLUSRV2(t *testing.T) {
	pdb := newTestPlayerDB(t)
	if _, err := pdb.Player.Exec(context.Background(),
		`INSERT INTO match_skill_rank (match_id, rating_type, rating_value, tier, tier_label, playlist_group, start_time)
		 VALUES ('m3_v2', 'LUSR_V2', 1500, 'Diamond', 'Diamant', 'h5_arena', NOW())`); err != nil {
		t.Fatalf("seed LUSR_V2: %v", err)
	}
	repo := NewCareerRepo(pdb)
	history, err := repo.GetLUSRHistory(context.Background())
	if err != nil {
		t.Fatalf("GetLUSRHistory: %v", err)
	}
	// Toujours 2 (m1 CSR + m2 LUSR) — la row LUSR_V2 est exclue.
	if len(history) != 2 {
		t.Fatalf("attendu 2 (CSR + LUSR, LUSR_V2 exclu), obtenu %d", len(history))
	}
	for _, cp := range history {
		if cp.RatingType == "LUSR_V2" {
			t.Errorf("LUSR_V2 ne doit jamais apparaître dans l'historique: %+v", cp)
		}
	}
}

// ---------------------------------------------------------------------------
// CitationsRepo — LoadMedalCitationMappings (Q36b, sur pdb.Metadata)
// ---------------------------------------------------------------------------

func TestCitationsRepo_LoadMedalCitationMappings_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewCitationsRepo(pdb)
	rows, err := repo.LoadMedalCitationMappings(context.Background())
	if err != nil {
		t.Fatalf("LoadMedalCitationMappings: %v", err)
	}
	// citation_mappings contient 1 entrée avec medal_id non-null et mapping_type='medal'
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

// ---------------------------------------------------------------------------
// DB — SQLDb() et Path() (accesseurs triviaux)
// ---------------------------------------------------------------------------

func TestDB_SQLDb(t *testing.T) {
	db := openMemDB(t)
	if db.SQLDb() == nil {
		t.Error("SQLDb() ne doit pas retourner nil")
	}
}

func TestDB_Path(t *testing.T) {
	db := openMemDB(t)
	if db.Path() != ":memory:" {
		t.Errorf("Path() = %q, want :memory:", db.Path())
	}
}

// ---------------------------------------------------------------------------
// MatchViewRepo — GetMatchMedals (Q14)
// ---------------------------------------------------------------------------

func TestMatchViewRepo_GetMatchMedals_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)
	// medals_earned est vide → 0 résultats
	medals, err := repo.GetMatchMedals(context.Background(), pTestXUID, "m1")
	if err != nil {
		t.Fatalf("GetMatchMedals: %v", err)
	}
	if len(medals) != 0 {
		t.Errorf("attendu 0 médailles, obtenu %d", len(medals))
	}
}

// ---------------------------------------------------------------------------
// MatchViewRepo — GetMatchEvents (Q21)
// ---------------------------------------------------------------------------

func TestMatchViewRepo_GetMatchEvents_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)
	events, err := repo.GetMatchEvents(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetMatchEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("attendu 0 events, obtenu %d", len(events))
	}
}

// ---------------------------------------------------------------------------
// MatchViewRepo — GetMatchKVPairs (Q20, vue v_killer_victim_full)
// ---------------------------------------------------------------------------

func TestMatchViewRepo_GetMatchKVPairs_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)
	// v_killer_victim_full retourne toujours 0 lignes (stub WHERE FALSE)
	pairs, err := repo.GetMatchKVPairs(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetMatchKVPairs: %v", err)
	}
	if len(pairs) != 0 {
		t.Errorf("attendu 0 paires, obtenu %d", len(pairs))
	}
}

// ---------------------------------------------------------------------------
// PlayerPoolConfig — PoolKey
// ---------------------------------------------------------------------------

func TestPoolKey_WithTitleSlug(t *testing.T) {
	cfg := PlayerPoolConfig{TitleSlug: "halo_infinite", Gamertag: "HeroPlayer"}
	expected := "halo_infinite:HeroPlayer"
	if got := cfg.PoolKey(); got != expected {
		t.Errorf("PoolKey = %q, want %q", got, expected)
	}
}

func TestPoolKey_WithoutTitleSlug(t *testing.T) {
	cfg := PlayerPoolConfig{Gamertag: "HeroPlayer"}
	expected := "HeroPlayer"
	if got := cfg.PoolKey(); got != expected {
		t.Errorf("PoolKey = %q, want %q", got, expected)
	}
}

// ---------------------------------------------------------------------------
// CloseAll — pool global
// ---------------------------------------------------------------------------

func TestCloseAll_ClearsPool(t *testing.T) {
	db1 := openMemDB(t)
	db2 := openMemDB(t)
	db3 := openMemDB(t)
	pdb := &PlayerDB{Player: db1, Shared: db2, Metadata: db3}
	const testKey = "_test_close_all_key_"
	globalPool.Store(testKey, pdb)

	CloseAll()

	if _, ok := globalPool.Load(testKey); ok {
		t.Error("CloseAll n'a pas supprimé l'entrée du globalPool")
	}
}

// ---------------------------------------------------------------------------
// SquadRepo — LoadTeammateMatches (Q31)
// ---------------------------------------------------------------------------

func TestSquadRepo_LoadTeammateMatches_Empty(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSquadRepo(pdb)
	// Pas de coéquipier "xuid_other" → 0 matchs communs
	rows, err := repo.LoadTeammateMatches(context.Background(), pTestXUID, "xuid_other_999")
	if err != nil {
		t.Fatalf("LoadTeammateMatches: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0, obtenu %d", len(rows))
	}
}

func TestSquadRepo_LoadTeammateMatches_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants
		 (match_id,xuid,gamertag,outcome,kills,deaths,team_id,kda,accuracy,time_played_seconds,team_mmr)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		"m1", "xuid_mate_003", "AllyPlayer", 2, 7, 3, 1, 1.4, 0.5, 600, 1200.0)
	repo := NewSquadRepo(pdb)
	rows, err := repo.LoadTeammateMatches(ctx, pTestXUID, "xuid_mate_003")
	if err != nil {
		t.Fatalf("LoadTeammateMatches: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

// ---------------------------------------------------------------------------
// SquadRepo — LoadImpactEvents (nil input → early return)
// ---------------------------------------------------------------------------

func TestSquadRepo_LoadImpactEvents_NilMatchIDs(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSquadRepo(pdb)
	rows, err := repo.LoadImpactEvents(context.Background(), nil)
	if err != nil {
		t.Fatalf("LoadImpactEvents nil: %v", err)
	}
	if rows != nil {
		t.Errorf("attendu nil, obtenu %v", rows)
	}
}

func TestSquadRepo_LoadImpactEvents_WithMatchID(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSquadRepo(pdb)
	// highlight_events est vide → 0 events mais pas d'erreur
	rows, err := repo.LoadImpactEvents(context.Background(), []string{"m1"})
	if err != nil {
		t.Fatalf("LoadImpactEvents: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("attendu 0 events, obtenu %d", len(rows))
	}
}

// ---------------------------------------------------------------------------
// SquadRepo — LoadSynthesisHeatmap (Q33)
// ---------------------------------------------------------------------------

func TestSquadRepo_LoadSynthesisHeatmap_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSquadRepo(pdb)
	rows, err := repo.LoadSynthesisHeatmap(context.Background(), pTestXUID)
	if err != nil {
		t.Fatalf("LoadSynthesisHeatmap: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}

// ---------------------------------------------------------------------------
// SquadRepo — LoadSynthesisMatches (Q33b)
// ---------------------------------------------------------------------------

func TestSquadRepo_LoadSynthesisMatches_WithData(t *testing.T) {
	pdb := newTestPlayerDB(t)
	repo := NewSquadRepo(pdb)
	rows, err := repo.LoadSynthesisMatches(context.Background(), pTestXUID)
	if err != nil {
		t.Fatalf("LoadSynthesisMatches: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("attendu 1, obtenu %d", len(rows))
	}
}
