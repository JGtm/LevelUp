//go:build integration

// Package persist — player_persister_test.go : tests TDD pour PlayerPersister.
//
// Contrat à valider :
//
//  1. Persist d'un PlayerBatch avec Enrichment + SkillRank + LUSRComponents
//     + Citations + PersonalScoreAwards + CareerProgression → toutes les
//     tables peuplées en 1 transaction.
//  2. Partial enrichment : SEULEMENT certains champs pointer non-nil → INSERT
//     dynamique n'inclut QUE ces colonnes (property "1 champ pointer = 1 colonne").
//  3. match_id déjà dans player_match_enrichment → no-op (idempotent).
//  4. Atomicité : INSERT échoue mid-batch → rollback complet.
//  5. LUSR vs CSR : SkillRankInsert.RatingType supporte les deux.

package persist

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/sync"
)

// ─── Helpers ───────────────────────────────────────────────────────────────

// openPlayerTestDB ouvre une DuckDB in-memory avec le schéma player appliqué
// (migrations + bootstrap playerSchemaSQL pour les tables sequencées comme
// personal_score_awards qui ne sont créées que via EnsurePlayerSchema).
func openPlayerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Bootstrap FIRST : crée les tables de base via playerSchemaSQL
	// (player_match_enrichment, match_skill_rank, personal_score_awards
	// avec sequence, etc.). Les migrations addColumnIfMissing qui suivent
	// dépendent de l'existence de ces tables — celles qui skip silencieusement
	// si la table n'existe pas (ex. engagement_score) ne s'appliqueraient pas
	// dans l'ordre inverse.
	if err := sync.EnsurePlayerSchema(t.Context(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	// Migrations player officielles (ajoutent les colonnes optionnelles)
	if err := migration.RunForDB(db, migration.TargetPlayer); err != nil {
		t.Fatalf("migrate player: %v", err)
	}
	return db
}

func helperPlayerBatch(matchID string) *MatchBatch {
	float64Ptr := func(v float64) *float64 { return &v }
	intPtr := func(v int) *int { return &v }
	strPtr := func(v string) *string { return &v }
	boolPtr := func(v bool) *bool { return &v }
	tPtr := func(v time.Time) *time.Time { return &v }

	b := NewBatchBuilder("halo_infinite", "Alice", "1111", "test")
	b.SetEnrichment(&EnrichmentRow{
		MatchID:                   matchID,
		PerformanceScore:          float64Ptr(75.2),
		PerformanceChain:          strPtr("arena_slayer"),
		DominanceFlag:             intPtr(1),
		EngagementScore:           float64Ptr(62.5),
		EngagementScoreConfidence: strPtr("full"),
		ModeCategory:              strPtr("PvP_unranked"),
		SessionID:                 strPtr("S001"),
		SessionLabel:              strPtr("Session 1"),
		IsWithFriends:             boolPtr(true),
		HadBotTeammate:            boolPtr(false),
		TeammatesSignature:        strPtr("9876543210"),
		UpdatedAt:                 tPtr(time.Now().UTC()),
	})
	b.SetSkillRank(&SkillRankInsert{
		MatchID:     matchID,
		RatingType:  "LUSR",
		RatingValue: float64Ptr(28.5),
		Tier:        strPtr("Onyx"),
		TierLabel:   strPtr("Onyx I"),
		RatingDelta: float64Ptr(+0.32),
	})
	b.AddLUSRComponents([]LUSRComponentInsert{
		{MatchID: matchID, ComponentName: "kills_vs_expected", Value: 0.45, Weight: 0.31},
		{MatchID: matchID, ComponentName: "deaths_vs_expected", Value: -0.12, Weight: 0.28},
	})
	b.AddCitations([]CitationInsert{
		{MatchID: matchID, CitationNameNorm: "killing_spree", Value: 3},
	})
	b.AddPersonalScoreAwards([]PersonalScoreAwardInsert{
		{
			MatchID: matchID, XUID: "1111",
			AwardName: "FlagCarrier", AwardCategory: "ObjectivePlayer",
			AwardCount: 2, AwardScore: 500,
		},
	})
	return b.Build()
}

// ─── Test 1 : Persist complet ──────────────────────────────────────────────

func TestPlayerPersister_FullBatch_InsertsAllTables(t *testing.T) {
	db := openPlayerTestDB(t)
	p := NewPlayerPersister(db)

	batch := helperPlayerBatch("pm_full_001")
	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	tests := []struct {
		table string
		where string
		args  []any
		want  int
	}{
		{"player_match_enrichment", "match_id = ?", []any{"pm_full_001"}, 1},
		{"match_skill_rank", "match_id = ?", []any{"pm_full_001"}, 1},
		{"lusr_component_history", "match_id = ?", []any{"pm_full_001"}, 2},
		{"match_citations", "match_id = ?", []any{"pm_full_001"}, 1},
		{"personal_score_awards", "match_id = ?", []any{"pm_full_001"}, 1},
	}
	for _, tc := range tests {
		var n int
		q := "SELECT COUNT(*) FROM " + tc.table + " WHERE " + tc.where
		if err := db.QueryRow(q, tc.args...).Scan(&n); err != nil {
			t.Errorf("%s: %v", tc.table, err)
			continue
		}
		if n != tc.want {
			t.Errorf("%s : got %d rows, want %d", tc.table, n, tc.want)
		}
	}
}

// ─── Test 2 : Partial enrichment — INSERT dynamique ───────────────────────

func TestPlayerPersister_PartialEnrichment_InsertsOnlyNonNilColumns(t *testing.T) {
	db := openPlayerTestDB(t)
	p := NewPlayerPersister(db)

	// Batch avec UNIQUEMENT performance_score + dominance_flag (1 numeric + 1 int)
	float64Ptr := func(v float64) *float64 { return &v }
	intPtr := func(v int) *int { return &v }

	builder := NewBatchBuilder("halo_infinite", "Alice", "1111", "test")
	builder.SetEnrichment(&EnrichmentRow{
		MatchID:          "pm_partial_001",
		PerformanceScore: float64Ptr(50.0),
		DominanceFlag:    intPtr(2),
		// tous les autres champs pointer = nil → ne doivent PAS apparaître dans l'INSERT
	})
	if err := p.Persist(context.Background(), builder.Build()); err != nil {
		t.Fatalf("Persist: %v", err)
	}

	// Vérifier que les colonnes non-set sont NULL en DB (DEFAULT pour bools / NULL pour les autres).
	var perfScore sql.NullFloat64
	var dominance sql.NullInt32
	var sessionID sql.NullString
	var engagementScore sql.NullFloat64
	var teammatesSig sql.NullString

	err := db.QueryRow(`
		SELECT performance_score, dominance_flag, session_id, engagement_score, teammates_signature
		FROM player_match_enrichment WHERE match_id = ?`, "pm_partial_001",
	).Scan(&perfScore, &dominance, &sessionID, &engagementScore, &teammatesSig)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if !perfScore.Valid || perfScore.Float64 != 50.0 {
		t.Errorf("performance_score = %+v, want 50.0", perfScore)
	}
	if !dominance.Valid || dominance.Int32 != 2 {
		t.Errorf("dominance_flag = %+v, want 2", dominance)
	}
	if sessionID.Valid {
		t.Errorf("session_id devrait être NULL (non set), got %q", sessionID.String)
	}
	if engagementScore.Valid {
		t.Errorf("engagement_score devrait être NULL (non set), got %f", engagementScore.Float64)
	}
	if teammatesSig.Valid {
		t.Errorf("teammates_signature devrait être NULL (non set), got %q", teammatesSig.String)
	}
}

// ─── Test 3 : DuplicateMatchID → idempotent ────────────────────────────────

func TestPlayerPersister_DuplicateMatchID_Idempotent(t *testing.T) {
	db := openPlayerTestDB(t)
	p := NewPlayerPersister(db)

	float64Ptr := func(v float64) *float64 { return &v }

	b1 := helperPlayerBatch("pm_dup_001")
	if err := p.Persist(context.Background(), b1); err != nil {
		t.Fatalf("1er Persist: %v", err)
	}

	// 2e batch — même match_id, performance_score modifié
	b2 := helperPlayerBatch("pm_dup_001")
	b2.PlayerData.Enrichment.PerformanceScore = float64Ptr(99.9)

	if err := p.Persist(context.Background(), b2); err != nil {
		t.Fatalf("2e Persist (idempotent attendu): %v", err)
	}

	// performance_score doit rester 75.2 (1er batch)
	var score float64
	if err := db.QueryRow(`SELECT performance_score FROM player_match_enrichment WHERE match_id = ?`,
		"pm_dup_001").Scan(&score); err != nil {
		t.Fatal(err)
	}
	if score != 75.2 {
		t.Errorf("performance_score = %f, want 75.2 (INSERT-only)", score)
	}
}

// ─── Test 4 : Atomicité — INSERT échoue mid-batch → rollback complet ──────

func TestPlayerPersister_AtomicityOnFailure_RollsBackAll(t *testing.T) {
	db := openPlayerTestDB(t)
	p := NewPlayerPersister(db)

	// Batch avec 2 LUSR components ayant la MEME PK (match_id, component_name)
	// → PK conflict sur le 2e → ROLLBACK
	float64Ptr := func(v float64) *float64 { return &v }

	builder := NewBatchBuilder("halo_infinite", "Alice", "1111", "test")
	builder.SetEnrichment(&EnrichmentRow{
		MatchID:          "pm_atomic_001",
		PerformanceScore: float64Ptr(80),
	})
	builder.SetSkillRank(&SkillRankInsert{
		MatchID: "pm_atomic_001", RatingType: "LUSR",
	})
	builder.AddLUSRComponents([]LUSRComponentInsert{
		{MatchID: "pm_atomic_001", ComponentName: "X", Value: 0.5, Weight: 0.5},
		{MatchID: "pm_atomic_001", ComponentName: "X", Value: 0.6, Weight: 0.5}, // PK conflict
	})

	err := p.Persist(context.Background(), builder.Build())
	if err == nil {
		t.Fatal("Persist devrait échouer sur PK conflict LUSR components")
	}

	tables := []string{
		"player_match_enrichment",
		"match_skill_rank",
		"lusr_component_history",
	}
	for _, tbl := range tables {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM "+tbl+" WHERE match_id = ?",
			"pm_atomic_001").Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("atomicité cassée — %s a %d rows pour le batch foiré", tbl, n)
		}
	}
}

// ─── Test 5 : NilBatch → erreur ────────────────────────────────────────────

func TestPlayerPersister_NilBatch_ReturnsError(t *testing.T) {
	db := openPlayerTestDB(t)
	p := NewPlayerPersister(db)
	if err := p.Persist(context.Background(), nil); err == nil {
		t.Error("Persist(nil) devrait retourner une erreur")
	}
}

// ─── Test 6 : EmptyEnrichment → no-op ──────────────────────────────────────

func TestPlayerPersister_NoEnrichment_NoOp(t *testing.T) {
	db := openPlayerTestDB(t)
	p := NewPlayerPersister(db)

	builder := NewBatchBuilder("halo_infinite", "Alice", "1111", "test")
	batch := builder.Build() // no Enrichment set

	if err := p.Persist(context.Background(), batch); err != nil {
		t.Fatalf("Persist no-op: %v", err)
	}

	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM player_match_enrichment").Scan(&n)
	if n != 0 {
		t.Errorf("player_match_enrichment doit rester vide (no-op), got %d rows", n)
	}
}

// ─── Test 7 : Property — l'INSERT SQL ne contient PAS les colonnes nil ────

func TestPlayerPersister_EnrichmentFields_OmitsNilPointers(t *testing.T) {
	float64Ptr := func(v float64) *float64 { return &v }

	row := &EnrichmentRow{
		MatchID:          "p1",
		PerformanceScore: float64Ptr(80),
		// le reste = nil
	}
	fields := enrichmentFields(row)

	// Doit contenir match_id + performance_score, et SEULEMENT ça.
	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.name)
	}
	got := strings.Join(names, ",")
	want := "match_id,performance_score"
	if got != want {
		t.Errorf("enrichmentFields = [%s], want [%s]", got, want)
	}
}
