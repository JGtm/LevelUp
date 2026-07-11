package migration

// schema_snapshot.go — outil de SNAPSHOT de schéma DuckDB normalisé et déterministe
// (chantier squash N4, M1). Extrait le schéma COMPLET d'une DB (tables, colonnes,
// contraintes, index, vues, séquences) et le sérialise de façon NORMALISÉE :
//
//   - objets de premier niveau (tables, vues, index, séquences, contraintes) triés
//     par une clé lexicale stable → indépendant de l'ordre d'exécution des steps là
//     où cet ordre n'a aucun effet observable ;
//   - colonnes d'une table conservées dans leur ordre POSITIONNEL (column_index) —
//     l'ordre des colonnes EST observable (SELECT *, INSERT positionnel), on ne le
//     trie donc pas : deux schémas ne différant que par l'ordre des colonnes DOIVENT
//     produire un snapshot différent (sinon fausse équivalence).
//
// SCHÉMA SEUL, zéro DONNÉE : aucune ligne n'est lue (pas de SELECT sur les tables
// métier). Le snapshot est donc insensible aux seeds/backfills — c'est voulu : il
// sert à prouver l'égalité de SCHÉMA entre (historique complet) et (baseline + reste)
// pour le squash. Corollaire : un registre dont la valeur tient à des SEEDS (données)
// n'est pas un bon candidat au squash via ce seul invariant (cf. plan M0e).
//
// Réutilisable : appelé par le test d'invariant (squash_invariant_test.go, M2) et
// destiné à un futur cmd/schema-snapshot (M5c — snapshot d'une DB fichier avant/après
// boot). Dé-risque aussi E7 (prouver que Ensure*Schema et les migrations produisent le
// même schéma) — cf. plan §4 Synergies.

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// SchemaSnapshot retourne une représentation textuelle normalisée et DÉTERMINISTE du
// schéma de db (deux appels sur deux DB provisionnées à l'identique → chaîne
// identique octet pour octet). db doit être ouverte ; aucune écriture n'est faite.
func SchemaSnapshot(db *sql.DB) (string, error) {
	var b strings.Builder
	for _, section := range []struct {
		title string
		fn    func(*sql.DB) ([]string, error)
	}{
		{"TABLES", snapshotTables},
		{"COLUMNS", snapshotColumns},
		{"CONSTRAINTS", snapshotConstraints},
		{"INDEXES", snapshotIndexes},
		{"VIEWS", snapshotViews},
		{"SEQUENCES", snapshotSequences},
	} {
		lines, err := section.fn(db)
		if err != nil {
			return "", fmt.Errorf("snapshot %s: %w", section.title, err)
		}
		sort.Strings(lines)
		b.WriteString("=== ")
		b.WriteString(section.title)
		b.WriteString(" ===\n")
		for _, l := range lines {
			b.WriteString(l)
			b.WriteByte('\n')
		}
	}
	return b.String(), nil
}

// collectLines exécute q et applique scan à chaque ligne, retournant les chaînes
// produites (non triées — l'appelant/SchemaSnapshot trie).
func collectLines(db *sql.DB, q string, scan func(*sql.Rows) (string, error)) ([]string, error) {
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		s, err := scan(rows)
		if err != nil {
			return nil, err
		}
		if s != "" {
			out = append(out, s)
		}
	}
	return out, rows.Err()
}

func snapshotTables(db *sql.DB) ([]string, error) {
	// internal=false : exclut les tables système DuckDB. temporary exclu.
	return collectLines(db, `
		SELECT schema_name, table_name, has_primary_key
		FROM duckdb_tables() WHERE internal = false AND temporary = false`,
		func(r *sql.Rows) (string, error) {
			var schema, name string
			var hasPK bool
			if err := r.Scan(&schema, &name, &hasPK); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s.%s pk=%t", schema, name, hasPK), nil
		})
}

func snapshotColumns(db *sql.DB) ([]string, error) {
	// Clé de tri = schema.table + column_index zéro-paddé → l'ordre positionnel des
	// colonnes est préservé DANS chaque table après le sort lexical global.
	return collectLines(db, `
		SELECT c.schema_name, c.table_name, c.column_index, c.column_name,
		       c.data_type, c.is_nullable, COALESCE(c.column_default, '')
		FROM duckdb_columns() c
		JOIN duckdb_tables() t
		  ON t.schema_name = c.schema_name AND t.table_name = c.table_name
		WHERE t.internal = false AND t.temporary = false`,
		func(r *sql.Rows) (string, error) {
			var schema, table, colName, dataType, def string
			var colIdx int
			var nullable bool
			if err := r.Scan(&schema, &table, &colIdx, &colName, &dataType, &nullable, &def); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s.%s|%04d %s %s null=%t default=%q",
				schema, table, colIdx, colName, dataType, nullable, def), nil
		})
}

func snapshotConstraints(db *sql.DB) ([]string, error) {
	return collectLines(db, `
		SELECT schema_name, table_name, constraint_type,
		       COALESCE(constraint_text, ''),
		       COALESCE(CAST(constraint_column_names AS VARCHAR), '')
		FROM duckdb_constraints()`,
		func(r *sql.Rows) (string, error) {
			var schema, table, ctype, ctext, ccols string
			if err := r.Scan(&schema, &table, &ctype, &ctext, &ccols); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s.%s %s cols=%s text=%q", schema, table, ctype, ccols, ctext), nil
		})
}

func snapshotIndexes(db *sql.DB) ([]string, error) {
	return collectLines(db, `
		SELECT schema_name, table_name, index_name, is_unique, is_primary, COALESCE(sql, '')
		FROM duckdb_indexes()`,
		func(r *sql.Rows) (string, error) {
			var schema, table, name, indexSQL string
			var unique, primary bool
			if err := r.Scan(&schema, &table, &name, &unique, &primary, &indexSQL); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s.%s %s unique=%t primary=%t sql=%q",
				schema, table, name, unique, primary, normalizeSQL(indexSQL)), nil
		})
}

func snapshotViews(db *sql.DB) ([]string, error) {
	return collectLines(db, `
		SELECT schema_name, view_name, COALESCE(sql, '')
		FROM duckdb_views() WHERE internal = false AND temporary = false`,
		func(r *sql.Rows) (string, error) {
			var schema, name, viewSQL string
			if err := r.Scan(&schema, &name, &viewSQL); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s.%s sql=%q", schema, name, normalizeSQL(viewSQL)), nil
		})
}

func snapshotSequences(db *sql.DB) ([]string, error) {
	return collectLines(db, `
		SELECT schema_name, sequence_name, start_value, min_value, max_value, increment_by, cycle
		FROM duckdb_sequences()`,
		func(r *sql.Rows) (string, error) {
			var schema, name string
			var start, minV, maxV, incr int64
			var cycle bool
			if err := r.Scan(&schema, &name, &start, &minV, &maxV, &incr, &cycle); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s.%s start=%d min=%d max=%d incr=%d cycle=%t",
				schema, name, start, minV, maxV, incr, cycle), nil
		})
}

// normalizeSQL réduit le bruit de mise en forme d'une définition SQL (vues, index)
// que DuckDB régénère : espaces multiples/retours ligne → un espace unique, trim.
// DuckDB ré-expanse déjà `SELECT *` et canonicalise ; cette étape absorbe les
// variations de whitespace résiduelles pour un snapshot stable.
func normalizeSQL(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
