// Package duckdb — cross_game_repo.go : co-occurrence cross-titre pour le badge
// cross-jeu du hub Relations (Phase 3b). Lecture seule sur le shared d'un AUTRE
// titre via SharedReader. UNE seule requête batch par titre (jamais une DB par
// relation). xuid global (ADR 0008) → la jointure se fait par xuid.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// crossGameTimeout : limite hard de la requête de co-occurrence cross-titre.
// Court : le badge est best-effort, on omet plutôt que de bloquer /relations.
const crossGameTimeout = 10 * time.Second

// q31CrossGameCooccurrenceTpl compte, sur le shared d'un AUTRE titre, le nombre
// de matchs communs entre le joueur courant (me.xuid = ?) et chaque opp_xuid de
// l'ensemble candidat, filtré au seuil. xuid global (ADR 0008). Bots exclus.
//
// %s = placeholders de l'ensemble candidat (clause IN). Args (dans l'ordre) :
// me.xuid, puis chaque opp_xuid, puis le seuil (HAVING).
const q31CrossGameCooccurrenceTpl = `
SELECT opp.xuid AS opp_xuid, COUNT(DISTINCT opp.match_id) AS together
FROM match_participants me
JOIN match_participants opp
  ON opp.match_id = me.match_id
 AND opp.xuid <> me.xuid
WHERE me.xuid = ?
  AND opp.xuid IN (%s)
  AND opp.xuid NOT LIKE 'bid(%%'
GROUP BY opp.xuid
HAVING COUNT(DISTINCT opp.match_id) >= ?`

// CountCrossTitleCooccurrences retourne, pour chaque oppXUID croisé >= minMatches
// avec myXUID sur le shared lu par `db`, le nombre de matchs communs. UNE
// requête batch. Lecture seule. Retourne une map vide (jamais nil-panic) si
// oppXUIDs est vide. Toute erreur d'accès est propagée au caller, qui la traite
// en best-effort (le caller cross-jeu avale et logge). `db` est un handle RO
// (ou RW emprunté en lecture) ouvert par le caller, qui en garde la propriété.
func CountCrossTitleCooccurrences(
	ctx context.Context,
	db *sql.DB,
	myXUID string,
	oppXUIDs []string,
	minMatches int,
) (map[string]int, error) {
	out := make(map[string]int)
	if db == nil || myXUID == "" || len(oppXUIDs) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, crossGameTimeout)
	defer cancel()

	sqlText := fmt.Sprintf(q31CrossGameCooccurrenceTpl, Placeholders(len(oppXUIDs)))
	args := make([]any, 0, 2+len(oppXUIDs))
	args = append(args, myXUID)
	args = append(args, ToAnySlice(oppXUIDs)...)
	args = append(args, minMatches)

	rows, err := db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("CountCrossTitleCooccurrences: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var xuid string
		var together int
		if err := rows.Scan(&xuid, &together); err != nil {
			return nil, fmt.Errorf("CountCrossTitleCooccurrences scan: %w", err)
		}
		out[xuid] = together
	}
	return out, rows.Err()
}
