// Package duckdb — engagement_score_repo_integration_test.go : tests
// d'integration pour EngagementScoreRepo via DuckDB :memory: avec migration
// engagement Phase 2 appliquee.
//
// Phase 8 du plan engagement.
package duckdb_test

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
	ddb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
)

// setupEngagementDB ouvre une DB in-memory, cree les tables minimales et
// applique le schema engagement Phase 2. Retourne un PlayerDB minimal cible
// pour les tests repo.
func setupEngagementDB(t *testing.T) *ddb.PlayerDB {
	t.Helper()

	db, err := ddb.OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()

	// Tables minimales requises.
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS player_match_enrichment (
			match_id VARCHAR PRIMARY KEY,
			performance_score DOUBLE,
			session_id VARCHAR,
			session_label VARCHAR,
			is_with_friends BOOLEAN DEFAULT FALSE,
			teammates_signature VARCHAR,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,
		// Application du schema Phase 2 engagement (mirror steps_engagement.go).
		`ALTER TABLE player_match_enrichment ADD COLUMN engagement_score DOUBLE`,
		`ALTER TABLE player_match_enrichment ADD COLUMN engagement_score_brut DOUBLE`,
		`ALTER TABLE player_match_enrichment ADD COLUMN engagement_score_confidence VARCHAR`,
		`ALTER TABLE player_match_enrichment ADD COLUMN mode_category VARCHAR`,
		`ALTER TABLE player_match_enrichment ADD COLUMN xuid VARCHAR`,
		`CREATE TABLE engagement_coefficients (
			xuid             VARCHAR NOT NULL,
			mode_category    VARCHAR NOT NULL,
			coef_team_share  DOUBLE NOT NULL,
			coef_lobby_share DOUBLE NOT NULL,
			n_matches        INTEGER NOT NULL,
			last_updated     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (xuid, mode_category)
		)`,
		// Modele lobby-anchored v2 (mirror create_engagement_response_bins_table).
		`CREATE TABLE engagement_response_bins (
			xuid          VARCHAR NOT NULL,
			mode_category VARCHAR NOT NULL,
			intensity_bin VARCHAR NOT NULL,
			lower_bound   DOUBLE NOT NULL,
			upper_bound   DOUBLE NOT NULL,
			coef_lobby    DOUBLE NOT NULL,
			n_matches     INTEGER NOT NULL,
			last_updated  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (xuid, mode_category, intensity_bin)
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(ctx, s); err != nil {
			t.Fatalf("setup ddl %q: %v", s, err)
		}
	}

	// Append-only #23046 : convertit player_match_enrichment (id PK + stage +
	// written_at) et crée la vue player_match_enrichment_latest (lue par le repo).
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}

	return &ddb.PlayerDB{Player: db, XUID: "xuid-test"}
}

// =============================================================================
// SaveEngagementCoefficient + LoadEngagementCoefficient
// =============================================================================

func TestEngagementRepo_SaveAndLoadCoefficient(t *testing.T) {
	pdb := setupEngagementDB(t)
	repo := ddb.NewEngagementScoreRepo(pdb)
	ctx := context.Background()

	// 1. Cold start : pas de coef stocke.
	got, err := repo.LoadEngagementCoefficient(ctx, "xuid-1", "PvP_ranked")
	if err != nil {
		t.Fatalf("LoadEngagementCoefficient (empty): %v", err)
	}
	if got != nil {
		t.Errorf("expected nil coef on cold start, got %+v", got)
	}

	// 2. Save un coef.
	if err := repo.SaveEngagementCoefficient(ctx, domainCoef("xuid-1", "PvP_ranked", 1.05, 200)); err != nil {
		t.Fatalf("SaveEngagementCoefficient: %v", err)
	}

	// 3. Load -> doit retourner les valeurs sauvees (coef_team_share non lu).
	got, err = repo.LoadEngagementCoefficient(ctx, "xuid-1", "PvP_ranked")
	if err != nil {
		t.Fatalf("LoadEngagementCoefficient: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil coef after save")
	}
	if got.CoefLobbyShare != 1.05 || got.NMatches != 200 {
		t.Errorf("unexpected coef values: %+v", got)
	}

	// 4. UPSERT : sauver a nouveau avec valeurs differentes -> doit remplacer.
	if err := repo.SaveEngagementCoefficient(ctx, domainCoef("xuid-1", "PvP_ranked", 1.15, 250)); err != nil {
		t.Fatalf("SaveEngagementCoefficient (upsert): %v", err)
	}
	got, _ = repo.LoadEngagementCoefficient(ctx, "xuid-1", "PvP_ranked")
	if got == nil || got.CoefLobbyShare != 1.15 || got.NMatches != 250 {
		t.Errorf("expected upserted values, got %+v", got)
	}
}

// =============================================================================
// SaveResponseBins + LoadResponseBins (modele lobby-anchored v2)
// =============================================================================

func TestEngagementRepo_SaveAndLoadResponseBins(t *testing.T) {
	pdb := setupEngagementDB(t)
	repo := ddb.NewEngagementScoreRepo(pdb)
	ctx := context.Background()

	// 1. Cold start : aucun bin.
	got, err := repo.LoadResponseBins(ctx, "xuid-1", "PvP_ranked")
	if err != nil {
		t.Fatalf("LoadResponseBins (empty): %v", err)
	}
	if got != nil {
		t.Errorf("expected nil bins on cold start, got %+v", got)
	}

	// 2. Save 3 bins.
	bins := domain.EngagementResponseBins{
		XUID:         "xuid-1",
		ModeCategory: "PvP_ranked",
		Bins: []domain.EngagementIntensityBin{
			{Bin: "calme", LowerBound: 0, UpperBound: 4, CoefLobby: 1.5, NMatches: 20},
			{Bin: "standard", LowerBound: 4, UpperBound: 7, CoefLobby: 1.0, NMatches: 18},
			{Bin: "chaotique", LowerBound: 7, UpperBound: 12, CoefLobby: 0.5, NMatches: 22},
		},
	}
	if err := repo.SaveResponseBins(ctx, bins); err != nil {
		t.Fatalf("SaveResponseBins: %v", err)
	}

	// 3. Load -> 3 bins ordonnes par lower_bound, resolution correcte.
	got, err = repo.LoadResponseBins(ctx, "xuid-1", "PvP_ranked")
	if err != nil {
		t.Fatalf("LoadResponseBins: %v", err)
	}
	if got == nil || len(got.Bins) != 3 {
		t.Fatalf("expected 3 bins, got %+v", got)
	}
	if b, ok := got.ResolveBin(10.0); !ok || b.Bin != "chaotique" || b.CoefLobby != 0.5 {
		t.Errorf("ResolveBin(10) want chaotique/0.5, got %+v (ok=%v)", b, ok)
	}
	if b, ok := got.ResolveBin(1.0); !ok || b.Bin != "calme" {
		t.Errorf("ResolveBin(1) want calme, got %+v", b)
	}

	// 4. UPSERT : re-save avec valeurs differentes -> remplace (pas de doublon).
	bins.Bins[2].CoefLobby = 0.3
	bins.Bins[2].NMatches = 25
	if err := repo.SaveResponseBins(ctx, bins); err != nil {
		t.Fatalf("SaveResponseBins (upsert): %v", err)
	}
	got, _ = repo.LoadResponseBins(ctx, "xuid-1", "PvP_ranked")
	if got == nil || len(got.Bins) != 3 {
		t.Fatalf("expected still 3 bins after upsert, got %+v", got)
	}
	if b, _ := got.ResolveBin(10.0); b.CoefLobby != 0.3 || b.NMatches != 25 {
		t.Errorf("expected upserted chaotique coef 0.3/n25, got %+v", b)
	}
}

// =============================================================================
// LoadAllCoefficients
// =============================================================================

func TestEngagementRepo_LoadAllCoefficients(t *testing.T) {
	pdb := setupEngagementDB(t)
	repo := ddb.NewEngagementScoreRepo(pdb)
	ctx := context.Background()

	// Save 2 coefs sur differentes categories.
	_ = repo.SaveEngagementCoefficient(ctx, domainCoef("xuid-1", "PvP_ranked", 1.05, 200))
	_ = repo.SaveEngagementCoefficient(ctx, domainCoef("xuid-1", "PvP_unranked", 0.92, 150))
	// Coef d'un autre joueur — ne doit PAS etre retourne.
	_ = repo.SaveEngagementCoefficient(ctx, domainCoef("xuid-OTHER", "PvP_ranked", 1.40, 300))

	coefs, err := repo.LoadAllCoefficients(ctx, "xuid-1")
	if err != nil {
		t.Fatalf("LoadAllCoefficients: %v", err)
	}
	if len(coefs) != 2 {
		t.Errorf("expected 2 coefs for xuid-1, got %d: %+v", len(coefs), coefs)
	}
}

// =============================================================================
// LoadPlayerHistory
// =============================================================================

func TestEngagementRepo_LoadPlayerHistory(t *testing.T) {
	pdb := setupEngagementDB(t)
	repo := ddb.NewEngagementScoreRepo(pdb)
	ctx := context.Background()

	// Inserer 3 enrichments avec engagement_score_brut.
	_, _ = pdb.Player.Exec(ctx, `
		INSERT INTO player_match_enrichment (match_id, xuid, mode_category, engagement_score_brut)
		VALUES
			('m1', 'xuid-1', 'PvP_ranked', 0.5),
			('m2', 'xuid-1', 'PvP_ranked', -0.3),
			('m3', 'xuid-1', 'PvP_ranked', 1.0),
			('m4', 'xuid-1', 'PvP_unranked', 0.8)
	`)

	// Query history.
	history, err := repo.LoadPlayerHistory(ctx, port.EngagementHistoryFilter{
		XUID:         "xuid-1",
		ModeCategory: "PvP_ranked",
		Limit:        10,
	})
	if err != nil {
		t.Fatalf("LoadPlayerHistory: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("expected 3 PvP_ranked history entries, got %d", len(history))
	}

	// Test exclude.
	history, _ = repo.LoadPlayerHistory(ctx, port.EngagementHistoryFilter{
		XUID:           "xuid-1",
		ModeCategory:   "PvP_ranked",
		Limit:          10,
		ExcludeMatchID: "m2",
	})
	if len(history) != 2 {
		t.Errorf("expected 2 entries after exclude, got %d", len(history))
	}
	for _, h := range history {
		if h.MatchID == "m2" {
			t.Error("excluded match should not appear")
		}
	}
}

// =============================================================================
// HasEngagementScore
// =============================================================================

func TestEngagementRepo_HasEngagementScore(t *testing.T) {
	pdb := setupEngagementDB(t)
	repo := ddb.NewEngagementScoreRepo(pdb)
	ctx := context.Background()

	_, _ = pdb.Player.Exec(ctx, `
		INSERT INTO player_match_enrichment (match_id, xuid, engagement_score)
		VALUES ('m1', 'xuid-1', 65.0)
	`)
	_, _ = pdb.Player.Exec(ctx, `
		INSERT INTO player_match_enrichment (match_id, xuid)
		VALUES ('m2', 'xuid-1')
	`)

	has, err := repo.HasEngagementScore(ctx, "xuid-1", "m1")
	if err != nil {
		t.Fatalf("HasEngagementScore m1: %v", err)
	}
	if !has {
		t.Error("expected true for m1 (has engagement_score)")
	}

	has, err = repo.HasEngagementScore(ctx, "xuid-1", "m2")
	if err != nil {
		t.Fatalf("HasEngagementScore m2: %v", err)
	}
	if has {
		t.Error("expected false for m2 (engagement_score IS NULL)")
	}

	has, err = repo.HasEngagementScore(ctx, "xuid-1", "missing")
	if err != nil {
		t.Fatalf("HasEngagementScore missing: %v", err)
	}
	if has {
		t.Error("expected false for unknown match")
	}
}

// =============================================================================
// Helpers
// =============================================================================

// domainCoef construit un EngagementCoefficient minimal pour les tests.
// coef_team_share n'est plus porté par le type (inerte, D5).
func domainCoef(xuid, mode string, lobby float64, n int) domain.EngagementCoefficient {
	return domain.EngagementCoefficient{
		XUID:           xuid,
		ModeCategory:   mode,
		CoefLobbyShare: lobby,
		NMatches:       n,
		LastUpdated:    time.Now().UTC(),
	}
}
