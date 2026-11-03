package duckdbbackup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// HeldRunner exécute fn avec un handle de LECTURE de la base, sous le bail d'écriture de son chemin, et rend le
// bail (et, si elle l'a ouvert elle-même, le handle) au retour de fn. Une erreur du bail ou de l'ouverture est
// rendue telle quelle ; celle de fn aussi.
type HeldRunner func(ctx context.Context, fn func(db *sql.DB) error) error

// tableFailure enveloppe l'échec de fn (l'export d'une table) pour le distinguer d'un échec du bail ou de
// l'ouverture, que le HeldRunner rend seul.
type tableFailure struct{ err error }

func (e tableFailure) Error() string { return e.err.Error() }
func (e tableFailure) Unwrap() error { return e.err }

// HeldExport décrit l'export d'une base par [ExportTargetTablesHeld].
type HeldExport struct {
	// Key : la clé de la base (journaux, schéma).
	Key string
	// OutputDir : le dossier qui reçoit les Parquet et le schéma.
	OutputDir string
	// Level : le niveau zstd (zéro ou moins = 9).
	Level int
	// TableTimeout : l'échéance d'une table, bail compris (zéro = aucune échéance propre).
	TableTimeout time.Duration
}

// ExportTargetTablesHeld exporte les tables de la base `h.Key` vers h.OutputDir comme ExportTargetTables, mais
// sous un bail PRIS ET RENDU PAR TABLE (run) : un écrivain à attente courte (synchronisation, écriture des
// réglages) ne reste jamais bloqué le temps d'une base entière, seulement celui de la table en cours. Chaque
// table dispose de h.TableTimeout : son dépassement l'écarte (listée en second
// résultat, la sauvegarde est alors partielle) sans retenir le bail plus longtemps.
//
// Un échec du bail ou de l'ouverture (base occupée, poignée impossible) interrompt tout l'export et est
// rendu tel quel : l'appelant décide de reprendre la base entière plus tard. Le schéma est relevé sous un
// premier bail, avant les tables.
func ExportTargetTablesHeld(ctx context.Context, run HeldRunner, h HeldExport) ([]TableExport, []string, error) {
	key, outputDir := h.Key, h.OutputDir
	if h.Level <= 0 {
		h.Level = 9
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("mkdir %s: %w", outputDir, err)
	}
	start := time.Now()
	ts := start.UTC().Format("20060102_150405")
	var tables []string
	err := run(ctx, func(db *sql.DB) error {
		var lerr error
		if tables, lerr = listTables(ctx, db); lerr != nil {
			return tableFailure{fmt.Errorf("list tables %s: %w", key, lerr)}
		}
		writeSchemaDDL(ctx, db, key, filepath.Join(outputDir, ddlFilePrefix+ts+ddlFileSuffix))
		return nil
	})
	if err != nil {
		return nil, nil, unwrapRunnerError(err)
	}
	slog.DebugContext(ctx, "backup: export démarré", "key", key, "tables", len(tables))
	var exported []TableExport
	var failed []string
	for _, table := range tables {
		outPath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.parquet", table, ts))
		rows, err := exportTableHeld(ctx, run, h, table, outPath)
		var tf tableFailure
		switch {
		case err == nil:
			exported = append(exported, TableExport{Table: table, File: outPath, Rows: rows})
		case errors.As(err, &tf):
			slog.WarnContext(ctx, "backup: table non exportée", "key", key, "table", table, "err", tf.err)
			_ = os.Remove(outPath)
			failed = append(failed, table)
		default:
			return nil, nil, err
		}
	}
	slog.InfoContext(ctx, "backup: export terminé", "key", key, "tables_ok", len(exported), "tables_failed", len(failed),
		"duration", time.Since(start).Round(time.Millisecond))
	if len(exported) == 0 && len(failed) > 0 {
		return nil, failed, fmt.Errorf("export %s: aucune des %d tables exportée (ex: %s)", key, len(failed), failed[0])
	}
	return exported, failed, nil
}

// exportTableHeld exporte UNE table sous son propre bail et sa propre échéance. Une erreur de l'export ou
// l'échéance dépassée rend un [tableFailure] ; toute autre erreur vient du bail ou de l'ouverture.
func exportTableHeld(ctx context.Context, run HeldRunner, h HeldExport, table, outPath string) (int64, error) {
	tctx := ctx
	timeout := h.TableTimeout
	if timeout > 0 {
		var cancel context.CancelFunc
		tctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	var rows int64
	err := run(tctx, func(db *sql.DB) error {
		var eerr error
		if rows, eerr = exportTable(tctx, db, table, outPath, h.Level); eerr != nil {
			return tableFailure{eerr}
		}
		return nil
	})
	if err != nil && ctx.Err() == nil && errors.Is(tctx.Err(), context.DeadlineExceeded) {
		return 0, tableFailure{fmt.Errorf("échéance de %s dépassée : %w", timeout, err)}
	}
	return rows, err
}

// unwrapRunnerError rend l'erreur du bail ou, si l'échec vient de fn, celle de fn.
func unwrapRunnerError(err error) error {
	var tf tableFailure
	if errors.As(err, &tf) {
		return tf.err
	}
	return err
}
