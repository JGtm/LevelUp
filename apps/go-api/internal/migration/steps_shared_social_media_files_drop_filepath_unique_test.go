//go:build integration

// Package migration — tests d'intégration CGO pour applyMediaFilesDropFilePathUnique
// (rebuild media_files sans UNIQUE(file_path), éradication ART #23046, blast MAX).
//
// Couvre les fixes issus de la vérification adversariale (2026-06-20) :
//   - UNIQUE retiré, PK + idx_mf_player_stem préservés, idx_mf_kind droppé ;
//   - data + types (TIMESTAMPTZ) préservés ;
//   - DEFAULT id (nextval) restauré → INSERT sans id obtient un id auto (pas NULL) ;
//   - autres DEFAULTs restaurés (indexed_at/created_at) ;
//   - continuité de séquence (pas de collision id) ;
//   - doublon file_path désormais accepté ; 3 UPDATE file_path OK ;
//   - idempotence ; orphan-recovery ; no-op sans table.

package migration

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// setupLegacyMediaFilesWithUnique reproduit le schéma LIVE prod (ensureMediaTables) :
// id INTEGER + séquence, file_path UNIQUE, idx_mf_player_stem + idx_mf_kind, colonnes
// complètes avec DEFAULTs. Seed rowCount rows.
func setupLegacyMediaFilesWithUnique(t *testing.T, rowCount int) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE SEQUENCE media_files_id_seq START 1;
		CREATE TABLE media_files (
			id INTEGER PRIMARY KEY DEFAULT nextval('media_files_id_seq'),
			player_slug VARCHAR,
			file_path VARCHAR UNIQUE,
			file_name VARCHAR,
			file_hash VARCHAR,
			kind VARCHAR,
			thumbnail_path VARCHAR,
			capture_start_utc TIMESTAMPTZ,
			capture_end_utc TIMESTAMPTZ,
			duration_seconds DOUBLE,
			status VARCHAR,
			mtime TIMESTAMPTZ,
			discord_notified BOOLEAN DEFAULT FALSE,
			indexed_at TIMESTAMPTZ DEFAULT NOW(),
			file_stem VARCHAR,
			file_ext VARCHAR,
			file_size INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_mf_player_stem ON media_files(player_slug, file_stem);
		CREATE INDEX idx_mf_kind ON media_files(kind);
	`); err != nil {
		t.Fatalf("create legacy: %v", err)
	}
	for i := 0; i < rowCount; i++ {
		if _, err := db.Exec(`
			INSERT INTO media_files (player_slug, file_path, file_name, file_stem, file_ext, kind, capture_start_utc)
			VALUES (?, ?, ?, ?, '.mp4', 'video', TIMESTAMPTZ '2026-01-01 12:00:00+00')
		`, "PlayerA", fmt.Sprintf("PlayerA/clip%d.mp4", i), fmt.Sprintf("clip%d.mp4", i), fmt.Sprintf("clip%d", i)); err != nil {
			t.Fatalf("seed row %d: %v", i, err)
		}
	}
	return db
}

func mediaFilesRowCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM media_files`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func mediaFilesIndexExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM duckdb_indexes() WHERE table_name='media_files' AND index_name=?`, name,
	).Scan(&n); err != nil {
		t.Fatalf("index check %s: %v", name, err)
	}
	return n > 0
}

// TestMediaFilesDropUnique_Basic — la migration retire UNIQUE, garde PK +
// idx_mf_player_stem, droppe idx_mf_kind, préserve les rows.
func TestMediaFilesDropUnique_Basic(t *testing.T) {
	db := setupLegacyMediaFilesWithUnique(t, 8)

	hasUnique, err := mediaFilesHasFilePathUnique(bootCtx(), db)
	if err != nil {
		t.Fatal(err)
	}
	if !hasUnique {
		t.Fatal("setup invalide : UNIQUE(file_path) absent avant migration")
	}

	if err := applyMediaFilesDropFilePathUnique(db); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	hasUnique, err = mediaFilesHasFilePathUnique(bootCtx(), db)
	if err != nil {
		t.Fatal(err)
	}
	if hasUnique {
		t.Error("UNIQUE(file_path) toujours présent après migration")
	}

	hasPK, err := hasPrimaryKey(db, "media_files")
	if err != nil {
		t.Fatal(err)
	}
	if !hasPK {
		t.Error("PRIMARY KEY(id) perdue après migration")
	}

	if !mediaFilesIndexExists(t, db, "idx_mf_player_stem") {
		t.Error("idx_mf_player_stem absent après migration (doit être préservé)")
	}
	if mediaFilesIndexExists(t, db, "idx_mf_kind") {
		t.Error("idx_mf_kind présent après migration (doit être droppé — kind muté)")
	}

	if n := mediaFilesRowCount(t, db); n != 8 {
		t.Errorf("rows = %d, want 8 (zéro perte)", n)
	}
}

// TestMediaFilesDropUnique_IDDefaultAndSequence — fix BLOCKER : après migration,
// un INSERT sans id obtient un id auto (DEFAULT nextval restauré) > max existant.
func TestMediaFilesDropUnique_IDDefaultAndSequence(t *testing.T) {
	db := setupLegacyMediaFilesWithUnique(t, 5)
	if err := applyMediaFilesDropFilePathUnique(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var maxBefore int64
	if err := db.QueryRow(`SELECT MAX(id) FROM media_files`).Scan(&maxBefore); err != nil {
		t.Fatal(err)
	}

	// INSERT sans id (les writers prod omettent id → dépendent du DEFAULT nextval).
	if _, err := db.Exec(`
		INSERT INTO media_files (player_slug, file_path, file_name, file_stem, kind)
		VALUES ('PlayerA', 'PlayerA/new.mp4', 'new.mp4', 'new', 'video')
	`); err != nil {
		t.Fatalf("INSERT sans id (DEFAULT nextval manquant ?): %v", err)
	}

	var newID sql.NullInt64
	if err := db.QueryRow(`SELECT id FROM media_files WHERE file_path='PlayerA/new.mp4'`).Scan(&newID); err != nil {
		t.Fatal(err)
	}
	if !newID.Valid {
		t.Fatal("id NULL après INSERT (DEFAULT nextval non restauré = BLOCKER)")
	}
	if newID.Int64 <= maxBefore {
		t.Errorf("collision séquence : nouvel id=%d <= max existant=%d", newID.Int64, maxBefore)
	}
}

// TestMediaFilesDropUnique_DefaultsRestored — les DEFAULTs perdus par le CTAS sont
// restaurés : un INSERT omettant discord_notified/indexed_at/created_at obtient les
// valeurs par défaut.
func TestMediaFilesDropUnique_DefaultsRestored(t *testing.T) {
	db := setupLegacyMediaFilesWithUnique(t, 1)
	if err := applyMediaFilesDropFilePathUnique(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO media_files (player_slug, file_path, file_name, file_stem, kind)
		VALUES ('PlayerA', 'PlayerA/def.mp4', 'def.mp4', 'def', 'video')
	`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var notified sql.NullBool
	var indexedNull, createdNull int
	if err := db.QueryRow(`
		SELECT discord_notified,
		       (indexed_at IS NULL)::INT,
		       (created_at IS NULL)::INT
		FROM media_files WHERE file_path='PlayerA/def.mp4'
	`).Scan(&notified, &indexedNull, &createdNull); err != nil {
		t.Fatal(err)
	}
	if !notified.Valid || notified.Bool {
		t.Errorf("discord_notified = %v, want FALSE (DEFAULT non restauré)", notified)
	}
	if indexedNull != 0 {
		t.Error("indexed_at NULL (DEFAULT NOW() non restauré)")
	}
	if createdNull != 0 {
		t.Error("created_at NULL (DEFAULT CURRENT_TIMESTAMP non restauré)")
	}
}

// TestMediaFilesDropUnique_TimestamptzPreserved — le CTAS préserve TIMESTAMP WITH
// TIME ZONE (pattern timezone canonique) — pas de downcast en TIMESTAMP.
func TestMediaFilesDropUnique_TimestamptzPreserved(t *testing.T) {
	db := setupLegacyMediaFilesWithUnique(t, 2)
	if err := applyMediaFilesDropFilePathUnique(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	var dtype string
	if err := db.QueryRow(`
		SELECT data_type FROM information_schema.columns
		WHERE table_name='media_files' AND column_name='capture_start_utc'
	`).Scan(&dtype); err != nil {
		t.Fatal(err)
	}
	if dtype != "TIMESTAMP WITH TIME ZONE" {
		t.Errorf("capture_start_utc data_type = %q, want TIMESTAMP WITH TIME ZONE (downcast = casse timezone canonique)", dtype)
	}
}

// TestMediaFilesDropUnique_DuplicateFilePathAllowed — après retrait de UNIQUE, deux
// lignes même file_path sont acceptées (la dédup passe en applicatif côté writers).
func TestMediaFilesDropUnique_DuplicateFilePathAllowed(t *testing.T) {
	db := setupLegacyMediaFilesWithUnique(t, 0)
	if err := applyMediaFilesDropFilePathUnique(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := db.Exec(`
			INSERT INTO media_files (player_slug, file_path, file_name, file_stem, kind)
			VALUES ('PlayerA', 'PlayerA/dup.mp4', 'dup.mp4', 'dup', 'video')
		`); err != nil {
			t.Fatalf("insert dup %d (UNIQUE pas retiré ?): %v", i, err)
		}
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM media_files WHERE file_path='PlayerA/dup.mp4'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("doublons file_path = %d, want 2 (UNIQUE retiré)", n)
	}
}

// TestMediaFilesDropUnique_UpdateFilePathWorks — les 3 UPDATE qui mutent file_path
// (le vecteur ART) s'exécutent sans erreur après retrait de la contrainte.
func TestMediaFilesDropUnique_UpdateFilePathWorks(t *testing.T) {
	db := setupLegacyMediaFilesWithUnique(t, 3)
	if err := applyMediaFilesDropFilePathUnique(db); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// UPDATE WHERE id (conversion / reconcile).
	if _, err := db.Exec(`UPDATE media_files SET file_path='PlayerA/renamed.webm', kind='video' WHERE id=1`); err != nil {
		t.Fatalf("UPDATE file_path WHERE id: %v", err)
	}
	// UPDATE WHERE file_path (finalizeMediaHLS).
	if _, err := db.Exec(`UPDATE media_files SET file_path='PlayerA/hls.m3u8' WHERE file_path='PlayerA/clip1.mp4'`); err != nil {
		t.Fatalf("UPDATE file_path WHERE file_path: %v", err)
	}
}

// TestMediaFilesDropUnique_Idempotent — re-appliquer la migration est un no-op
// (UNIQUE déjà retiré) et préserve les rows.
func TestMediaFilesDropUnique_Idempotent(t *testing.T) {
	db := setupLegacyMediaFilesWithUnique(t, 4)
	if err := applyMediaFilesDropFilePathUnique(db); err != nil {
		t.Fatalf("apply #1: %v", err)
	}
	n1 := mediaFilesRowCount(t, db)
	if err := applyMediaFilesDropFilePathUnique(db); err != nil {
		t.Fatalf("apply #2 (idempotent): %v", err)
	}
	if n2 := mediaFilesRowCount(t, db); n2 != n1 {
		t.Errorf("count change après re-apply : %d → %d", n1, n2)
	}
	hasUnique, _ := mediaFilesHasFilePathUnique(bootCtx(), db)
	if hasUnique {
		t.Error("UNIQUE réapparue après 2e apply")
	}
}

// TestMediaFilesDropUnique_OrphanRecovery — un crash mid-swap antérieur (media_files
// absente + media_files__rebuild présente) est réparé en tête de migration.
func TestMediaFilesDropUnique_OrphanRecovery(t *testing.T) {
	db := setupLegacyMediaFilesWithUnique(t, 6)
	// Simule un orphan RÉALISTE : __rebuild créé par CTAS (sans contrainte ni
	// dépendance, comme dans le swap), table principale droppée — l'état qu'un
	// crash mid-swap non transactionnel aurait laissé.
	if _, err := db.Exec(`CREATE TABLE media_files__rebuild AS SELECT * FROM media_files`); err != nil {
		t.Fatalf("create orphan __rebuild: %v", err)
	}
	if _, err := db.Exec(`DROP TABLE media_files`); err != nil {
		t.Fatalf("drop main: %v", err)
	}
	if err := applyMediaFilesDropFilePathUnique(db); err != nil {
		t.Fatalf("apply (recovery): %v", err)
	}
	// media_files restaurée + rows préservées + PK restaurée + UNIQUE absente.
	if n := mediaFilesRowCount(t, db); n != 6 {
		t.Errorf("rows après recovery = %d, want 6", n)
	}
	hasPK, _ := hasPrimaryKey(db, "media_files")
	if !hasPK {
		t.Error("PK absente après recovery (le rebuild post-recovery doit la restaurer)")
	}
	hasUnique, _ := mediaFilesHasFilePathUnique(bootCtx(), db)
	if hasUnique {
		t.Error("UNIQUE toujours présente après recovery+rebuild")
	}
}

// TestMediaFilesDropUnique_NoTable_NoOp — DB sans media_files → no-op propre.
func TestMediaFilesDropUnique_NoTable_NoOp(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := applyMediaFilesDropFilePathUnique(db); err != nil {
		t.Errorf("migration sans table devrait être no-op, got: %v", err)
	}
}
