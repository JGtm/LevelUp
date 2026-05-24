package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Package-level helpers pour la migration B1 (sprint sharedprovider, commit
// 8k.0). Utilisés par les repos qui exécutent leurs queries shared via
// `pdb.SharedReadDB().Get(ctx)` au lieu de l'ancien pattern ATTACH +
// `r.pdb.Player.Query(... "JOIN shared.X ...")`.
//
// Ces helpers factorisent les patterns récurrents de la migration :
//   1. Charger un set d'IDs depuis player (ex: match_ids du joueur)
//   2. Charger les rows shared.X correspondantes via une IN clause
//   3. Merger en Go via une map[ID]Row
//
// L'utilisation typique :
//
//	matchIDs := loadPlayerMatchIDs(...)
//	ph := Placeholders(len(matchIDs))
//	q := fmt.Sprintf("SELECT match_id, x FROM match_registry WHERE match_id IN (%s)", ph)
//
//	db, release, err := r.pdb.SharedReadDB().Get(ctx)
//	if err != nil { return nil, err }
//	defer release()
//	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
//	...

// Placeholders construit la chaîne "?, ?, ?, ..." pour une IN clause SQL.
// Retourne "" si n <= 0 (le caller doit gérer le cas de slice vide en
// amont — exécuter une query avec IN vide cause une erreur SQL).
//
// Exemples :
//
//	Placeholders(0) → ""
//	Placeholders(1) → "?"
//	Placeholders(3) → "?, ?, ?"
//
// Pattern d'usage :
//
//	if len(ids) == 0 {
//	    return nil, nil // ou retour neutre selon le cas métier
//	}
//	q := fmt.Sprintf("SELECT * FROM X WHERE id IN (%s)", Placeholders(len(ids)))
//	rows, err := db.QueryContext(ctx, q, ToAnySlice(ids)...)
func Placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?, ", n), ", ")
}

// ToAnySlice convertit une slice typée en []any pour passage variadique à
// `db.QueryContext(ctx, q, args...)`. Évite la boucle manuelle de conversion
// que chaque caller devrait écrire sinon.
//
// Pattern d'usage :
//
//	matchIDs := []string{"m1", "m2", "m3"}
//	q := fmt.Sprintf("... WHERE match_id IN (%s)", Placeholders(len(matchIDs)))
//	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
//
// Utiliser de préférence ce helper plutôt que :
//
//	args := make([]any, len(matchIDs))
//	for i, id := range matchIDs { args[i] = id }
func ToAnySlice[T any](s []T) []any {
	if len(s) == 0 {
		return nil
	}
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

// MatchEnrichment porte les colonnes player_match_enrichment couramment
// utilisées pour hydrater des rows shared après un split+merge cross-DB
// (filters_repo, match_history_repo, squad_repo, player_matches_repo).
//
// Tous les champs sont sql.Null* pour uniformiser : chaque caller convertit
// ensuite vers son type cible (*string, *int, *float64, bool) selon son
// modèle de domaine.
//
// Note schéma : session_id est VARCHAR en prod (cf. internal/migration/
// steps_player.go) — c'est l'ID textuel d'une session, pas un INTEGER PK.
type MatchEnrichment struct {
	SessionID        sql.NullString
	SessionLabel     sql.NullString
	IsWithFriends    bool
	IsExcluded       bool
	PerformanceScore sql.NullFloat64
	DominanceFlag    int
	HadBotTeammate   bool
}

// LoadPlayerMatchEnrichments retourne les 7 colonnes player_match_enrichment
// pour une liste de match_ids. Helper unifié pour toutes les migrations
// split+merge cross-DB qui ont besoin d'hydrater leurs rows shared avec
// les enrichments du joueur courant.
//
// Comportement :
//   - matchIDs vide ou nil : retourne nil, nil (pas d'erreur)
//   - SQL retourne 0 rows : map vide
//   - Erreur DB : propagée
//
// Pattern d'usage :
//
//	enrichments, err := LoadPlayerMatchEnrichments(ctx, pdb.Player, matchIDs)
//	if err != nil { return nil, err }
//	for i := range rows {
//	    if e, ok := enrichments[rows[i].MatchID]; ok {
//	        rows[i].IsWithFriends = e.IsWithFriends
//	        if e.SessionLabel.Valid { ... }
//	    }
//	}
func LoadPlayerMatchEnrichments(ctx context.Context, playerDB *DB, matchIDs []string) (map[string]MatchEnrichment, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(`
		SELECT match_id, session_id, session_label,
		       COALESCE(is_with_friends, FALSE),
		       COALESCE(is_excluded, FALSE),
		       performance_score,
		       COALESCE(dominance_flag, 0),
		       COALESCE(had_bot_teammate, FALSE)
		FROM player_match_enrichment
		WHERE match_id IN (%s)`, Placeholders(len(matchIDs)))

	// QueryRecovered (Phase 5 du refactor ART) : si la handle a été
	// invalidée par un crash FATAL (le scénario du crash home 2026-05-24
	// 20:41:04), Reopen() est appelé automatiquement et la requête est
	// retentée. Évite la cascade `sql: database is closed` côté home.
	rows, err := playerDB.QueryRecovered(ctx, query, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("LoadPlayerMatchEnrichments: %w", err)
	}
	defer rows.Close()

	out := make(map[string]MatchEnrichment, len(matchIDs))
	for rows.Next() {
		var (
			mid string
			e   MatchEnrichment
		)
		if err := rows.Scan(&mid, &e.SessionID, &e.SessionLabel,
			&e.IsWithFriends, &e.IsExcluded, &e.PerformanceScore,
			&e.DominanceFlag, &e.HadBotTeammate); err != nil {
			return nil, fmt.Errorf("LoadPlayerMatchEnrichments scan: %w", err)
		}
		out[mid] = e
	}
	return out, rows.Err()
}
