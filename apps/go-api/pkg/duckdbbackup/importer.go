package duckdbbackup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2" // DuckDB driver
)

// ImportResult summarises what was imported from a staging directory.
type ImportResult struct {
	Restored []string // keys successfully imported
	Skipped  []string // keys with no target path mapping
	// FallbackTables : tables ("clé:table") absentes du schéma canonique de leur base,
	// importées par CREATE TABLE AS — donc SANS clé ni défaut. Chacune est journalisée en WARN.
	FallbackTables []string
	// SourceDDLTables : tables ("clé:table") absentes du schéma canonique, recréées avec leurs
	// clés depuis le DDL que porte la sauvegarde.
	SourceDDLTables []string
	// DroppedRows : lignes écartées à l'import (doublons d'une clé du schéma canonique, clé
	// primaire NULL). Chaque table concernée est journalisée en WARN avec son compte.
	DroppedRows int64
	DurationMs  int64
}

// CanonicalSchema pose le schéma canonique d'une base AVANT l'import de ses tables, puis la
// finalise APRÈS. C'est par lui que la base restaurée retrouve ses clés primaires et UNIQUE,
// ses défauts, ses séquences, ses index et ses vues : un Parquet ne porte que des lignes.
//
// Provision reçoit la base neuve (vide). Elle rend ErrNoCanonicalSchema quand la clé n'a pas
// de schéma connu : toutes les tables de la base sont alors importées par le repli (WARN).
type CanonicalSchema interface {
	Provision(ctx context.Context, key string, db *sql.DB) error
	Finalize(ctx context.Context, key string, db *sql.DB) error
}

// ErrNoCanonicalSchema : la clé de sauvegarde ne correspond à aucun schéma connu.
var ErrNoCanonicalSchema = errors.New("aucun schéma canonique pour cette clé de sauvegarde")

// tsPattern strips the timestamp suffix appended by ExportTarget to table filenames.
// e.g. "match_registry_20260525_132028.parquet" → "match_registry"
var tsPattern = regexp.MustCompile(`_\d{8}_\d{6}\.parquet$`)

// importer porte l'état d'un import de staging.
type importer struct {
	stagingDir string
	keyToPath  func(string) string
	schema     CanonicalSchema
	dryRun     bool
	result     *ImportResult
	ddl        map[string]string // DDL de la sauvegarde de la clé en cours (table → CREATE TABLE)
}

// ImportFromStaging walks stagingDir and imports each key's Parquet files into DuckDB.
//
// keyToPath maps a backup key (e.g. "halo_infinite:shared_matches_v2") to the
// target DuckDB file path. Keys with no mapping (empty string) are skipped.
//
// schema pose le schéma canonique de chaque base avant d'y verser ses tables (obligatoire :
// sans lui, une base restaurée perdrait toutes ses clés).
//
// When dryRun is true the function logs what it would do but makes no changes.
func ImportFromStaging(ctx context.Context, stagingDir string, keyToPath func(string) string,
	schema CanonicalSchema, dryRun bool) (*ImportResult, error) {
	if schema == nil {
		return nil, errors.New("import: schéma canonique requis (une base restaurée sans lui perd ses clés)")
	}
	start := time.Now()
	im := &importer{stagingDir: stagingDir, keyToPath: keyToPath, schema: schema, dryRun: dryRun,
		result: &ImportResult{}}

	top, err := os.ReadDir(stagingDir)
	if err != nil {
		return nil, fmt.Errorf("lecture staging: %w", err)
	}

	for _, e := range top {
		if !e.IsDir() {
			continue
		}
		if walkErr := im.walkKeyDirs(ctx, e.Name()); walkErr != nil {
			return im.result, walkErr
		}
	}

	im.result.DurationMs = time.Since(start).Milliseconds()
	return im.result, nil
}

// walkKeyDirs recurses the staging tree and imports leaf directories
// (directories that directly contain .parquet files).
func (im *importer) walkKeyDirs(ctx context.Context, rel string) error {
	full := filepath.Join(im.stagingDir, rel)
	entries, err := os.ReadDir(full)
	if err != nil {
		slog.WarnContext(ctx, "restore: dossier de staging illisible — ignoré", "dir", full, "err", err)
		return nil
	}

	hasParquet := false
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".parquet") {
			hasParquet = true
			break
		}
	}

	if hasParquet {
		// Leaf: convert filesystem path separators back to ":" to reconstruct the key.
		key := strings.ReplaceAll(filepath.ToSlash(rel), "/", ":")
		return im.importLeaf(ctx, key, full)
	}

	// Not a leaf — recurse.
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := im.walkKeyDirs(ctx, filepath.Join(rel, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// importLeaf importe le dossier Parquet d'une clé dans la base que keyToPath lui associe.
func (im *importer) importLeaf(ctx context.Context, key, parquetDir string) error {
	dbPath := im.keyToPath(key)
	if dbPath == "" {
		slog.WarnContext(ctx, "restore: clé sans mapping — ignorée", "key", key)
		im.result.Skipped = append(im.result.Skipped, key)
		return nil
	}
	if im.dryRun {
		slog.InfoContext(ctx, "restore: [dry-run] importerait", "key", key, "db", dbPath)
		im.result.Restored = append(im.result.Restored, key)
		return nil
	}
	if err := im.importKeyDB(ctx, key, parquetDir, dbPath); err != nil {
		return fmt.Errorf("import %s: %w", key, err)
	}
	slog.InfoContext(ctx, "restore: DB importée", "key", key, "db", dbPath)
	im.result.Restored = append(im.result.Restored, key)
	return nil
}

// importKeyDB imports all Parquet files from parquetDir into a fresh DuckDB at dbPath.
// The existing file (if any) is renamed to .bak; removed on success, restored on failure.
func (im *importer) importKeyDB(ctx context.Context, key, parquetDir, dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	bakPath := dbPath + ".bak"
	hasBak := false
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Rename(dbPath, bakPath); err != nil {
			return fmt.Errorf("sauvegarde existant: %w", err)
		}
		hasBak = true
	}

	if err := im.doImport(ctx, key, parquetDir, dbPath); err != nil {
		if hasBak {
			im.restoreBackup(ctx, dbPath, bakPath)
		}
		return err
	}

	if hasBak {
		if err := os.Remove(bakPath); err != nil {
			slog.WarnContext(ctx, "restore: ancienne base (.bak) non supprimée", "path", bakPath, "err", err)
		}
	}
	return nil
}

// restoreBackup remet en place la base d'avant l'import après un échec.
func (im *importer) restoreBackup(ctx context.Context, dbPath, bakPath string) {
	for _, p := range []string{dbPath, dbPath + ".wal"} {
		if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
			slog.ErrorContext(ctx, "restore: base partielle non supprimée", "path", p, "err", err)
		}
	}
	if err := os.Rename(bakPath, dbPath); err != nil {
		slog.ErrorContext(ctx, "restore: ancienne base NON remise en place", "bak", bakPath, "db", dbPath, "err", err)
	}
}

// doImport crée la base neuve à dbPath, y pose le schéma canonique, y verse chaque table
// Parquet puis la finalise.
func (im *importer) doImport(ctx context.Context, key, parquetDir, dbPath string) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("créer DuckDB %s: %w", dbPath, err)
	}
	defer func() { _ = db.Close() }()

	canonical := true
	if err := im.schema.Provision(ctx, key, db); err != nil {
		if !errors.Is(err, ErrNoCanonicalSchema) {
			return fmt.Errorf("schéma canonique: %w", err)
		}
		canonical = false
		slog.WarnContext(ctx, "restore: aucun schéma canonique pour cette base — tables importées sans clé",
			"key", key, "db", dbPath)
	}

	if im.ddl, err = loadTableDDL(parquetDir); err != nil {
		return err
	}
	files, err := parquetFiles(parquetDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := im.importTable(ctx, db, key, f.table, f.path); err != nil {
			return fmt.Errorf("table %q: %w", f.table, err)
		}
	}
	if canonical {
		if err := im.schema.Finalize(ctx, key, db); err != nil {
			return fmt.Errorf("finalisation du schéma: %w", err)
		}
	}
	slog.InfoContext(ctx, "restore: import terminé", "tables", len(files), "db", dbPath)
	return nil
}

// parquetFile : une table à importer et son fichier.
type parquetFile struct{ table, path string }

// parquetFiles liste, pour chaque table, le fichier Parquet le plus récent de dir
// (récursivement), triés par nom de table. Le staging peut garder plusieurs exports d'une même
// table (un par cycle, `<table>_<horodatage>.parquet`) : seul le dernier fait foi.
func parquetFiles(dir string) ([]parquetFile, error) {
	latest := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(d.Name(), ".parquet") {
			return nil
		}
		table := tsPattern.ReplaceAllString(d.Name(), "")
		if prev, ok := latest[table]; !ok || filepath.Base(path) > filepath.Base(prev) {
			latest[table] = path
		}
		return nil
	})
	out := make([]parquetFile, 0, len(latest))
	for table, path := range latest {
		out = append(out, parquetFile{table: table, path: path})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].table < out[j].table })
	return out, err
}
