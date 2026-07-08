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
	player := newTestDB(playerSQL, ":memory:")
	meta := newTestDB(metaSQL, ":memory:")

	ctx := context.Background()
	for _, q := range []string{
		`CREATE SCHEMA IF NOT EXISTS shared`,
		`CREATE TABLE shared.match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMP, end_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ, end_time_utc TIMESTAMPTZ,
			real_start_time TIMESTAMP,
			playlist_id VARCHAR,
			map_id VARCHAR, map_version_id VARCHAR,
			pair_id VARCHAR,
			game_variant_id VARCHAR,
			map_name VARCHAR, map_name_fr VARCHAR,
			game_variant_name VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR,
			playlist_name VARCHAR, playlist_name_fr VARCHAR,
			is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER, playable_duration_seconds INTEGER,
			team_0_score INTEGER, team_1_score INTEGER)`,
		// Vue root-level : les queries SharedReader migrées P5/P7 lisent
		// `FROM match_registry` (sans préfixe), car la conn cible
		// directement le catalogue shared_matches_v2 en prod. Le schéma
		// `shared` est conservé pour les inserts de test legacy.
		`CREATE VIEW match_registry AS SELECT * FROM shared.match_registry`,
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

	// SharedReader pointe vers `player` (qui contient le faux schéma `shared`
	// créé via le seed local). Cohérent avec le pattern du helper
	// newTestPlayerDB. La vraie topologie prod est couverte par les tests
	// sentinel.
	return &PlayerDB{
		Player:       player,
		Metadata:     meta,
		SharedReader: LegacySharedReader(player),
		XUID:         "test-xuid",
		Gamertag:     "test-gt",
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

// TestGetMatchMeta_ResolvesUUIDMapAndPlaylistViaAssetTranslations couvre la
// résolution map+playlist via asset_translations quand match_registry stocke
// des UUIDs bruts. La résolution du mode passe désormais par
// analysis.ResolveModeUI (formule unifiée avec la home), ce test ne porte
// donc que sur map et playlist.
func TestGetMatchMeta_ResolvesUUIDMapAndPlaylistViaAssetTranslations(t *testing.T) {
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
}

// TestGetMatchMeta_T0Ms vérifie le calcul de l'offset T0 (Match Timeline T0,
// Phase 3) dans Q13 : real_start_time renseigné → T0Ms = écart ms avec
// start_time_utc ; real_start_time absent → T0Ms nil (fallback runtime T0=0).
func TestGetMatchMeta_T0Ms(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	// m_t0 : countdown de 28s ; m_no_t0 : pas de real_start_time.
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, real_start_time, pair_name)
		VALUES ('m_t0', '2026-05-05 23:16:00', '2026-05-05 23:16:00+00',
		        '2026-05-05 23:16:28', 'qp-slayer')`); err != nil {
		t.Fatalf("insert m_t0: %v", err)
	}
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, pair_name)
		VALUES ('m_no_t0', '2026-05-05 23:16:00', '2026-05-05 23:16:00+00', 'qp-slayer')`); err != nil {
		t.Fatalf("insert m_no_t0: %v", err)
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")

	metaT0, err := repo.GetMatchMeta(ctx, "m_t0")
	if err != nil {
		t.Fatalf("GetMatchMeta(m_t0): %v", err)
	}
	if metaT0.T0Ms == nil {
		t.Fatalf("m_t0: T0Ms should be non-nil")
	}
	if *metaT0.T0Ms != 28000 {
		t.Errorf("m_t0: T0Ms want 28000ms, got %d", *metaT0.T0Ms)
	}

	metaNo, err := repo.GetMatchMeta(ctx, "m_no_t0")
	if err != nil {
		t.Fatalf("GetMatchMeta(m_no_t0): %v", err)
	}
	if metaNo.T0Ms != nil {
		t.Errorf("m_no_t0: T0Ms want nil, got %d", *metaNo.T0Ms)
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

// TestGetMatchMeta_NoTranslation : sans entrée asset_translations, MapNameFR
// reste nil (la cascade map est conservée). Pour le mode, le contrat est
// désormais le même que la home : ResolveModeUI applique NormalizeModeLabel
// sur pair_name brut — un UUID inconnu remonte tel quel jusqu'à l'UI, ce qui
// est le comportement attendu (cohérent home/match-view).
func TestGetMatchMeta_NoTranslation(t *testing.T) {
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
	if meta.ModeNameFR == nil || *meta.ModeNameFR != "unknown-pair-uuid" {
		t.Errorf("ModeNameFR = %v, want \"unknown-pair-uuid\" (NormalizeModeLabel sur brut)", meta.ModeNameFR)
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
		// Vues root-level : Q21 (GetMatchEvents) tourne désormais via SharedReader
		// sans préfixe `shared.` (ADR 0016). Doublons les vues côté racine.
		`CREATE VIEW highlight_events AS SELECT * FROM shared.highlight_events`,
		`CREATE VIEW v_gamertag_lookup AS SELECT * FROM shared.v_gamertag_lookup`,
	} {
		if _, err := pdb.Player.Exec(ctx, q); err != nil {
			t.Fatalf("seed shared: %v\nSQL: %s", err, q)
		}
	}
	for _, q := range []string{
		`INSERT INTO shared.xuid_aliases (xuid, gamertag) VALUES ('2535472884034919', 'JGtm')`,
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

// TestGetMatchMeta_NormalizedPairName_ExtractsSubmode : sans traduction dans
// mode_name_tr, ModeNameFR est le pair_name brut (COALESCE pair_name_fr →
// pair_name). Le frontend normalise via normalizeModeLabel("Arena:Slayer") →
// "Slayer", cohérent avec le chemin home/match-history.
func TestGetMatchMeta_NormalizedPairName_ExtractsSubmode(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, pair_name)
		VALUES ('m4', '2026-04-01 22:00:00', '2026-04-01 22:00:00+00', 'Arena:Slayer')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	meta, err := repo.GetMatchMeta(ctx, "m4")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	if meta.ModeNameFR == nil || *meta.ModeNameFR != "Arena:Slayer" {
		t.Errorf("ModeNameFR = %v, want Arena:Slayer (frontend normalise via normalizeModeLabel)", meta.ModeNameFR)
	}
}

// TestGetMatchMeta_LegacyPairNameFRPassedThrough : pour les matchs d'avant le
// 23 mars 2026, pair_name_fr contenait des libellés legacy avec suffixe
// " on <map>" (ex. "Slayer on Streets"). Le backend retransmet la valeur telle
// quelle (aligné sur home/match-history) ; le frontend normalise via
// normalizeModeLabel("Slayer on Streets") → "Slayer".
func TestGetMatchMeta_LegacyPairNameFRPassedThrough(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, pair_name, pair_name_fr)
		VALUES ('m5', '2026-03-01 18:00:00', '2026-03-01 18:00:00+00',
		        'Arena:Slayer', 'Slayer on Streets')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	meta, err := repo.GetMatchMeta(ctx, "m5")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	if meta.ModeNameFR == nil || *meta.ModeNameFR != "Slayer on Streets" {
		t.Errorf("ModeNameFR = %v, want 'Slayer on Streets' (frontend normalise, cf. normalizeModeLabel)", meta.ModeNameFR)
	}
}

// TestGetMatchMeta_TranslatesModeFRViaModeNameTr : pair_name_fr est NULL en DB
// (jamais écrit par le sync), mais mode_name_tr contient la traduction.
// GetMatchMeta doit produire "Capture du drapeau" et non "CTF" (EN).
// C'est le chemin qui permet le titre "Capture du drapeau sur Forbidden"
// dans le frontend (buildMatchHeadingStr(map_ui, mode_ui, "fr")).
func TestGetMatchMeta_TranslatesModeFRViaModeNameTr(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, map_id, map_name, pair_name)
		VALUES ('m6', '2026-05-01 20:00:00', '2026-05-01 20:00:00+00',
		        'forbidden-uuid', 'Forbidden', 'Arena:CTF on Forbidden')`); err != nil {
		t.Fatalf("insert match: %v", err)
	}
	if _, err := pdb.Metadata.Exec(ctx,
		`INSERT INTO mode_name_tr (lang, mode_en, name) VALUES ('fr', 'CTF', 'Capture du drapeau')`); err != nil {
		t.Fatalf("seed mode_name_tr: %v", err)
	}
	if _, err := pdb.Metadata.Exec(ctx,
		`INSERT INTO asset_translations VALUES ('forbidden-uuid', 'map', 'fr-FR', 'Forbidden', '', now())`); err != nil {
		t.Fatalf("seed map: %v", err)
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	meta, err := repo.GetMatchMeta(ctx, "m6")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	// Mode : doit être traduit FR via mode_name_tr, pas l'EN normalisé.
	if meta.ModeNameFR == nil || *meta.ModeNameFR != "Capture du drapeau" {
		t.Errorf("ModeNameFR = %v, want %q (mode_name_tr lookup)", meta.ModeNameFR, "Capture du drapeau")
	}
	// Map : toujours résolue via asset_translations.
	if meta.MapNameFR == nil || *meta.MapNameFR != "Forbidden" {
		t.Errorf("MapNameFR = %v, want Forbidden", meta.MapNameFR)
	}
	// Titre attendu côté frontend : "Capture du drapeau sur Forbidden"
	// (buildMatchHeadingStr(MapNameFR, ModeNameFR, "fr")).
}

// TestGetMatchMeta_ExtractsModeVariantFR : un pair_name variante non canonique
// ("Legacy Slayer BR on Narrows", absent tel quel de mode_name_tr) doit quand
// même donner le mode FR via extraction du mode connu ("Slayer") + traduction
// ("Assassin"). Rattrapage des variantes d'arme/saison.
func TestGetMatchMeta_ExtractsModeVariantFR(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, pair_name)
		VALUES ('m8', '2026-05-01 22:00:00', '2026-05-01 22:00:00+00',
		        'Legacy Slayer BR on Narrows')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// mode_name_tr connaît "Slayer" → "Assassin" mais PAS la variante complète.
	if _, err := pdb.Metadata.Exec(ctx,
		`INSERT INTO mode_name_tr (lang, mode_en, name) VALUES ('fr', 'Slayer', 'Assassin')`); err != nil {
		t.Fatalf("seed mode_name_tr: %v", err)
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	meta, err := repo.GetMatchMeta(ctx, "m8")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	if meta.ModeNameFR == nil || *meta.ModeNameFR != "Assassin" {
		t.Errorf("ModeNameFR = %v, want %q (Legacy Slayer BR → Slayer → Assassin)", meta.ModeNameFR, "Assassin")
	}
}

// TestGetMatchMeta_ModeNameFRFallbackToEN : mode_name_tr absent → ModeNameFR
// est le pair_name brut (COALESCE). Aligné sur home/match-history qui retournent
// aussi le pair_name brut. Frontend normalise via normalizeModeLabel →
// "Slayer" ; titre "Slayer sur Live Fire" (EN leak toléré).
func TestGetMatchMeta_ModeNameFRFallbackToEN(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, pair_name)
		VALUES ('m7', '2026-05-01 21:00:00', '2026-05-01 21:00:00+00',
		        'Arena:Slayer on Live Fire')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	meta, err := repo.GetMatchMeta(ctx, "m7")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	if meta.ModeNameFR == nil || *meta.ModeNameFR != "Arena:Slayer on Live Fire" {
		t.Errorf("ModeNameFR = %v, want 'Arena:Slayer on Live Fire' (frontend normalise, EN leak toléré)", meta.ModeNameFR)
	}
}

// TestGetMatchMeta_MapImageURLFallsBackToEndpoint : sans image locale curée mais
// avec map_version_id, MapImageURL pointe vers l'endpoint framework KindMapImage
// (?v=version) → le front déclenche le fetch DiscoveryUGC lazy + cache.
func TestGetMatchMeta_MapImageURLFallsBackToEndpoint(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	ctx := context.Background()

	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, map_id, map_version_id)
		VALUES ('m_img', '2026-05-01 23:00:00', '2026-05-01 23:00:00+00', 'map-x', 'ver-x')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	repo := NewMatchViewRepo(pdb, "test-xuid")
	meta, err := repo.GetMatchMeta(ctx, "m_img")
	if err != nil {
		t.Fatalf("GetMatchMeta: %v", err)
	}
	want := "/api/v1/assets/maps/halo_infinite/map-x/image?v=ver-x"
	if meta.MapImageURL == nil || *meta.MapImageURL != want {
		t.Errorf("MapImageURL = %v, want %q", meta.MapImageURL, want)
	}
}

// newEmptySnapshotSharedReader construit un SharedReader adossé à une DB in-memory
// isolée exposant un `match_registry` root-level (colonnes lues par Q13), pré-rempli
// avec les lignes fournies. Simule le snapshot immuable de MatchView, distinct du live.
func newEmptySnapshotSharedReader(t *testing.T, seed ...string) SharedReader {
	t.Helper()
	snapSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	t.Cleanup(func() { snapSQL.Close() })
	snap := newTestDB(snapSQL, ":memory:")
	if _, err := snap.Exec(context.Background(), `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP, end_time TIMESTAMP,
		start_time_utc TIMESTAMPTZ, end_time_utc TIMESTAMPTZ, real_start_time TIMESTAMP,
		playlist_id VARCHAR, map_id VARCHAR, map_version_id VARCHAR, pair_id VARCHAR,
		game_variant_id VARCHAR, map_name VARCHAR, map_name_fr VARCHAR,
		game_variant_name VARCHAR, pair_name VARCHAR, pair_name_fr VARCHAR,
		playlist_name VARCHAR, playlist_name_fr VARCHAR,
		is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE,
		duration_seconds INTEGER, playable_duration_seconds INTEGER,
		team_0_score INTEGER, team_1_score INTEGER)`); err != nil {
		t.Fatalf("seed snapshot schema: %v", err)
	}
	for _, q := range seed {
		if _, err := snap.Exec(context.Background(), q); err != nil {
			t.Fatalf("seed snapshot row: %v\nSQL: %s", err, q)
		}
	}
	return LegacySharedReader(snap)
}

// TestGetMatchMeta_SnapshotMissFallsBackToLive prouve le correctif GH2-A1.
//
// Contexte : la vue détaillée d'un match (MatchViewRepo) lit les faits shared
// match-immutables depuis un SNAPSHOT Parquet découplé du B-swap. Le bouton
// « Voir les matchs » (FilterOmnibar, page Séries temporelles) construit sa liste
// via /filters/match-ids qui lit le shared LIVE. Un match joué après le dernier cut
// du snapshot (ou exclu comme "partial" au moment du cut) est donc dans le live mais
// ABSENT du snapshot → GetMatchMeta renvoyait sql.ErrNoRows → 404 sur le 1er match
// de la liste. Correctif : fallback snapshot→live per-requête (forceLive).
func TestGetMatchMeta_SnapshotMissFallsBackToLive(t *testing.T) {
	pdb := newMetaResolveTestPDB(t) // pdb.SharedReadDB() = LIVE (player)
	ctx := context.Background()

	// LIVE : contient le match récent (absent du snapshot).
	if _, err := pdb.Player.Exec(ctx, `
		INSERT INTO shared.match_registry
			(match_id, start_time, start_time_utc, map_name, pair_name, playlist_name, duration_seconds)
		VALUES ('live-only', '2026-07-03 18:57:00', '2026-07-03 18:57:00+00',
		        'Catalyst', 'Slayer', 'Quick Play', 495)`); err != nil {
		t.Fatalf("insert live match: %v", err)
	}

	// SNAPSHOT : DB isolée, ne contient PAS 'live-only' (juste un décor).
	snapReader := newEmptySnapshotSharedReader(t,
		`INSERT INTO match_registry (match_id, start_time, start_time_utc, map_name, pair_name, playlist_name)
		 VALUES ('in-snap', '2026-06-01 00:00:00', '2026-06-01 00:00:00+00', 'Aquarius', 'Slayer', 'Ranked')`)

	repo := NewMatchViewRepo(pdb, "test-xuid").WithSharedReader(snapReader)

	// Contrôle (non-régression) : un match PRÉSENT dans le snapshot est servi sans
	// jamais toucher au live → forceLive reste désarmé.
	if _, err := repo.GetMatchMeta(ctx, "in-snap"); err != nil {
		t.Fatalf("match présent dans le snapshot devrait résoudre: %v", err)
	}
	if repo.forceLive {
		t.Fatalf("forceLive ne doit PAS être armé quand le snapshot contient le match")
	}

	// Cœur GH2-A1 : match absent du snapshot mais présent en live → servi (fallback).
	meta, err := repo.GetMatchMeta(ctx, "live-only")
	if err != nil {
		t.Fatalf("GH2-A1: 'live-only' doit être servi via fallback live, err=%v", err)
	}
	if meta == nil || meta.MatchID != "live-only" {
		t.Fatalf("meta.MatchID = %v, want 'live-only'", meta)
	}
	if !repo.forceLive {
		t.Fatalf("forceLive doit être armé après un snapshot-miss servi par le live")
	}

	// Non-régression : un match absent du snapshot ET du live reste introuvable
	// (vraie 404, pas de faux positif du fallback).
	repo2 := NewMatchViewRepo(pdb, "test-xuid").WithSharedReader(snapReader)
	if _, err := repo2.GetMatchMeta(ctx, "ghost"); err == nil {
		t.Fatalf("match absent du snapshot ET du live doit rester introuvable")
	}
}
