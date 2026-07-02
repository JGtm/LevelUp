// Package sync — exclusion_filter.go : helper de lecture du flag is_excluded.
//
// Les matchs marqués `is_excluded = TRUE` dans player_match_enrichment ne doivent
// pas peser dans le calcul du performance_score (fenêtre glissante par chaîne)
// ni dans la cascade LUSR (TrueSkill 2 séquentiel par playlist_group). Ce module
// fournit le helper canonique de lecture utilisé par les deux batches.
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// loadExcludedMatchIDs renvoie l'ensemble des match_id marqués is_excluded = TRUE
// dans player_match_enrichment. Map vide (non-nil) si la requête réussit mais ne
// retourne rien — l'appelant peut toujours lire `m[id]` sans nil-check.
//
// Tolérance schéma : si la table `player_match_enrichment` ou la colonne
// `is_excluded` n'existe pas (fixture de test partielle, DB legacy pré-migration
// `add_match_exclusion_flag`), retourne une map vide + log Debug plutôt que
// d'échouer le batch. En prod la colonne est créée par migration au boot.
//
// Erreurs SQL inattendues (autres que schéma) : propagées normalement.
func loadExcludedMatchIDs(ctx context.Context, playerDB *sql.DB) (map[string]bool, error) {
	result := make(map[string]bool)
	rows, err := playerDB.QueryContext(ctx,
		`SELECT match_id FROM player_match_enrichment_latest WHERE COALESCE(is_excluded, FALSE) = TRUE`)
	if err != nil {
		if isSchemaMissingErr(err) {
			slog.DebugContext(ctx, "loadExcludedMatchIDs: schéma absent — aucun match exclu pris en compte",
				"err", err)
			return result, nil
		}
		return nil, fmt.Errorf("loadExcludedMatchIDs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var mid string
		if err := rows.Scan(&mid); err != nil {
			return nil, fmt.Errorf("loadExcludedMatchIDs scan: %w", err)
		}
		result[mid] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("loadExcludedMatchIDs rows: %w", err)
	}
	return result, nil
}

// isSchemaMissingErr détecte les erreurs DuckDB de type "table introuvable" ou
// "colonne introuvable". Volontairement étroit (motifs littéraux) : on ne veut
// pas avaler les erreurs SQL légitimes (disque plein, contention…).
func isSchemaMissingErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "not found in FROM clause") ||
		strings.Contains(msg, "Catalog Error")
}
