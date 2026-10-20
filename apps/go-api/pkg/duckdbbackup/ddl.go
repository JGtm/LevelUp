package duckdbbackup

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Le DDL des tables (CREATE TABLE avec clés, défauts, NOT NULL) accompagne leurs Parquet dans
// le staging : un Parquet ne porte que des lignes. À la restauration, il recrée avec ses clés
// une table que le schéma canonique ne connaît pas (table créée hors migrations par un module
// à l'exécution). Fichier `_ddl_<horodatage>.json` : objet {table: "CREATE TABLE …"}.
const (
	ddlFilePrefix = "_ddl_"
	ddlFileSuffix = ".json"
)

// writeTableDDL écrit le DDL des tables de base de db à path. Best-effort : un échec est
// journalisé en WARN sans faire échouer la sauvegarde (les lignes restent sauvegardées ; la
// restauration retombera, pour les tables hors schéma canonique, sur une table sans clé).
func writeTableDDL(ctx context.Context, db *sql.DB, key, path string) {
	ddl, err := readTableDDL(ctx, db)
	if err == nil {
		var data []byte
		if data, err = json.MarshalIndent(ddl, "", "  "); err == nil {
			err = os.WriteFile(path, data, 0o644)
		}
	}
	if err != nil {
		slog.WarnContext(ctx, "backup: DDL des tables non écrit (restauration sans clé pour les tables hors schéma canonique)",
			"key", key, "err", err)
	}
}

// readTableDDL rend le CREATE TABLE de chaque table de base du catalogue courant (schéma main).
func readTableDDL(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT table_name, sql FROM duckdb_tables()
		WHERE database_name = current_database() AND schema_name = 'main' AND NOT temporary`)
	if err != nil {
		return nil, fmt.Errorf("DDL des tables: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]string{}
	for rows.Next() {
		var name string
		var ddl sql.NullString
		if err := rows.Scan(&name, &ddl); err != nil {
			return nil, fmt.Errorf("DDL des tables: lecture: %w", err)
		}
		if ddl.Valid && ddl.String != "" {
			out[name] = ddl.String
		}
	}
	return out, rows.Err()
}

// loadTableDDL lit le DDL le plus récent du dossier Parquet d'une clé. Aucun fichier (sauvegarde
// antérieure à son introduction) : map vide, sans erreur.
func loadTableDDL(dir string) (map[string]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, ddlFilePrefix+"*"+ddlFileSuffix))
	if err != nil || len(matches) == 0 {
		return map[string]string{}, err
	}
	sort.Strings(matches)
	latest := matches[len(matches)-1]
	data, err := os.ReadFile(latest)
	if err != nil {
		return nil, fmt.Errorf("lecture du DDL %s: %w", latest, err)
	}
	ddl := map[string]string{}
	if err := json.Unmarshal(data, &ddl); err != nil {
		return nil, fmt.Errorf("DDL %s illisible: %w", latest, err)
	}
	for table, stmt := range ddl {
		if !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(stmt)), "CREATE TABLE") {
			return nil, fmt.Errorf("DDL %s : instruction inattendue pour %s", latest, table)
		}
	}
	return ddl, nil
}
