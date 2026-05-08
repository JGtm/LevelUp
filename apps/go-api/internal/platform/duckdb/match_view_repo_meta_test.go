// Package duckdb — match_view_repo_meta_test.go : tests pour la résolution
// unifiée des noms d'asset dans GetMatchMeta (map, pair, playlist).
//
// Ces tests reproduisent le scénario empirique observé en prod 2026-05-08 :
// match_registry.map_name et pair_name contiennent des UUIDs bruts (l'API
// Halo n'a pas résolu les noms), mais asset_translations a les entrées FR.
// Le resolver unifié doit retourner les noms propres pour la match-view.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"levelup/go-api/internal/analysis"
	titlepkg "levelup/go-api/internal/domain/title"
)

// newMetaResolveTestPDB construit un PlayerDB minimal (Player + Metadata)
// pour exercer GetMatchMeta. Player héberge le schéma `shared` (où Q13
// requête `shared.match_registry`) ; Metadata contient asset_translations
// et mode_name_tr.
func newMetaResolveTestPDB(t *testing.T) *PlayerDB {
	t.Helper()
	playerSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	t.Cleanup(func() { playerSQL.Close() })
	metaSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open meta: %v", err)
	}
	t.Cleanup(func() { metaSQL.Close() })
	player := &DB{sqlDB: playerSQL, path: ":memory:"}
	meta := &DB{sqlDB: metaSQL, path: ":memory:"}

	ctx := context.Background()
	for _, q := range []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE shared.match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP, end_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ, end_time_utc TIMESTAMPTZ,
			playlist_id VARCHAR,
			map_id VARCHAR,
			pair_id VARCHAR,
			game_variant_id VARCHAR,
			map_name VARCHAR, map_name_fr VARCHAR,
			game_variant_name VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR,
			playlist_name VARCHAR, playlist_name_fr VARCHAR,
			is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER, playable_duration_seconds INTEGER,
			team_0_score INTEGER, team_1_score INTEGER)`,
	} {
		if _, err := player.Exec(ctx, q); err != nil {
			t.Fatalf("seed player schema: %v\nSQL: %s", err, q)
		}
	}
	for _, q := range []string{
		`CREATE TABLE asset_translations (
			asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR,
			name VARCHAR, description VARCHAR DEFAULT '', fetched_at TIMESTAMPTZ DEFAULT now(),
			PRIMARY KEY (asset_id, asset_type, lang))`,
		`CREATE TABLE mode_name_tr (lang VARCHAR, mode_en VARCHAR, name VARCHAR)`,
		`CREATE TABLE map_images_registry (title_id VARCHAR, map_id VARCHAR, local_path VARCHAR,
			fetched_at TIMESTAMPTZ DEFAULT now(),
			PRIMARY KEY (title_id, map_id))`,
	} {
		if _, err := meta.Exec(ctx, q); err != nil {
			t.Fatalf("seed meta schema: %v\nSQL: %s", err, q)
		}
	}

	return &PlayerDB{
		Player:    player,
		Metadata:  meta,
		XUID:      "test-xuid",
		Gamertag:  "test-gt",
		TitleSlug: titlepkg.DefaultSlug,
	}
}

// TestGetMatchMeta_ResolvesUUIDPairAndMapViaAssetTranslations reproduit
// EXACTEMENT le scénario de la capture d'écran 2026-05-08 :
//
//	match_registry :
//	  map_id    = "shiro-uuid"  map_name    = "shiro-uuid"  (UUID brut)
//	  pair_id   = "qp-slayer"   pair_name   = "qp-slayer"   (UUID brut)
//	  *_name_fr = NULL
//
//	asset_translations :
//	  ("shiro-uuid", "map", "fr-FR", "Shiro")
//	  ("qp-slayer", "pair", "fr-FR", "Partie rapide : Assassin")
//
// Attendu : MapNameFR="Shiro", ModeNameFR contient le nom FR du mode (et NON
// l'UUID brut). Le bug pré-fix aurait laissé ModeNameFR=nil → service
// affichait "qp-slayer" comme on le voyait à l'écran.
func TestGetMatchMeta_ResolvesUUIDPairAndMapViaAssetTranslations(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc,
			 map_id, pair_id, playlist_id,
			 map_name, pair_name, playlist_name)
		VALUES ('m1', '2026-05-05 23:16:00', '2026-05-05 23:16:00+00',
		        'shiro-uuid', 'qp-slayer', 'pl-quickplay',
		        'shiro-uuid', 'qp-slayer', 'qp-quickplay')`); err != nil {
		t.Fatalf("insert match: %v", err)
	}
	for _, q := range []string{
		`INSERT INTO asset_translations VALUES ('shiro-uuid','map','fr-FR','Shiro','',now())`,
		`INSERT INTO asset_translations VALUES ('qp-slayer','pair','fr-FR','Partie rapide : Assassin','',now())`,
		`INSERT INTO asset_translations VALUES ('pl-quickplay','playlist','fr-FR','Partie rapide','',now())`,
	} {
		if _, err := pdb.Metadata.Exec(ctx, q); err != nil {
			t.Fatalf("seed translations: %v\nSQL: %s", err, q)
		}
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	meta, err := repo.GetMatchMeta(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	if meta.MapNameFR == nil || *meta.MapNameFR != "Shiro" {
		t.Errorf("MapNameFR = %v, want Shiro", meta.MapNameFR)
	}
	if meta.PlaylistNameFR == nil || *meta.PlaylistNameFR != "Partie rapide" {
		t.Errorf("PlaylistNameFR = %v, want Partie rapide", meta.PlaylistNameFR)
	}
	// Pour le mode, le pair_name FR est "Partie rapide : Assassin". La cascade
	// resolveModeNameFR essaie d'abord NormalizeModeLabel(pair_name brut="qp-slayer")
	// → fail, puis asset_translations[pair_id, FR] → "Partie rapide : Assassin"
	// → NormalizeModeLabel → "Partie rapide" → mode_name_tr (vide) → fallback
	// retour direct = "Partie rapide : Assassin".
	if meta.ModeNameFR == nil {
		t.Fatal("ModeNameFR = nil, want non-nil (cascade asset_translations[pair])")
	}
	if *meta.ModeNameFR == "qp-slayer" {
		t.Errorf("ModeNameFR = UUID brut %q : la résolution n'a PAS marché", *meta.ModeNameFR)
	}
}

// TestGetMatchMeta_FallsBackToENWhenNoFR vérifie la cascade FR→EN. Quand
// asset_translations n'a que en-US (cas typique d'un asset jamais traduit
// par populate-assets), le resolver doit retourner le nom EN plutôt que
// l'UUID brut.
func TestGetMatchMeta_FallsBackToENWhenNoFR(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, map_id, map_name, pair_name, playlist_name)
		VALUES ('m2', '2026-05-05 22:00:00', '2026-05-05 22:00:00+00',
		        'forbidden-uuid', 'forbidden-uuid', 'qp-slayer', 'pl-quickplay')`); err != nil {
		t.Fatalf("insert match: %v", err)
	}
	if _, err := pdb.Metadata.Exec(ctx,
		`INSERT INTO asset_translations VALUES ('forbidden-uuid','map','en-US','Forbidden','',now())`,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	meta, err := repo.GetMatchMeta(ctx, "m2")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	if meta.MapNameFR == nil || *meta.MapNameFR != "Forbidden" {
		t.Errorf("MapNameFR (cascade EN) = %v, want Forbidden", meta.MapNameFR)
	}
}

// TestGetMatchMeta_NoTranslation_ReturnsNil vérifie que sans entrée
// asset_translations, le resolver renvoie nil sans erreur (le service décidera
// du fallback final côté UI).
func TestGetMatchMeta_NoTranslation_ReturnsNil(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, map_id, map_name, pair_name)
		VALUES ('m3', '2026-05-05 21:00:00', '2026-05-05 21:00:00+00',
		        'unknown-uuid', 'unknown-uuid', 'unknown-pair-uuid')`); err != nil {
		t.Fatalf("insert match: %v", err)
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	meta, err := repo.GetMatchMeta(ctx, "m3")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	if meta.MapNameFR != nil {
		t.Errorf("MapNameFR = %v, want nil (asset inconnu)", meta.MapNameFR)
	}
	if meta.ModeNameFR != nil {
		t.Errorf("ModeNameFR = %v, want nil (asset inconnu)", meta.ModeNameFR)
	}
}

// TestGetMatchEvents_ResolvesGamertagViaView reproduit le bug "Faits marquants"
// 2026-05-08 : highlight_events stockait juste le xuid, le frontend affichait
// "Premier sang 2535472884034919 · 0:43" au lieu de "Premier sang JGtm · 0:43".
//
// Q21 JOIN désormais sur v_gamertag_lookup. Le test crée la vue avec son
// rendu réel (noms officiels bots + COALESCE alias/participants) et
// vérifie 3 cas :
//  1. xuid réel dans xuid_aliases → gamertag = "JGtm"
//  2. bot bid(7.0) → gamertag = "343 PardonMy" (nom officiel)
//  3. xuid orphelin (pas dans la vue) → gamertag = nil (caller fallback xuid)
func TestGetMatchEvents_ResolvesGamertagViaView(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	// Seed shared : tables nécessaires + vue v_gamertag_lookup avec la même
	// logique que la migration prod (noms officiels bots + cascade aliases/participants).
	xuidExpr := "COALESCE(xa.xuid, mp.xuid)"
	viewSQL := fmt.Sprintf(`CREATE OR REPLACE VIEW shared.v_gamertag_lookup AS
			SELECT
				%s AS xuid,
				CASE
					WHEN %s LIKE 'bid(%%'
						THEN %s
					WHEN xa.gamertag IS NOT NULL AND xa.gamertag != ''
						THEN xa.gamertag
					WHEN mp.gamertag IS NOT NULL AND mp.gamertag != ''
						THEN mp.gamertag
					ELSE %s
				END AS gamertag
			FROM shared.xuid_aliases xa
			FULL OUTER JOIN (
				SELECT xuid, MAX(gamertag) AS gamertag FROM shared.match_participants GROUP BY xuid
			) mp ON xa.xuid = mp.xuid`,
		xuidExpr, xuidExpr, analysis.BotSQLCase(xuidExpr), xuidExpr,
	)
	for _, q := range []string{
		`CREATE TABLE shared.highlight_events (
			match_id VARCHAR, event_type VARCHAR, time_ms BIGINT, xuid VARCHAR, type_hint VARCHAR)`,
		`CREATE TABLE shared.match_participants (
			match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE shared.xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`,
		viewSQL,
	} {
		if _, err := pdb.Player.Exec(ctx, q); err != nil {
			t.Fatalf("seed shared: %v\nSQL: %s", err, q)
		}
	}
	for _, q := range []string{
		`INSERT INTO shared.xuid_aliases VALUES ('2535472884034919', 'JGtm')`,
		`INSERT INTO shared.match_participants VALUES ('m1', 'bid(7.0)', NULL)`,
		`INSERT INTO shared.highlight_events VALUES ('m1', 'first_blood', 43000, '2535472884034919', 'kill')`,
		`INSERT INTO shared.highlight_events VALUES ('m1', 'killing_spree', 60000, 'bid(7.0)', 'spree')`,
		`INSERT INTO shared.highlight_events VALUES ('m1', 'kill', 90000, '9999999999999999', 'kill')`, // orphelin
	} {
		if _, err := pdb.Player.Exec(ctx, q); err != nil {
			t.Fatalf("seed events: %v\nSQL: %s", err, q)
		}
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	events, err := repo.GetMatchEvents(ctx, "m1")
	if err != nil {
		t.Fatalf("GetMatchEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	// Vérifier la résolution dans l'ordre time_ms ASC
	if events[0].Gamertag == nil || *events[0].Gamertag != "JGtm" {
		t.Errorf("events[0].Gamertag = %v, want JGtm (xuid_aliases)", events[0].Gamertag)
	}
	if events[1].Gamertag == nil || *events[1].Gamertag != "343 PardonMy" {
		t.Errorf("events[1].Gamertag = %v, want '343 PardonMy' (bid(7.0) → nom officiel)", events[1].Gamertag)
	}
	// Orphelin : LEFT JOIN renvoie NULL — le service décidera du fallback (xuid brut)
	if events[2].Gamertag != nil {
		t.Errorf("events[2].Gamertag (orphelin) = %v, want nil (caller fallback xuid)", events[2].Gamertag)
	}
}

// TestGetMatchMeta_NormalizedPairName_ResolvesViaModeNameTr couvre le cas
// nominal pré-régression : pair_name est une chaîne formatée "Arena:Slayer"
// (pas un UUID), NormalizeModeLabel extrait "Slayer", mode_name_tr le mappe
// vers "Assassin" en FR.
func TestGetMatchMeta_NormalizedPairName_ResolvesViaModeNameTr(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, pair_name)
		VALUES ('m4', '2026-04-01 22:00:00', '2026-04-01 22:00:00+00', 'Arena:Slayer')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := pdb.Metadata.Exec(ctx,
		`INSERT INTO mode_name_tr VALUES ('fr','Slayer','Assassin')`,
	); err != nil {
		t.Fatalf("seed mode_name_tr: %v", err)
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	meta, err := repo.GetMatchMeta(ctx, "m4")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	if meta.ModeNameFR == nil || *meta.ModeNameFR != "Assassin" {
		t.Errorf("ModeNameFR = %v, want Assassin (cascade nominal)", meta.ModeNameFR)
	}
}
