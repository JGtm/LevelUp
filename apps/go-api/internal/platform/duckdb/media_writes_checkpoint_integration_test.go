//go:build cgo

// Package duckdb — media_writes_checkpoint_integration_test.go (ADR 0021 Phase 4.1).
//
// Tests d'intégration qui valident que les écritures media/likes/favoris sur
// shared_social SURVIVENT à un Close brutal (sans Close gracieux propre),
// grâce au CHECKPOINT immédiat post-write ajouté en Phase 3.2.
//
// Stratégie :
//
//  1. Créer une DB shared_social fraîche avec le schéma minimal nécessaire.
//  2. Exécuter une écriture via le path testé (SetMediaMatchAssociation,
//     SetMediaLike, ToggleSharedLike — tous les paths "HIGH" de l'audit).
//  3. Close brutal de la DB (sans CHECKPOINT explicite — simule un crash).
//  4. Re-open en mode RO et vérifier que la donnée est présente.
//
// Si la donnée est absente après reopen, c'est que le WAL contenait la write
// non-checkpointed → fix Phase 3.2 ne marche pas.

package duckdb

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// createSharedSocialSchemaForMediaTests crée un sous-ensemble minimal du schéma
// shared_social pour les tests d'intégration des paths d'écriture media.
//
// Le schéma migration complet a une divergence connue avec le schéma de prod
// (cf. audit). On reproduit ici le schéma de prod (id INTEGER auto, etc.) pour
// que MediaRepo.SetMediaMatchAssociation fonctionne.
func createSharedSocialSchemaForMediaTests(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "shared_social.duckdb")
	db, err := OpenReadWriteShared(path, "")
	if err != nil {
		t.Fatalf("OpenReadWriteShared: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ddl := `
		CREATE SEQUENCE IF NOT EXISTS media_id_seq START 1;
		CREATE TABLE IF NOT EXISTS media_files (
			id INTEGER DEFAULT nextval('media_id_seq') PRIMARY KEY,
			player_slug VARCHAR,
			file_path VARCHAR NOT NULL UNIQUE,
			file_name VARCHAR,
			kind VARCHAR DEFAULT 'video',
			thumbnail_path VARCHAR,
			-- status : présente en prod (ALTER de steps_shared_social) et filtrée par
			-- toute lecture applicative depuis l'item 3.1 (MediaVisiblePredicate).
			status VARCHAR,
			mtime TIMESTAMP WITH TIME ZONE,
			indexed_at TIMESTAMP WITH TIME ZONE DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS media_match_associations (
			media_file_id INTEGER,
			match_id VARCHAR,
			delta_seconds INTEGER DEFAULT 0,
			is_manual BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (media_file_id, match_id)
		);
		CREATE SEQUENCE IF NOT EXISTS media_match_associations_history_id_seq START 1;
		CREATE TABLE IF NOT EXISTS media_match_associations_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('media_match_associations_history_id_seq'),
			media_file_id BIGINT NOT NULL, match_id VARCHAR NOT NULL, delta_seconds INTEGER,
			is_manual BOOLEAN NOT NULL DEFAULT FALSE, is_active BOOLEAN NOT NULL DEFAULT TRUE,
			associated_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE OR REPLACE VIEW media_match_associations_latest AS
			WITH lpp AS (
				SELECT media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at,
					ROW_NUMBER() OVER (PARTITION BY media_file_id, match_id ORDER BY written_at DESC, id DESC) AS rn
				FROM media_match_associations_history),
			act AS (SELECT * FROM lpp WHERE rn = 1 AND is_active = TRUE),
			hm AS (SELECT media_file_id, bool_or(is_manual) AS has_manual FROM act GROUP BY media_file_id)
			SELECT a.media_file_id, a.match_id, a.delta_seconds, a.is_manual, a.associated_at, a.written_at
			FROM act a JOIN hm ON hm.media_file_id = a.media_file_id
			WHERE a.is_manual = hm.has_manual;
		CREATE TABLE IF NOT EXISTS media_likes (
			media_path VARCHAR NOT NULL,
			liker_slug VARCHAR NOT NULL,
			liker_gamertag VARCHAR,
			liked_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (media_path, liker_slug)
		);
		CREATE SEQUENCE IF NOT EXISTS media_likes_history_id_seq START 1;
		CREATE TABLE IF NOT EXISTS media_likes_history (
			id BIGINT PRIMARY KEY DEFAULT nextval('media_likes_history_id_seq'),
			media_path VARCHAR NOT NULL, liker_slug VARCHAR NOT NULL, liker_gamertag VARCHAR,
			is_liked BOOLEAN NOT NULL, liked_at TIMESTAMP,
			written_at TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		);
		CREATE OR REPLACE VIEW media_likes_latest AS
			SELECT id, media_path, liker_slug, liker_gamertag, is_liked, liked_at, written_at
			FROM media_likes_history
			QUALIFY ROW_NUMBER() OVER (PARTITION BY media_path, liker_slug
				ORDER BY written_at DESC, id DESC) = 1;
		CREATE TABLE IF NOT EXISTS match_favorites (
			player_slug VARCHAR NOT NULL,
			match_id VARCHAR NOT NULL,
			favorited_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (player_slug, match_id)
		);
	`
	if _, err := db.SQLDb().Exec(ddl); err != nil {
		t.Fatalf("seed shared_social ddl: %v", err)
	}
	// Insérer un média baseline pour les tests qui en ont besoin.
	if _, err := db.SQLDb().Exec(`
		INSERT INTO media_files (player_slug, file_path, file_name)
		VALUES ('test-player', '/test/media.mp4', 'media.mp4')
	`); err != nil {
		t.Fatalf("insert baseline media: %v", err)
	}
	// CHECKPOINT initial pour vider le DDL du WAL.
	if _, err := db.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint initial: %v", err)
	}
	return db
}

// reopenAndCount close la DB de test et la rouvre en RO pour valider que
// les writes ont bien été flushées sur disque (pas restées dans le WAL).
func reopenAndCount(t *testing.T, db *DB, query string, args ...any) int64 {
	t.Helper()
	path := db.Path()
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// Reopen RO — si la donnée est dans le WAL non-flushé, elle disparaîtra.
	roDB, err := sql.Open("duckdb", path+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatalf("reopen RO: %v", err)
	}
	defer roDB.Close()
	var n int64
	if err := roDB.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

// TestCheckpointSharedSocial_FlushesWALToFile : write + CHECKPOINT + Close
// brutal (sans 2e CHECKPOINT) + Reopen → la donnée doit être présente.
func TestCheckpointSharedSocial_FlushesWALToFile(t *testing.T) {
	db := createSharedSocialSchemaForMediaTests(t)
	ctx := context.Background()

	// Write : insère un like via INSERT direct (path legacy, simule ToggleSharedLike
	// fallback). Le INSERT seul laisse le WAL pending.
	if _, err := db.Exec(ctx,
		`INSERT INTO media_likes (media_path, liker_slug, liker_gamertag) VALUES (?, ?, ?)`,
		"/test/media.mp4", "alice", "Alice",
	); err != nil {
		t.Fatalf("insert like: %v", err)
	}

	// CHECKPOINT explicite (le fix Phase 3.2) — flush WAL sur disque.
	if err := CheckpointSharedSocial(ctx, db); err != nil {
		t.Fatalf("CheckpointSharedSocial: %v", err)
	}

	// Reopen RO et compter — si CHECKPOINT a marché, le like est présent.
	got := reopenAndCount(t, db, `SELECT COUNT(*) FROM media_likes WHERE media_path = ?`, "/test/media.mp4")
	if got != 1 {
		t.Errorf("attendu 1 like après CHECKPOINT + reopen, got %d", got)
	}
}

// TestCheckpointSharedSocial_NilDB_NoOp : passer nil doit être no-op (pas de panic).
func TestCheckpointSharedSocial_NilDB_NoOp(t *testing.T) {
	if err := CheckpointSharedSocial(context.Background(), nil); err != nil {
		t.Errorf("CheckpointSharedSocial(nil) doit retourner nil, got %v", err)
	}
}

// createMinimalSharedDB crée une DB shared_matches_v2 minimale avec match_registry
// vide — suffisant pour que SetMediaMatchAssociation puisse faire son lookup
// d'enrich sans paniquer (le Scan retourne 0 rows, mais la conn existe).
func createMinimalSharedDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.duckdb")
	db, err := OpenReadWrite(path)
	if err != nil {
		t.Fatalf("OpenReadWrite shared: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.SQLDb().Exec(`
		CREATE TABLE IF NOT EXISTS match_registry (
			match_id VARCHAR PRIMARY KEY,
			map_name VARCHAR, map_name_fr VARCHAR,
			pair_name VARCHAR, pair_name_fr VARCHAR
		);
	`); err != nil {
		t.Fatalf("seed shared match_registry: %v", err)
	}
	return db
}

// TestSetMediaMatchAssociation_PersistsAfterCheckpoint : exerce le path
// SetMediaMatchAssociation (Phase 3.2 patché) + reopen RO + vérifie persistance.
//
// Ce test valide concrètement que le CHECKPOINT ajouté ligne 43 de
// media_repo_writes.go fait son boulot.
func TestSetMediaMatchAssociation_PersistsAfterCheckpoint(t *testing.T) {
	socialDB := createSharedSocialSchemaForMediaTests(t)
	sharedDB := createMinimalSharedDB(t)

	pdb := &PlayerDB{
		SharedSocial: socialDB,
		Shared:       sharedDB,
		Gamertag:     "test-player",
		XUID:         "0000000000000001",
	}
	repo := NewMediaRepo(pdb)

	ctx := context.Background()
	// Le DELETE+INSERT+CHECKPOINT sur shared_social s'exécute.
	// L'enrich match_registry tourne sur la DB shared minimale (0 row → mapName/modeName nil),
	// pas d'erreur fatale.
	_, _, _ = repo.SetMediaMatchAssociation(ctx, "/test/media.mp4", "match-abc-123")

	// Reopen RO — si CHECKPOINT a marché, l'association est sur disque.
	got := reopenAndCount(t, socialDB,
		`SELECT COUNT(*) FROM media_match_associations_latest WHERE match_id = ?`,
		"match-abc-123",
	)
	if got != 1 {
		t.Errorf("attendu 1 association après SetMediaMatchAssociation + reopen, got %d", got)
	}
}

// NB (2026-08-04) : TestSetMediaLike_LegacyPath_PersistsAfterCheckpoint a été
// SUPPRIMÉ ici. Il vérifiait la durabilité de l'UPDATE media_files.liked, une
// colonne GLOBALE retirée du chemin de like avec le passage au par-viewer. La
// durabilité du chemin legacy est intégralement couverte par le test suivant :
// l'unique écriture de like y est désormais l'event append-only par liker.

// TestToggleSharedLike_FallbackPath_PersistsAfterCheckpoint : exerce
// ToggleSharedLike branche legacy (Persister == nil) + reopen.
func TestToggleSharedLike_FallbackPath_PersistsAfterCheckpoint(t *testing.T) {
	socialDB := createSharedSocialSchemaForMediaTests(t)
	pdb := &PlayerDB{SharedSocial: socialDB, Gamertag: "test-player"}
	// SocialPersister non setté → branche fallback.
	repo := NewMediaRepo(pdb)

	ctx := context.Background()
	if err := repo.ToggleSharedLike(ctx, "/test/media.mp4", "bob", "BobGamertag", true); err != nil {
		t.Fatalf("ToggleSharedLike add: %v", err)
	}

	got := reopenAndCount(t, socialDB,
		`SELECT COUNT(*) FROM media_likes_latest WHERE media_path = ? AND liker_slug = ? AND is_liked = TRUE`,
		"/test/media.mp4", "bob",
	)
	if got != 1 {
		t.Errorf("attendu 1 like persisté après ToggleSharedLike + reopen, got %d", got)
	}
}
