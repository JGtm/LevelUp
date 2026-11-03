package parquetbackup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/pkg/duckdbbackup"
)

// Délais d'attente du bail d'écriture d'une base.
const (
	// defaultLeaseWait : premier passage, la même porte que le CHECKPOINT périodique (une base dont un
	// lot d'écriture est en cours est sautée, jamais attendue longtemps).
	defaultLeaseWait = 2 * time.Second
	// defaultRetryWait : second passage, sur les seules bases sautées, une fois les autres exportées.
	defaultRetryWait = 60 * time.Second
	// defaultTableTimeout : échéance d'UNE table (le bail n'est tenu que le temps d'une table ; au-delà, la table
	// est écartée et le jeu est partiel). Une table de l'ordre du Gio s'exporte en quelques secondes.
	defaultTableTimeout = 2 * time.Minute
)

// Codes des raisons de saut d'une base dans le manifeste.
const (
	reasonBusy         = "busy"
	reasonExportFailed = "export_failed"
)

// ErrNothingExported : aucune base n'a pu être lue, il n'y a pas de jeu à garder.
var ErrNothingExported = errors.New("parquetbackup: aucune base exportée")

// Options configure un jeu de sauvegarde.
type Options struct {
	// Paths : le résolveur de chemins (bases et fichiers de configuration).
	Paths *title.PathResolver
	// AuthDir : le dossier d'authentification (users.json, groups.json).
	AuthDir string
	// Dest : le dossier choisi par l'utilisateur ; le jeu y est créé.
	Dest string
	// Keep : nombre de jeux conservés après celui-ci (>= 1).
	Keep int
	// AppVersion : version de l'application, inscrite au manifeste.
	AppVersion string
	// Now : l'horloge (tests) ; nil = time.Now.
	Now func() time.Time
	// LeaseWait et RetryWait : attentes du bail (zéro = valeurs par défaut).
	LeaseWait, RetryWait time.Duration
	// TableTimeout : échéance d'une table, bail compris (zéro = [defaultTableTimeout]).
	TableTimeout time.Duration
}

func (o *Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

func (o *Options) leaseWait() time.Duration {
	if o.LeaseWait > 0 {
		return o.LeaseWait
	}
	return defaultLeaseWait
}

func (o *Options) tableTimeout() time.Duration {
	if o.TableTimeout > 0 {
		return o.TableTimeout
	}
	return defaultTableTimeout
}

func (o *Options) retryWait() time.Duration {
	if o.RetryWait > 0 {
		return o.RetryWait
	}
	return defaultRetryWait
}

// Run écrit un jeu de sauvegarde dans opts.Dest, applique la rotation et rend son manifeste. Les bases se
// lisent seulement (voir doc.go). Un jeu dont une base reste occupée ou échoue est rendu PARTIEL ; si
// aucune base n'a pu être exportée, rien n'est gardé et l'erreur est [ErrNothingExported].
func Run(ctx context.Context, opts Options) (*Manifest, error) {
	if err := os.MkdirAll(opts.Dest, 0o755); err != nil {
		return nil, fmt.Errorf("dossier de sauvegarde %s : %w", opts.Dest, err)
	}
	removePartials(ctx, opts.Dest)

	start := opts.now()
	name, err := freshSetName(opts.Dest, start)
	if err != nil {
		return nil, err
	}
	partial := filepath.Join(opts.Dest, partialPrefix+name)
	if err := os.MkdirAll(partial, 0o755); err != nil {
		return nil, fmt.Errorf("jeu de sauvegarde : %w", err)
	}
	m, err := build(ctx, opts, partial, name, start)
	if err != nil {
		_ = os.RemoveAll(partial)
		return nil, err
	}
	final := filepath.Join(opts.Dest, name)
	if err := os.Rename(partial, final); err != nil {
		_ = os.RemoveAll(partial)
		return nil, fmt.Errorf("jeu de sauvegarde, renommage final : %w", err)
	}
	removed := rotate(ctx, opts.Dest, opts.Keep)
	slog.InfoContext(ctx, "backup parquet: jeu écrit",
		"set", final, "bases", len(m.Databases), "lignes", m.TotalRows, "octets", m.TotalBytes,
		"partiel", m.Partial, "rotation_supprimes", removed, "duree_ms", m.DurationMs)
	return m, nil
}

// freshSetName rend le nom du jeu de l'instant start, suffixé si une sauvegarde de la même minute existe.
func freshSetName(dest string, start time.Time) (string, error) {
	base := SetPrefix + start.Format(setTimeLayout)
	for i := 1; i < 100; i++ {
		name := base
		if i > 1 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		if _, err := os.Stat(filepath.Join(dest, name)); errors.Is(err, os.ErrNotExist) {
			return name, nil
		}
	}
	return "", fmt.Errorf("trop de jeux à la même minute dans %s", dest)
}

// build remplit le dossier partiel : bases, configuration, manifeste.
func build(ctx context.Context, opts Options, partial, name string, start time.Time) (*Manifest, error) {
	dbs, err := ops.BackupDatabases(opts.Paths)
	if err != nil {
		return nil, fmt.Errorf("liste des bases : %w", err)
	}
	m := &Manifest{
		Format: ManifestFormat, Name: name, CreatedAt: start.UTC(), AppVersion: opts.AppVersion,
		Excluded: []string{ExcludedFilmArchive},
	}
	entries := exportAll(ctx, opts, dbs, partial)
	exported := 0
	for _, db := range dbs {
		e := entries[db.Key]
		m.Databases = append(m.Databases, *e)
		if len(e.FailedTables) > 0 {
			m.Partial = true
		}
		if e.Skipped {
			m.Partial = true
			continue
		}
		exported++
		for _, t := range e.Tables {
			m.TotalBytes += t.Bytes
			m.TotalRows += t.Rows
		}
	}
	if len(dbs) > 0 && exported == 0 {
		return nil, ErrNothingExported
	}
	if m.Config, err = copyConfigFiles(partial, configSources(opts.Paths, opts.AuthDir)); err != nil {
		return nil, err
	}
	for _, c := range m.Config {
		m.TotalBytes += c.Bytes
	}
	m.DurationMs = opts.now().Sub(start).Milliseconds()
	if err := writeManifest(partial, m); err != nil {
		return nil, err
	}
	return m, nil
}

// exportAll exporte chaque base : un premier passage à attente courte, puis un second, plus patient, sur
// les seules bases dont le bail était tenu.
func exportAll(ctx context.Context, opts Options, dbs []ops.BackupDB, partial string) map[string]*DatabaseEntry {
	entries := make(map[string]*DatabaseEntry, len(dbs))
	var busy []ops.BackupDB
	for _, db := range dbs {
		e, err := exportOne(ctx, db, partial, opts.leaseWait(), opts.tableTimeout())
		if errors.Is(err, duckdb.ErrBaseOccupee) {
			busy = append(busy, db)
		}
		entries[db.Key] = e
	}
	for _, db := range busy {
		entries[db.Key], _ = exportOne(ctx, db, partial, opts.retryWait(), opts.tableTimeout())
	}
	return entries
}

// exportOne exporte une base dans <partial>/db/<clé>. Une base occupée rend [duckdb.ErrBaseOccupee] (et
// une entrée « sautée », la base entière est alors reprise plus tard) ; tout autre échec est journalisé et rend
// une entrée « sautée » sans erreur : une base défaillante ne fait pas perdre les autres.
func exportOne(ctx context.Context, db ops.BackupDB, partial string, wait, tableTimeout time.Duration) (*DatabaseEntry, error) {
	rel := dbDirName + "/" + strings.ReplaceAll(db.Key, ":", "/")
	outDir := filepath.Join(partial, filepath.FromSlash(rel))
	entry := &DatabaseEntry{Key: db.Key}
	// Le bail est pris et rendu PAR TABLE (et la lecture n'est pas laissée ouverte entre deux tables) : un écrivain à
	// attente courte n'attend jamais le temps d'une base entière.
	run := func(ctx context.Context, fn func(*sql.DB) error) error {
		return duckdb.WithHeldRead(ctx, db.Path, wait, fn)
	}
	exports, failedTables, err := duckdbbackup.ExportTargetTablesHeld(ctx, run,
		duckdbbackup.HeldExport{Key: db.Key, OutputDir: outDir, Level: compressionLevel, TableTimeout: tableTimeout})
	switch {
	case errors.Is(err, duckdb.ErrBaseOccupee):
		slog.WarnContext(ctx, "backup parquet: base occupée, sautée", "key", db.Key, "attente", wait)
		entry.Skipped, entry.Reason = true, reasonBusy
		_ = os.RemoveAll(outDir)
		return entry, err
	case err != nil:
		slog.ErrorContext(ctx, "backup parquet: export de la base échoué", "key", db.Key, "err", err)
		_ = os.RemoveAll(outDir)
		entry.Skipped, entry.Reason = true, reasonExportFailed
		return entry, nil
	}
	entry.FailedTables = failedTables
	if err := describe(partial, rel, exports, entry); err != nil {
		slog.ErrorContext(ctx, "backup parquet: mesure des fichiers échouée", "key", db.Key, "err", err)
		_ = os.RemoveAll(outDir)
		entry.Tables, entry.Schema = nil, nil
		entry.Skipped, entry.Reason = true, reasonExportFailed
	}
	return entry, nil
}

// describe mesure les fichiers écrits d'une base (Parquet et schéma) et remplit son entrée.
func describe(partial, rel string, exports []duckdbbackup.TableExport, entry *DatabaseEntry) error {
	for _, ex := range exports {
		fe, err := fileEntryOf(partial, rel+"/"+filepath.Base(ex.File))
		if err != nil {
			return err
		}
		entry.Tables = append(entry.Tables, TableEntry{FileEntry: fe, Table: ex.Table, Rows: ex.Rows})
	}
	ddl, err := filepath.Glob(filepath.Join(partial, filepath.FromSlash(rel), "_ddl_*.json"))
	if err != nil || len(ddl) == 0 {
		return err // pas de schéma : la restauration se fera sans clé (journalisé à l'export)
	}
	fe, err := fileEntryOf(partial, rel+"/"+filepath.Base(ddl[0]))
	if err != nil {
		return err
	}
	entry.Schema = &fe
	return nil
}
