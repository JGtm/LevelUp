package migration

// steps_written_at_utc_default.go — REPARE le DEFAULT de `written_at` sur les bases
// EXISTANTES (lot de suivi S2, suite du constat R1).
//
// Le defaut : `written_at TIMESTAMP ... DEFAULT now()` (ou CURRENT_TIMESTAMP). Les deux
// expressions rendent un TIMESTAMPTZ que DuckDB coerce vers une colonne TIMESTAMP NAIVE
// par le fuseau de SESSION. Sur un poste a UTC+2, toute ligne ecrite SANS valeur
// explicite se date donc deux heures dans le FUTUR, alors que les ecrivains applicatifs
// posent `time.Now().UTC()`. Or `written_at` est la colonne de TRI des vues `<table>_latest`
// (ADR 0026) : melanger deux horloges sur cette colonne, c'est melanger deux PRESEANCES.
// Une ligne datee au fuseau local gagne l'arbitrage contre toute ligne UTC ecrite dans les
// deux heures qui suivent — l'enrichissement disparait de la lecture sans erreur, sans
// compteur, sans qu'un seul nom ni un seul compte ne bouge (mecanisme demontre par R1).
//
// Le DDL source est corrige a la racine (forme canonique WrittenAtDefaultUTC partout,
// tenue par written_at_utc_guard_test.go), ce qui suffit aux bases NEUVES. Ce step traite
// les bases DEJA CREEES : `CREATE TABLE IF NOT EXISTS` ne retouche jamais une table
// existante, seul un ALTER le fait.
//
// Data-driven a dessein : il repare TOUTE colonne `written_at` naive dont le DEFAULT est
// sensible au fuseau, sans liste de tables a maintenir. Enregistre sur les cinq targets
// (une base peut porter n'importe quel sous-ensemble de ces tables) ; no-op complet quand
// il n'y a rien a reparer. Idempotent : apres l'ALTER, le DEFAULT normalise par DuckDB
// (`CAST(main.timezone('UTC', now()) AS TIMESTAMP)`) ne matche plus le predicat.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// WrittenAtDefaultUTC — forme canonique du DEFAULT de `written_at`. Tout DDL du depot
// doit l'utiliser telle quelle (garde-rail : written_at_utc_guard_test.go).
const WrittenAtDefaultUTC = "CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)"

// writtenAtDefectiveDefaultsSQL liste les colonnes `written_at` a reparer dans la base
// COURANTE. Filtres, dans l'ordre d'importance :
//   - `database_name = current_database()` : duckdb_columns() voit AUSSI les bases
//     ATTACHees (metadata l'est sur shared) — sans ce filtre on tenterait un ALTER sur une
//     base tenue en lecture seule par un autre chemin.
//   - `data_type = 'TIMESTAMP'` : une colonne TIMESTAMPTZ n'a pas le defaut (elle garde
//     l'instant absolu) ; la reparer serait un changement de semantique gratuit.
//   - DEFAULT sensible au fuseau ET pas deja converti (idempotence).
const writtenAtDefectiveDefaultsSQL = `
	SELECT c.table_name
	FROM duckdb_columns() c
	JOIN duckdb_tables() t
	  ON t.database_name = c.database_name
	 AND t.schema_name   = c.schema_name
	 AND t.table_name    = c.table_name
	WHERE c.database_name = current_database()
	  AND c.schema_name   = 'main'
	  AND c.column_name   = 'written_at'
	  AND c.data_type     = 'TIMESTAMP'
	  AND c.column_default IS NOT NULL
	  AND (lower(c.column_default) LIKE '%now()%'
	    OR lower(c.column_default) LIKE '%current_timestamp%')
	  AND lower(c.column_default) NOT LIKE '%timezone(''utc''%'
	ORDER BY c.table_name`

// EnsureWrittenAtDefaultsUTC bascule vers WrittenAtDefaultUTC le DEFAULT de toute colonne
// `written_at` naive encore datee sur le fuseau de session. Retourne le nombre de tables
// reparees. Exporte : rejoue aussi hors runner (outillage de reparation ponctuelle).
func EnsureWrittenAtDefaultsUTC(db *sql.DB) (int, error) {
	ctx := context.Background()
	tables, err := writtenAtDefectiveTables(ctx, db)
	if err != nil {
		return 0, err
	}
	for _, table := range tables {
		// Nom d'identifiant issu du catalogue DuckDB lui-meme (pas d'entree utilisateur),
		// quote pour supporter un nom sensible a la casse.
		stmt := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN written_at SET DEFAULT %s`,
			table, WrittenAtDefaultUTC)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return 0, fmt.Errorf("migration: written_at DEFAULT UTC sur %s: %w", table, err)
		}
		slog.InfoContext(ctx, "migration: written_at DEFAULT bascule en UTC explicite",
			"table", table)
	}
	return len(tables), nil
}

// writtenAtDefectiveTables lit le catalogue et retourne les tables a reparer.
func writtenAtDefectiveTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, writtenAtDefectiveDefaultsSQL)
	if err != nil {
		return nil, fmt.Errorf("migration: detection written_at DEFAULT: %w", err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("migration: scan written_at DEFAULT: %w", err)
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

// writtenAtUTCStepName construit le nom du step par target (un nom = une ligne
// schema_migrations, et chaque base a son propre ledger).
func writtenAtUTCStepName(target TargetDB) string {
	return "written_at_default_utc_" + string(target)
}

func init() {
	for _, target := range []TargetDB{
		TargetShared, TargetPlayer, TargetSharedPvE, TargetSharedSocial, TargetMetadata,
	} {
		Register(Migration{
			Name:        writtenAtUTCStepName(target),
			TargetDB:    target,
			Description: "written_at : DEFAULT en UTC explicite (fin du melange de fuseaux sur la colonne de tri des vues _latest)",
			ApplySchema: func(db *sql.DB) error {
				_, err := EnsureWrittenAtDefaultsUTC(db)
				return err
			},
		})
	}
}
