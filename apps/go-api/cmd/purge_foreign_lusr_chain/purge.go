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
// Les vues reposées ne sont PAS recopiées : elles sont CAPTURÉES dans la base elle-même
// avant le swap (duckdb_views) puis rejouées. Une DDL recopiée dérive en silence dès que la
// migration du titre évolue ; celle-ci est par construction celle de la base traitée.
// Aucun index n'est reposé : match_skill_rank n'a plus d'index secondaire
// (drop_msr_secondary_art_indexes_v1, ratchet noSecondaryIndexTables), et un index qu'un
// binaire plus ancien aurait recréé disparaît avec l'ancienne table au lieu d'être rejoué.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/migration"
)

// chainCensus — photographie des lignes d'une chaîne dans une player DB.
type chainCensus struct {
	TotalRows           int            // toutes lignes de match_skill_rank
	ForeignRaw          int            // lignes de la chaîne visée, par SCAN FORCÉ
	ForeignByRatingType map[string]int // ventilation LUSR / LUSR_V2 / ...
	ForeignLatest       int            // lignes de la chaîne encore GAGNANTES dans la vue
}

// censusForeignChain recense la chaîne visée dans la table brute ET dans la vue
// _latest. Les deux comptes racontent deux choses différentes : le brut est ce que la
// purge retire, `_latest` est ce que les lecteurs applicatifs voient encore.
//
// SCAN FORCÉ PARTOUT (`playlist_group || ” = ?`) : un `playlist_group = ?` nu était
// servi par idx_msr_playlist, et un index ART désynchronisé rendait alors un compte
// MINORÉ — le P0 du 2026-09-13 (JGtm : 22 lignes annoncées pour 1 826 réelles). Les
// index secondaires de match_skill_rank sont retirés depuis le 2026-09-27
// (drop_msr_secondary_art_indexes_v1) ; le scan forcé reste, parce que l'outil ouvre
// la base sans le soin d'EnsurePlayerSchema, et qu'un binaire plus ancien a pu les
// recréer.
func censusForeignChain(ctx context.Context, db *sql.DB, chain string) (chainCensus, error) {
	c := chainCensus{ForeignByRatingType: map[string]int{}}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_skill_rank`).Scan(&c.TotalRows); err != nil {
		return c, fmt.Errorf("recensement total: %w", err)
	}
	rows, err := db.QueryContext(ctx,
		`SELECT rating_type, COUNT(*) FROM match_skill_rank
		 WHERE playlist_group || '' = ? GROUP BY rating_type ORDER BY rating_type`, chain)
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
		`SELECT COUNT(*) FROM match_skill_rank_latest WHERE playlist_group || '' = ?`,
		chain).Scan(&c.ForeignLatest); err != nil {
		return c, fmt.Errorf("recensement _latest: %w", err)
	}
	return c, nil
}

// purgeForeignChain reconstruit match_skill_rank SANS les lignes de la chaîne visée.
//
// Deux précautions dans le prédicat du CTAS, chacune pour un défaut constaté :
//   - `IS DISTINCT FROM` et non `<>` : en SQL, `NULL <> 'x'` vaut NULL, donc un `<>`
//     nu JETTERAIT toutes les lignes à playlist_group NULL (les CSR, entre autres) ;
//   - `playlist_group || ”` et non la colonne nue : la concaténation interdit au
//     planner de servir le prédicat par idx_msr_playlist. Sur une base dont l'index
//     est désynchronisé (P0 du 2026-09-13), un filtre indexé CONSERVERAIT les lignes
//     étrangères que l'index ne voit pas — la purge serait silencieusement partielle.
//     Index retiré le 2026-09-27 ; la garde reste (cf. censusForeignChain).
func purgeForeignChain(ctx context.Context, db *sql.DB, chain string, before chainCensus) error {
	var staleIndexes int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM duckdb_indexes() WHERE table_name = 'match_skill_rank'`).Scan(&staleIndexes); err != nil {
		return fmt.Errorf("compte des index secondaires: %w", err)
	}
	views, err := captureDependentViews(ctx, db)
	if err != nil {
		return err
	}
	if len(views) == 0 {
		return fmt.Errorf("aucune vue ne référence match_skill_rank — base non migrée ? " +
			"purge refusée : les lecteurs applicatifs ne seraient pas restaurés")
	}
	slog.InfoContext(ctx, "purge_foreign_lusr_chain: DDL capturée avant le swap",
		"stale_indexes_dropped", staleIndexes, "views", len(views), "view_names", viewNames(views))

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
		 SELECT * FROM match_skill_rank WHERE playlist_group || '' IS DISTINCT FROM ?`, chain); err != nil {
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

	stmts := make([]string, 0, 8+len(views))
	for _, v := range views {
		stmts = append(stmts, `DROP VIEW IF EXISTS `+v.name)
	}
	stmts = append(stmts,
		`DROP TABLE match_skill_rank`,
		`ALTER TABLE match_skill_rank__purge RENAME TO match_skill_rank`,
		`CREATE SEQUENCE IF NOT EXISTS msr_seq START 1`,
		`ALTER TABLE match_skill_rank ADD PRIMARY KEY (id)`,
		`ALTER TABLE match_skill_rank ALTER COLUMN id SET DEFAULT nextval('msr_seq')`,
		`ALTER TABLE match_skill_rank ALTER COLUMN written_at SET DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)`,
	)
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("étape du swap (%.60s): %w", stmt, err)
		}
	}
	if err := recreateViews(ctx, tx, views); err != nil {
		return err
	}
	if err := assertViewsRestored(ctx, tx, len(views)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit du swap: %w", err)
	}
	committed = true

	// Le swap repose msr_seq par CREATE SEQUENCE IF NOT EXISTS ... START 1 : une séquence
	// absente renaîtrait à 1 sous des ids déjà pris (« Duplicate key » à chaque insertion LUSR
	// suivante). Réalignement explicite sur max(id), échec = erreur de l'outil.
	if _, err := migration.AlignSequencesToColumns(ctx, db); err != nil {
		return fmt.Errorf("alignement des séquences après swap: %w", err)
	}
	// CHECKPOINT : sans lui le WAL peut être perdu à la fermeture (leçon ADR 0022).
	if _, err := db.ExecContext(ctx, `CHECKPOINT`); err != nil {
		return fmt.Errorf("CHECKPOINT après swap: %w", err)
	}
	slog.InfoContext(ctx, "purge_foreign_lusr_chain: reconstruction commitée",
		"chain", chain, "rows_before", before.TotalRows, "rows_after", rebuilt,
		"rows_removed", before.ForeignRaw, "stale_indexes_dropped", staleIndexes,
		"views_restored", len(views), "view_names", viewNames(views))
	return nil
}

// dependentView — une vue non interne dont le SQL référence match_skill_rank.
type dependentView struct {
	name string
	ddl  string
}

func viewNames(views []dependentView) []string {
	out := make([]string, 0, len(views))
	for _, v := range views {
		out = append(out, v.name)
	}
	return out
}

// captureDependentViews relève TOUTES les vues non internes dont le SQL référence
// match_skill_rank — filtre sur le SQL, jamais sur un nom.
//
// Défaut corrigé le 2026-09-13 (C.9) : l'outil ne capturait que le nom
// `match_skill_rank_latest`, et depuis C.3 bis la table en porte DEUX
// (`match_skill_rank_latest_by_type` est née avec le graphe d'évolution). Filtrer
// par nom, c'est perdre en silence toute vue ajoutée après l'écriture de l'outil —
// et une vue perdue ne se voit qu'au premier lecteur qui tombe sur
// « table does not exist ». Le filtre par SQL suit le schéma, il ne le devine pas.
//
// À SAVOIR sur DuckDB (mesuré) : `DROP TABLE` ne supprime PAS une vue dépendante —
// elle survit au catalogue et se re-lie à la table recréée par le RENAME. Le swap
// FONCTIONNERAIT donc même sans les DROP VIEW ; on les fait quand même, pour que la
// vue rejouée soit celle qu'on a capturée et non une vue laissée liée par accident.
func captureDependentViews(ctx context.Context, db *sql.DB) ([]dependentView, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT view_name, sql FROM duckdb_views()
		WHERE internal = FALSE AND sql IS NOT NULL
		  AND lower(sql) LIKE '%match_skill_rank%'
		ORDER BY view_name`)
	if err != nil {
		return nil, fmt.Errorf("capture des vues dépendantes: %w", err)
	}
	defer rows.Close()
	var out []dependentView
	for rows.Next() {
		var v dependentView
		if err := rows.Scan(&v.name, &v.ddl); err != nil {
			return nil, fmt.Errorf("capture des vues dépendantes (scan): %w", err)
		}
		if v.name == "" || v.ddl == "" {
			continue
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// recreateViews rejoue les DDL capturées. Une vue peut en référencer une autre :
// on repasse tant que la passe précédente a fait progresser, et on échoue en nommant
// les vues restantes plutôt que de laisser un lecteur découvrir le trou.
func recreateViews(ctx context.Context, tx *sql.Tx, views []dependentView) error {
	pending := make([]dependentView, len(views))
	copy(pending, views)
	for len(pending) > 0 {
		var failed []dependentView
		var lastErr error
		for _, v := range pending {
			if _, err := tx.ExecContext(ctx, v.ddl); err != nil {
				failed = append(failed, v)
				lastErr = err
			}
		}
		if len(failed) == len(pending) {
			return fmt.Errorf("recréation des vues bloquée sur %v (dépendances circulaires ?): %w",
				viewNames(failed), lastErr)
		}
		pending = failed
	}
	return nil
}

// assertViewsRestored : garde de cardinalité sur les VUES, jumelle de celle sur les
// lignes. Elle s'exécute DANS la transaction : un manque fait rollback du swap entier
// plutôt que de laisser une base sans son lecteur.
func assertViewsRestored(ctx context.Context, tx *sql.Tx, want int) error {
	var got int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM duckdb_views()
		WHERE internal = FALSE AND sql IS NOT NULL AND lower(sql) LIKE '%match_skill_rank%'`).Scan(&got); err != nil {
		return fmt.Errorf("recomptage des vues dépendantes: %w", err)
	}
	if got != want {
		return fmt.Errorf("swap abandonné : %d vue(s) dépendante(s) restaurée(s) sur %d "+
			"(rollback, aucune vue perdue)", got, want)
	}
	return nil
}
