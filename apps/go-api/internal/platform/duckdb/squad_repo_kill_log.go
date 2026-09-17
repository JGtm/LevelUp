// Package duckdb — squad_repo_kill_log.go : journal des morts de l'escouade (Q32e), source
// du badge d'impact « Voleur » (analysis.ComputeThiefBadge).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// Q32eSquadKillLogTemplate : les morts PUBLIABLES (lecture ligne à ligne : le badge nomme
// un tueur et un assistant) des matchs fournis, restreintes à ce que le badge consulte :
//
//   - victime membre de l'escouade (vérifier que l'assistant est vivant autour du kill) ;
//   - tueur ET assistant membres de l'escouade (candidats au vol).
//
// `assist_known` est exigé pour la seconde branche : sans lui, un assistant NULL veut dire
// « on ne sait pas », jamais « pas d'assistant ». Les parts ne sont ni plafonnées ni
// complétées : NULL reste NULL (le vol exige une part mesurée).
//
// LES MORTS DE BOT SONT ÉCARTÉES ICI, comme en Q32c (`queries_squad.go`) : elles arrivent du
// film avec un `victim_xuid` NULL (orphelines — le kill-feed de l'API est humain seul, cf.
// `persist/kill_events_merge.go`). La première branche les exclut déjà par construction
// (`IN (...)` est faux sur NULL) ; la seconde ne parlant que du tueur et de l'assistant, elle
// les acceptait — un vol n'a de sens qu'entre joueurs, donc on écarte au plus près de la
// source, où l'intention est lisible.
//
// Les '%s' positionnels sont, DANS CET ORDRE : match_ids, xuids (victime), xuids (tueur),
// xuids (assistant). Ne PAS utiliser directement — passer par LoadSquadKillLog().
const Q32eSquadKillLogTemplate = `
SELECT
    match_id,
    time_ms,
    feed_killer_xuid,
    victim_xuid,
    assist_xuid,
    killer_damage_pct
FROM ` + KillEventsCanonicalTable + `
WHERE match_id IN (%s)
  AND publishable
  AND (
        victim_xuid IN (%s)
     OR (victim_xuid IS NOT NULL
         AND assist_known AND feed_killer_xuid IN (%s) AND assist_xuid IN (%s))
  )
ORDER BY match_id, time_ms`

// LoadSquadKillLog : cf. port.SquadRepository. Dégradation gracieuse alignée sur Q32d :
// reader indisponible ou table absente → (nil, nil), loggé.
func (r *SquadRepo) LoadSquadKillLog(
	ctx context.Context,
	matchIDs, squadXUIDs []string,
) ([]domain.SquadKillLogRow, error) {
	if len(matchIDs) == 0 || len(squadXUIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		slog.WarnContext(ctx, "teammates: journal des morts indisponible (shared reader)",
			"matchs", len(matchIDs), "err", err)
		return nil, nil
	}
	defer release()

	query, args := buildSquadKillLogQuery(matchIDs, squadXUIDs)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		slog.WarnContext(ctx, "teammates: journal des morts indisponible (Q32e)",
			"matchs", len(matchIDs), "err", err)
		return nil, nil
	}
	defer rows.Close()
	return scanSquadKillLog(rows)
}

// buildSquadKillLogQuery rend Q32e et ses arguments, dans l'ordre des '%s' du gabarit.
func buildSquadKillLogQuery(matchIDs, squadXUIDs []string) (string, []interface{}) {
	query := fmt.Sprintf(Q32eSquadKillLogTemplate,
		placeholderList(len(matchIDs)),
		placeholderList(len(squadXUIDs)),
		placeholderList(len(squadXUIDs)),
		placeholderList(len(squadXUIDs)),
	)
	args := make([]interface{}, 0, len(matchIDs)+3*len(squadXUIDs))
	args = appendStrArgs(args, matchIDs)
	args = appendStrArgs(args, squadXUIDs)
	args = appendStrArgs(args, squadXUIDs)
	args = appendStrArgs(args, squadXUIDs)
	return query, args
}

// scanSquadKillLog lit le résultat de Q32e.
func scanSquadKillLog(rows *sql.Rows) ([]domain.SquadKillLogRow, error) {
	var out []domain.SquadKillLogRow
	for rows.Next() {
		var (
			row                  domain.SquadKillLogRow
			killer, victim, asst sql.NullString
			killerPct            sql.NullInt64
		)
		if err := rows.Scan(&row.MatchID, &row.TimeMS, &killer, &victim, &asst, &killerPct); err != nil {
			return nil, fmt.Errorf("SquadRepo.LoadSquadKillLog scan: %w", err)
		}
		row.KillerXUID, row.VictimXUID, row.AssistXUID = killer.String, victim.String, asst.String
		if killerPct.Valid {
			v := int(killerPct.Int64)
			row.KillerDamagePct = &v
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
