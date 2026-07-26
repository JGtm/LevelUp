//go:build integration

// Package ops — seed_demo_integration_test.go : test end-to-end avec DuckDB live.
//
// Crée des fixtures source (shared + player + metadata), lance SeedDemo, et
// vérifie les outputs : tables filtrées + xuid anonymisé + configs JSON.
//
// CGO_ENABLED=1 requis (driver duckdb).
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// seedSourceDBs crée des DBs source factices (3 matchs, 2 joueurs) prêtes pour
// SeedDemo. Retourne le tempdir + les 3 chemins source.
func seedSourceDBs(t *testing.T) (tmpDir, srcPlayer, srcShared, srcMeta string) {
	t.Helper()
	tmpDir = t.TempDir()
	srcPlayer = filepath.Join(tmpDir, "src_player.duckdb")
	srcShared = filepath.Join(tmpDir, "src_shared.duckdb")
	srcMeta = filepath.Join(tmpDir, "src_meta.duckdb")

	const sourceXUID = "1111111111111111"

	// metadata.duckdb : minimal (juste 1 table avec 1 row pour vérifier la copie).
	metaDB, err := sql.Open("duckdb", srcMeta)
	if err != nil {
		t.Fatal(err)
	}
	defer metaDB.Close()
	if _, err := metaDB.Exec(`CREATE TABLE career_ranks (rank_id INTEGER, rank_name VARCHAR)`); err != nil {
		t.Fatal(err)
	}
	if _, err := metaDB.Exec(`INSERT INTO career_ranks VALUES (1, 'Recruit'), (2, 'Iron')`); err != nil {
		t.Fatal(err)
	}

	// shared_matches_v2.duckdb : 3 matchs, 2 joueurs.
	sharedDB, err := sql.Open("duckdb", srcShared)
	if err != nil {
		t.Fatal(err)
	}
	defer sharedDB.Close()
	// Schéma complet via migrations canoniques : un fixture minimal fait main
	// dérive vite du schéma réel et casse la chaîne de migrations relancée par
	// seed-demo (vues matérialisées, colonnes append-only…). On crée donc la DB
	// source au schéma courant, puis on insère les données (colonnes explicites).
	if err := migration.RunForDB(sharedDB, migration.TargetShared); err != nil {
		t.Fatalf("migrations shared (fixture): %v", err)
	}

	// 3 matchs : m1 (le plus récent), m2 (milieu), m3 (le plus ancien).
	mustExec(t, sharedDB, `
		INSERT INTO match_registry (match_id, start_time, map_name, playlist_name, pair_name) VALUES
		('m1', TIMESTAMP '2026-05-22 18:00:00', 'Aquarius', 'Ranked Slayer', 'Ranked:Slayer'),
		('m2', TIMESTAMP '2026-05-21 18:00:00', 'Bazaar', 'Open Crossplay', 'Open:CTF'),
		('m3', TIMESTAMP '2026-05-20 18:00:00', 'Live Fire', 'Open Crossplay', 'Open:Slayer')`)
	mustExec(t, sharedDB, `
		INSERT INTO match_participants (match_id, xuid, gamertag, kills, deaths) VALUES
		('m1', '`+sourceXUID+`', 'JGtm', 15, 8),
		('m1', '2222222222222222', 'Other', 10, 12),
		('m2', '`+sourceXUID+`', 'JGtm', 8, 10),
		('m3', '`+sourceXUID+`', 'JGtm', 20, 5)`)
	mustExec(t, sharedDB, `
		INSERT INTO medals_earned (match_id, xuid, medal_name_id) VALUES
		('m1', '`+sourceXUID+`', 100),
		('m2', '`+sourceXUID+`', 200),
		('m3', '`+sourceXUID+`', 100)`)
	mustExec(t, sharedDB, `
		INSERT INTO xuid_aliases (xuid, gamertag) VALUES
		('`+sourceXUID+`', 'JGtm'),
		('2222222222222222', 'Other')`)

	// player stats.duckdb
	playerDB, err := sql.Open("duckdb", srcPlayer)
	if err != nil {
		t.Fatal(err)
	}
	defer playerDB.Close()
	// Schéma player aligné sur la prod (cf. internal/sync/schema.go + steps_player.go) :
	// - player_match_enrichment : PAS de colonne xuid (player DB mono-joueur)
	// - match_skill_rank : PAS de colonne xuid (idem)
	// - career_progression : A une colonne xuid (anonymisée)
	if err := migration.RunForDB(playerDB, migration.TargetPlayer); err != nil {
		t.Fatalf("migrations player (fixture): %v", err)
	}

	mustExec(t, playerDB, `
		INSERT INTO player_match_enrichment (match_id, session_id, performance_score) VALUES
		('m1', 'sess1', 78.5),
		('m2', 'sess1', 65.0),
		('m3', 'sess2', 82.3)`)
	mustExec(t, playerDB, `INSERT INTO match_citations (match_id, citation_name_norm, value) VALUES ('m1', 'kills', 15), ('m2', 'kills', 8)`)
	mustExec(t, playerDB, `INSERT INTO sessions (session_id, label) VALUES (1, 'Session 1'), (2, 'Session 2')`)
	mustExec(t, playerDB, `INSERT INTO career_progression (xuid, rank, recorded_at) VALUES ('`+sourceXUID+`', 10, TIMESTAMP '2026-05-22 18:00:00')`)
	mustExec(t, playerDB, `INSERT INTO sync_meta (key, value) VALUES ('xuid', '`+sourceXUID+`'), ('msal_token_cache', 'SECRET_DO_NOT_LEAK')`)
	mustExec(t, playerDB, `INSERT INTO match_skill_rank (match_id, rating_type, rating_value) VALUES ('m1', 'csr', 1200.5)`)

	return tmpDir, srcPlayer, srcShared, srcMeta
}

func mustExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("exec %q: %v", q[:min(60, len(q))], err)
	}
}

func TestSeedDemo_EndToEnd_HappyPath(t *testing.T) {
	tmpDir, srcPlayer, srcShared, srcMeta := seedSourceDBs(t)
	const sourceXUID = "1111111111111111"
	outDir := filepath.Join(tmpDir, "out")

	opts := SeedDemoOptions{
		SourcePlayerDB: srcPlayer,
		SourceSharedDB: srcShared,
		SourceMetaDB:   srcMeta,
		SourceXUID:     sourceXUID,
		OutDir:         outDir,
		MaxMatches:     2, // limite à 2 → on garde m1 + m2 (les plus récents)
		SourceLabel:    "JGtm",
		ServiceTag:     "SPTA",
		IncludeMedia:   false, // testé séparément
	}
	res, err := SeedDemo(context.Background(), opts)
	if err != nil {
		t.Fatalf("SeedDemo: %v", err)
	}

	// ── Sélection matches : m1 + m2 (m3 exclu par LIMIT 2)
	if len(res.MatchIDs) != 2 {
		t.Errorf("MatchIDs count = %d, want 2", len(res.MatchIDs))
	}
	wantMatches := map[string]bool{"m1": true, "m2": true}
	for _, id := range res.MatchIDs {
		if !wantMatches[id] {
			t.Errorf("unexpected matchID: %s (m3 doit être exclu)", id)
		}
	}

	// ── metadata copiée
	outMeta := filepath.Join(outDir, "warehouse", "metadata.duckdb")
	if _, err := os.Stat(outMeta); err != nil {
		t.Fatalf("metadata.duckdb absent: %v", err)
	}
	verifyMetaCopied(t, outMeta)

	// ── shared filtré + anonymisé
	outShared := filepath.Join(outDir, "warehouse", "shared_matches_v2.duckdb")
	verifySharedExtracted(t, outShared, sourceXUID)

	// ── player filtré + anonymisé
	outPlayer := filepath.Join(outDir, "players", DefaultDemoGamertag, "stats.duckdb")
	verifyPlayerExtracted(t, outPlayer, sourceXUID)

	// ── échantillons Prestige (player DB + shared_social)
	verifyPrestigeSeeded(t, outDir, res)

	// ── configs JSON
	verifyConfigsWritten(t, outDir, "JGtm", "SPTA", false)
}

// verifyPrestigeSeeded : les tables Prestige/Progression de la démo sont PEUPLÉES
// après un seed-demo (sans elles, les pages Ascension/Prestige sont vides), et les
// invariants de cohérence tiennent en base :
//   - total de points de progression == somme des événements qui l'ont produit ;
//   - un événement « match » par match du corpus ;
//   - aucun arc/objectif rattaché à une identité autre que le joueur démo ;
//   - les objectifs couvrent plusieurs stades du cycle de vie.
func verifyPrestigeSeeded(t *testing.T, outDir string, res SeedDemoResult) {
	t.Helper()
	playerDB, err := sql.Open("duckdb", filepath.Join(outDir, "players", DefaultDemoGamertag, "stats.duckdb")+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	defer playerDB.Close()

	for _, tbl := range []string{"arc", "challenge", "improvement_campaign",
		"prestige_telemetry", "baseline_state", "streak_history", "record_history"} {
		var n int
		if err := playerDB.QueryRow(`SELECT COUNT(*) FROM ` + tbl).Scan(&n); err != nil {
			t.Errorf("player démo : %s illisible: %v", tbl, err)
			continue
		}
		if n == 0 {
			t.Errorf("player démo : %s vide — la page correspondante n'afficherait rien", tbl)
		}
	}
	// La vue append-only des séries doit servir exactement les séries seedées.
	var streakRows int
	if err := playerDB.QueryRow(`SELECT COUNT(*) FROM streak_latest`).Scan(&streakRows); err != nil {
		t.Errorf("streak_latest illisible: %v", err)
	} else if streakRows == 0 {
		t.Error("streak_latest vide — les séries seedées ne seraient pas servies")
	}
	// Anti-fuite : aucune identité autre que celle du joueur démo.
	var foreign int
	if err := playerDB.QueryRow(
		`SELECT COUNT(*) FROM challenge WHERE user_id <> ?`, DefaultDemoMainSlug).Scan(&foreign); err != nil {
		t.Errorf("challenge.user_id illisible: %v", err)
	} else if foreign != 0 {
		t.Errorf("challenge : %d ligne(s) avec un user_id étranger au joueur démo", foreign)
	}
	var statuses int
	if err := playerDB.QueryRow(`SELECT COUNT(DISTINCT status) FROM challenge`).Scan(&statuses); err != nil {
		t.Errorf("challenge.status illisible: %v", err)
	} else if statuses < 3 {
		t.Errorf("challenge : %d statut(s) distinct(s), want >= 3 (actif/complété/terminé)", statuses)
	}

	socialDB, err := sql.Open("duckdb", filepath.Join(outDir, "warehouse", "shared_social.duckdb")+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	defer socialDB.Close()

	var eventCount, eventSum, matchEvents int
	if err := socialDB.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(pp_amount), 0),
		       COALESCE(SUM(CASE WHEN source_type = 'match' THEN 1 ELSE 0 END), 0)
		FROM prestige_events`).Scan(&eventCount, &eventSum, &matchEvents); err != nil {
		t.Fatalf("prestige_events illisible: %v", err)
	}
	if eventCount == 0 {
		t.Fatal("prestige_events vide — aucun point de progression à montrer")
	}
	if matchEvents != len(res.MatchIDs) {
		t.Errorf("événements match = %d, want %d (1 par match du corpus démo)",
			matchEvents, len(res.MatchIDs))
	}
	var totalPP int
	if err := socialDB.QueryRow(
		`SELECT total_pp FROM user_prestige_latest WHERE user_id = ?`, DefaultDemoMainSlug).Scan(&totalPP); err != nil {
		t.Fatalf("user_prestige_latest illisible: %v", err)
	}
	if totalPP != eventSum {
		t.Errorf("total_pp = %d mais somme des événements = %d (le total contredirait le journal)",
			totalPP, eventSum)
	}
	for _, tbl := range []string{"squad", "squad_member_latest",
		"squad_challenge", "squad_challenge_participant_latest", "player_records_latest"} {
		var n int
		if err := socialDB.QueryRow(`SELECT COUNT(*) FROM ` + tbl).Scan(&n); err != nil {
			t.Errorf("shared_social démo : %s illisible: %v", tbl, err)
			continue
		}
		if n == 0 {
			t.Errorf("shared_social démo : %s vide", tbl)
		}
	}
}

func TestSeedDemo_EndToEnd_NoMatchesForXUID(t *testing.T) {
	_, srcPlayer, srcShared, srcMeta := seedSourceDBs(t)
	tmpDir := t.TempDir()
	opts := SeedDemoOptions{
		SourcePlayerDB: srcPlayer,
		SourceSharedDB: srcShared,
		SourceMetaDB:   srcMeta,
		SourceXUID:     "9999999999999999", // xuid inexistant
		OutDir:         filepath.Join(tmpDir, "out"),
		MaxMatches:     10,
		SourceLabel:    "Phantom",
	}
	_, err := SeedDemo(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error for unknown xuid")
	}
	if err != nil && !contains(err.Error(), "aucun match") {
		t.Errorf("err = %v, want 'aucun match'", err)
	}
}

func TestSeedDemo_EndToEnd_OutDirCreated(t *testing.T) {
	tmpDir, srcPlayer, srcShared, srcMeta := seedSourceDBs(t)
	// OutDir niché 3 niveaux profond, doit être créé récursivement.
	outDir := filepath.Join(tmpDir, "a", "b", "c", "demo")
	opts := SeedDemoOptions{
		SourcePlayerDB: srcPlayer,
		SourceSharedDB: srcShared,
		SourceMetaDB:   srcMeta,
		SourceXUID:     "1111111111111111",
		OutDir:         outDir,
		MaxMatches:     5,
		SourceLabel:    "JGtm",
	}
	if _, err := SeedDemo(context.Background(), opts); err != nil {
		t.Fatalf("SeedDemo: %v", err)
	}
	if _, err := os.Stat(outDir); err != nil {
		t.Errorf("outDir not created: %v", err)
	}
}

// ── Helpers de vérification ──────────────────────────────────────────────────

func verifyMetaCopied(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM career_ranks`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("metadata.career_ranks = %d, want 2 (copie complète)", n)
	}
}

func verifySharedExtracted(t *testing.T, path, sourceXUID string) {
	t.Helper()
	db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// match_registry : 2 matchs (m1 + m2).
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("match_registry rows = %d, want 2", n)
	}

	// m3 ne doit pas être dans le set.
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_registry WHERE match_id = 'm3'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("m3 leaked in shared filtré")
	}

	// match_participants : sourceXUID doit être anonymisé en DefaultDemoXUID.
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE xuid = ?`, sourceXUID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("sourceXUID encore présent dans match_participants (%d rows)", n)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_participants WHERE xuid = ?`, DefaultDemoXUID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("demoXUID dans match_participants = %d, want 2 (1 par match m1+m2)", n)
	}

	// xuid_aliases : sourceXUID anonymisé aussi.
	if err := db.QueryRow(`SELECT COUNT(*) FROM xuid_aliases WHERE xuid = ?`, sourceXUID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("sourceXUID encore présent dans xuid_aliases")
	}

	// vue v_match_full doit exister.
	if err := db.QueryRow(`SELECT COUNT(*) FROM v_match_full`).Scan(&n); err != nil {
		t.Errorf("v_match_full inaccessible: %v", err)
	}
	if n != 2 {
		t.Errorf("v_match_full rows = %d, want 2", n)
	}
}

func verifyPlayerExtracted(t *testing.T, path, sourceXUID string) {
	t.Helper()
	db, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// player_match_enrichment : 2 rows (m1 + m2), m3 exclu.
	// Note : pas de colonne xuid (player DB mono-joueur, xuid implicite via path).
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_match_enrichment`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("player_match_enrichment rows = %d, want 2", n)
	}

	// sync_meta : msal_token_cache exclu, xuid mis à jour.
	if err := db.QueryRow(`SELECT COUNT(*) FROM sync_meta WHERE key = 'msal_token_cache'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("msal_token_cache LEAKED dans sync_meta démo")
	}
	var demoXUIDValue string
	if err := db.QueryRow(`SELECT value FROM sync_meta WHERE key = 'xuid'`).Scan(&demoXUIDValue); err != nil {
		t.Fatal(err)
	}
	if demoXUIDValue != DefaultDemoXUID {
		t.Errorf("sync_meta.xuid = %q, want %q", demoXUIDValue, DefaultDemoXUID)
	}

	// sessions : copié intégralement (2 rows : sess1 + sess2).
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("sessions rows = %d, want 2 (copie intégrale)", n)
	}

	// match_skill_rank : 1 row (m1). Pas de colonne xuid (player DB mono-joueur).
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_skill_rank`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("match_skill_rank rows = %d, want 1 (m1 ranked seulement)", n)
	}

	// career_progression : anonymisé (a une vraie colonne xuid).
	if err := db.QueryRow(`SELECT COUNT(*) FROM career_progression WHERE xuid = ?`, sourceXUID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("sourceXUID dans career_progression, anonymisation manquée")
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM career_progression WHERE xuid = ?`, DefaultDemoXUID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("career_progression[xuid=DEMO] = %d, want 1", n)
	}
}

func verifyConfigsWritten(t *testing.T, outDir, sourceGamertag, serviceTag string, mediaEnabled bool) {
	t.Helper()
	profilesPath := filepath.Join(outDir, "db_profiles.json")
	data, err := os.ReadFile(profilesPath)
	if err != nil {
		t.Fatalf("read profiles: %v", err)
	}
	var profiles map[string]any
	if err := json.Unmarshal(data, &profiles); err != nil {
		t.Fatalf("parse profiles: %v", err)
	}
	// Profils démo keyés par GAMERTAG (multi-roster) : le main = DefaultDemoMainGamertag
	// ("DemoPlayer"), distinct du répertoire DB "DEMO" (cf. demoDirForIndex / DemoRoster[0]).
	// Résolution version-aware : v3 (profiles.{titleSlug}.{gamertag}, écrit par la CLI
	// multi-titre) puis v2.1 (profiles.{gamertag}, écrit par SeedDemo direct).
	profMap, _ := profiles["profiles"].(map[string]any)
	var demoProfile map[string]any
	if byTitle, ok := profMap[titlePkg.DefaultSlug].(map[string]any); ok {
		demoProfile, _ = byTitle[DefaultDemoMainGamertag].(map[string]any)
	}
	if demoProfile == nil {
		demoProfile, _ = profMap[DefaultDemoMainGamertag].(map[string]any)
	}
	if demoProfile == nil {
		t.Fatalf("profile %q absent", DefaultDemoMainGamertag)
	}
	if v, _ := demoProfile["xuid"].(string); v != DefaultDemoXUID {
		t.Errorf("%s.xuid = %q, want %q", DefaultDemoMainGamertag, v, DefaultDemoXUID)
	}
	// waypoint_player = gamertag démo (jamais le gamertag source réel : anti-fuite).
	if v, _ := demoProfile["waypoint_player"].(string); v != DefaultDemoMainGamertag {
		t.Errorf("%s.waypoint_player = %q, want %q (gamertag démo, pas la source)",
			DefaultDemoMainGamertag, v, DefaultDemoMainGamertag)
	}
	// Invariant anti-fuite : le gamertag source réel ne doit apparaître nulle part.
	if contains(string(data), sourceGamertag) {
		t.Errorf("gamertag source %q fuit dans db_profiles.json", sourceGamertag)
	}

	settingsPath := filepath.Join(outDir, "app_settings.json")
	settingsData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(settingsData, &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	if v, _ := settings["profile_service_tag"].(string); v != serviceTag {
		t.Errorf("service_tag = %q, want %q", v, serviceTag)
	}
	if v, _ := settings["media_enabled"].(bool); v != mediaEnabled {
		t.Errorf("media_enabled = %v, want %v", v, mediaEnabled)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// Sanity ensure time.Time import usage (utilisé par seedSourceDBs ailleurs).
var _ = time.Now

// TestLoadVideoRealMaps valide la résolution VRAIE carte d'une vidéo via l'association
// prod (media_match_associations → match → map du shared_matches_v2 ATTACHé). C'est le
// cœur du fix maps média (remplace l'ancienne heuristique temporelle qui se trompait).
func TestLoadVideoRealMaps(t *testing.T) {
	dir := t.TempDir()
	smPath := filepath.Join(dir, "sm.duckdb")
	ssPath := filepath.Join(dir, "ss.duckdb")
	ctx := context.Background()

	sm, err := sql.Open("duckdb", smPath)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, sm, `CREATE TABLE match_registry (match_id VARCHAR, map_name VARCHAR)`)
	mustExec(t, sm, `INSERT INTO match_registry VALUES ('m1','Illusion'), ('m2','Bazaar')`)
	sm.Close()

	ss, err := sql.Open("duckdb", ssPath)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, ss, `CREATE TABLE media_files (id VARCHAR, file_name VARCHAR, file_stem VARCHAR)`)
	// Append-only média : le reader lit la vue media_match_associations_latest (sur _history).
	mustExec(t, ss, `CREATE SEQUENCE media_match_associations_history_id_seq START 1`)
	mustExec(t, ss, `CREATE TABLE media_match_associations_history (
		id BIGINT PRIMARY KEY DEFAULT nextval('media_match_associations_history_id_seq'),
		media_file_id VARCHAR NOT NULL, match_id VARCHAR NOT NULL, delta_seconds INTEGER,
		is_manual BOOLEAN NOT NULL DEFAULT FALSE, is_active BOOLEAN NOT NULL DEFAULT TRUE,
		associated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		written_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`)
	mustExec(t, ss, `CREATE OR REPLACE VIEW media_match_associations_latest AS
		WITH lpp AS (SELECT media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at,
			ROW_NUMBER() OVER (PARTITION BY media_file_id, match_id ORDER BY written_at DESC, id DESC) AS rn
			FROM media_match_associations_history),
		act AS (SELECT * FROM lpp WHERE rn = 1 AND is_active = TRUE),
		hm AS (SELECT media_file_id, bool_or(is_manual) AS has_manual FROM act GROUP BY media_file_id)
		SELECT a.media_file_id, a.match_id, a.delta_seconds, a.is_manual, a.associated_at, a.written_at
		FROM act a JOIN hm ON hm.media_file_id = a.media_file_id WHERE a.is_manual = hm.has_manual`)
	mustExec(t, ss, `INSERT INTO media_files VALUES
		('a','Halo Infinite 2026-02-18 17-37-18.mkv','Halo Infinite 2026-02-18 17-37-18'),
		('b','Replay clip.mp4','Replay clip')`)
	mustExec(t, ss, `INSERT INTO media_match_associations_history (media_file_id, match_id) VALUES ('a','m1'), ('b','m2')`)
	ss.Close()

	got, err := loadVideoRealMaps(ctx, ssPath, smPath)
	if err != nil {
		t.Fatal(err)
	}
	// La vidéo Halo Infinite associée à m1 → vraie carte Illusion.
	if got["Halo Infinite 2026-02-18 17-37-18"] != "Illusion" {
		t.Errorf("vraie carte = %q, want Illusion", got["Halo Infinite 2026-02-18 17-37-18"])
	}
	// Média non "Halo Infinite" exclu (filtre file_name LIKE 'Halo Infinite%').
	if _, ok := got["Replay clip"]; ok {
		t.Error("média non Halo Infinite ne devrait pas être inclus")
	}
}
