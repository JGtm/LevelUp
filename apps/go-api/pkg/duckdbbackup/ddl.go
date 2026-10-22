package duckdbbackup

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/platform/atomicfile"
)

// Le schéma de la base (séquences, tables avec leurs clés, défauts et NOT NULL, index, vues)
// accompagne ses Parquet dans le staging : un Parquet ne porte que des lignes. Il est pris au
// MÊME instant que les données et que le registre des migrations : la restauration rend donc la
// base dans l'état de son époque, et le cycle de migrations du démarrage suivant la fait avancer
// comme il l'aurait fait sur la base d'origine. Fichier `_ddl_<horodatage>.json`.
const (
	ddlFilePrefix = "_ddl_"
	ddlFileSuffix = ".json"
)

// schemaDDL : le schéma exporté d'une base.
type schemaDDL struct {
	Sequences []string          `json:"sequences"` // CREATE SEQUENCE (compteur courant compris)
	Tables    map[string]string `json:"tables"`    // table → CREATE TABLE
	Indexes   []string          `json:"indexes"`   // CREATE [UNIQUE] INDEX (hors clés)
	Views     []string          `json:"views"`     // CREATE VIEW
}

// schemaQueries : objets du schéma main de la base courante, avec leur DDL (nom, sql).
var schemaQueries = struct{ sequences, tables, indexes, views string }{
	sequences: `SELECT sequence_name, sql FROM duckdb_sequences()
		WHERE database_name = current_database() AND schema_name = 'main' AND NOT temporary
		ORDER BY sequence_name`,
	tables: `SELECT table_name, sql FROM duckdb_tables()
		WHERE database_name = current_database() AND schema_name = 'main' AND NOT temporary
		ORDER BY table_name`,
	indexes: `SELECT index_name, sql FROM duckdb_indexes()
		WHERE database_name = current_database() AND schema_name = 'main'
		ORDER BY index_name`,
	views: `SELECT view_name, sql FROM duckdb_views()
		WHERE database_name = current_database() AND schema_name = 'main'
		  AND NOT internal AND NOT temporary
		ORDER BY view_name`,
}

// writeSchemaDDL écrit le schéma de db à path, atomiquement (un fichier interrompu ne peut pas
// être pris pour un schéma complet). Best-effort : un échec est journalisé en WARN sans faire
// échouer la sauvegarde — les lignes restent sauvegardées, la restauration de cette sauvegarde
// ne rendra que les tables et leurs lignes (CREATE TABLE AS : ni clé, ni défaut, ni séquence, ni vue).
func writeSchemaDDL(ctx context.Context, db *sql.DB, key, path string) {
	schema, err := readSchemaDDL(ctx, db)
	if err == nil {
		var data []byte
		if data, err = json.MarshalIndent(schema, "", "  "); err == nil {
			err = atomicfile.WriteFileStrict(path, data, 0o644)
		}
	}
	if err != nil {
		slog.WarnContext(ctx, "backup: schéma de la base non écrit (restauration sans clé pour cette sauvegarde)",
			"key", key, "err", err)
	}
}

// readSchemaDDL relève le schéma de la base courante.
func readSchemaDDL(ctx context.Context, db *sql.DB) (*schemaDDL, error) {
	s := &schemaDDL{Tables: map[string]string{}}
	collect := func(q string, add func(name, stmt string)) error {
		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			return fmt.Errorf("schéma: %w", err)
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var name string
			var stmt sql.NullString
			if err := rows.Scan(&name, &stmt); err != nil {
				return fmt.Errorf("schéma: lecture: %w", err)
			}
			if stmt.Valid && strings.TrimSpace(stmt.String) != "" {
				add(name, stmt.String)
			}
		}
		return rows.Err()
	}
	steps := []struct {
		q   string
		add func(name, stmt string)
	}{
		{schemaQueries.sequences, func(_, st string) { s.Sequences = append(s.Sequences, st) }},
		{schemaQueries.tables, func(n, st string) { s.Tables[n] = st }},
		{schemaQueries.indexes, func(_, st string) { s.Indexes = append(s.Indexes, st) }},
		{schemaQueries.views, func(_, st string) { s.Views = append(s.Views, st) }},
	}
	for _, step := range steps {
		if err := collect(step.q, step.add); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// loadSchemaDDL lit le schéma exporté par le MÊME cycle que les Parquet retenus (horodatage
// `ts`) : un schéma d'un autre export ne décrit pas ces lignes et n'est jamais rejoué. Aucun
// schéma de ce cycle : (nil, other, nil), other indiquant qu'un schéma d'un autre export est
// présent (ignoré). Fichier illisible ou incomplet : erreur — l'appelant replie la clé sur des
// tables sans clé, sans arrêter la restauration.
func loadSchemaDDL(dir, ts string) (s *schemaDDL, other bool, err error) {
	matches, err := filepath.Glob(filepath.Join(dir, ddlFilePrefix+"*"+ddlFileSuffix))
	if err != nil {
		return nil, false, err
	}
	want := filepath.Join(dir, ddlFilePrefix+ts+ddlFileSuffix)
	for _, m := range matches {
		if ts != "" && filepath.Base(m) == filepath.Base(want) {
			s, err := readSchemaFile(m)
			return s, false, err
		}
	}
	return nil, len(matches) > 0, nil
}

// readSchemaFile lit et valide un fichier de schéma exporté.
func readSchemaFile(path string) (*schemaDDL, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture du schéma %s: %w", path, err)
	}
	var s schemaDDL
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("schéma %s illisible: %w", path, err)
	}
	if len(s.Tables) == 0 {
		return nil, fmt.Errorf("schéma %s sans table", path)
	}
	for table, stmt := range s.Tables {
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(stmt)), "CREATE TABLE") {
			return nil, fmt.Errorf("schéma %s : instruction inattendue pour %s", path, table)
		}
	}
	return &s, nil
}
