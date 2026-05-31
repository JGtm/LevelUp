package duckdbbackup

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2" // DuckDB driver
)

// IntegrityResult holds the outcome of a PRAGMA integrity_check on a single DB.
type IntegrityResult struct {
	OK        bool      `json:"ok"`
	Detail    string    `json:"detail,omitempty"` // first error line if !OK
	CheckedAt time.Time `json:"checked_at"`
}

// CheckIntegrity runs PRAGMA integrity_check on t.
// Always returns a result; never panics or returns an error.
// Open failures set OK=false (can't read the file at all).
// Query errors (e.g. pragma not supported by this DuckDB version) set OK=true
// as they are inconclusive — not a corruption signal.
// The caller should log a warning when OK is false.
//
// Si t.OpenDB est défini, réutilise la connexion fournie (cas des fichiers
// détenus en RW par le serveur). Sinon ouvre une connexion read-only autonome.
func CheckIntegrity(ctx context.Context, t Target) IntegrityResult {
	res := IntegrityResult{CheckedAt: time.Now().UTC()}

	db, release, err := openTarget(ctx, t)
	if err != nil {
		res.Detail = fmt.Sprintf("open: %v", err)
		return res
	}
	defer release()

	rows, err := db.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		// pragma not supported by this DuckDB version — inconclusive, not alarming
		slog.DebugContext(ctx, "backup: integrity_check indisponible", "key", t.Key, "err", err)
		res.OK = true
		return res
	}
	defer rows.Close()

	var first string
	for rows.Next() {
		var line string
		if scanErr := rows.Scan(&line); scanErr != nil {
			break
		}
		if first == "" {
			first = line
		}
	}

	if strings.EqualFold(first, "ok") || first == "" {
		res.OK = true
		return res
	}
	res.Detail = first
	return res
}

// ExportTarget exports all BASE TABLE tables from t to outputDir as Parquet+zstd.
//
// Si t.OpenDB est défini, réutilise la connexion fournie — requis pour les
// fichiers déjà détenus en RW par le serveur (metadata, shared_social) car
// DuckDB refuse une seconde ouverture avec ?access_mode=read_only sur le même
// fichier in-process. Sinon ouvre une connexion read-only autonome (cas des
// DBs détenues en RO ou fermées).
//
// Returns the number of tables exported.
func ExportTarget(ctx context.Context, t Target, outputDir string, compressionLevel int) (int, error) {
	if compressionLevel <= 0 {
		compressionLevel = 9
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir %s: %w", outputDir, err)
	}

	db, release, err := openTarget(ctx, t)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", t.Path, err)
	}
	defer release()

	tables, err := listTables(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("list tables %s: %w", t.Key, err)
	}

	slog.DebugContext(ctx, "backup: export démarré", "key", t.Key, "tables", len(tables))
	start := time.Now()
	ts := start.UTC().Format("20060102_150405")
	exported := 0
	var failed []string
	for _, table := range tables {
		outPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.parquet", table, ts))
		if err := exportTable(ctx, db, table, outPath, compressionLevel); err != nil {
			// Échec d'une table : on WARN et on continue. Un objet résiduel ou
			// corrompu (ex: vue legacy, table d'un catalogue attaché qui a
			// échappé au filtre listTables) ne doit jamais faire perdre tout le
			// backup du target — un backup partiel vaut mieux qu'aucun backup.
			slog.WarnContext(ctx, "backup: export table échoué (ignoré)",
				"key", t.Key, "table", table, "err", err)
			failed = append(failed, table)
			continue
		}
		exported++
	}
	slog.InfoContext(ctx, "backup: export terminé",
		"key", t.Key,
		"tables_ok", exported,
		"tables_failed", len(failed),
		"duration", time.Since(start).Round(time.Millisecond))
	// On ne remonte une erreur dure que si AUCUNE table n'a pu être exportée
	// (signal d'un problème global : conn morte, dossier non writable...).
	if exported == 0 && len(failed) > 0 {
		return 0, fmt.Errorf("export %s: aucune des %d tables exportée (ex: %s)",
			t.Key, len(failed), failed[0])
	}
	return exported, nil
}

// openTarget retourne une connexion sur t et la fonction de libération
// associée. Si t.OpenDB est défini, délègue (emprunt non-possédant — le
// release est un no-op côté DuckDB). Sinon ouvre une connexion read-only
// autonome qui sera fermée par release().
func openTarget(ctx context.Context, t Target) (*sql.DB, func(), error) {
	if t.OpenDB != nil {
		db, release, err := t.OpenDB(ctx)
		if err != nil {
			return nil, nil, err
		}
		if release == nil {
			release = func() {}
		}
		return db, release, nil
	}
	db, err := sql.Open("duckdb", t.Path+"?access_mode=read_only")
	if err != nil {
		return nil, nil, err
	}
	return db, func() { _ = db.Close() }, nil
}

func listTables(ctx context.Context, db *sql.DB) ([]string, error) {
	// Scope au SEUL catalogue courant + schéma main : information_schema.tables
	// remonte aussi les tables des catalogues ATTACHés (global xbox_aliases,
	// shared...) quand la conn est celle du pool. Sans ce filtre, on listait par
	// ex. `xuid_aliases` (global) puis `COPY "xuid_aliases"` échouait car non
	// résolvable dans le catalogue player ("Table does not exist").
	rows, err := db.QueryContext(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_type = 'BASE TABLE'
		  AND table_catalog = current_database()
		  AND table_schema = 'main'
		ORDER BY table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func exportTable(ctx context.Context, db *sql.DB, table, outPath string, compressionLevel int) error {
	q := fmt.Sprintf(
		`COPY %q TO '%s' (FORMAT PARQUET, COMPRESSION 'zstd', COMPRESSION_LEVEL %d)`,
		table, outPath, compressionLevel,
	)
	_, err := db.ExecContext(ctx, q)
	return err
}
