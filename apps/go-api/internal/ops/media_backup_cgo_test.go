//go:build cgo

// Package ops — media_backup_cgo_test.go : tests CGO pour les helpers media,
// listBaseTables, et findLatestParquetFiles.
//
// Lancer avec : go test ./internal/ops/ -v (CGO_ENABLED=1 requis)
package ops

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
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
