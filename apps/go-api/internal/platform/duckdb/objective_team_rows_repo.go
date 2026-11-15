// Package duckdb — objective_team_rows_repo.go : lignes (match, joueur) de
// `match_objective_stats_latest` pour la part d'un joueur dans l'objectif de son équipe
// (narrative.ComputeObjectiveTeamShare, variante de la note de performance à l'étude).
//
// Une ligne porte les valeurs BRUTES de toutes les colonnes d'action et de durée : c'est la
// famille du MATCH, décidée côté analysis, qui choisit lesquelles comptent. Les listes de
// colonnes sont générées depuis les tables narrative (objectiveIndexSelectColumns — source
// unique, aucune liste parallèle). Le camp vient de match_participants. TOUS les joueurs des
// matchs du joueur sont chargés (les deux camps) : l'équipe se reconstitue côté analysis.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/platform/duckdb/sqlfrag"
)

// LoadObjectiveTeamRowsForPlayer retourne, par match, les lignes d'objectif de tous les
// participants des matchs où `xuid` a joué. Lecture par la vue `_latest` UNIQUEMENT (ADR
// 0026). Une base sans la vue (titre sans match_objective_stats, Halo 5) rend une map vide et
// aucune erreur : la métrique est alors absente de la note, jamais un échec. Les matchs de la
// Campagne sont exclus, comme dans le chargement des notes (loadHistoryForPerf).
func LoadObjectiveTeamRowsForPlayer(ctx context.Context, sharedDB *sql.DB, xuid string) (map[string][]narrative.ObjectiveTeamRow, error) {
	out := map[string][]narrative.ObjectiveTeamRow{}
	if xuid == "" {
		return out, nil
	}
	var present int
	if err := sharedDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'match_objective_stats_latest'`,
	).Scan(&present); err != nil {
		return out, fmt.Errorf("LoadObjectiveTeamRowsForPlayer: schéma: %w", err)
	}
	if present == 0 {
		return out, nil
	}

	actionCols, holdCols := objectiveIndexSelectColumns()
	cols := append(append([]string{}, actionCols...), holdCols...)
	cols = append(cols, narrative.ObjectiveColTimesSelectedAsVIP)

	var sb strings.Builder
	sb.WriteString("SELECT o.match_id, o.xuid, " + objectiveFamilyCaseSQL() + " AS family, mp.team_id, " +
		"COALESCE(mp.time_played_seconds, 0)::DOUBLE")
	for _, c := range cols {
		sb.WriteString(",\n\t\tCOALESCE(o." + c + ", 0)::DOUBLE")
	}
	sb.WriteString(`
		FROM match_objective_stats_latest o
		JOIN match_participants mp ON mp.match_id = o.match_id AND mp.xuid = o.xuid
		WHERE o.match_id IN (SELECT match_id FROM match_participants WHERE xuid = ?)`)
	sb.WriteString(sqlfrag.SQLExcludeCampaignByMatchIDCtx(ctx, "o.match_id"))

	rows, err := sharedDB.QueryContext(ctx, sb.String(), xuid)
	if err != nil {
		return out, fmt.Errorf("LoadObjectiveTeamRowsForPlayer: requête: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			matchID, rowXUID string
			family           *string
			team             sql.NullInt64
			timePlayed       float64
		)
		vals := make([]float64, len(cols))
		ptrs := []any{&matchID, &rowXUID, &family, &team, &timePlayed}
		for i := range vals {
			ptrs = append(ptrs, &vals[i])
		}
		if err := rows.Scan(ptrs...); err != nil {
			return out, fmt.Errorf("LoadObjectiveTeamRowsForPlayer: scan: %w", err)
		}
		r := narrative.ObjectiveTeamRow{
			XUID: rowXUID, TeamID: narrative.NoObjectiveTeam, TimePlayedSeconds: timePlayed,
			Values: make(map[string]float64, len(cols)),
		}
		if team.Valid {
			r.TeamID = int(team.Int64)
		}
		if family != nil {
			r.Family = narrative.ObjectiveFamily(*family)
		}
		for i, c := range cols {
			r.Values[c] = vals[i]
		}
		out[matchID] = append(out[matchID], r)
	}
	return out, rows.Err()
}
