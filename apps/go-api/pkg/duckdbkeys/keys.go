// Package duckdbkeys lit les clés déclarées (PRIMARY KEY, UNIQUE) et les colonnes d'une table
// DuckDB, et recopie des lignes vers une table à clés en écartant les doublons de façon
// déterministe.
//
// Appelants : la réparation, au démarrage, des tables dont la base a perdu les clés que son
// schéma déclare (internal/migration), l'écriture sans ON CONFLICT (Upsert) et la citation
// d'identifiants de la restauration (pkg/duckdbbackup). Le paquet ne dépend d'aucun paquet du projet.
//
// Toutes les lectures portent sur la base COURANTE (`current_database()`), schéma `main` : un
// catalogue attaché n'est jamais lu ni écrit.
package duckdbkeys

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Querier est la surface commune de *sql.DB, *sql.Tx et *sql.Conn utilisée ici.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Key est une contrainte de clé d'une table : la clé primaire ou une contrainte UNIQUE.
type Key struct {
	Primary bool
	Columns []string
}

// Column décrit une colonne telle que le catalogue DuckDB la rend.
type Column struct {
	Name    string
	Type    string
	Default string // texte du DEFAULT, "" si aucun
	NotNull bool
}

// Ident cite un identifiant DuckDB (guillemets doublés).
func Ident(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// Literal cite une chaîne SQL (apostrophes doublées), pour les fonctions qui exigent un
// littéral (ex. `read_parquet`).
func Literal(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// TableKeys rend les clés (PRIMARY KEY d'abord, puis UNIQUE dans l'ordre du catalogue) de la
// table `table` de la base courante. Une table sans clé rend une liste vide.
func TableKeys(ctx context.Context, q Querier, table string) ([]Key, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT constraint_type = 'PRIMARY KEY', to_json(constraint_column_names)::VARCHAR
		FROM duckdb_constraints()
		WHERE database_name = current_database() AND schema_name = 'main'
		  AND table_name = ? AND constraint_type IN ('PRIMARY KEY', 'UNIQUE')
		ORDER BY constraint_type = 'PRIMARY KEY' DESC, constraint_index`, table)
	if err != nil {
		return nil, fmt.Errorf("clés de %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var keys []Key
	seen := map[string]bool{}
	for rows.Next() {
		var k Key
		var colsJSON string
		if err := rows.Scan(&k.Primary, &colsJSON); err != nil {
			return nil, fmt.Errorf("clés de %s: lecture: %w", table, err)
		}
		if err := json.Unmarshal([]byte(colsJSON), &k.Columns); err != nil {
			return nil, fmt.Errorf("clés de %s: colonnes %q: %w", table, colsJSON, err)
		}
		sig := fmt.Sprint(k.Primary, k.Columns)
		if len(k.Columns) == 0 || seen[sig] {
			continue
		}
		seen[sig] = true
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// TableColumns rend les colonnes de la table `table` de la base courante, dans leur ordre.
func TableColumns(ctx context.Context, q Querier, table string) ([]Column, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT column_name, data_type, COALESCE(column_default, ''), NOT is_nullable
		FROM duckdb_columns()
		WHERE database_name = current_database() AND schema_name = 'main' AND table_name = ?
		ORDER BY column_index`, table)
	if err != nil {
		return nil, fmt.Errorf("colonnes de %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var cols []Column
	for rows.Next() {
		var c Column
		if err := rows.Scan(&c.Name, &c.Type, &c.Default, &c.NotNull); err != nil {
			return nil, fmt.Errorf("colonnes de %s: lecture: %w", table, err)
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// ColumnNames rend les noms de cols, dans leur ordre.
func ColumnNames(cols []Column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Name
	}
	return out
}

// recencyPreference : noms d'horodatage préférés pour départager deux lignes d'une même clé,
// du plus parlant (dernière mise à jour) au moins parlant (création).
var recencyPreference = []string{
	"updated_at", "written_at", "last_seen_at", "last_seen", "last_fetched_at",
	"fetched_at", "recorded_at", "created_at", "applied_at",
}

// RecencyColumn choisit la colonne d'horodatage qui départage les doublons d'une clé (la ligne
// la plus récente est gardée) : la première de recencyPreference qui existe avec un type
// TIMESTAMP/DATE, sinon la première colonne de ce type, sinon "" (aucun horodatage).
func RecencyColumn(cols []Column) string {
	timed := map[string]bool{}
	first := ""
	for _, c := range cols {
		t := strings.ToUpper(c.Type)
		if strings.HasPrefix(t, "TIMESTAMP") || t == "DATE" {
			timed[c.Name] = true
			if first == "" {
				first = c.Name
			}
		}
	}
	for _, name := range recencyPreference {
		if timed[name] {
			return name
		}
	}
	return first
}
