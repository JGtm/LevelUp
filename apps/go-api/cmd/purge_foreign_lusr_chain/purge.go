//go:build cgo

package main

// purge.go — recensement + reconstruction CTAS transactionnelle de match_skill_rank.
//
// La reconstruction est modelée sur migration.rebuildAppendOnlyTx (helper non exporté,
// et spécialisé dans la CONVERSION legacy→append-only : il no-ope sur une table déjà
// convertie, donc inutilisable pour un filtrage de lignes). Ce qui en est repris à
// l'identique : transaction unique, garde de cardinalité AVANT le DROP, rollback
// intégral, table de travail `__purge` déposée dans la même TX.
//
// Le DDL reposé n'est PAS recopié : il est CAPTURÉ dans la base elle-même avant le swap
// (duckdb_indexes / duckdb_views) puis rejoué. Une DDL recopiée dérive en silence dès
// que la migration du titre évolue ; celle-ci est par construction celle de la base
// traitée.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

// chainCensus — photographie des lignes d'une chaîne dans une player DB.
type chainCensus struct {
	TotalRows           int            // toutes lignes de match_skill_rank
	ForeignRaw          int            // lignes de la chaîne visée, TABLE BRUTE
	ForeignByRatingType map[string]int // ventilation LUSR / LUSR_V2 / ...
	ForeignLatest       int            // lignes de la chaîne encore GAGNANTES dans la vue
}

// censusForeignChain recense la chaîne visée dans la table brute ET dans la vue
// _latest. Les deux comptes racontent deux choses différentes : le brut est ce que la
// purge retire, `_latest` est ce que les lecteurs applicatifs voient encore.
func censusForeignChain(ctx context.Context, db *sql.DB, chain string) (chainCensus, error) {
	c := chainCensus{ForeignByRatingType: map[string]int{}}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank`).Scan(&c.TotalRows); err != nil {
		return c, fmt.Errorf("recensement total: %w", err)
	}
	rows, err := db.QueryContext(ctx,
		`SELECT rating_type, COUNT(*) FROM match_skill_rank
		 WHERE playlist_group = ? GROUP BY rating_type ORDER BY rating_type`, chain)
	if err != nil {
		return c, fmt.Errorf("recensement brut: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rt string
		var n int
		if err := rows.Scan(&rt, &n); err != nil {
			return c, fmt.Errorf("recensement brut scan: %w", err)
		}
		c.ForeignByRatingType[rt] = n
		c.ForeignRaw += n
	}
	if err := rows.Err(); err != nil {
		return c, fmt.Errorf("recensement brut rows: %w", err)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank_latest WHERE playlist_group = ?`,
		chain).Scan(&c.ForeignLatest); err != nil {
		return c, fmt.Errorf("recensement _latest: %w", err)
	}
	return c, nil
}

// purgeForeignChain reconstruit match_skill_rank SANS les lignes de la chaîne visée.
//
// `playlist_group IS DISTINCT FROM ?` et non `<> ?` : en SQL, `NULL <> 'x'` vaut NULL,
// donc un `<>` nu JETTERAIT toutes les lignes à playlist_group NULL (les CSR, entre
// autres). La garde de cardinalité l'attraperait, mais le filtre doit être juste.
func purgeForeignChain(ctx context.Context, db *sql.DB, chain string, before chainCensus) error {
	indexDDL, err := captureDDL(ctx, db,
		`SELECT sql FROM duckdb_indexes() WHERE table_name = 'match_skill_rank' AND sql IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("capture des index: %w", err)
	}
	viewDDL, err := captureDDL(ctx, db,
		`SELECT sql FROM duckdb_views() WHERE view_name = 'match_skill_rank_latest' AND sql IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("capture de la vue _latest: %w", err)
	}
	if len(viewDDL) != 1 {
		return fmt.Errorf("vue match_skill_rank_latest introuvable (%d définition(s)) — "+
			"base non migrée ? purge refusée, la vue ne serait pas restaurée", len(viewDDL))
	}
	slog.InfoContext(ctx, "purge_foreign_lusr_chain: DDL capturée avant le swap",
		"indexes", len(indexDDL), "views", len(viewDDL))

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS match_skill_rank__purge`); err != nil {
		return fmt.Errorf("drop table de travail: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`CREATE TABLE match_skill_rank__purge AS
		 SELECT * FROM match_skill_rank WHERE playlist_group IS DISTINCT FROM ?`, chain); err != nil {
		return fmt.Errorf("CTAS filtré: %w", err)
	}

	var rebuilt int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank__purge`).Scan(&rebuilt); err != nil {
		return fmt.Errorf("count table de travail: %w", err)
	}
	want := before.TotalRows - before.ForeignRaw
	if rebuilt != want {
		return fmt.Errorf("swap abandonné : rebuilt=%d != total(%d) - étrangères(%d) = %d "+
			"(rollback, zéro perte)", rebuilt, before.TotalRows, before.ForeignRaw, want)
	}

	stmts := []string{
		`DROP VIEW IF EXISTS match_skill_rank_latest`,
		`DROP TABLE match_skill_rank`,
		`ALTER TABLE match_skill_rank__purge RENAME TO match_skill_rank`,
		`CREATE SEQUENCE IF NOT EXISTS msr_seq START 1`,
		`ALTER TABLE match_skill_rank ADD PRIMARY KEY (id)`,
		`ALTER TABLE match_skill_rank ALTER COLUMN id SET DEFAULT nextval('msr_seq')`,
		`ALTER TABLE match_skill_rank ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
	}
	stmts = append(stmts, indexDDL...)
	stmts = append(stmts, viewDDL...)
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("étape du swap (%.60s): %w", stmt, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit du swap: %w", err)
	}
	committed = true

	// CHECKPOINT : sans lui le WAL peut être perdu à la fermeture (leçon ADR 0022).
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("CHECKPOINT après swap: %w", err)
	}
	slog.InfoContext(ctx, "purge_foreign_lusr_chain: reconstruction commitée",
		"chain", chain, "rows_before", before.TotalRows, "rows_after", rebuilt,
		"rows_removed", before.ForeignRaw, "indexes_restored", len(indexDDL))
	return nil
}

// captureDDL lit des instructions DDL depuis les catalogues DuckDB.
func captureDDL(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var ddl string
		if err := rows.Scan(&ddl); err != nil {
			return nil, err
		}
		if ddl != "" {
			out = append(out, ddl)
		}
	}
	return out, rows.Err()
}
