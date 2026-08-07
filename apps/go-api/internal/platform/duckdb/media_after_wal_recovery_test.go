//go:build cgo

// Package duckdb — media_after_wal_recovery_test.go (ADR 0021 Phase 4.1).
//
// Tests d'intégration qui simulent un cycle complet "écriture → quarantaine
// WAL → recovery → re-query" pour vérifier que les data utilisateur
// (likes, favoris, associations) survivent à la procédure de recovery.

package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupWALRecoveryFixture crée une DB shared_social fraîche, fait quelques
// writes via les paths legacy (chemin direct + CHECKPOINT), puis simule un
// scénario "WAL orphelin" en copiant un WAL corrompu à côté du .duckdb.
//
// Retourne le path de la DB. Le test peut ensuite appeler
// openSharedSocialWithWALRecovery pour valider la séquence quarantaine + retry.
func setupWALRecoveryFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")

	socialDB, err := OpenReadWriteShared(dbPath, "")
	if err != nil {
		t.Fatalf("init social: %v", err)
	}
	// Seed minimal schema + data.
	if _, err := socialDB.SQLDb().Exec(`
		CREATE SEQUENCE IF NOT EXISTS media_id_seq START 1;
		CREATE TABLE IF NOT EXISTS media_files (
			id INTEGER DEFAULT nextval('media_id_seq') PRIMARY KEY,
			player_slug VARCHAR,
			file_path VARCHAR UNIQUE,
			file_name VARCHAR,
			kind VARCHAR DEFAULT 'video'
		);
		CREATE TABLE IF NOT EXISTS media_likes (
			media_path VARCHAR NOT NULL,
			liker_slug VARCHAR NOT NULL,
			liker_gamertag VARCHAR,
			liked_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (media_path, liker_slug)
		);
		CREATE TABLE IF NOT EXISTS match_favorites (
			player_slug VARCHAR NOT NULL,
			match_id VARCHAR NOT NULL,
			favorited_at TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
			PRIMARY KEY (player_slug, match_id)
		);
	`); err != nil {
		t.Fatalf("seed schema: %v", err)
	}
	if _, err := socialDB.SQLDb().Exec(`
		INSERT INTO media_files (player_slug, file_path, file_name) VALUES
		('alice', '/m/a.mp4', 'a.mp4'),
		('bob',   '/m/b.mp4', 'b.mp4'),
		('carol', '/m/c.mp4', 'c.mp4'),
		('dave',  '/m/d.mp4', 'd.mp4');
	`); err != nil {
		t.Fatalf("seed data: %v", err)
	}
	// Force CHECKPOINT pour que tout aille dans le fichier principal.
	if _, err := socialDB.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("seed checkpoint: %v", err)
	}
	if err := socialDB.Close(); err != nil {
		t.Fatalf("seed close: %v", err)
	}
	return dbPath
}

// simulateWALOrphan crée un .wal "synthétique" à côté du .duckdb en injectant
// le mock opener qui retourne une erreur WAL replay au 1er essai. Retourne
// une fonction restore qui rétablit l'opener original.
//
// Différence vs Phase 0 (fixture réelle DuckDB-corrupted) : ici on teste la
// SÉQUENCE recovery sans dépendre du bug DuckDB exact. La fixture Phase 0
// teste le scénario réel ; Phase 4.1 teste les invariants applicatifs
// (data survit, comportement correct sous quarantaine).
func simulateWALOrphan(t *testing.T, dbPath string) func() {
	t.Helper()
	// Crée un .wal "orphelin" synthétique pour donner à la fonction de
	// quarantaine quelque chose à renommer.
	walPath := dbPath + ".wal"
	if err := os.WriteFile(walPath, []byte("synthetic-orphan-wal"), 0o644); err != nil {
		t.Fatalf("create synthetic wal: %v", err)
	}
	original := openSharedSocialFn
	callCount := 0
	openSharedSocialFn = func(p string, tz ...string) (*DB, error) {
		callCount++
		if callCount == 1 {
			return nil, fmt.Errorf(
				`INTERNAL Error: %s "%s.wal": Calling DatabaseManager::GetDefaultDatabase with no default database set`,
				errWALReplayMarker, p,
			)
		}
		return OpenReadWriteShared(p, tz...)
	}
	return func() { openSharedSocialFn = original }
}

// TestMediaService_LikeSurvives_WALRecovery (ADR 0021 Phase 4.1).
//
// Scénario : un like est inséré sur la DB → simulate WAL orphelin → recovery →
// re-query : le like doit toujours être présent.
//
// Le like porte sur media_likes (une ligne PAR LIKER) : depuis le 2026-08-04
// c'est le seul support de l'état du cœur, media_files.liked — le booléen global
// que ce test mutait auparavant — ayant été droppé du schéma.
func TestMediaService_LikeSurvives_WALRecovery(t *testing.T) {
	dbPath := setupWALRecoveryFixture(t)

	// Like via path direct (simule SetMediaLike sans Persister).
	socialDB, err := OpenReadWriteShared(dbPath, "")
	if err != nil {
		t.Fatalf("open pre-like: %v", err)
	}
	if _, err := socialDB.SQLDb().Exec(
		`INSERT INTO media_likes (media_path, liker_slug, liker_gamertag) VALUES (?, ?, ?)`,
		"/m/a.mp4", "alice", "alice",
	); err != nil {
		t.Fatalf("set like: %v", err)
	}
	if _, err := socialDB.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint pre-recovery: %v", err)
	}
	if err := socialDB.Close(); err != nil {
		t.Fatalf("close pre-recovery: %v", err)
	}

	// Simule WAL orphelin + recovery.
	restore := simulateWALOrphan(t, dbPath)
	defer restore()

	db := openSharedSocialWithWALRecovery(context.Background(), dbPath, "", "test-like-survives")
	if db == nil {
		t.Fatal("recovery devrait réussir (opener mock retourne OK au 2e essai)")
	}
	defer db.Close()

	var likes int
	if err := db.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM media_likes WHERE media_path = ? AND liker_slug = ?`,
		"/m/a.mp4", "alice",
	).Scan(&likes); err != nil {
		t.Fatalf("query post-recovery: %v", err)
	}
	if likes != 1 {
		t.Errorf("like perdu après cycle WAL recovery (%d, want 1) — la donnée pré-recovery n'a pas survécu", likes)
	}
}

// TestMediaService_FavoriteSurvives_WALRecovery : idem pour match_favorites.
func TestMediaService_FavoriteSurvives_WALRecovery(t *testing.T) {
	dbPath := setupWALRecoveryFixture(t)

	socialDB, err := OpenReadWriteShared(dbPath, "")
	if err != nil {
		t.Fatalf("open pre-favorite: %v", err)
	}
	if _, err := socialDB.SQLDb().Exec(
		`INSERT INTO match_favorites (player_slug, match_id) VALUES (?, ?)`,
		"alice", "match-xyz",
	); err != nil {
		t.Fatalf("set favorite: %v", err)
	}
	if _, err := socialDB.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint pre-recovery: %v", err)
	}
	if err := socialDB.Close(); err != nil {
		t.Fatalf("close pre-recovery: %v", err)
	}

	restore := simulateWALOrphan(t, dbPath)
	defer restore()

	db := openSharedSocialWithWALRecovery(context.Background(), dbPath, "", "test-fav-survives")
	if db == nil {
		t.Fatal("recovery devrait réussir")
	}
	defer db.Close()

	var n int
	if err := db.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM match_favorites WHERE player_slug = ? AND match_id = ?`,
		"alice", "match-xyz",
	).Scan(&n); err != nil {
		t.Fatalf("query post-recovery: %v", err)
	}
	if n != 1 {
		t.Errorf("favori perdu après cycle WAL recovery (got %d, want 1)", n)
	}
}

// TestMediaService_LikeAtomic_DuringQuarantine : lance un like (via le path
// avec CHECKPOINT immédiat) PENDANT que la DB vient d'être quarantinée (et
// donc rouverte). Doit soit réussir, soit échouer proprement — JAMAIS
// corrompre l'état.
func TestMediaService_LikeAtomic_DuringQuarantine(t *testing.T) {
	dbPath := setupWALRecoveryFixture(t)

	restore := simulateWALOrphan(t, dbPath)
	defer restore()

	db := openSharedSocialWithWALRecovery(context.Background(), dbPath, "", "test-atomic")
	if db == nil {
		t.Fatal("recovery setup KO")
	}
	defer db.Close()

	// Like atomique post-recovery.
	tx, err := db.SQLDb().BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.ExecContext(context.Background(),
		`INSERT INTO media_likes (media_path, liker_slug, liker_gamertag) VALUES (?, ?, ?)`,
		"/m/b.mp4", "bob", "bob",
	); err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert like: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	// CHECKPOINT immédiat (équivalent CommitWithCheckpoint).
	if _, err := db.SQLDb().Exec("CHECKPOINT"); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	// Lecture in-place via la même conn (évite le conflit RW+RO côté DuckDB).
	const qLikeCount = `SELECT COUNT(*) FROM media_likes WHERE media_path = ? AND liker_slug = ?`
	var likes int
	if err := db.SQLDb().QueryRow(qLikeCount, "/m/b.mp4", "bob").Scan(&likes); err != nil {
		t.Fatalf("query post-checkpoint: %v", err)
	}
	if likes != 1 {
		t.Errorf("like post-quarantine non persisté (lecture in-place, %d want 1)", likes)
	}

	// Bonus : ferme la conn RW et reopen RO pour preuve disque.
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	roDB, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		t.Fatalf("reopen RO: %v", err)
	}
	defer roDB.Close()
	var likesRO int
	if err := roDB.QueryRow(qLikeCount, "/m/b.mp4", "bob").Scan(&likesRO); err != nil {
		t.Fatalf("query RO: %v", err)
	}
	if likesRO != 1 {
		t.Errorf("like post-quarantine non persisté sur disque après close (%d want 1)", likesRO)
	}
}

// TestMediaService_4Players_ConcurrentLikes_AfterRecovery : 4 goroutines
// (1 par "joueur") font des likes en parallèle sur 4 médias distincts
// APRÈS la recovery. Vérifie qu'aucune data race + tous les likes persistent.
func TestMediaService_4Players_ConcurrentLikes_AfterRecovery(t *testing.T) {
	dbPath := setupWALRecoveryFixture(t)

	restore := simulateWALOrphan(t, dbPath)
	defer restore()

	db := openSharedSocialWithWALRecovery(context.Background(), dbPath, "", "test-4players")
	if db == nil {
		t.Fatal("recovery setup KO")
	}
	defer db.Close()

	players := []struct {
		slug      string
		mediaPath string
	}{
		{"alice", "/m/a.mp4"},
		{"bob", "/m/b.mp4"},
		{"carol", "/m/c.mp4"},
		{"dave", "/m/d.mp4"},
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(players))
	for _, p := range players {
		wg.Add(1)
		go func(slug, mediaPath string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := db.SQLDb().ExecContext(ctx,
				`INSERT INTO media_likes (media_path, liker_slug, liker_gamertag) VALUES (?, ?, ?)`,
				mediaPath, slug, slug,
			); err != nil {
				errs <- fmt.Errorf("%s: %w", slug, err)
			}
		}(p.slug, p.mediaPath)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent like error: %v", err)
		}
	}

	// Compter les likes.
	var n int
	if err := db.SQLDb().QueryRow(`SELECT COUNT(*) FROM media_likes`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != len(players) {
		t.Errorf("attendu %d likes concurrents persistés, got %d", len(players), n)
	}
}
