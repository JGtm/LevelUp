//go:build cgo

// Package ops — media_backup_cgo_test.go : tests CGO pour les helpers media,
// listBaseTables, et findLatestParquetFiles.
//
// Lancer avec : go test ./internal/ops/ -v (CGO_ENABLED=1 requis)
package ops

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ─────────────────────────────────────────────────────────────────────────────
// fileHash
// ─────────────────────────────────────────────────────────────────────────────

func TestFileHash_Valid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.bin")
	if err := os.WriteFile(p, []byte("hello levelup"), 0600); err != nil {
		t.Fatal(err)
	}
	h, err := fileHash(p)
	if err != nil {
		t.Fatalf("fileHash inattendu: %v", err)
	}
	if len(h) != 16 {
		t.Errorf("hash length = %d, want 16", len(h))
	}
	// Idempotent
	h2, _ := fileHash(p)
	if h != h2 {
		t.Error("fileHash n'est pas idempotent")
	}
}

func TestFileHash_Absent(t *testing.T) {
	_, err := fileHash("/nonexistent/file.bin")
	if err == nil {
		t.Error("expected error pour fichier absent")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// walkMediaDir
// ─────────────────────────────────────────────────────────────────────────────

func TestWalkMediaDir_Absent(t *testing.T) {
	files, err := walkMediaDir("/nonexistent/media")
	if err != nil {
		t.Errorf("inattendu: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil pour dir absent, got %v", files)
	}
}

func TestWalkMediaDir_Empty(t *testing.T) {
	dir := t.TempDir()
	files, err := walkMediaDir(dir)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 fichiers, got %d", len(files))
	}
}

func TestWalkMediaDir_WithMedia(t *testing.T) {
	dir := t.TempDir()
	// Créer des fichiers avec extensions supportées
	for _, name := range []string{"clip.mp4", "screenshot.png", "doc.pdf"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	files, err := walkMediaDir(dir)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	// clip.mp4 + screenshot.png = 2 fichiers supportés, doc.pdf ignoré
	if len(files) != 2 {
		t.Errorf("expected 2 fichiers media, got %d: %v", len(files), files)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// listBaseTables (CGO / DuckDB)
// ─────────────────────────────────────────────────────────────────────────────

func TestListBaseTables_Empty(t *testing.T) {
	_, db := openDiagDB(t)
	tables, err := listBaseTables(db)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(tables))
	}
}

func TestListBaseTables_WithTables(t *testing.T) {
	_, db := openDiagDB(t)
	for _, q := range []string{
		"CREATE TABLE alpha (id INTEGER)",
		"CREATE TABLE beta (name TEXT)",
		"CREATE VIEW v_alpha AS SELECT * FROM alpha",
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	tables, err := listBaseTables(db)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	// alpha + beta mais PAS la vue v_alpha
	if len(tables) != 2 {
		t.Errorf("expected 2 tables, got %d: %v", len(tables), tables)
	}
	for _, name := range tables {
		if name == "v_alpha" {
			t.Error("les vues ne doivent pas apparaître dans listBaseTables")
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ensureMediaTables (CGO / DuckDB)
// ─────────────────────────────────────────────────────────────────────────────

func TestEnsureMediaTables_CreatesTable(t *testing.T) {
	_, db := openDiagDB(t)
	if err := ensureMediaTables(db); err != nil {
		t.Fatalf("ensureMediaTables inattendu: %v", err)
	}
	// Vérifier que les tables existent via information_schema
	for _, table := range []string{"media_files", "media_match_associations"} {
		var cnt int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='main' AND table_name=?", table,
		).Scan(&cnt)
		if err != nil || cnt == 0 {
			t.Errorf("table %q absente après ensureMediaTables", table)
		}
	}
}

func TestEnsureMediaTables_Idempotent(t *testing.T) {
	_, db := openDiagDB(t)
	for i := 0; i < 3; i++ {
		if err := ensureMediaTables(db); err != nil {
			t.Fatalf("ensureMediaTables iteration %d: %v", i, err)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// loadKnownHashes (CGO / DuckDB)
// ─────────────────────────────────────────────────────────────────────────────

func TestLoadKnownHashes_NoTable(t *testing.T) {
	_, db := openDiagDB(t)
	// Sans table media_files → erreur SQL attendue
	_, err := loadKnownHashes(db)
	if err == nil {
		t.Error("expected error (table absente)")
	}
}

func TestLoadKnownHashes_EmptyTable(t *testing.T) {
	_, db := openDiagDB(t)
	if err := ensureMediaTables(db); err != nil {
		t.Fatal(err)
	}
	hashes, err := loadKnownHashes(db)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(hashes) != 0 {
		t.Errorf("expected 0 hashes, got %d", len(hashes))
	}
}

func TestLoadKnownHashes_WithData(t *testing.T) {
	_, db := openDiagDB(t)
	if err := ensureMediaTables(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO media_files (id, file_path, file_hash, kind) VALUES (1, '/a.mp4', 'abc123', 'video'), (2, '/b.mp4', 'def456', 'video')`); err != nil {
		t.Fatal(err)
	}
	hashes, err := loadKnownHashes(db)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if len(hashes) != 2 {
		t.Errorf("expected 2 hashes, got %d", len(hashes))
	}
	if !hashes["abc123"] || !hashes["def456"] {
		t.Errorf("hashes manquants: %v", hashes)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// findLatestParquetFiles (filesystem pur)
// ─────────────────────────────────────────────────────────────────────────────

func TestFindLatestParquetFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	files, ts, err := findLatestParquetFiles(dir)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if ts != "" {
		t.Errorf("expected empty ts, got %q", ts)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 fichiers, got %d", len(files))
	}
}

func TestFindLatestParquetFiles_WithMetadata(t *testing.T) {
	dir := t.TempDir()
	ts := "20250101_120000"
	// Créer un backup_metadata_<ts>.json et des .parquet correspondants
	metaPath := filepath.Join(dir, "backup_metadata_"+ts+".json")
	if err := os.WriteFile(metaPath, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	parquetPath := filepath.Join(dir, "player_match_enrichment_"+ts+".parquet")
	if err := os.WriteFile(parquetPath, []byte("fake parquet"), 0600); err != nil {
		t.Fatal(err)
	}

	files, gotTS, err := findLatestParquetFiles(dir)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if gotTS != ts {
		t.Errorf("ts = %q, want %q", gotTS, ts)
	}
	if _, ok := files["player_match_enrichment"]; !ok {
		t.Errorf("table manquante dans files: %v", files)
	}
}

func TestFindLatestParquetFiles_FallbackParquet(t *testing.T) {
	dir := t.TempDir()
	ts := "20250201_090000"
	// Pas de metadata JSON — uniquement un .parquet (fallback)
	parquetPath := filepath.Join(dir, "sessions_"+ts+".parquet")
	if err := os.WriteFile(parquetPath, []byte("fake"), 0600); err != nil {
		t.Fatal(err)
	}
	_, gotTS, err := findLatestParquetFiles(dir)
	if err != nil {
		t.Fatalf("inattendu: %v", err)
	}
	if gotTS == "" {
		t.Error("expected non-empty ts depuis fallback .parquet")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// IndexMedia — concurrence (mutex par chemin DB)
// ─────────────────────────────────────────────────────────────────────────────

// TestIndexMedia_ConcurrentSameDB_NoRace vérifie que deux IndexMedia simultanés
// sur la même shared_social.duckdb ne produisent pas d'erreur ATTACH/DETACH.
// Sans le mutex indexLock, duckdb-go partage la même instance interne et
// le second ATTACH échoue avec "already attached".
func TestIndexMedia_ConcurrentSameDB_NoRace(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")

	// Deux sous-répertoires avec chacun un fichier média
	for _, sub := range []string{"player1", "player2"} {
		capDir := filepath.Join(dir, sub)
		if err := os.MkdirAll(capDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(capDir, "clip.mp4"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	makeOpts := func(sub string) MediaIndexOptions {
		return MediaIndexOptions{
			SharedSocialDBPath: dbPath,
			CapturesDir:        filepath.Join(dir, sub),
			BufferMin:          2,
		}
	}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, sub := range []string{"player1", "player2"} {
		wg.Add(1)
		go func(idx int, s string) {
			defer wg.Done()
			_, errs[idx] = IndexMedia(makeOpts(s))
		}(i, sub)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: IndexMedia error: %v", i, err)
		}
	}
}

// TestIndexMedia_ConcurrentSameDB_SameDir simule deux uploads depuis le même joueur
// (deux navigateurs) : même DB et même répertoire, trois goroutines simultanées.
func TestIndexMedia_ConcurrentSameDB_SameDir(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shared_social.duckdb")
	capDir := filepath.Join(dir, "captures")
	if err := os.MkdirAll(capDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(capDir, "clip.mp4"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	optsVal := MediaIndexOptions{
		SharedSocialDBPath: dbPath,
		CapturesDir:        capDir,
		BufferMin:          2,
	}

	var wg sync.WaitGroup
	errs := make([]error, 3)
	for i := range 3 {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = IndexMedia(optsVal)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: IndexMedia error: %v", i, err)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// insertMediaFile — chaîne de priorité des timestamps
// ─────────────────────────────────────────────────────────────────────────────

// openInsertTestDB crée une DB DuckDB temporaire avec les tables médias.
func openInsertTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.duckdb")
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := ensureMediaTables(db); err != nil {
		db.Close()
		t.Fatalf("ensureMediaTables: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, dir
}

// readCaptureAt retourne capture_start_utc en UTC (string) pour le hash donné.
func readCaptureAt(t *testing.T, db *sql.DB, hash string) string {
	t.Helper()
	var v string
	// AT TIME ZONE 'UTC' force la représentation UTC indépendamment du TimeZone de session.
	if err := db.QueryRow(
		"SELECT CAST(capture_start_utc AT TIME ZONE 'UTC' AS VARCHAR) FROM media_files WHERE file_hash = ?", hash,
	).Scan(&v); err != nil {
		t.Fatalf("QueryRow capture_start_utc (hash=%s): %v", hash, err)
	}
	return v
}

// TestInsertMediaFile_Priority1_XboxFilename vérifie que le datetime Xbox
// est utilisé en priorité même si un captureTimeUnix client est fourni.
func TestInsertMediaFile_Priority1_XboxFilename(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Skip("timezone Europe/Paris non disponible")
	}
	db, dir := openInsertTestDB(t)

	// Nom Xbox (CET décembre = UTC+1) : 21:30:45 Paris → 20:30:45 UTC
	name := "Halo Infinite 2024.12.15 - 21.30.45.01.mp4"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// captureTimeUnix pointe sur 2000-01-01 (doit être ignoré au profit du filename)
	clientTs := int64(946684800)
	if err := insertMediaFile(db, path, "hash_p1", "spartan", &clientTs, loc); err != nil {
		t.Fatalf("insertMediaFile: %v", err)
	}

	v := readCaptureAt(t, db, "hash_p1")
	// 20:30:45 UTC (pas 2000-01-01)
	if !strings.Contains(v, "20:30:45") {
		t.Errorf("Priority1: capture_start_utc = %q, want \"20:30:45\" UTC (Xbox filename)", v)
	}
	if strings.Contains(v, "2000") {
		t.Errorf("Priority1: client ts (2000-01-01) ne doit pas être utilisé, got %q", v)
	}
}

// TestInsertMediaFile_Priority2_ClientTimestamp vérifie que file.lastModified
// est utilisé quand le filename ne matche pas le pattern Xbox.
func TestInsertMediaFile_Priority2_ClientTimestamp(t *testing.T) {
	db, dir := openInsertTestDB(t)

	name := "OBS-recording.mp4"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2024-06-01 12:00:00 UTC
	clientTs := int64(1717243200)
	if err := insertMediaFile(db, path, "hash_p2", "spartan", &clientTs, nil); err != nil {
		t.Fatalf("insertMediaFile: %v", err)
	}

	v := readCaptureAt(t, db, "hash_p2")
	if !strings.Contains(v, "2024-06-01") || !strings.Contains(v, "12:00:00") {
		t.Errorf("Priority2: capture_start_utc = %q, want \"2024-06-01 12:00:00\" UTC", v)
	}
}

// TestInsertMediaFile_Priority3_MtimeFallback vérifie le fallback sur mtime
// quand ni le filename ni le captureTimeUnix ne sont disponibles.
func TestInsertMediaFile_Priority3_MtimeFallback(t *testing.T) {
	db, dir := openInsertTestDB(t)

	name := "OBS-fallback.mp4"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	beforeWrite := time.Now().UTC().Add(-time.Second)
	if err := insertMediaFile(db, path, "hash_p3", "spartan", nil, nil); err != nil {
		t.Fatalf("insertMediaFile: %v", err)
	}
	afterWrite := time.Now().UTC().Add(time.Second)

	v := readCaptureAt(t, db, "hash_p3")
	// Vérifier que la date extraite est dans la fenêtre d'écriture.
	parsed, err := time.Parse("2006-01-02 15:04:05-07:00", v)
	if err != nil {
		// DuckDB peut retourner "+00" sans les deux points → essayer d'autres formats
		parsed, err = time.Parse("2006-01-02 15:04:05+00", v)
	}
	if err == nil {
		if parsed.Before(beforeWrite) || parsed.After(afterWrite) {
			t.Errorf("Priority3: mtime fallback %q hors fenêtre [%v, %v]", v, beforeWrite, afterWrite)
		}
	} else {
		// Vérification minimale : valeur non vide
		if v == "" {
			t.Error("Priority3: capture_start_utc ne doit pas être vide (fallback mtime)")
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AssociateMediaWithMatches — timezone SET TimeZone + fenêtre BETWEEN
// ─────────────────────────────────────────────────────────────────────────────

// TestAssociateMediaWithMatches_TimezoneWindow vérifie que la fenêtre BETWEEN fonctionne
// avec start_time_utc (TIMESTAMPTZ UTC garanti, post-migration add_start_time_utc_to_match_registry).
// Scénario : match CEST 22:00→22:15 Paris = 20:00→20:15 UTC, capture à 20:08 UTC.
func TestAssociateMediaWithMatches_TimezoneWindow(t *testing.T) {
	dir := t.TempDir()

	// DB de la galerie (media_files + associations)
	socialPath := filepath.Join(dir, "shared_social.duckdb")
	dbSocial, err := sql.Open("duckdb", socialPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbSocial.Close()
	if err := ensureMediaTables(dbSocial); err != nil {
		t.Fatal(err)
	}

	// DB des matchs : match_registry avec start_time_utc/end_time_utc (post-migration).
	matchesPath := filepath.Join(dir, "shared_matches.duckdb")
	dbMatches, err := sql.Open("duckdb", matchesPath)
	if err != nil {
		t.Fatal(err)
	}
	defer dbMatches.Close()

	_, err = dbMatches.Exec(`
		CREATE TABLE match_registry (
			match_id       VARCHAR PRIMARY KEY,
			start_time     TIMESTAMP,
			end_time       TIMESTAMP,
			start_time_utc TIMESTAMPTZ,
			end_time_utc   TIMESTAMPTZ
		)
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Match CEST (été, UTC+2) : 22:00→22:15 Paris naïf = 20:00→20:15 UTC.
	// start_time_utc/end_time_utc stockent la valeur UTC corrigée.
	_, err = dbMatches.Exec(`
		INSERT INTO match_registry VALUES (
			'match-tz-test',
			TIMESTAMP '2024-07-20 22:00:00',
			TIMESTAMP '2024-07-20 22:15:00',
			TIMESTAMPTZ '2024-07-20 20:00:00+00',
			TIMESTAMPTZ '2024-07-20 20:15:00+00'
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	dbMatches.Close()

	// Insérer un média capturé à 20:08 UTC (dans la fenêtre UTC)
	captureUTC := time.Date(2024, 7, 20, 20, 8, 0, 0, time.UTC)
	_, err = dbSocial.Exec(`
		INSERT INTO media_files (player_slug, file_path, file_name, file_hash, kind, capture_start_utc)
		VALUES ('spartan', '/cap/clip.mp4', 'clip.mp4', 'hash-tz', 'video', ?)
	`, captureUTC)
	if err != nil {
		t.Fatalf("INSERT media_files: %v", err)
	}

	n, err := AssociateMediaWithMatches(dbSocial, matchesPath, 2, "Europe/Paris")
	if err != nil {
		t.Fatalf("AssociateMediaWithMatches: %v", err)
	}
	if n != 1 {
		t.Errorf("AssociateMediaWithMatches = %d associations, want 1 (start_time_utc UTC)", n)
	}
}
