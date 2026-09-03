//go:build integration

package duckdb

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// applyWorldLeaderboardMigration applique les migrations du classement mondial
// (création de la table + vue _latest par batch) sur une DB de test, dans l'ordre.
// Depuis Phase 1.5 b18 ces 3 steps sont title-owned → résolus via StepsFor (+ fallback
// global ForTarget pour robustesse si l'un redevenait global).
func applyWorldLeaderboardMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	wanted := []string{
		"create_world_csr_leaderboard_snapshots",
		"world_csr_leaderboard_latest_by_batch",
		"add_title_slug_to_world_csr_leaderboard",
		"add_xuid_to_world_csr_leaderboard",
		"create_world_player_season_stats",
		"create_world_player_no_data",
	}
	byName := map[string]migration.Migration{}
	for _, m := range migration.ForTarget(migration.TargetShared) {
		byName[m.Name] = m
	}
	for _, m := range halomigrations.StepsFor(migration.TargetShared) {
		byName[m.Name] = m
	}
	for _, name := range wanted {
		m, ok := byName[name]
		if !ok {
			t.Fatalf("migration %s introuvable", name)
		}
		if err := m.ApplySchema(db); err != nil {
			t.Fatalf("ApplySchema(%s): %v", name, err)
		}
	}
}

// TestGetCSRWorldLeaderboard_PrivateMasking valide le masquage des joueurs privés
// (world_player_no_data) à l'affichage ET la garde anti-classement-vide : une saison
// SANS aucun enrichi (historique expiré) n'est PAS masquée.
func TestGetCSRWorldLeaderboard_PrivateMasking(t *testing.T) {
	shared := openMemDB(t)
	applyWorldLeaderboardMigration(t, shared.SQLDb())
	ctx := context.Background()

	if _, err := shared.SQLDb().ExecContext(ctx, `
		INSERT INTO world_csr_leaderboard_snapshots
			(season_id, playlist_id, rank, gamertag, csr_value, title_slug, fetched_at)
		VALUES
			('s1','pl1',1,'P1',2000,'halo_infinite',TIMESTAMP '2026-01-01 00:00:00'),
			('s1','pl1',2,'P2',1900,'halo_infinite',TIMESTAMP '2026-01-01 00:00:00'),
			('s1','pl1',3,'P3',1800,'halo_infinite',TIMESTAMP '2026-01-01 00:00:00'),
			('s2','pl1',1,'Q1',2000,'halo_infinite',TIMESTAMP '2026-01-01 00:00:00'),
			('s2','pl1',2,'Q2',1900,'halo_infinite',TIMESTAMP '2026-01-01 00:00:00');
	`); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}
	repo := NewLeaderboardRepo(&PlayerDB{Shared: shared})

	// s1 : au moins un enrichi (P1) → masquage ACTIF ; P2 marqué privé → doit disparaître.
	if _, err := InsertPlayerSeasonStats(ctx, shared.SQLDb(), []domain.WorldPlayerSeasonStats{
		{TitleSlug: "halo_infinite", SeasonID: "s1", PlaylistID: "pl1", Gamertag: "P1", MatchCount: 5},
	}); err != nil {
		t.Fatalf("insert enrich: %v", err)
	}
	if _, err := InsertWorldNoDataPlayers(ctx, shared.SQLDb(), "halo_infinite", "s1", []string{"P2"}); err != nil {
		t.Fatalf("mark private: %v", err)
	}
	got, err := repo.GetCSRWorldLeaderboard(ctx, "halo_infinite", "s1", "pl1", 10)
	if err != nil {
		t.Fatalf("s1: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("s1 → %d entrées, want 2 (P2 privé masqué)", len(got))
	}
	for _, e := range got {
		if e.Gamertag == "P2" {
			t.Errorf("P2 (privé) ne doit pas apparaître dans le classement")
		}
	}

	// s2 : AUCUN enrichi (saison expirée) ; Q1 marqué privé → garde : PAS de masquage.
	if _, err := InsertWorldNoDataPlayers(ctx, shared.SQLDb(), "halo_infinite", "s2", []string{"Q1"}); err != nil {
		t.Fatalf("mark private s2: %v", err)
	}
	got2, err := repo.GetCSRWorldLeaderboard(ctx, "halo_infinite", "s2", "pl1", 10)
	if err != nil {
		t.Fatalf("s2: %v", err)
	}
	if len(got2) != 2 {
		t.Errorf("s2 → %d entrées, want 2 (saison expirée : pas de masquage malgré Q1 privé)", len(got2))
	}
}

// TestInsertWorldCSRSnapshot_AppendOnlyAndLatestView valide que l'insertion est
// append-only et que la vue _latest retourne le dernier snapshot par rang.
func TestInsertWorldCSRSnapshot_AppendOnlyAndLatestView(t *testing.T) {
	db := openMemDB(t).SQLDb()
	applyWorldLeaderboardMigration(t, db)
	ctx := context.Background()

	t0 := time.Date(2026, 6, 8, 10, 0, 0, 0, time.UTC)
	batch1 := []domain.LeaderboardEntry{
		{Season: "csrseason13-2", Playlist: "p", Rank: 1, Gamertag: "Twissted Mindss", CSRValue: 2180, Tier: "Onyx", FetchedAt: t0},
		{Season: "csrseason13-2", Playlist: "p", Rank: 2, Gamertag: "OR81TAL", CSRValue: 2097, Tier: "Onyx", FetchedAt: t0},
	}
	n, err := InsertWorldCSRSnapshot(ctx, db, "halo_infinite", batch1)
	if err != nil {
		t.Fatalf("InsertWorldCSRSnapshot batch1: %v", err)
	}
	if n != 2 {
		t.Fatalf("inserted = %d, want 2", n)
	}

	// Vue _latest : 2 lignes, rang 1 = Twissted Mindss.
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM world_csr_leaderboard_latest WHERE season_id='csrseason13-2' AND playlist_id='p'`,
	).Scan(&count); err != nil {
		t.Fatalf("count latest: %v", err)
	}
	if count != 2 {
		t.Fatalf("latest count = %d, want 2", count)
	}

	// Nouveau snapshot plus récent : le rang 1 change d'occupant.
	t1 := t0.Add(24 * time.Hour)
	batch2 := []domain.LeaderboardEntry{
		{Season: "csrseason13-2", Playlist: "p", Rank: 1, Gamertag: "NewKing", CSRValue: 2222, Tier: "Onyx", FetchedAt: t1},
	}
	if _, err := InsertWorldCSRSnapshot(ctx, db, "halo_infinite", batch2); err != nil {
		t.Fatalf("InsertWorldCSRSnapshot batch2: %v", err)
	}

	// Append-only : la table garde les 3 lignes (2 + 1).
	var raw int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM world_csr_leaderboard_snapshots`).Scan(&raw); err != nil {
		t.Fatalf("count raw: %v", err)
	}
	if raw != 3 {
		t.Fatalf("raw rows = %d, want 3 (append-only)", raw)
	}

	// La vue _latest retourne le nouvel occupant du rang 1.
	var gt string
	var csr int
	if err := db.QueryRowContext(ctx,
		`SELECT gamertag, csr_value FROM world_csr_leaderboard_latest WHERE season_id='csrseason13-2' AND playlist_id='p' AND rank=1`,
	).Scan(&gt, &csr); err != nil {
		t.Fatalf("query rank 1: %v", err)
	}
	if gt != "NewKing" || csr != 2222 {
		t.Fatalf("rank 1 latest = (%q, %d), want (NewKing, 2222)", gt, csr)
	}

	// Fix Frankenstein : la vue groupe par batch (fetched_at), pas par rang. Le
	// batch2 (1 ligne, t1) REMPLACE entièrement batch1 pour (csrseason13-2, p) :
	// la queue de batch1 (rang 2) ne doit PAS subsister dans _latest.
	var latestCount int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM world_csr_leaderboard_latest WHERE season_id='csrseason13-2' AND playlist_id='p'`,
	).Scan(&latestCount); err != nil {
		t.Fatalf("count latest après batch2: %v", err)
	}
	if latestCount != 1 {
		t.Fatalf("latest count après batch2 = %d, want 1 (batch remplacé, pas fusionné)", latestCount)
	}
}

// TestWorldCSRServedBatchStats valide la lecture de qualité du lot SERVI (garde-fou
// D1) : comptage des lignes et des xuid sur le DERNIER batch seulement, absence de
// lot servi signalée sans erreur, et isolation par (titre, saison, playlist).
func TestWorldCSRServedBatchStats(t *testing.T) {
	db := openMemDB(t).SQLDb()
	applyWorldLeaderboardMigration(t, db)
	ctx := context.Background()

	// Aucun lot servi → (0,0,false) SANS erreur : première capture, rien à protéger.
	stats, ok, err := WorldCSRServedBatchStats(ctx, db, "halo_infinite", "csrseason13-3", "pl-a")
	if err != nil {
		t.Fatalf("lot absent: %v", err)
	}
	if ok || stats.Rows != 0 || stats.WithXUID != 0 {
		t.Errorf("lot absent → (%+v, ok=%v), attendu ({0 0}, false)", stats, ok)
	}

	t0 := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	// Lot sain servi : 3 lignes, 3 xuid. Une autre playlist en parallèle (isolation).
	if _, err := InsertWorldCSRSnapshot(ctx, db, "halo_infinite", []domain.LeaderboardEntry{
		{Season: "csrseason13-3", Playlist: "pl-a", Rank: 1, Gamertag: "A", XUID: "x1", CSRValue: 2000, FetchedAt: t0},
		{Season: "csrseason13-3", Playlist: "pl-a", Rank: 2, Gamertag: "B", XUID: "x2", CSRValue: 1900, FetchedAt: t0},
		{Season: "csrseason13-3", Playlist: "pl-a", Rank: 3, Gamertag: "C", XUID: "x3", CSRValue: 1800, FetchedAt: t0},
		{Season: "csrseason13-3", Playlist: "pl-b", Rank: 1, Gamertag: "Z", XUID: "x9", CSRValue: 1700, FetchedAt: t0},
	}); err != nil {
		t.Fatalf("insert lot sain: %v", err)
	}
	stats, ok, err = WorldCSRServedBatchStats(ctx, db, "halo_infinite", "csrseason13-3", "pl-a")
	if err != nil || !ok {
		t.Fatalf("lot sain: stats=%+v ok=%v err=%v", stats, ok, err)
	}
	if stats.Rows != 3 || stats.WithXUID != 3 {
		t.Errorf("lot sain = %+v, attendu {Rows:3 WithXUID:3}", stats)
	}
	if cov := stats.XUIDCoverage(); cov != 1 {
		t.Errorf("couverture xuid = %v, attendu 1", cov)
	}

	// Lot plus récent SANS xuid : la vue sert le dernier batch → les stats doivent
	// décrire CE lot (2 lignes, 0 xuid), pas l'ancien ni la somme des deux.
	if _, err := InsertWorldCSRSnapshot(ctx, db, "halo_infinite", []domain.LeaderboardEntry{
		{Season: "csrseason13-3", Playlist: "pl-a", Rank: 1, Gamertag: "A", CSRValue: 2000, FetchedAt: t0.Add(24 * time.Hour)},
		{Season: "csrseason13-3", Playlist: "pl-a", Rank: 2, Gamertag: "B", CSRValue: 1900, FetchedAt: t0.Add(24 * time.Hour)},
	}); err != nil {
		t.Fatalf("insert lot dégradé: %v", err)
	}
	stats, ok, err = WorldCSRServedBatchStats(ctx, db, "halo_infinite", "csrseason13-3", "pl-a")
	if err != nil || !ok {
		t.Fatalf("lot dégradé: stats=%+v ok=%v err=%v", stats, ok, err)
	}
	if stats.Rows != 2 || stats.WithXUID != 0 {
		t.Errorf("lot servi = %+v, attendu {Rows:2 WithXUID:0} (dernier batch seulement)", stats)
	}
	if cov := stats.XUIDCoverage(); cov != 0 {
		t.Errorf("couverture xuid = %v, attendu 0", cov)
	}

	// Isolation : la playlist voisine et un autre titre ne sont pas affectés.
	if other, ok2, err2 := WorldCSRServedBatchStats(ctx, db, "halo_infinite", "csrseason13-3", "pl-b"); err2 != nil || !ok2 || other.Rows != 1 {
		t.Errorf("pl-b = %+v ok=%v err=%v, attendu {Rows:1 WithXUID:1}", other, ok2, err2)
	}
	if _, ok3, err3 := WorldCSRServedBatchStats(ctx, db, "autre_titre", "csrseason13-3", "pl-a"); err3 != nil || ok3 {
		t.Errorf("autre titre → ok=%v err=%v, attendu false/nil (isolation par titre)", ok3, err3)
	}
}

// TestGetWorldLeaderboardCatalog valide la remontée des saisons/playlists
// distinctes présentes en base + la résolution du libellé de playlist.
func TestGetWorldLeaderboardCatalog(t *testing.T) {
	shared := openMemDB(t)
	applyWorldLeaderboardMigration(t, shared.SQLDb())
	ctx := context.Background()

	arenaID := "edfef3ac-9cbe-4fa2-b949-8f29deafd483" // Ranked Arena (NameFR "Arène classée")

	// Leaderboard scrappé : 4 saisons (dont csrseason6-1 à un chiffre pour valider le tri
	// NUMÉRIQUE, et csrseason4-1 archivée), 2 playlists.
	lbRows := []domain.LeaderboardEntry{
		{Season: "csrseason13-2", Playlist: arenaID, Rank: 1, Gamertag: "A", CSRValue: 2000, Tier: "Onyx", FetchedAt: time.Now().UTC()},
		{Season: "csrseason12-1", Playlist: arenaID, Rank: 1, Gamertag: "B", CSRValue: 1900, Tier: "Onyx", FetchedAt: time.Now().UTC()},
		{Season: "csrseason6-1", Playlist: arenaID, Rank: 1, Gamertag: "C", CSRValue: 1800, Tier: "Diamond", FetchedAt: time.Now().UTC()},
		{Season: "csrseason6-1", Playlist: "unknown-pl", Rank: 1, Gamertag: "D", CSRValue: 1750, Tier: "Diamond", FetchedAt: time.Now().UTC()},
		{Season: "csrseason4-1", Playlist: arenaID, Rank: 1, Gamertag: "Z", CSRValue: 1700, Tier: "Diamond", FetchedAt: time.Now().UTC()},
	}
	if _, err := InsertWorldCSRSnapshot(ctx, shared.SQLDb(), "halo_infinite", lbRows); err != nil {
		t.Fatalf("InsertWorldCSRSnapshot: %v", err)
	}

	// Stats enrichies pour 3 saisons ; PAS csrseason4-1 (archivée → restera Enriched=false).
	statRows := []domain.WorldPlayerSeasonStats{
		{TitleSlug: "halo_infinite", Gamertag: "A", SeasonID: "csrseason13-2", PlaylistID: arenaID, MatchCount: 10, WinCount: 6},
		{TitleSlug: "halo_infinite", Gamertag: "B", SeasonID: "csrseason12-1", PlaylistID: arenaID, MatchCount: 8, WinCount: 4},
		{TitleSlug: "halo_infinite", Gamertag: "C", SeasonID: "csrseason6-1", PlaylistID: arenaID, MatchCount: 5, WinCount: 2},
	}
	if _, err := InsertPlayerSeasonStats(ctx, shared.SQLDb(), statRows); err != nil {
		t.Fatalf("InsertPlayerSeasonStats: %v", err)
	}

	repo := NewLeaderboardRepo(&PlayerDB{Shared: shared})
	cat, err := repo.GetWorldLeaderboardCatalog(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("GetWorldLeaderboardCatalog: %v", err)
	}

	// Option C : TOUTES les saisons du leaderboard sont exposées, triées NUMÉRIQUEMENT
	// récent-d'abord (6-1 en DERNIER, pas en tête comme un tri lexicographique) ; chaque
	// saison porte Enriched (csrseason4-1 sans stats → false, affichée en classement seul).
	want := []struct {
		id       string
		enriched bool
	}{
		{"csrseason13-2", true}, {"csrseason12-1", true}, {"csrseason6-1", true}, {"csrseason4-1", false},
	}
	if len(cat.Seasons) != len(want) {
		t.Fatalf("seasons = %+v, attendu %d (toutes les saisons du leaderboard)", cat.Seasons, len(want))
	}
	for i, w := range want {
		if cat.Seasons[i].ID != w.id || cat.Seasons[i].Enriched != w.enriched {
			t.Errorf("season[%d] = {%s, enriched=%v}, attendu {%s, enriched=%v}",
				i, cat.Seasons[i].ID, cat.Seasons[i].Enriched, w.id, w.enriched)
		}
	}
	// Playlists : 2 distinctes (depuis le leaderboard) ; la playlist connue a son libellé FR.
	if len(cat.Playlists) != 2 {
		t.Fatalf("playlists = %d, attendu 2 (%+v)", len(cat.Playlists), cat.Playlists)
	}
	var arenaLabel, unknownLabel string
	for _, p := range cat.Playlists {
		switch p.ID {
		case arenaID:
			arenaLabel = p.DisplayName
		case "unknown-pl":
			unknownLabel = p.DisplayName
		}
	}
	if arenaLabel != "Arène classée" {
		t.Errorf("libellé Arena = %q, attendu \"Arène classée\"", arenaLabel)
	}
	if unknownLabel != "unknown-pl" {
		t.Errorf("libellé inconnu = %q, attendu fallback sur l'asset_id", unknownLabel)
	}
}

// TestGetWorldLeaderboardCatalog_PlaylistIDsPerSeason (Lot 4.2) : chaque saison
// porte les playlists RÉELLEMENT relevées pour elle. Sans ça, le front croise les
// deux listes plates et propose des couples jamais capturés (tableau vide).
func TestGetWorldLeaderboardCatalog_PlaylistIDsPerSeason(t *testing.T) {
	shared := openMemDB(t)
	applyWorldLeaderboardMigration(t, shared.SQLDb())
	ctx := context.Background()

	now := time.Now().UTC()
	// 13-3 relevée sur 2 playlists, 13-2 sur une seule : le couple (13-2, pl-b)
	// n'existe pas et ne doit apparaître nulle part.
	rows := []domain.LeaderboardEntry{
		{Season: "csrseason13-3", Playlist: "pl-a", Rank: 1, Gamertag: "A", CSRValue: 2000, Tier: "Onyx", FetchedAt: now},
		{Season: "csrseason13-3", Playlist: "pl-b", Rank: 1, Gamertag: "B", CSRValue: 1900, Tier: "Onyx", FetchedAt: now},
		{Season: "csrseason13-2", Playlist: "pl-a", Rank: 1, Gamertag: "C", CSRValue: 1800, Tier: "Diamond", FetchedAt: now},
	}
	if _, err := InsertWorldCSRSnapshot(ctx, shared.SQLDb(), "halo_infinite", rows); err != nil {
		t.Fatalf("InsertWorldCSRSnapshot: %v", err)
	}

	cat, err := NewLeaderboardRepo(&PlayerDB{Shared: shared}).GetWorldLeaderboardCatalog(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("GetWorldLeaderboardCatalog: %v", err)
	}
	got := map[string][]string{}
	for _, s := range cat.Seasons {
		got[s.ID] = s.PlaylistIDs
	}
	if want := []string{"pl-a", "pl-b"}; !slices.Equal(got["csrseason13-3"], want) {
		t.Errorf("13-3 playlist_ids = %v, attendu %v", got["csrseason13-3"], want)
	}
	if want := []string{"pl-a"}; !slices.Equal(got["csrseason13-2"], want) {
		t.Errorf("13-2 playlist_ids = %v, attendu %v (le couple (13-2, pl-b) n'existe pas)", got["csrseason13-2"], want)
	}
	// Les listes plates restent inchangées (compat) : 2 saisons, 2 playlists.
	if len(cat.Seasons) != 2 || len(cat.Playlists) != 2 {
		t.Errorf("listes plates = %d saisons / %d playlists, attendu 2 / 2", len(cat.Seasons), len(cat.Playlists))
	}
	// Les playlists (liste plate) ne portent JAMAIS de couples.
	for _, p := range cat.Playlists {
		if len(p.PlaylistIDs) != 0 {
			t.Errorf("playlist %s porte playlist_ids=%v, attendu vide", p.ID, p.PlaylistIDs)
		}
	}
}
