// Package duckdb — gameplay_seconds.go : la DURÉE DE JEU d'un match, socle du flag
// « Prolongation ».
//
// La durée de jeu est celle de la chronologie du match (ADR 0050) : la durée du registre moins
// le coup d'envoi, par le fragment unique sqlfrag.SQLGameplayDurationMs (coup d'envoi inconnu :
// la durée du match). Elle ne se déduit plus du temps joué des participants : celui-ci est
// calculé sur le même coup d'envoi, mais un joueur parti avant la fin ou arrivé après le début
// le raccourcit, et le recalcul de ce temps n'est pas une mesure du match.
//
// La page d'UN match n'a pas besoin de cette lecture : sa chronologie (service.matchViewTimeline)
// porte déjà la durée et le coup d'envoi. Elle sert l'historique et l'Explorateur, dont les lignes
// ne portent pas le coup d'envoi.
//
// Lecture BEST-EFFORT, jamais fatale : une colonne absente (DB non migrée) ou une requête en
// échec logge un WARN et rend une map vide → aucun flag, aucune page cassée (leçon G1, incident
// prod 25/07 : un LEFT JOIN inconditionnel sur une vue manquante faisait tomber toute la
// Match View).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/platform/duckdb/sqlfrag"
)

// qGameplayMsByMatchTpl : `%s` = placeholders de la clause IN (match_id). Exécutée sur
// SharedReader (ADR 0016) — pas de préfixe `shared.`. Aucun filtre temporel : le fuseau
// canonique est celui que le fragment de durée porte déjà.
var qGameplayMsByMatchTpl = `
SELECT r.match_id, ` + sqlfrag.SQLGameplayDurationMs("r") + ` AS gameplay_ms
FROM match_registry r
WHERE r.match_id IN (%s)`

// LoadGameplaySecondsByMatch retourne la durée de jeu (secondes, tronquées comme
// MatchTimeline.GameplayDurationSeconds) par match_id. Un match sans durée de jeu exploitable
// (durée absente ou nulle) est absent de la map : l'appelant n'estime rien. Best-effort : map
// vide (jamais d'erreur) si la lecture échoue — l'appelant sert alors la page sans flag
// « Prolongation ».
func LoadGameplaySecondsByMatch(ctx context.Context, db *sql.DB, matchIDs []string) map[string]int {
	out := make(map[string]int, len(matchIDs))
	if db == nil || len(matchIDs) == 0 {
		return out
	}
	q := fmt.Sprintf(qGameplayMsByMatchTpl, Placeholders(len(matchIDs)))
	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "gameplay_seconds: lecture indisponible, flag prolongation dégradé",
			"matches_count", len(matchIDs), "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var (
			matchID string
			ms      sql.NullInt64
		)
		if err := rows.Scan(&matchID, &ms); err != nil {
			slog.WarnContext(ctx, "gameplay_seconds: scan échoué, ligne ignorée", "err", err)
			continue
		}
		if !ms.Valid || ms.Int64 <= 0 {
			continue // aucune durée de jeu exploitable → pas d'estimation
		}
		out[matchID] = int(ms.Int64 / 1000)
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "gameplay_seconds: itération interrompue, flag prolongation dégradé", "err", err)
	}
	return out
}
