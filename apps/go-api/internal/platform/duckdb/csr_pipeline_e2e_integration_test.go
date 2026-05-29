//go:build integration

// Package duckdb — csr_pipeline_e2e_integration_test.go : tests E2E intégration
// pour la pipeline CSR end-to-end (Phase 6+9+display).
//
// Couvre les 5 scénarios planifiés au Niveau 3 du plan pipeline CSR :
//  1. Home highest_csr placement S13 → threshold 5, badge unranked mappé
//  2. Home recent_playlist_ranks mixed states (matured + placement + social)
//  3. Career CSRs merge catalogue + snapshots (synthetic rows pour playlists non jouées)
//  4. Diag endpoint detect coverage gap
//  5. Display threshold dynamique S2 historique (threshold=10) vs S13 (threshold=5)
//
// Setup : temp DuckDB shared + player + metadata avec migrations appliquées,
// seeded via INSERT direct (pas de mock Halo client — couvre uniquement les
// chemins read/display, pas le sync).
package duckdb_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/migration"
	ddb "levelup/go-api/internal/platform/duckdb"
	syncpkg "levelup/go-api/internal/sync"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	pipelineTestXUID = "2533274823110022"
	pipelineTestSlug = "TestPlayer"
)

// pipelineEnv bundles temp DBs + repos pour les tests E2E.
type pipelineEnv struct {
	playerDB   *ddb.DB
	sharedDB   *ddb.DB
	metadataDB *ddb.DB
	pdb        *ddb.PlayerDB
	homeRepo   *ddb.HomeRepo
	careerRepo *ddb.CareerRepo
	coverRepo  *ddb.CSRCoverageRepo
}

// setupPipelineEnv crée 3 DBs temp, applique les schemas + migrations, et
// retourne un environnement câblé pour tester les repos display.
func setupPipelineEnv(t *testing.T) *pipelineEnv {
	t.Helper()
	tmp := t.TempDir()

	// Player DB
	playerPath := filepath.Join(tmp, "stats.duckdb")
	playerDB, err := ddb.OpenReadWrite(playerPath)
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	t.Cleanup(func() { _ = playerDB.Close() })
	if err := syncpkg.EnsurePlayerSchema(context.Background(), playerDB.SQLDb()); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	// Appliquer les migrations player (ajoute measurement_matches_remaining sur
	// match_skill_rank, utilisée par CSRCoverageRepo + display).
	_ = migration.All()
	if err := migration.RunForDB(playerDB.SQLDb(), migration.TargetPlayer); err != nil {
		t.Fatalf("RunForDB player: %v", err)
	}

	// Shared DB
	sharedPath := filepath.Join(tmp, "shared.duckdb")
	sharedDB, err := ddb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { _ = sharedDB.Close() })
	if err := syncpkg.EnsureSharedSchema(context.Background(), sharedDB.SQLDb()); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}

	// Metadata DB : besoin de csr_placement_thresholds + playlists_catalog.
	// On applique TOUTES les migrations TargetMetadata via RunForDB.
	metaPath := filepath.Join(tmp, "metadata.duckdb")
	metaDB, err := ddb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatalf("open metadata: %v", err)
	}
	t.Cleanup(func() { _ = metaDB.Close() })
	_ = migration.All() // force registration init() side-effects
	if err := migration.RunForDB(metaDB.SQLDb(), migration.TargetMetadata); err != nil {
		t.Fatalf("RunForDB metadata: %v", err)
	}

	pdb := &ddb.PlayerDB{
		Player:       playerDB,
		Shared:       sharedDB,
		SharedReader: ddb.LegacySharedReader(sharedDB),
		Metadata:     metaDB,
		XUID:         pipelineTestXUID,
		TitleSlug:    "halo_infinite",
	}

	thresholds := ddb.NewCSRThresholdsRepo(metaDB)
	currentSeason := "CsrSeason13-1"

	return &pipelineEnv{
		playerDB:   playerDB,
		sharedDB:   sharedDB,
		metadataDB: metaDB,
		pdb:        pdb,
		homeRepo:   ddb.NewHomeRepo(pdb).WithCSRThresholds(thresholds, currentSeason),
		careerRepo: ddb.NewCareerRepo(pdb).WithCSRThresholds(thresholds, currentSeason),
		coverRepo:  ddb.NewCSRCoverageRepo(pdb),
	}
}

// seedMatchRegistry insère un match dans shared.match_registry + un participant
// dans match_participants pour le XUID test. Helper compact.
func seedMatchRegistry(t *testing.T, env *pipelineEnv, matchID, playlistID, playlistName string, isRanked bool, seasonID string, startTime string) {
	t.Helper()
	if _, err := env.sharedDB.SQLDb().Exec(`
		INSERT INTO match_registry (match_id, start_time, playlist_id, playlist_name, pair_name, is_ranked, season_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, matchID, startTime, playlistID, playlistName, "Slayer on Recharge", isRanked, seasonID); err != nil {
		t.Fatalf("seed match_registry %s: %v", matchID, err)
	}
	if _, err := env.sharedDB.SQLDb().Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
		VALUES (?, ?, ?, 0, 2)
	`, matchID, pipelineTestXUID, pipelineTestSlug); err != nil {
		t.Fatalf("seed match_participants %s: %v", matchID, err)
	}
}

// seedMSR insère une ligne match_skill_rank (player DB).
func seedMSR(t *testing.T, env *pipelineEnv, matchID, ratingType string, ratingValue float64, tier, tierLabel string, subTier, measurementRem int) {
	t.Helper()
	if _, err := env.playerDB.SQLDb().Exec(`
		INSERT INTO match_skill_rank (match_id, rating_type, rating_value, tier, sub_tier, tier_label, playlist_group, start_time)
		VALUES (?, ?, ?, ?, ?, ?, 'ranked', CURRENT_TIMESTAMP)
	`, matchID, ratingType, ratingValue, tier, subTier, tierLabel); err != nil {
		t.Fatalf("seed match_skill_rank %s: %v", matchID, err)
	}
}

// seedCSRSnapshot insère une ligne player_csr_snapshots (player DB).
func seedCSRSnapshot(t *testing.T, env *pipelineEnv, playlistID, playlistName, seasonID string, currentValue float64, currentTier string, currentSubTier, currentRem int, alltimeValue float64) {
	t.Helper()
	if _, err := env.playerDB.SQLDb().Exec(`
		INSERT INTO player_csr_snapshots (
			playlist_id, playlist_name, queue, input, season_id,
			current_value, current_tier, current_sub_tier, current_measurement_remaining,
			alltime_value, alltime_tier, alltime_sub_tier
		)
		VALUES (?, ?, '', '', ?, ?, ?, ?, ?, ?, ?, ?)
	`, playlistID, playlistName, seasonID,
		currentValue, currentTier, currentSubTier, currentRem,
		alltimeValue, currentTier, currentSubTier); err != nil {
		t.Fatalf("seed player_csr_snapshots %s: %v", playlistID, err)
	}
}

// seedCatalogPlaylist insère une playlist dans metadata.playlists_catalog.
func seedCatalogPlaylist(t *testing.T, env *pipelineEnv, playlistID, name string, isRanked bool) {
	t.Helper()
	exp := "social"
	if isRanked {
		exp = "ranked"
	}
	if _, err := env.metadataDB.SQLDb().Exec(`
		INSERT OR REPLACE INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical, experience, is_ranked, is_active)
		VALUES ('halo_infinite', ?, ?, ?, ?, TRUE)
	`, playlistID, name, exp, isRanked); err != nil {
		t.Fatalf("seed playlists_catalog %s: %v", playlistID, err)
	}
}

// ── Scénario 1 : Home Recent Playlists — placement S13 threshold 5 ──────────

func TestE2EPipeline_HomeRecentPlaylists_S13_Placement_Threshold5(t *testing.T) {
	env := setupPipelineEnv(t)
	seedMatchRegistry(t, env, "m1", "pl-ranked", "Ranked Arena", true, "CsrSeason13-1", "2026-04-15T12:00:00Z")
	seedMSR(t, env, "m1", "CSR", 0, "Placement", "Placement (2 restants)", 0, 2)
	seedCSRSnapshot(t, env, "pl-ranked", "Ranked Arena", "CsrSeason13-1", 0, "", 0, 2, 0)

	items, err := env.homeRepo.LoadRecentPlaylistRanks(context.Background(), "fr")
	if err != nil {
		t.Fatalf("LoadRecentPlaylistRanks: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	it := items[0]
	if it.PlacementTotal == nil || *it.PlacementTotal != 5 {
		t.Errorf("PlacementTotal: want 5, got %v", it.PlacementTotal)
	}
	if it.MeasurementMatchesRemaining == nil || *it.MeasurementMatchesRemaining != 2 {
		t.Errorf("MeasurementMatchesRemaining: want 2, got %v", it.MeasurementMatchesRemaining)
	}
	// completed = 5-2 = 3 → unranked_(3*10/5)=unranked_6.png
	if it.BadgeImageURL == nil || !strings.Contains(*it.BadgeImageURL, "unranked_6") {
		t.Errorf("BadgeImageURL: want unranked_6.png, got %v", it.BadgeImageURL)
	}
	if it.RatingValue != nil {
		t.Errorf("RatingValue: want nil (placement), got %v", *it.RatingValue)
	}
}

// ── Scénario 2 : Home Recent Playlists — mixed states ───────────────────────

func TestE2EPipeline_HomeRecentPlaylists_MixedStates(t *testing.T) {
	env := setupPipelineEnv(t)
	// 1 matured Gold (S13 → threshold 5)
	seedMatchRegistry(t, env, "m_gold", "pl-arena", "Ranked Arena", true, "CsrSeason13-1", "2026-04-15T12:00:00Z")
	seedMSR(t, env, "m_gold", "CSR", 1100, "Gold", "Or 4", 4, 0)
	// 1 placement (autre playlist, S13)
	seedMatchRegistry(t, env, "m_plac", "pl-slayer", "Ranked Slayer", true, "CsrSeason13-1", "2026-04-14T12:00:00Z")
	seedMSR(t, env, "m_plac", "CSR", 0, "Placement", "Placement (3 restants)", 0, 3)
	seedCSRSnapshot(t, env, "pl-slayer", "Ranked Slayer", "CsrSeason13-1", 0, "", 0, 3, 0)
	// 1 sociale (Quick Play)
	seedMatchRegistry(t, env, "m_qp", "pl-qp", "Quick Play", false, "CsrSeason13-1", "2026-04-13T12:00:00Z")
	seedMSR(t, env, "m_qp", "LUSR", 1450, "", "", 0, 0)

	items, err := env.homeRepo.LoadRecentPlaylistRanks(context.Background(), "fr")
	if err != nil {
		t.Fatalf("LoadRecentPlaylistRanks: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("want 3 items, got %d", len(items))
	}
	byName := map[string]int{}
	for i, it := range items {
		byName[it.PlaylistName] = i
	}
	// Gold matured : placement_total exposé (info), pas de measurement_remaining
	gold := items[byName["Ranked Arena"]]
	if gold.TierLabel == nil || *gold.TierLabel != "Or 4" {
		t.Errorf("Ranked Arena tier_label: want Or 4, got %v", gold.TierLabel)
	}
	if gold.MeasurementMatchesRemaining != nil {
		t.Errorf("Ranked Arena MeasurementMatchesRemaining: want nil (matured), got %v", *gold.MeasurementMatchesRemaining)
	}
	if gold.PlacementTotal == nil || *gold.PlacementTotal != 5 {
		t.Errorf("Ranked Arena PlacementTotal: want 5 (S13 info), got %v", gold.PlacementTotal)
	}
	// Placement : measurement_remaining + placement_total
	plac := items[byName["Ranked Slayer"]]
	if plac.MeasurementMatchesRemaining == nil || *plac.MeasurementMatchesRemaining != 3 {
		t.Errorf("Ranked Slayer remaining: want 3, got %v", plac.MeasurementMatchesRemaining)
	}
	if plac.PlacementTotal == nil || *plac.PlacementTotal != 5 {
		t.Errorf("Ranked Slayer PlacementTotal: want 5, got %v", plac.PlacementTotal)
	}
	// Sociale : pas de placement_total exposé (item.IsRanked=false)
	social := items[byName["Quick Play"]]
	if social.IsRanked {
		t.Error("Quick Play should be is_ranked=false")
	}
	if social.PlacementTotal != nil {
		t.Errorf("Quick Play PlacementTotal: want nil (social), got %v", social.PlacementTotal)
	}
}

// ── Scénario 3 : Career CSRs merge catalogue + snapshots ────────────────────

func TestE2EPipeline_CareerCSRs_MergeCatalogPlusSnapshots(t *testing.T) {
	env := setupPipelineEnv(t)
	// La migration seed_ranked_playlists_catalog pré-remplit playlists_catalog
	// avec les playlists ranked réelles HI (4 actives). On repart d'un catalogue
	// vide pour valider le merge sur un jeu contrôlé de 3 playlists fictives.
	if _, err := env.metadataDB.SQLDb().Exec(`DELETE FROM playlists_catalog`); err != nil {
		t.Fatalf("reset playlists_catalog: %v", err)
	}
	// Catalogue : 3 playlists ranked actives
	seedCatalogPlaylist(t, env, "pl-arena", "Ranked Arena", true)
	seedCatalogPlaylist(t, env, "pl-slayer", "Ranked Slayer", true)
	seedCatalogPlaylist(t, env, "pl-doubles", "Ranked Doubles", true)
	// 2 snapshots existants (Arena Gold + Slayer placement)
	seedCSRSnapshot(t, env, "pl-arena", "Ranked Arena", "CsrSeason13-1", 1100, "Gold", 4, 0, 1100)
	seedCSRSnapshot(t, env, "pl-slayer", "Ranked Slayer", "CsrSeason13-1", 0, "", 0, 2, 0)
	// pl-doubles : pas joué → doit ressortir comme placement synthétique 0/5

	playlists, err := env.careerRepo.GetCSRSnapshots(context.Background(), "")
	if err != nil {
		t.Fatalf("GetCSRSnapshots: %v", err)
	}
	if len(playlists) != 3 {
		t.Fatalf("want 3 playlists (2 snapshots + 1 catalog-only), got %d : %+v", len(playlists), playlists)
	}
	byID := map[string]int{}
	for i, p := range playlists {
		byID[p.PlaylistID] = i
	}
	for _, id := range []string{"pl-arena", "pl-slayer", "pl-doubles"} {
		if _, ok := byID[id]; !ok {
			t.Errorf("playlist %s manquante dans le résultat", id)
		}
	}
	doubles := playlists[byID["pl-doubles"]]
	// PlacementTotal=5 + MeasurementRemaining=5 (placement synthétique S13 courant)
	if doubles.Current.PlacementTotal != 5 {
		t.Errorf("pl-doubles.Current.PlacementTotal: want 5, got %d", doubles.Current.PlacementTotal)
	}
	if doubles.Current.MeasurementMatchesRemaining != 5 {
		t.Errorf("pl-doubles.Current.MeasurementMatchesRemaining: want 5, got %d", doubles.Current.MeasurementMatchesRemaining)
	}
	if doubles.Current.Tier != "" {
		t.Errorf("pl-doubles.Current.Tier: want empty (placement), got %q", doubles.Current.Tier)
	}
	// Arena matured : threshold S13=5 exposé
	arena := playlists[byID["pl-arena"]]
	if arena.Current.PlacementTotal != 5 {
		t.Errorf("pl-arena.Current.PlacementTotal: want 5, got %d", arena.Current.PlacementTotal)
	}
	if arena.Current.Tier != "Gold" {
		t.Errorf("pl-arena.Current.Tier: want Gold, got %q", arena.Current.Tier)
	}
}

// ── Scénario 4 : Diag endpoint detect coverage gap ──────────────────────────

func TestE2EPipeline_DiagEndpoint_DetectsCoverageGap(t *testing.T) {
	env := setupPipelineEnv(t)
	// 5 matchs ranked dans match_registry pour le joueur
	for i, mid := range []string{"m1", "m2", "m3", "m4", "m5"} {
		seedMatchRegistry(t, env, mid, "pl-arena", "Ranked Arena", true, "CsrSeason13-1", "2026-04-1"+string(rune('0'+i))+"T12:00:00Z")
	}
	// MSR CSR seulement pour 2 matchs (gap = 3)
	seedMSR(t, env, "m1", "CSR", 1100, "Gold", "Or 4", 4, 0)
	seedMSR(t, env, "m2", "CSR", 0, "Placement", "Placement (3 restants)", 0, 3)

	cov, err := env.coverRepo.GetCoverage(context.Background(), pipelineTestSlug, pipelineTestXUID)
	if err != nil {
		t.Fatalf("GetCoverage: %v", err)
	}
	if cov.MatchSkillRankCSR.RankedMatchesInRegistry != 5 {
		t.Errorf("RankedMatchesInRegistry: want 5, got %d", cov.MatchSkillRankCSR.RankedMatchesInRegistry)
	}
	if cov.MatchSkillRankCSR.Total != 2 {
		t.Errorf("MatchSkillRankCSR.Total: want 2, got %d", cov.MatchSkillRankCSR.Total)
	}
	if cov.MatchSkillRankCSR.CoverageGap != 3 {
		t.Errorf("CoverageGap: want 3, got %d", cov.MatchSkillRankCSR.CoverageGap)
	}
	if !cov.NeedsBackfill {
		t.Error("NeedsBackfill: want true (gap > 0)")
	}
	if cov.MatchSkillRankCSR.Matured != 1 {
		t.Errorf("Matured: want 1, got %d", cov.MatchSkillRankCSR.Matured)
	}
	if cov.MatchSkillRankCSR.Placement != 1 {
		t.Errorf("Placement: want 1, got %d", cov.MatchSkillRankCSR.Placement)
	}
}

// ── Scénario 5 : Display threshold dynamique S2 vs S13 ──────────────────────

func TestE2EPipeline_DisplayThresholdDynamic_S2vsS13(t *testing.T) {
	env := setupPipelineEnv(t)
	// 2 matchs sur 2 playlists distinctes (LoadRecentPlaylistRanks ne retourne
	// QU'UNE row par playlist_id distinct).
	// Match 1 : "Ranked Arena (S2)" — playlist_id distinct, threshold doit être 10
	seedMatchRegistry(t, env, "m_s2", "pl-s2-arena", "Ranked Arena (S2)", true, "CsrSeason2", "2022-08-10T12:00:00Z")
	seedMSR(t, env, "m_s2", "CSR", 0, "Placement", "Placement (4 restants)", 0, 4)
	seedCSRSnapshot(t, env, "pl-s2-arena", "Ranked Arena (S2)", "CsrSeason2", 0, "", 0, 4, 0)
	// Match 2 : "Ranked Arena S13" — autre playlist_id, threshold 5
	seedMatchRegistry(t, env, "m_s13", "pl-s13-arena", "Ranked Arena S13", true, "CsrSeason13-1", "2026-04-15T12:00:00Z")
	seedMSR(t, env, "m_s13", "CSR", 0, "Placement", "Placement (2 restants)", 0, 2)
	seedCSRSnapshot(t, env, "pl-s13-arena", "Ranked Arena S13", "CsrSeason13-1", 0, "", 0, 2, 0)

	items, err := env.homeRepo.LoadRecentPlaylistRanks(context.Background(), "fr")
	if err != nil {
		t.Fatalf("LoadRecentPlaylistRanks: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	byName := map[string]int{}
	for i, it := range items {
		byName[it.PlaylistName] = i
	}
	// S2 historique → threshold 10
	s2 := items[byName["Ranked Arena (S2)"]]
	if s2.PlacementTotal == nil || *s2.PlacementTotal != 10 {
		t.Errorf("S2 PlacementTotal: want 10 (historique), got %v", s2.PlacementTotal)
	}
	// S2 completed = 10-4 = 6 → unranked_(6*10/10)=unranked_6.png
	if s2.BadgeImageURL == nil || !strings.Contains(*s2.BadgeImageURL, "unranked_6") {
		t.Errorf("S2 badge: want unranked_6.png (mapping identité threshold=10), got %v", s2.BadgeImageURL)
	}
	// S13 → threshold 5
	s13 := items[byName["Ranked Arena S13"]]
	if s13.PlacementTotal == nil || *s13.PlacementTotal != 5 {
		t.Errorf("S13 PlacementTotal: want 5, got %v", s13.PlacementTotal)
	}
	// S13 completed = 5-2 = 3 → unranked_(3*10/5)=unranked_6.png
	if s13.BadgeImageURL == nil || !strings.Contains(*s13.BadgeImageURL, "unranked_6") {
		t.Errorf("S13 badge: want unranked_6.png (mapping 3/5 → 6/10), got %v", s13.BadgeImageURL)
	}
	// Smoke check : les 2 retournent unranked_6.png mais via 2 mapping différents.
	// Garde-fou : remaining différent doit ressortir
	if s2.MeasurementMatchesRemaining == nil || *s2.MeasurementMatchesRemaining != 4 {
		t.Errorf("S2 remaining: want 4, got %v", s2.MeasurementMatchesRemaining)
	}
	if s13.MeasurementMatchesRemaining == nil || *s13.MeasurementMatchesRemaining != 2 {
		t.Errorf("S13 remaining: want 2, got %v", s13.MeasurementMatchesRemaining)
	}
}

// ── Scénario 6 : Home Recent Playlists — playlist_id NULL groupé par playlist_name ──

// TestE2EPipeline_HomeRecentPlaylists_NullPlaylistIdGroupedByName vérifie que
// Q26gPlaylistPhaseBShared retourne plusieurs playlists distinctes même quand
// playlist_id est NULL dans match_registry (cas Social + anciens matchs). Le
// groupement par COALESCE(playlist_id, playlist_name) doit retourner jusqu'à
// LIMIT 3 groupes distincts.
func TestE2EPipeline_HomeRecentPlaylists_NullPlaylistIdGroupedByName(t *testing.T) {
	env := setupPipelineEnv(t)

	// Match 1 : playlist_id NULL, playlist_name "Quick Play" (social)
	if _, err := env.sharedDB.SQLDb().Exec(`
		INSERT INTO match_registry (match_id, start_time, playlist_id, playlist_name, pair_name, is_ranked, season_id)
		VALUES (?, ?, NULL, ?, ?, ?, ?)
	`, "null_m1", "2026-04-15T12:00:00Z", "Quick Play", "Slayer on Recharge", false, "CsrSeason13-1"); err != nil {
		t.Fatalf("seed null_m1: %v", err)
	}
	if _, err := env.sharedDB.SQLDb().Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
		VALUES (?, ?, ?, 0, 2)
	`, "null_m1", pipelineTestXUID, pipelineTestSlug); err != nil {
		t.Fatalf("seed null_m1 participant: %v", err)
	}

	// Match 2 : playlist_id NULL, playlist_name "Social Slayer" (social différent)
	if _, err := env.sharedDB.SQLDb().Exec(`
		INSERT INTO match_registry (match_id, start_time, playlist_id, playlist_name, pair_name, is_ranked, season_id)
		VALUES (?, ?, NULL, ?, ?, ?, ?)
	`, "null_m2", "2026-04-14T12:00:00Z", "Social Slayer", "Slayer on Aquarius", false, "CsrSeason13-1"); err != nil {
		t.Fatalf("seed null_m2: %v", err)
	}
	if _, err := env.sharedDB.SQLDb().Exec(`
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
		VALUES (?, ?, ?, 0, 2)
	`, "null_m2", pipelineTestXUID, pipelineTestSlug); err != nil {
		t.Fatalf("seed null_m2 participant: %v", err)
	}

	// Match 3 : playlist_id renseigné, playlist ranked
	seedMatchRegistry(t, env, "null_m3", "pl-ranked", "Ranked Arena", true, "CsrSeason13-1", "2026-04-13T12:00:00Z")

	// Insérer des MSR pour chaque match (nécessaire pour Phase A1)
	seedMSR(t, env, "null_m1", "LUSR", 1400, "", "", 0, 0)
	seedMSR(t, env, "null_m2", "LUSR", 1380, "", "", 0, 0)
	seedMSR(t, env, "null_m3", "LUSR", 1500, "", "", 0, 0)

	items, err := env.homeRepo.LoadRecentPlaylistRanks(context.Background(), "fr")
	if err != nil {
		t.Fatalf("LoadRecentPlaylistRanks: %v", err)
	}
	// Avant le fix : playlist_id NULL → filtre WHERE excluait les 2 matchs sociaux
	// → 1 seul item retourné (pl-ranked seulement). Après le fix : 3 groupes.
	if len(items) != 3 {
		names := make([]string, len(items))
		for i, it := range items {
			names[i] = it.PlaylistName
		}
		t.Fatalf("want 3 playlists (2 null-playlist_id + 1 ranked), got %d : %v", len(items), names)
	}
	byName := map[string]bool{}
	for _, it := range items {
		byName[it.PlaylistName] = true
	}
	for _, want := range []string{"Quick Play", "Social Slayer", "Ranked Arena"} {
		if !byName[want] {
			t.Errorf("playlist %q manquante dans le résultat", want)
		}
	}
}

// Garde-fous imports — utilisés par les helpers seed.
var (
	_ = sql.ErrNoRows
	_ = os.Open
)
