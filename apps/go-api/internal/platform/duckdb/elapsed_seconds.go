// Package duckdb — elapsed_seconds.go : estimateur de la DURÉE DE JEU OBSERVÉE
// d'un match (elapsed_seconds), socle du flag « Prolongation ».
//
// Pourquoi pas match_registry.duration_seconds : cette colonne porte la durée de
// la SESSION (countdown, écrans de fin, joueurs qui traînent), pas le temps de
// jeu. L'estimateur fiable est le temps joué des participants qui ont fait le
// match ENTIER : leur médiane est le mur de temps du match.
//
//	elapsed_seconds = MEDIAN(time_played_seconds) des participants
//	                  present_at_beginning AND present_at_completion
//	                  ; repli MAX(time_played_seconds) si aucun tel participant
//
// Le repli couvre les matchs dont les colonnes de participation n'ont jamais été
// backfillées (NULL partout) : le joueur resté le plus longtemps borne le match
// par en dessous. Médiane (et non moyenne) : insensible aux quitteurs.
//
// Lecture BEST-EFFORT, jamais fatale : une colonne absente (DB non migrée) ou une
// requête en échec logge un WARN et rend une map vide → aucun flag, aucune page
// cassée. C'est la leçon G1 (incident prod 25/07 : un LEFT JOIN inconditionnel
// sur une vue manquante faisait tomber toute la Match View).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
)

// qElapsedSecondsByMatchTpl : `%s` = placeholders de la clause IN (match_id).
// Exécutée sur SharedReader (ADR 0016) — pas de préfixe `shared.`.
// Aucun filtre temporel → pas de fragment timezone canonique à appliquer.
const qElapsedSecondsByMatchTpl = `
SELECT
    p.match_id,
    COALESCE(
        MEDIAN(p.time_played_seconds) FILTER (
            WHERE COALESCE(p.present_at_beginning, FALSE)
              AND COALESCE(p.present_at_completion, FALSE)
        ),
        MAX(p.time_played_seconds)
    ) AS elapsed_seconds
FROM match_participants p
WHERE p.match_id IN (%s)
GROUP BY p.match_id`

// LoadElapsedSecondsByMatch retourne la durée de jeu observée (secondes, arrondie)
// par match_id. Best-effort : map vide (jamais d'erreur) si la lecture échoue —
// l'appelant sert alors la page sans flag « Prolongation ».
func LoadElapsedSecondsByMatch(ctx context.Context, db *sql.DB, matchIDs []string) map[string]int {
	out := make(map[string]int, len(matchIDs))
	if db == nil || len(matchIDs) == 0 {
		return out
	}
	q := fmt.Sprintf(qElapsedSecondsByMatchTpl, Placeholders(len(matchIDs)))
	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		slog.WarnContext(ctx, "elapsed_seconds: lecture indisponible, flag prolongation dégradé",
			"matches_count", len(matchIDs), "err", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var (
			matchID string
			elapsed sql.NullFloat64
		)
		if err := rows.Scan(&matchID, &elapsed); err != nil {
			slog.WarnContext(ctx, "elapsed_seconds: scan échoué, ligne ignorée", "err", err)
			continue
		}
		if !elapsed.Valid || elapsed.Float64 <= 0 {
			continue // aucun temps joué exploitable → pas d'estimation
		}
		out[matchID] = int(math.Round(elapsed.Float64))
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "elapsed_seconds: itération interrompue, flag prolongation dégradé", "err", err)
	}
	return out
}

// LoadElapsedSecondsForMatch est le cas mono-match (Match View). Retourne
// (0, false) quand la durée n'est pas estimable.
func LoadElapsedSecondsForMatch(ctx context.Context, db *sql.DB, matchID string) (int, bool) {
	byMatch := LoadElapsedSecondsByMatch(ctx, db, []string{matchID})
	v, ok := byMatch[matchID]
	return v, ok
}
