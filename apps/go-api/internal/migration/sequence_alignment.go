package migration

// sequence_alignment.go — une séquence servant de DEFAULT ne rend jamais une valeur déjà prise.
//
// Contrat : pour chaque séquence de la base courante utilisée comme `DEFAULT nextval('…')`
// d'une colonne entière, la prochaine valeur rendue est strictement supérieure au maximum de
// toutes les colonnes qu'elle alimente. Une séquence en retard (base legacy dont la séquence a
// été recréée à START 1 après coup, valeurs consommées par des transactions annulées puis
// perdues à l'arrêt brutal, rows copiées avec leurs ids) fait échouer CHAQUE insertion
// suivante en « Duplicate key » sur la clé primaire : l'écrivain retente au cycle suivant et
// rééchoue, sans fin.
//
// Technique : DuckDB n'a ni `ALTER SEQUENCE … RESTART` ni `setval`, et `CREATE OR REPLACE
// SEQUENCE` est refusé tant qu'une colonne en dépend (Dependency Error). On CONSOMME donc
// `nextval` le nombre de fois nécessaire, en une seule instruction (`SELECT max(nextval(…))
// FROM range(k)`). C'est sûr : la consommation ne recule jamais une séquence, ne touche à
// aucune ligne de table, et une insertion concurrente ne peut que consommer en plus (trou
// d'ids, sans conséquence pour une clé technique).
//
// Prochaine valeur : lue dans `duckdb_sequences().sql` (`… START <compteur> …`), qui porte le
// compteur courant. Les colonnes `start_value` / `last_value` ne le portent PAS de façon
// fiable : après rejeu du WAL elles restent à leur valeur du dernier point de reprise
// (mesuré : 1 / 1 pour un compteur réel à 6) ; s'y fier ferait consommer à tort sur une base
// saine. TestAlignSequences_LitLeCompteurReelApresRejeuDuWAL tient ce contrat.
//
// Ce que la fonction ne fait jamais : modifier une ligne (pas d'UPDATE des ids NULL ou en
// double — l'état des données est rapporté, pas réparé ici), créer ou supprimer un objet de
// schéma (le soin reste un no-op pour schemadrift).

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/observability"
)

// maxSequenceCatchUp borne le rattrapage d'une séquence en une passe : au-delà, l'écart
// signale une colonne alimentée par autre chose que sa séquence (ids importés en masse), qui
// mérite un diagnostic plutôt qu'une consommation aveugle de millions de valeurs.
const maxSequenceCatchUp = 10_000_000

// sequenceAlignedCounter — compteur expvar (/debug/vars, clé "levelup") des séquences
// réellement avancées par AlignSequencesToColumns.
const sequenceAlignedCounter = "duckdb_sequence_aligned_total"

// sequenceAlignFailedCounter — compteur expvar des passes d'alignement en échec.
const sequenceAlignFailedCounter = "duckdb_sequence_align_failed_total"

// sequenceUse — une séquence et les colonnes qu'elle alimente par DEFAULT.
type sequenceUse struct {
	name      string
	increment int64
	maxValue  int64
	cycle     bool
	columns   []tableColumn
}

type tableColumn struct {
	table  string
	column string
}

// SequenceAlignment rapporte une séquence avancée par AlignSequencesToColumns.
type SequenceAlignment struct {
	Sequence   string
	ColumnMax  int64
	NextBefore int64
	NextAfter  int64
	Consumed   int64
}

// sequenceUsesSQL — séquences de la base courante servant de DEFAULT à une colonne entière.
// Le nom est extrait du DEFAULT (`nextval('nom')` ou `nextval('schema.nom')`) ; la jointure
// se fait par nom dans la même base et le même schéma.
const sequenceUsesSQL = `
SELECT s.sequence_name, s.increment_by, s.max_value, s.cycle, c.table_name, c.column_name
FROM duckdb_columns() c
JOIN duckdb_tables() t
  ON t.database_name = c.database_name AND t.schema_name = c.schema_name AND t.table_name = c.table_name
JOIN duckdb_sequences() s
  ON s.database_name = c.database_name AND s.schema_name = c.schema_name
 AND s.sequence_name = regexp_extract(c.column_default, '^nextval\(''(?:[^''.]+\.)?([^''.]+)''\)$', 1)
WHERE c.database_name = current_database()
  AND NOT t.temporary
  AND c.data_type IN ('BIGINT', 'INTEGER', 'SMALLINT', 'TINYINT')
ORDER BY s.sequence_name, c.table_name, c.column_name`

// AlignSequencesToColumns avance chaque séquence en retard sur le maximum des colonnes
// qu'elle alimente (contrat en tête de fichier). Idempotente : une base saine, ou une base
// déjà alignée, n'est pas touchée (aucune valeur consommée, aucun log). Chaque séquence
// avancée est journalisée (base, séquence, avant, après) et comptée. Rend la liste des
// séquences avancées. Une séquence en échec n'empêche pas d'aligner les suivantes : la passe
// va au bout et rend les erreurs réunies (errors.Join).
func AlignSequencesToColumns(ctx context.Context, db *sql.DB) ([]SequenceAlignment, error) {
	uses, err := loadSequenceUses(ctx, db)
	if err != nil {
		return nil, err
	}
	var aligned []SequenceAlignment
	var errs []error
	for _, u := range uses {
		a, moved, err := alignSequence(ctx, db, u)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !moved {
			continue
		}
		aligned = append(aligned, a)
		observability.IncCounter(sequenceAlignedCounter)
		slog.InfoContext(ctx, "séquence alignée sur le max de sa colonne",
			"db", databasePath(ctx, db), "sequence", a.Sequence, "columns", u.columnList(),
			"next_before", a.NextBefore, "column_max", a.ColumnMax,
			"next_after", a.NextAfter, "consumed", a.Consumed)
	}
	return aligned, errors.Join(errs...)
}

// AlignSequencesBestEffort applique AlignSequencesToColumns à l'ouverture d'une base sans
// pouvoir la bloquer : une séquence en retard fait échouer des écritures, une ouverture
// refusée les ferait TOUTES échouer. L'échec est journalisé en ERROR et compté, jamais avalé.
func AlignSequencesBestEffort(ctx context.Context, db *sql.DB, caller string) {
	if _, err := AlignSequencesToColumns(ctx, db); err != nil {
		observability.IncCounter(sequenceAlignFailedCounter)
		slog.ErrorContext(ctx, "alignement des séquences échoué — collisions de clé possibles à l'insertion",
			"err", err, "db", databasePath(ctx, db), "caller", caller)
	}
}

func loadSequenceUses(ctx context.Context, db *sql.DB) ([]sequenceUse, error) {
	rows, err := db.QueryContext(ctx, sequenceUsesSQL)
	if err != nil {
		return nil, fmt.Errorf("séquences: lecture du catalogue: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var uses []sequenceUse
	for rows.Next() {
		var u sequenceUse
		var tc tableColumn
		if err := rows.Scan(&u.name, &u.increment, &u.maxValue, &u.cycle, &tc.table, &tc.column); err != nil {
			return nil, fmt.Errorf("séquences: scan du catalogue: %w", err)
		}
		if n := len(uses); n > 0 && uses[n-1].name == u.name {
			uses[n-1].columns = append(uses[n-1].columns, tc)
			continue
		}
		u.columns = []tableColumn{tc}
		uses = append(uses, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("séquences: itération du catalogue: %w", err)
	}
	return uses, nil
}

// alignSequence avance u si sa prochaine valeur ne dépasse pas le max de ses colonnes.
// moved=false : séquence saine (ou colonnes vides), rien n'a été consommé.
func alignSequence(ctx context.Context, db *sql.DB, u sequenceUse) (SequenceAlignment, bool, error) {
	a := SequenceAlignment{Sequence: u.name}
	colMax, found, err := u.columnMax(ctx, db)
	if err != nil || !found {
		return a, false, err
	}
	next, err := sequenceNextValue(ctx, db, u.name)
	if err != nil {
		return a, false, err
	}
	if next > colMax {
		return a, false, nil
	}
	if u.increment <= 0 || u.cycle {
		return a, false, fmt.Errorf("séquence %s en retard (prochaine %d <= max %d) mais non alignable "+
			"(increment %d, cycle %t)", u.name, next, colMax, u.increment, u.cycle)
	}
	if colMax > u.maxValue-u.increment {
		return a, false, fmt.Errorf("séquence %s : max de colonne %d au-delà de MAXVALUE %d", u.name, colMax, u.maxValue)
	}
	k := (colMax-next)/u.increment + 1
	if k > maxSequenceCatchUp {
		return a, false, fmt.Errorf("séquence %s : rattrapage de %d valeurs refusé (borne %d) — prochaine %d, max %d",
			u.name, k, maxSequenceCatchUp, next, colMax)
	}
	var last int64
	if err := db.QueryRowContext(ctx, `SELECT max(nextval(?)) FROM range(?)`, u.name, k).Scan(&last); err != nil {
		return a, false, fmt.Errorf("séquence %s : consommation de %d valeurs: %w", u.name, k, err)
	}
	after, err := sequenceNextValue(ctx, db, u.name)
	if err != nil {
		return a, false, err
	}
	if after <= colMax {
		return a, false, fmt.Errorf("séquence %s toujours en retard après consommation (prochaine %d, max %d)",
			u.name, after, colMax)
	}
	return SequenceAlignment{Sequence: u.name, ColumnMax: colMax, NextBefore: next, NextAfter: after, Consumed: k}, true, nil
}

// columnMax rend le maximum de toutes les colonnes alimentées par la séquence ;
// found=false quand toutes sont vides (ou entièrement NULL).
func (u sequenceUse) columnMax(ctx context.Context, db *sql.DB) (maxVal int64, found bool, err error) {
	for _, tc := range u.columns {
		var m sql.NullInt64
		q := fmt.Sprintf(`SELECT max(%s) FROM %s`, quoteIdentifier(tc.column), quoteIdentifier(tc.table))
		if err := db.QueryRowContext(ctx, q).Scan(&m); err != nil {
			return 0, false, fmt.Errorf("séquence %s : max de %s.%s: %w", u.name, tc.table, tc.column, err)
		}
		if m.Valid && (!found || m.Int64 > maxVal) {
			maxVal, found = m.Int64, true
		}
	}
	return maxVal, found, nil
}

func (u sequenceUse) columnList() string {
	parts := make([]string, len(u.columns))
	for i, tc := range u.columns {
		parts[i] = tc.table + "." + tc.column
	}
	return strings.Join(parts, ",")
}

// sequenceNextValue lit la prochaine valeur que rendra nextval, sans la consommer (cf. en-tête
// pour le choix de la colonne `sql`).
func sequenceNextValue(ctx context.Context, db *sql.DB, name string) (int64, error) {
	var next sql.NullInt64
	err := db.QueryRowContext(ctx, `
		SELECT TRY_CAST(regexp_extract(sql, ' START (-?[0-9]+)', 1) AS BIGINT)
		FROM duckdb_sequences()
		WHERE database_name = current_database() AND sequence_name = ?`, name).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("séquence %s : lecture du compteur: %w", name, err)
	}
	if !next.Valid {
		return 0, fmt.Errorf("séquence %s : compteur illisible dans duckdb_sequences().sql", name)
	}
	return next.Int64, nil
}

// databasePath — chemin du fichier de la base courante pour les logs (vide pour :memory: ou
// si le catalogue ne répond pas ; l'échec est journalisé en DEBUG, la clé reste vide).
func databasePath(ctx context.Context, db *sql.DB) string {
	var path string
	if err := db.QueryRowContext(ctx,
		`SELECT COALESCE(path, '') FROM duckdb_databases() WHERE database_name = current_database()`).Scan(&path); err != nil {
		slog.DebugContext(ctx, "séquences: chemin de la base indisponible", "err", err)
		return ""
	}
	return path
}

// quoteIdentifier cite un identifiant DuckDB (guillemets doublés).
func quoteIdentifier(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
