//go:build integration

package snapshot

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/ops"
)

// fakeLiveReader : SharedReader live de repli, retourne une DB sentinelle.
type fakeLiveReader struct {
	db    *sql.DB
	calls int
}

func (f *fakeLiveReader) Get(context.Context) (*sql.DB, func(), error) {
	f.calls++
	return f.db, func() {}, nil
}

// TestSnapshotPreferredSharedReader_FallbackLive : sans snapshot, Get sert le live.
func TestSnapshotPreferredSharedReader_FallbackLive(t *testing.T) {
	paths := title.NewPathResolver(t.TempDir(), nil)
	live := &fakeLiveReader{db: openSnapMemDB(t)}
	r := NewSnapshotPreferredSharedReader(paths, title.DefaultSlug, live)

	db, release, err := r.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	release()
	if db != live.db || live.calls != 1 {
		t.Fatalf("attendu repli live (calls=%d), got db live=%v", live.calls, db == live.db)
	}
}

// seedReaderSnapshot produit un snapshot complet (schéma shared + 1 match ready) et
// retourne le PathResolver pour le reader.
func seedReaderSnapshot(t *testing.T) *title.PathResolver {
	t.Helper()
	paths := title.NewPathResolver(t.TempDir(), nil)

	shared := openSnapMemDB(t)
	// Schéma RÉALISTE (toutes colonnes lues par mv_player_matches) — OpenSnapshotShared
	// reconstruit mv_player_matches qui référence le schéma complet.
	snapExec(t, shared, `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, start_time TIMESTAMPTZ, end_time TIMESTAMPTZ,
		start_time_utc TIMESTAMPTZ, end_time_utc TIMESTAMPTZ,
		playlist_id VARCHAR, playlist_name VARCHAR, playlist_name_fr VARCHAR,
		map_id VARCHAR, map_name VARCHAR, map_name_fr VARCHAR,
		pair_name VARCHAR, pair_name_fr VARCHAR, pair_id VARCHAR,
		game_variant_id VARCHAR, game_variant_name VARCHAR, mode_category VARCHAR,
		is_ranked BOOLEAN, is_firefight BOOLEAN,
		duration_seconds INTEGER, playable_duration_seconds INTEGER,
		team_0_score INTEGER, team_1_score INTEGER, team_0_ps_score INTEGER, team_1_ps_score INTEGER,
		player_count INTEGER)`)
	snapExec(t, shared, `CREATE TABLE match_participants (
		match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR, team_id INTEGER, outcome INTEGER,
		rank INTEGER, score INTEGER, kills INTEGER, deaths INTEGER, assists INTEGER,
		kda DOUBLE, accuracy DOUBLE, shots_fired INTEGER, shots_hit INTEGER,
		damage_dealt DOUBLE, damage_taken DOUBLE, personal_score INTEGER,
		time_played_seconds INTEGER, avg_life_seconds DOUBLE, headshot_kills INTEGER,
		max_killing_spree INTEGER, grenade_kills INTEGER, melee_kills INTEGER,
		power_weapon_kills INTEGER, team_mmr DOUBLE, enemy_mmr DOUBLE,
		kills_expected DOUBLE, deaths_expected DOUBLE, backfill_bits BIGINT)`)
	snapExec(t, shared, `CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_id BIGINT)`)
	snapExec(t, shared, `CREATE TABLE highlight_events (match_id VARCHAR, xuid VARCHAR, event_type VARCHAR)`)
	snapExec(t, shared, `CREATE TABLE killer_victim_pairs (match_id VARCHAR, killer_xuid VARCHAR, killer_gamertag VARCHAR, victim_xuid VARCHAR, victim_gamertag VARCHAR)`)
	snapExec(t, shared, `CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`)
	snapExec(t, shared, `CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, weapon_id BIGINT, reconciled_as BIGINT, generation_id BIGINT)`)
	snapExec(t, shared, `CREATE TABLE match_csrs (match_id VARCHAR, xuid VARCHAR, rating_value FLOAT, written_at TIMESTAMP, id BIGINT)`)
	// Requise depuis l'ajout de match_objective_stats au snapshot (Q12 LEFT JOIN la
	// vue _latest) — table vide : le COPY exporte un parquet vide, la vue répond vide.
	snapExec(t, shared, `CREATE TABLE match_objective_stats (id BIGINT, match_id VARCHAR, xuid VARCHAR, flag_captures INTEGER, written_at TIMESTAMP)`)
	// Requise depuis l'ajout de match_kill_events au snapshot (Q21b/Q21c lisent la vue
	// _latest) — schéma canonique via la fonction de migration, même raison que
	// match_objective_stats : sans la table, l'export saute la relation (relationExists)
	// et OpenSnapshotShared refuse le snapshot → ce test servirait le live à tort.
	if err := migration.EnsureMatchKillEvents(shared); err != nil {
		t.Fatalf("mke: %v", err)
	}
	now := time.Now()
	snapExec(t, shared, `INSERT INTO match_registry (match_id, start_time, start_time_utc, is_ranked, is_firefight) VALUES ('m1', ?, ?, FALSE, FALSE)`, now, now)
	snapExec(t, shared, `INSERT INTO match_participants (match_id, xuid, gamertag, team_id, kills, deaths) VALUES ('m1', 'x1', 'GT1', 0, 7, 3)`)
	snapExec(t, shared, `INSERT INTO killer_victim_pairs (match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag) VALUES ('m1', 'x1', 'GT1', 'x2', 'GT2')`)
	snapExec(t, shared, `INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('x1', 'GT1')`)
	snapExec(t, shared, `INSERT INTO weapon_kills (match_id, xuid, weapon_id, reconciled_as, generation_id) VALUES ('m1', 'x1', 200, NULL, 0)`)

	player := openSnapMemDB(t)
	snapExec(t, player, `CREATE TABLE player_match_enrichment (match_id VARCHAR)`)
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(player); err != nil {
		t.Fatalf("pme: %v", err)
	}
	snapExec(t, player, `CREATE TABLE match_citations_latest (match_id VARCHAR)`)
	snapExec(t, player, `CREATE TABLE match_skill_rank_latest (match_id VARCHAR, rating_type VARCHAR)`)
	snapExec(t, player, `INSERT INTO player_match_enrichment (match_id, snapshot_ready_at, partial_reasons, stage) VALUES ('m1', ?, '[]', 'snapshot')`, now)

	res, err := ops.ProduceSnapshot(context.Background(), ops.SnapshotOptions{
		TitleSlug: title.DefaultSlug,
		Paths:     paths,
		Shared:    ops.SharedReadOpenerFunc(func(context.Context) (*sql.DB, func(), error) { return shared, func() {}, nil }),
		PlayerOpener: ops.PlayerReadOpenerFunc(func(_ context.Context, gt string) (*sql.DB, func(), error) {
			return player, func() {}, nil
		}),
		Players: []string{"Solo"},
	})
	if err != nil || !res.Produced {
		t.Fatalf("produce: res=%+v err=%v", res, err)
	}
	return paths
}

// TestSnapshotPreferredSharedReader_ServedAndCached : avec un snapshot, Get sert depuis
// le snapshot (pas le live) et réutilise le querier caché entre deux appels.
func TestSnapshotPreferredSharedReader_ServedAndCached(t *testing.T) {
	ctx := context.Background()
	paths := seedReaderSnapshot(t)
	live := &fakeLiveReader{db: openSnapMemDB(t)}
	r := NewSnapshotPreferredSharedReader(paths, title.DefaultSlug, live)
	defer r.Close()

	db1, rel1, err := r.Get(ctx)
	if err != nil {
		t.Fatalf("Get #1: %v", err)
	}
	rel1()
	if db1 == live.db {
		t.Fatal("Get #1 a servi le live, attendu le snapshot")
	}
	if live.calls != 0 {
		t.Fatalf("live.calls = %d, attendu 0 (servi depuis snapshot)", live.calls)
	}
	// La DB snapshot répond aux requêtes shared (m1 ready présent).
	var n int
	if err := db1.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_registry`).Scan(&n); err != nil {
		t.Fatalf("query snapshot: %v", err)
	}
	if n != 1 {
		t.Errorf("match_registry = %d, attendu 1", n)
	}
	// 2e Get : même querier caché (même *sql.DB).
	db2, rel2, _ := r.Get(ctx)
	rel2()
	if db2 != db1 {
		t.Error("2e Get a reconstruit le querier (cache versionné non réutilisé)")
	}
}

// TestSnapshotPreferredSharedReader_IncompleteFallback : si le snapshot existe mais est
// INCOMPLET (une table requise manque), OpenSnapshotShared retourne ErrSnapshotIncomplete
// → le reader retombe sur le live (jamais de schéma partiel servi), et le negative-cache
// évite de reconstruire la :memory à chaque Get.
func TestSnapshotPreferredSharedReader_IncompleteFallback(t *testing.T) {
	ctx := context.Background()
	paths := seedReaderSnapshot(t)
	// Casse la complétude : supprime un Parquet REQUIS de la v1.
	missing := filepath.Join(paths.SnapshotVersionDir(title.DefaultSlug, 1), "shared", "xuid_aliases.parquet")
	if err := os.Remove(missing); err != nil {
		t.Fatalf("remove required parquet: %v", err)
	}

	live := &fakeLiveReader{db: openSnapMemDB(t)}
	r := NewSnapshotPreferredSharedReader(paths, title.DefaultSlug, live)
	defer r.Close()

	db1, rel1, err := r.Get(ctx)
	if err != nil {
		t.Fatalf("Get #1: %v", err)
	}
	rel1()
	if db1 != live.db {
		t.Fatal("schéma incomplet : Get devait retomber sur le live")
	}
	// 2e Get : toujours live, ET le negative-cache a évité une nouvelle tentative de
	// reconstruction (failedVersion mémorisé).
	db2, rel2, _ := r.Get(ctx)
	rel2()
	if db2 != live.db {
		t.Fatal("2e Get : devait rester sur le live")
	}
	if r.failedVersion != 1 {
		t.Errorf("failedVersion = %d, attendu 1 (negative-cache)", r.failedVersion)
	}
}
