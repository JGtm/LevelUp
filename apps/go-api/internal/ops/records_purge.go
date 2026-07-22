// Package ops — records_purge.go : purge one-off des records de PB corrompus
// (valeur hors bornes de vraisemblance ou métrique hors catalogue, cf. A4/A5).
//
// Contexte (DEC-7) : des PB aberrants (accuracy « 7333 % », best_kda 107) ont
// été persistés avant l'ajout des bornes du detector (A4). Ces lignes vivent
// dans deux tables APPEND-ONLY :
//   - player_records_history (shared_social.duckdb, vue player_records_latest) ;
//   - record_history (stats.duckdb par joueur, timeline).
//
// Neutralisation = recette ADR 0026 (rebuild par CTAS filtré + swap
// transactionnel), JAMAIS de DELETE brut (surface ART #23046). On reconstruit la
// table en ne gardant QUE les lignes plausibles (métrique suivie ET valeur dans
// ses bornes) puis on recrée PK / index / vue à l'identique. Une clé dont toutes
// les versions étaient corrompues disparaît ; une clé dont seule la dernière
// version était corrompue retombe sur sa dernière version plausible (sémantique
// correcte de la vue _latest).
//
// Sûreté : garde de cardinalité (kept == before - removed) AVANT le commit,
// rollback intégral sur toute erreur. Le dry-run n'effectue AUCUNE mutation
// (comptage read-only seulement).
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"levelup/go-api/internal/progression/records"
)

// RecordsPurgeOffender décrit un groupe de lignes corrompues à retirer.
type RecordsPurgeOffender struct {
	Metric string
	Value  float64
	Count  int
}

// RecordsPurgeResult résume la purge d'une table.
type RecordsPurgeResult struct {
	Table     string
	Applied   bool
	Before    int
	Removed   int
	Offenders []RecordsPurgeOffender
}

// recordsPurgeSpec décrit une table append-only à purger et les DDL de
// reconstruction (PK / index / vue) rejoués APRÈS le swap CTAS filtré.
type recordsPurgeSpec struct {
	table       string
	postSwapDDL []string
}

// playerRecordsHistorySpec — table PB partagée (shared_social.duckdb).
// PK id (séquence player_records_history_id_seq), index idx_prh_lookup, vue
// player_records_latest (DISTINCT ON dernière written_at par xuid/metric/period).
var playerRecordsHistorySpec = recordsPurgeSpec{
	table: "player_records_history",
	postSwapDDL: []string{
		`ALTER TABLE player_records_history ADD PRIMARY KEY (id)`,
		`ALTER TABLE player_records_history ALTER COLUMN id SET DEFAULT nextval('player_records_history_id_seq')`,
		`CREATE INDEX IF NOT EXISTS idx_prh_lookup ON player_records_history(xuid, metric, period, written_at DESC)`,
		`CREATE OR REPLACE VIEW player_records_latest AS
			SELECT DISTINCT ON (xuid, metric, period)
				id, xuid, metric, period, value, achieved_at, achieved_match_id,
				previous_value, previous_achieved_at, written_at
			FROM player_records_history
			ORDER BY xuid, metric, period, written_at DESC, id DESC`,
	},
}

// recordHistorySpec — timeline par joueur (stats.duckdb). PK id (VARCHAR/UUID),
// deux index de lecture. Pas de vue _latest (timeline complète).
var recordHistorySpec = recordsPurgeSpec{
	table: "record_history",
	postSwapDDL: []string{
		`ALTER TABLE record_history ADD PRIMARY KEY (id)`,
		`CREATE INDEX IF NOT EXISTS idx_rec_hist_user_title_metric ON record_history(user_id, title_slug, metric)`,
		`CREATE INDEX IF NOT EXISTS idx_rec_hist_achieved_desc ON record_history(user_id, achieved_at DESC)`,
	},
}

// PurgePlayerRecordsHistory purge player_records_history sur une connexion
// shared_social ouverte en RW. apply=false → dry-run (aucune mutation).
func PurgePlayerRecordsHistory(ctx context.Context, db *sql.DB, apply bool) (RecordsPurgeResult, error) {
	return purgeRecordsTable(ctx, db, playerRecordsHistorySpec, apply)
}

// PurgeRecordHistory purge record_history sur une connexion stats joueur ouverte
// en RW. apply=false → dry-run (aucune mutation).
func PurgeRecordHistory(ctx context.Context, db *sql.DB, apply bool) (RecordsPurgeResult, error) {
	return purgeRecordsTable(ctx, db, recordHistorySpec, apply)
}

// keepPlausibleRowsSQL construit le prédicat WHERE gardant UNIQUEMENT les lignes
// dont la métrique est suivie ET la valeur dans ses bornes. Les métriques hors
// catalogue (ex best_kda) ne matchent aucune branche → exclues. Les noms de
// métriques viennent d'un jeu de constantes Go contrôlé (aucune injection).
func keepPlausibleRowsSQL() string {
	bounds := records.TrackedMetricBounds()
	parts := make([]string, 0, len(bounds))
	for _, b := range bounds {
		parts = append(parts, fmt.Sprintf("(metric = '%s' AND value BETWEEN %s AND %s)",
			b.Metric, formatBound(b.Min), formatBound(b.Max)))
	}
	return strings.Join(parts, " OR ")
}

func formatBound(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// purgeRecordsTable exécute le comptage (toujours) puis, si apply, le rebuild
// transactionnel filtré. Retourne un résultat même quand la table est absente
// (no-op) ou déjà propre (Removed=0).
func purgeRecordsTable(ctx context.Context, db *sql.DB, spec recordsPurgeSpec, apply bool) (RecordsPurgeResult, error) {
	res := RecordsPurgeResult{Table: spec.table, Applied: apply}
	exists, err := recordsTableExists(ctx, db, spec.table)
	if err != nil {
		return res, err
	}
	if !exists {
		return res, nil
	}

	keep := keepPlausibleRowsSQL()
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+spec.table).Scan(&res.Before); err != nil {
		return res, fmt.Errorf("purge %s: count before: %w", spec.table, err)
	}
	offenders, removed, err := collectOffenders(ctx, db, spec.table, keep)
	if err != nil {
		return res, err
	}
	res.Offenders = offenders
	res.Removed = removed
	if removed == 0 || !apply {
		return res, nil
	}
	if err := rebuildFilteredTx(ctx, db, spec, keep, res.Before, removed); err != nil {
		return res, err
	}
	slog.InfoContext(ctx, "records purge: table reconstruite (lignes corrompues retirées)",
		"table", spec.table, "before", res.Before, "removed", removed)
	return res, nil
}

// collectOffenders liste les lignes NON gardées, groupées par (metric, value).
func collectOffenders(ctx context.Context, db *sql.DB, table, keep string) ([]RecordsPurgeOffender, int, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(
		`SELECT metric, value, COUNT(*) FROM %s WHERE NOT (%s) GROUP BY metric, value ORDER BY metric, value`,
		table, keep))
	if err != nil {
		return nil, 0, fmt.Errorf("purge %s: collect offenders: %w", table, err)
	}
	defer rows.Close()
	var out []RecordsPurgeOffender
	total := 0
	for rows.Next() {
		var o RecordsPurgeOffender
		if err := rows.Scan(&o.Metric, &o.Value, &o.Count); err != nil {
			return nil, 0, fmt.Errorf("purge %s: scan offender: %w", table, err)
		}
		out = append(out, o)
		total += o.Count
	}
	return out, total, rows.Err()
}

// rebuildFilteredTx reconstruit la table en ne gardant que les lignes plausibles,
// dans une transaction avec garde de cardinalité et rollback intégral sur erreur.
func rebuildFilteredTx(ctx context.Context, db *sql.DB, spec recordsPurgeSpec, keep string, before, removed int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("purge %s: begin: %w", spec.table, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	tmp := spec.table + "__purge"
	if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+tmp); err != nil {
		return fmt.Errorf("purge %s: drop stale tmp: %w", spec.table, err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(
		"CREATE TABLE %s AS SELECT * FROM %s WHERE %s", tmp, spec.table, keep)); err != nil {
		return fmt.Errorf("purge %s: create filtered: %w", spec.table, err)
	}
	var kept int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+tmp).Scan(&kept); err != nil {
		return fmt.Errorf("purge %s: count kept: %w", spec.table, err)
	}
	if kept != before-removed {
		return fmt.Errorf("purge %s: garde cardinalité échouée (kept=%d, attendu=%d) — rollback",
			spec.table, kept, before-removed)
	}

	stmts := append([]string{
		"DROP TABLE " + spec.table,
		fmt.Sprintf("ALTER TABLE %s RENAME TO %s", tmp, spec.table),
	}, spec.postSwapDDL...)
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("purge %s: swap step: %w", spec.table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("purge %s: commit: %w", spec.table, err)
	}
	committed = true
	return nil
}

func recordsTableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, table).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("check table %s: %w", table, err)
	}
	return n > 0, nil
}
