package migration

// steps_player_repair_append_only_ids.go — repair_player_append_only_ids_v1 : rend une clé
// technique valide aux tables append-only joueur dont la colonne `id` porte des NULL ou des
// doublons, ou n'est pas clé primaire.
//
// Pourquoi. Ces tables déclarent PRIMARY KEY (id) alimentée par une séquence. Deux défauts ont
// été relevés sur des bases réelles (copies du 2026-10-08) :
//   - des ids en DOUBLE dans une clé déclarée (player_match_enrichment, player_csr_snapshots) :
//     des lignes DISTINCTES écrites après qu'une séquence est repartie en arrière ont reçu un id
//     déjà pris. L'index ART de la clé contient alors deux fois la même clé, état qu'un index
//     unique ne devrait jamais connaître (famille #23645) ;
//   - des ids NULL (match_skill_rank, player_csr_snapshots) et une player_csr_snapshots sans clé
//     ni défaut, restée hors de la conversion append-only (le marqueur `id` était présent).
// Aucune lecture applicative ne joint sur `id` (toutes passent par les vues _latest, où il ne
// sert qu'à départager deux versions du même instant) ; le risque est l'écriture et toute
// reconstruction future (ADD PRIMARY KEY échoue sur une table qui porte ces défauts).
//
// Remède SANS risque ART (ADR 0026) : aucun UPDATE ni DELETE. La table est reconstruite à côté
// (swapTableTx : une transaction, garde de cardinalité, rollback intégral) : la première ligne
// (ordre d'insertion, rowid) d'un id en double le garde, chaque autre ligne en double et chaque
// id NULL reçoit un id neuf de la séquence, réalignée d'abord au-dessus du max. Les DEFAULT et
// NOT NULL de l'ancienne table sont reposés, la clé primaire et le DEFAULT de l'id aussi. Les
// vues _latest se relient à la table renommée (DuckDB ne lie pas une vue à l'objet supprimé).
// No-op sur une table saine : le diagnostic précède toute écriture.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// appendOnlyIDTable — une table append-only joueur, la séquence de son id et sa colonne
// d'horloge (DEFAULT UTC garanti si l'ancienne table l'avait perdu).
type appendOnlyIDTable struct {
	table, seq, clock string
}

// clockWrittenAt — colonne d'horloge des tables append-only joueur (sauf lusr_component_history).
const clockWrittenAt = "written_at"

var appendOnlyIDTables = []appendOnlyIDTable{
	{"match_skill_rank", "msr_seq", clockWrittenAt},
	{"player_match_enrichment", "pme_seq", clockWrittenAt},
	{"player_csr_snapshots", "pcs_seq", clockWrittenAt},
	{"lusr_component_history", "lch_seq", "computed_at"},
	{"personal_score_awards", "personal_score_awards_id_seq", clockWrittenAt},
}

// idRepairSuffix — suffixe de la table de construction du swap.
const idRepairSuffix = "__idfix"

// idDefects — diagnostic de la colonne id d'une table.
type idDefects struct {
	rows, nulls, dups int64
	hasPK             bool
}

func (d idDefects) clean() bool { return d.nulls == 0 && d.dups == 0 && d.hasPK }

// columnRule — DEFAULT et NOT NULL d'une colonne, relus avant le swap pour être reposés.
type columnRule struct {
	name, dataType, defaultExpr string
	notNull                     bool
}

func applyRepairAppendOnlyIDs(db *sql.DB) error {
	ctx := bootCtx()
	for _, t := range appendOnlyIDTables {
		if err := repairAppendOnlyIDs(ctx, db, t); err != nil {
			return err
		}
	}
	return nil
}

func repairAppendOnlyIDs(ctx context.Context, db *sql.DB, t appendOnlyIDTable) error {
	if err := recoverOrphanTable(ctx, db, t.table, idRepairSuffix); err != nil {
		return err
	}
	if ok, err := tableExists(db, t.table); err != nil || !ok {
		return err
	}
	if ok, err := columnExists(db, t.table, "id"); err != nil || !ok {
		return err
	}
	d, err := loadIDDefects(ctx, db, t.table)
	if err != nil || d.clean() {
		return err
	}
	rules, err := loadColumnRules(ctx, db, t.table)
	if err != nil {
		return err
	}
	if err := prepareIDSequence(ctx, db, t); err != nil {
		return err
	}
	n, err := swapTableTx(ctx, db, tableSwap{
		Table:      t.table,
		Suffix:     idRepairSuffix,
		Expected:   fmt.Sprintf(`SELECT COUNT(*) FROM %s`, t.table),
		Build:      []string{buildIDRepairCTAS(t, rules)},
		PostRename: idRepairPostRename(t, rules),
		Verify:     verifyUniqueIDs(t.table),
	})
	if err != nil {
		return fmt.Errorf("réparation des ids de %s: %w", t.table, err)
	}
	slog.WarnContext(ctx, "migration: ids de table append-only réparés (swap, aucune ligne perdue)",
		"table", t.table, "rows", n.Rebuilt, "null_ids", d.nulls, "duplicate_ids", d.dups, "had_primary_key", d.hasPK)
	return nil
}

func loadIDDefects(ctx context.Context, db *sql.DB, table string) (idDefects, error) {
	var d idDefects
	if err := db.QueryRowContext(ctx, fmt.Sprintf(
		`SELECT COUNT(*), COUNT(*) - COUNT(id), COUNT(id) - COUNT(DISTINCT id) FROM %s`, table),
	).Scan(&d.rows, &d.nulls, &d.dups); err != nil {
		return d, fmt.Errorf("diagnostic des ids de %s: %w", table, err)
	}
	var pk int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM duckdb_constraints()
		WHERE database_name = current_database() AND schema_name = 'main' AND table_name = ?
		  AND constraint_type = 'PRIMARY KEY' AND constraint_column_names = ['id']`, table).Scan(&pk); err != nil {
		return d, fmt.Errorf("clé primaire de %s: %w", table, err)
	}
	d.hasPK = pk > 0
	return d, nil
}

func loadColumnRules(ctx context.Context, db *sql.DB, table string) ([]columnRule, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT column_name, data_type, COALESCE(column_default, ''), NOT is_nullable
		FROM duckdb_columns()
		WHERE database_name = current_database() AND schema_name = 'main' AND table_name = ?
		ORDER BY column_index`, table)
	if err != nil {
		return nil, fmt.Errorf("colonnes de %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var out []columnRule
	for rows.Next() {
		var r columnRule
		if err := rows.Scan(&r.name, &r.dataType, &r.defaultExpr, &r.notNull); err != nil {
			return nil, fmt.Errorf("colonnes de %s (scan): %w", table, err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// prepareIDSequence crée la séquence si elle manque et l'avance au-dessus du max des ids :
// un id neuf ne doit jamais reprendre un id existant.
func prepareIDSequence(ctx context.Context, db *sql.DB, t appendOnlyIDTable) error {
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE SEQUENCE IF NOT EXISTS %s START 1`, t.seq)); err != nil {
		return fmt.Errorf("séquence %s: %w", t.seq, err)
	}
	u := sequenceUse{name: t.seq, columns: []tableColumn{{table: t.table, column: "id"}}}
	if err := db.QueryRowContext(ctx, `
		SELECT increment_by, max_value, cycle FROM duckdb_sequences()
		WHERE database_name = current_database() AND sequence_name = ?`, t.seq).
		Scan(&u.increment, &u.maxValue, &u.cycle); err != nil {
		return fmt.Errorf("séquence %s (catalogue): %w", t.seq, err)
	}
	if _, _, err := alignSequence(ctx, db, u); err != nil {
		return fmt.Errorf("séquence %s (alignement): %w", t.seq, err)
	}
	return nil
}

// buildIDRepairCTAS : même colonnes, même ordre ; un id NULL ou une occurrence non première
// d'un id reçoit nextval, casté au type d'origine de la colonne.
func buildIDRepairCTAS(t appendOnlyIDTable, rules []columnRule) string {
	idType := "BIGINT"
	for _, r := range rules {
		if r.name == "id" {
			idType = r.dataType
		}
	}
	return fmt.Sprintf(`CREATE TABLE %[1]s%[2]s AS
		SELECT * EXCLUDE (__idfix_rn, __idfix_rowid) REPLACE (
			CAST(CASE WHEN id IS NULL OR __idfix_rn > 1 THEN nextval('%[3]s') ELSE id END AS %[4]s) AS id)
		FROM (
			SELECT *, rowid AS __idfix_rowid,
				ROW_NUMBER() OVER (PARTITION BY id ORDER BY rowid) AS __idfix_rn
			FROM %[1]s)
		ORDER BY __idfix_rowid`, t.table, idRepairSuffix, t.seq, idType)
}

// idRepairPostRename : clé primaire, DEFAULT de l'id, DEFAULT et NOT NULL relus sur l'ancienne
// table, et DEFAULT UTC de la colonne d'horloge si l'ancienne table n'en avait pas.
func idRepairPostRename(t appendOnlyIDTable, rules []columnRule) []string {
	stmts := []string{
		fmt.Sprintf(`ALTER TABLE %s ADD PRIMARY KEY (id)`, t.table),
		fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN id SET DEFAULT nextval('%s')`, t.table, t.seq),
	}
	for _, r := range rules {
		if r.name == "id" {
			continue
		}
		def := r.defaultExpr
		if def == "" && strings.EqualFold(r.name, t.clock) {
			def = TimestampDefaultUTC
		}
		if def != "" {
			stmts = append(stmts, fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s`,
				t.table, quoteIdentifier(r.name), def))
		}
		if r.notNull {
			stmts = append(stmts, fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s SET NOT NULL`,
				t.table, quoteIdentifier(r.name)))
		}
	}
	return stmts
}

func verifyUniqueIDs(table string) func(context.Context, *sql.Tx) error {
	return func(ctx context.Context, tx *sql.Tx) error {
		var nulls, dups int64
		if err := tx.QueryRowContext(ctx, fmt.Sprintf(
			`SELECT COUNT(*) - COUNT(id), COUNT(id) - COUNT(DISTINCT id) FROM %s`, table),
		).Scan(&nulls, &dups); err != nil {
			return err
		}
		if nulls != 0 || dups != 0 {
			return fmt.Errorf("%s : %d id NULL et %d id en double après reconstruction", table, nulls, dups)
		}
		return nil
	}
}
