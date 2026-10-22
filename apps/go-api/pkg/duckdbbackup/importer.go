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
	// FallbackTables : tables ("clé:table") importées par CREATE TABLE AS — donc SANS clé ni
	// défaut : sauvegarde sans schéma lisible, table absente de son schéma, ou schéma refusé.
	// Chacune est journalisée en WARN. Rien ne recrée ensuite ce que CREATE TABLE AS a perdu :
	// le registre des migrations restauré est complet, aucune étape ne rejoue.
	FallbackTables []string
	// SchemaFailures : séquences, index ou vues ("clé:<ordre DDL abrégé>") du schéma de la
	// sauvegarde que DuckDB a refusé de recréer. Chacun est journalisé en WARN.
	SchemaFailures []string
	DurationMs     int64
}

// FinalizeFunc s'applique à chaque base restaurée, après le versement de ses tables (ex.
// alignement des séquences sur les identifiants restaurés). nil : aucune finalisation.
type FinalizeFunc func(ctx context.Context, key string, db *sql.DB) error

// tsPattern strips the timestamp suffix appended by ExportTarget to table filenames.
// e.g. "match_registry_20260525_132028.parquet" → "match_registry"
var tsPattern = regexp.MustCompile(`_\d{8}_\d{6}\.parquet$`)

// importer porte l'état d'un import de staging.
type importer struct {
	stagingDir string
	keyToPath  func(string) string
	finalize   FinalizeFunc
	dryRun     bool
	result     *ImportResult
}

// ImportFromStaging walks stagingDir and imports each key's Parquet files into DuckDB.
//
// keyToPath maps a backup key (e.g. "halo_infinite:shared_matches_v2") to the
// target DuckDB file path. Keys with no mapping (empty string) are skipped.
//
// Chaque base est recréée dans l'état de son ÉPOQUE : le schéma exporté avec la sauvegarde
// (`_ddl_<horodatage>.json`, même instant que les lignes et que le registre des migrations)
// est rejoué, puis chaque table est remplie. Jamais le schéma du code courant : une étape de
// migration absente du registre restauré doit trouver la base telle qu'elle était, sinon une
// conversion qui se saute sur un schéma déjà à jour perdrait les lignes restées dans l'ancienne
// table. Le démarrage suivant applique les étapes manquantes. Sans schéma exporté (sauvegarde
// antérieure à son introduction) : tables et lignes seules par CREATE TABLE AS, sans clé, défaut,
// séquence ni vue, et rien ne les recrée ensuite.
//
// When dryRun is true the function logs what it would do but makes no changes.
func ImportFromStaging(ctx context.Context, stagingDir string, keyToPath func(string) string,
	finalize FinalizeFunc, dryRun bool) (*ImportResult, error) {
	start := time.Now()
	im := &importer{stagingDir: stagingDir, keyToPath: keyToPath, finalize: finalize, dryRun: dryRun,
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

// doImport crée la base neuve à dbPath : séquences du schéma de la sauvegarde, tables (créées
// depuis ce schéma puis remplies), index, vues, puis finalisation.
func (im *importer) doImport(ctx context.Context, key, parquetDir, dbPath string) error {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("créer DuckDB %s: %w", dbPath, err)
	}
	defer func() { _ = db.Close() }()

	files, err := parquetFiles(parquetDir)
	if err != nil {
		return err
	}
	cycle := latestCycle(files)
	ts := &tableSchema{key: key, cycle: cycle, ddl: loadKeySchema(ctx, key, parquetDir, cycle), result: im.result}
	ts.replay(ctx, db, ts.sequences())
	for _, f := range files {
		if err := ts.importTable(ctx, db, f); err != nil {
			return fmt.Errorf("table %q: %w", f.table, err)
		}
	}
	ts.replay(ctx, db, ts.indexes())
	ts.replayViews(ctx, db)
	if im.finalize != nil {
		if err := im.finalize(ctx, key, db); err != nil {
			return fmt.Errorf("finalisation: %w", err)
		}
	}
	slog.InfoContext(ctx, "restore: import terminé", "tables", len(files), "db", dbPath)
	return nil
}

// loadKeySchema rend le schéma exporté par le cycle des Parquet retenus, ou nil (repli de toute
// la clé sur des tables sans clé), avec le WARN qui dit pourquoi.
func loadKeySchema(ctx context.Context, key, dir, cycle string) *schemaDDL {
	const seules = " — tables et lignes seules (ni clé, ni défaut, ni séquence, ni vue)"
	schema, other, err := loadSchemaDDL(dir, cycle)
	switch {
	case err != nil:
		slog.WarnContext(ctx, "restore: schéma de la sauvegarde illisible"+seules, "key", key, "err", err)
		return nil
	case schema == nil && other:
		slog.WarnContext(ctx, "restore: schéma d'un autre export ignoré (aucun schéma pour le cycle des Parquet retenus)"+seules,
			"key", key, "cycle", cycle)
	case schema == nil:
		slog.WarnContext(ctx, "restore: sauvegarde sans schéma (antérieure à son export)"+seules, "key", key)
	}
	return schema
}

// parquetFile : une table à importer, son fichier et l'horodatage de son export.
type parquetFile struct{ table, path, ts string }

// tsCapture extrait l'horodatage d'export du nom d'un fichier Parquet.
var tsCapture = regexp.MustCompile(`_(\d{8}_\d{6})\.parquet$`)

// latestCycle rend l'horodatage du dernier export présent parmi files ("" si aucun).
func latestCycle(files []parquetFile) string {
	latest := ""
	for _, f := range files {
		if f.ts > latest {
			latest = f.ts
		}
	}
	return latest
}

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
		f := parquetFile{table: table, path: path}
		if m := tsCapture.FindStringSubmatch(filepath.Base(path)); m != nil {
			f.ts = m[1]
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].table < out[j].table })
	return out, err
}
