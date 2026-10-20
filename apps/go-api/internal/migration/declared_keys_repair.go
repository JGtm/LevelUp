package migration

// declared_keys_repair.go — en fin de cycle de migrations, rend à chaque table les clés
// (PRIMARY KEY, UNIQUE) que son schéma canonique déclare et que la base a perdues.
//
// Pourquoi : `CREATE TABLE IF NOT EXISTS` n'ajoute jamais une clé à une table existante, et une
// étape déjà inscrite au registre ne rejoue pas. Une table recréée sans ses contraintes (copie
// `CREATE TABLE AS` d'une restauration Parquet, repli d'un outil de réparation) les a donc
// perdues pour toujours, et chaque écriture `ON CONFLICT` / `INSERT OR IGNORE` sur elle échoue
// en Binder Error. Règle générale, pas une étape par table : le schéma canonique est celui que le
// MÊME jeu d'étapes produit sur une base vierge (declared_keys.go).
//
// Ce que la passe répare, et seulement cela : une table SANS AUCUNE clé (ni PRIMARY KEY ni
// UNIQUE) dont le schéma canonique déclare des clés. Une table qui porte déjà une clé, ou que le
// schéma déclare sans clé (par conception), n'est jamais touchée.
//
// Ce qu'elle refuse (journalisé en WARN/ERROR et compté, jamais silencieux) :
//   - une table append-only (une vue `<table>_latest` existe) : sa clé technique se rétablit par
//     une décision au cas par cas, pas au démarrage ;
//   - une table dont une colonne de clé manque, ou dont une colonne de clé est alimentée par une
//     séquence que la table réelle a perdue (la clé refuserait chaque nouvelle ligne) ;
//   - une table de plus de maxKeyRepairRows lignes (reconstruction trop lourde pour le démarrage).
//
// Reconstruction : dans UNE transaction, table neuve (colonnes, types, NOT NULL et défauts de la
// table réelle ; défaut perdu repris du schéma canonique ; clés du schéma canonique), recopie
// avec doublons écartés (duckdbkeys.DedupSelect : la ligne la plus récente gardée, sinon la plus
// petite selon toutes les colonnes), DROP de l'ancienne, renommage, index secondaires recréés.
// Un seul écrivain : la passe tourne dans le cycle de migrations du démarrage (runSteps), qui
// tient la base en écriture exclusive.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/pkg/duckdbkeys"
)

// maxKeyRepairRows : au-delà, reconstruire la table au démarrage retarderait le boot de façon
// déraisonnable (plusieurs millions de lignes recopiées) ; la réparation se fait alors hors
// ligne, sur décision.
const maxKeyRepairRows = 5_000_000

// Compteurs expvar (/debug/vars, clé "levelup").
const (
	keyRepairDoneCounter    = "migration_key_repair_tables_total"
	keyRepairSkippedCounter = "migration_key_repair_skipped_total"
	keyRepairFailedCounter  = "migration_key_repair_failed_total"
)

// keyRepairTable : une table à reconstruire et ce qu'il faut pour le faire.
type keyRepairTable struct {
	name    string
	keys    []duckdbkeys.Key
	cols    []duckdbkeys.Column // colonnes réelles, défauts perdus repris du schéma canonique
	indexes []tableIndex
}

// repairDeclaredKeys applique la passe à db pour le cycle (slug, target, steps). steps est le jeu
// que runSteps vient d'appliquer, dans l'ordre canonique. Ne rend rien : chaque échec est
// journalisé et compté, et n'empêche ni les autres tables ni le démarrage.
func repairDeclaredKeys(ctx context.Context, db *sql.DB, slug string, target TargetDB, steps []Migration) {
	if !referenceTrusted(slug, target) {
		slog.DebugContext(ctx, "migration: réparation des clés sautée (jeu d'étapes title-owned non câblé)",
			"title", slug, "target", target)
		return
	}
	candidates, err := keylessTables(ctx, db)
	if err != nil || len(candidates) == 0 {
		if err != nil {
			observability.IncCounter(keyRepairFailedCounter)
			slog.ErrorContext(ctx, "migration: réparation des clés impossible (lecture du catalogue)",
				"title", slug, "target", target, "err", err)
		}
		return
	}
	ref, err := referenceFor(ctx, slug, target, steps)
	if err != nil {
		observability.IncCounter(keyRepairFailedCounter)
		slog.ErrorContext(ctx, "migration: réparation des clés impossible (schéma de référence)",
			"title", slug, "target", target, "err", err)
		return
	}
	repaired := 0
	for _, name := range candidates {
		rt, ok := planKeyRepair(ctx, db, ref, name)
		if !ok {
			continue
		}
		if err := rebuildWithKeys(ctx, db, rt); err != nil {
			observability.IncCounter(keyRepairFailedCounter)
			slog.ErrorContext(ctx, "migration: réparation des clés échouée (table inchangée)",
				"title", slug, "target", target, "table", name, "err", err)
			continue
		}
		repaired++
	}
	if repaired > 0 {
		if _, err := db.ExecContext(ctx, "CHECKPOINT"); err != nil {
			slog.WarnContext(ctx, "migration: CHECKPOINT après réparation des clés échoué (non fatal)",
				"target", target, "err", err)
		}
	}
}

// keylessTables liste les tables de base de la base courante qui ne portent aucune clé.
func keylessTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT t.table_name FROM duckdb_tables() t
		WHERE t.database_name = current_database() AND t.schema_name = 'main' AND NOT t.temporary
		  AND NOT EXISTS (
		    SELECT 1 FROM duckdb_constraints() c
		    WHERE c.database_name = t.database_name AND c.schema_name = t.schema_name
		      AND c.table_name = t.table_name AND c.constraint_type IN ('PRIMARY KEY', 'UNIQUE'))
		ORDER BY t.table_name`)
	if err != nil {
		return nil, fmt.Errorf("tables sans clé: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("tables sans clé: lecture: %w", err)
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// planKeyRepair décide si la table sans clé `name` se répare, et prépare sa reconstruction.
// ok=false : rien à faire (le schéma canonique ne déclare pas de clé) ou refus journalisé.
func planKeyRepair(ctx context.Context, db *sql.DB, ref *keyReference, name string) (keyRepairTable, bool) {
	refTable, known := ref.tables[name]
	if !known || len(refTable.keys) == 0 {
		return keyRepairTable{}, false
	}
	refuse := func(reason string, args ...any) (keyRepairTable, bool) {
		observability.IncCounter(keyRepairSkippedCounter)
		slog.WarnContext(ctx, "migration: clé déclarée absente, réparation refusée — "+reason,
			append([]any{"table", name, "db", databasePath(ctx, db)}, args...)...)
		return keyRepairTable{}, false
	}
	appendOnly, err := viewExists(ctx, db, name+"_latest")
	if err != nil {
		return refuse("vues illisibles", "err", err)
	}
	if appendOnly {
		return refuse("table append-only (vue _latest) : décision au cas par cas")
	}
	cols, err := duckdbkeys.TableColumns(ctx, db, name)
	if err != nil {
		return refuse("colonnes illisibles", "err", err)
	}
	restoreCanonicalDefaults(cols, refTable)
	if reason, col := keyColumnsUnusable(cols, refTable); reason != "" {
		return refuse(reason, "column", col)
	}
	var rows int64
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+duckdbkeys.Ident(name)).Scan(&rows); err != nil {
		return refuse("comptage impossible", "err", err)
	}
	if rows > maxKeyRepairRows {
		observability.IncCounter(keyRepairSkippedCounter)
		slog.ErrorContext(ctx, "migration: clé déclarée absente sur une table trop grosse pour le démarrage — réparation hors ligne à décider",
			"table", name, "rows", rows, "max", maxKeyRepairRows, "db", databasePath(ctx, db))
		return keyRepairTable{}, false
	}
	indexes, err := tableSecondaryIndexes(db, name)
	if err != nil {
		return refuse("index illisibles", "err", err)
	}
	return keyRepairTable{name: name, keys: refTable.keys, cols: cols, indexes: indexes}, true
}

// restoreCanonicalDefaults reprend du schéma canonique le défaut qu'une colonne réelle a perdu
// (même type). Un défaut `nextval(…)` n'est jamais repris : la séquence peut manquer à la base.
func restoreCanonicalDefaults(cols []duckdbkeys.Column, refTable referenceTable) {
	for i, c := range cols {
		rc, ok := refTable.cols[c.Name]
		if ok && c.Default == "" && strings.EqualFold(rc.Type, c.Type) && !usesSequence(rc.Default) {
			cols[i].Default = rc.Default
		}
	}
}

// keyColumnsUnusable rend la raison (et la colonne) pour laquelle une clé canonique ne peut pas
// être posée sur les colonnes réelles : colonne absente, ou alimentée par une séquence dont la
// table réelle a perdu le défaut (la clé refuserait chaque nouvelle ligne). "" si utilisable.
func keyColumnsUnusable(cols []duckdbkeys.Column, refTable referenceTable) (reason, column string) {
	for _, k := range refTable.keys {
		for _, kc := range k.Columns {
			def, present := columnDefault(cols, kc)
			if !present {
				return "colonne de clé absente", kc
			}
			if usesSequence(refTable.cols[kc].Default) && def == "" {
				return "colonne de clé alimentée par une séquence perdue", kc
			}
		}
	}
	return "", ""
}

// usesSequence indique si un défaut tire une valeur d'une séquence.
func usesSequence(def string) bool { return strings.Contains(def, "nextval(") }

// columnDefault rend le défaut de la colonne `name` de cols et sa présence.
func columnDefault(cols []duckdbkeys.Column, name string) (string, bool) {
	for _, c := range cols {
		if c.Name == name {
			return c.Default, true
		}
	}
	return "", false
}

// viewExists indique si la base courante porte la vue `name` (schéma main).
func viewExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT count(*) FROM duckdb_views()
		WHERE database_name = current_database() AND schema_name = 'main' AND view_name = ?`, name).Scan(&n)
	return n > 0, err
}

// rebuildWithKeys reconstruit la table avec ses clés, dans une transaction (contrat en tête de
// fichier), et journalise ce qui a été gardé et écarté.
func rebuildWithKeys(ctx context.Context, db *sql.DB, rt keyRepairTable) error {
	start := time.Now()
	tmp := rt.name + "__keyfix"
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, keyedTableDDL(tmp, rt.cols, rt.keys)); err != nil {
		return fmt.Errorf("création de la table à clés: %w", err)
	}
	names := duckdbkeys.ColumnNames(rt.cols)
	st, err := duckdbkeys.CopyInto(ctx, tx, tmp, duckdbkeys.Ident(rt.name), names, rt.keys,
		duckdbkeys.RecencyColumn(rt.cols))
	if err != nil {
		return err
	}
	for _, stmt := range []string{
		"DROP TABLE " + duckdbkeys.Ident(rt.name),
		fmt.Sprintf("ALTER TABLE %s RENAME TO %s", duckdbkeys.Ident(tmp), duckdbkeys.Ident(rt.name)),
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("%s: %w", firstWords(stmt, 2), err)
		}
	}
	for _, idx := range rt.indexes {
		if _, err := tx.ExecContext(ctx, idx.ddl); err != nil {
			return fmt.Errorf("index %s: %w", idx.name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	observability.IncCounter(keyRepairDoneCounter)
	level := slog.LevelInfo
	if st.NullKeyDropped+st.DuplicatesDropped > 0 {
		level = slog.LevelWarn
	}
	slog.Log(ctx, level, "migration: clés déclarées rétablies",
		"table", rt.name, "keys", describeKeys(rt.keys), "rows_before", st.Source, "rows_kept", st.Copied,
		"null_key_dropped", st.NullKeyDropped, "duplicates_dropped", st.DuplicatesDropped,
		"divergent_keys", st.DivergentKeys, "duration_ms", time.Since(start).Milliseconds(),
		"db", databasePath(ctx, db))
	return nil
}

// keyedTableDDL rend le CREATE TABLE de `name` : colonnes (type, défaut, NOT NULL) puis clés.
func keyedTableDDL(name string, cols []duckdbkeys.Column, keys []duckdbkeys.Key) string {
	parts := make([]string, 0, len(cols)+len(keys))
	for _, c := range cols {
		def := duckdbkeys.Ident(c.Name) + " " + c.Type
		if c.Default != "" {
			def += " DEFAULT " + c.Default
		}
		if c.NotNull {
			def += " NOT NULL"
		}
		parts = append(parts, def)
	}
	for _, k := range keys {
		kind := "UNIQUE"
		if k.Primary {
			kind = "PRIMARY KEY"
		}
		quoted := make([]string, len(k.Columns))
		for i, c := range k.Columns {
			quoted[i] = duckdbkeys.Ident(c)
		}
		parts = append(parts, kind+" ("+strings.Join(quoted, ", ")+")")
	}
	return "CREATE TABLE " + duckdbkeys.Ident(name) + " (\n\t" + strings.Join(parts, ",\n\t") + "\n)"
}

// describeKeys rend les clés sous une forme lisible pour le journal.
func describeKeys(keys []duckdbkeys.Key) string {
	out := make([]string, len(keys))
	for i, k := range keys {
		kind := "UNIQUE"
		if k.Primary {
			kind = "PK"
		}
		out[i] = kind + "(" + strings.Join(k.Columns, ",") + ")"
	}
	return strings.Join(out, " ")
}
