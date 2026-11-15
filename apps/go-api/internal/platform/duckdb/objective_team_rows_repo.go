// Package duckdb — objective_team_rows_repo.go : lignes (match, joueur) de
// `match_objective_stats_latest` pour la part d'un joueur dans l'objectif de son équipe
// (narrative.ComputeObjectiveTeamShare : note de performance à objectif, note d'objectif des
// radars et des profils, défis de campagne).
//
// Une ligne porte les valeurs BRUTES de toutes les colonnes d'action et de durée : c'est la
// famille du MATCH, décidée côté analysis, qui choisit lesquelles comptent. Les listes de
// colonnes sont générées depuis les tables narrative (objectiveIndexSelectColumns — source
// unique, aucune liste parallèle). Le camp vient de match_participants. TOUS les joueurs des
// matchs demandés sont chargés (les deux camps) : l'équipe se reconstitue côté analysis. Un
// chargement = UNE requête par lot de matchs, jamais une requête par match.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/platform/duckdb/sqlfrag"
)

// objectiveTeamRowsChunk : nombre maximal d'identifiants de match par requête.
const objectiveTeamRowsChunk = 500

// objectiveTeamRowsTimeout borne une lecture de page.
const objectiveTeamRowsTimeout = 15 * time.Second

// objectiveTeamRowsSelect rend le SELECT commun et les colonnes lues, dans l'ordre du scan.
func objectiveTeamRowsSelect() (string, []string) {
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
		JOIN match_participants mp ON mp.match_id = o.match_id AND mp.xuid = o.xuid`)
	return sb.String(), cols
}

// objectiveStatsViewPresent dit si la vue existe : une base sans elle (titre sans
// match_objective_stats, Halo 5) rend une map vide et aucune erreur.
func objectiveStatsViewPresent(ctx context.Context, sharedDB *sql.DB, who string) (bool, error) {
	var present int
	if err := sharedDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'match_objective_stats_latest'`,
	).Scan(&present); err != nil {
		return false, fmt.Errorf("%s: schéma: %w", who, err)
	}
	return present > 0, nil
}

// scanObjectiveTeamRows lit les lignes d'une requête de objectiveTeamRowsSelect dans `out`.
func scanObjectiveTeamRows(rows *sql.Rows, cols []string, who string, out map[string][]narrative.ObjectiveTeamRow) error {
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
			return fmt.Errorf("%s: scan: %w", who, err)
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
	return rows.Err()
}

// LoadObjectiveTeamRowsForPlayer retourne, par match, les lignes d'objectif de tous les
// participants des matchs où `xuid` a joué. Lecture par la vue `_latest` UNIQUEMENT (ADR
// 0026). Une base sans la vue (titre sans match_objective_stats, Halo 5) rend une map vide et
// aucune erreur : la métrique est alors absente de la note, jamais un échec. Les matchs de la
// Campagne sont exclus, comme dans le chargement des notes (loadHistoryForPerf).
func LoadObjectiveTeamRowsForPlayer(ctx context.Context, sharedDB *sql.DB, xuid string) (map[string][]narrative.ObjectiveTeamRow, error) {
	const who = "LoadObjectiveTeamRowsForPlayer"
	out := map[string][]narrative.ObjectiveTeamRow{}
	if xuid == "" {
		return out, nil
	}
	present, err := objectiveStatsViewPresent(ctx, sharedDB, who)
	if err != nil || !present {
		return out, err
	}
	sel, cols := objectiveTeamRowsSelect()
	q := sel + `
		WHERE o.match_id IN (SELECT match_id FROM match_participants WHERE xuid = ?)` +
		sqlfrag.SQLExcludeCampaignByMatchIDCtx(ctx, "o.match_id")
	rows, err := sharedDB.QueryContext(ctx, q, xuid)
	if err != nil {
		return out, fmt.Errorf("%s: requête: %w", who, err)
	}
	defer rows.Close()
	return out, scanObjectiveTeamRows(rows, cols, who, out)
}

// LoadObjectiveTeamRowsForMatches retourne, par match, les lignes d'objectif de TOUS les
// participants des matchs donnés (les deux camps), par la vue `_latest` : les lignes d'équipe d'un
// scope fermé de matchs (session, fenêtre d'un profil, matchs partagés d'une escouade). Même
// dégradation que LoadObjectiveTeamRowsForPlayer : vue absente = map vide sans erreur. Une requête
// par lot de objectiveTeamRowsChunk matchs.
func LoadObjectiveTeamRowsForMatches(ctx context.Context, sharedDB *sql.DB, matchIDs []string) (map[string][]narrative.ObjectiveTeamRow, error) {
	const who = "LoadObjectiveTeamRowsForMatches"
	out := map[string][]narrative.ObjectiveTeamRow{}
	if len(matchIDs) == 0 {
		return out, nil
	}
	present, err := objectiveStatsViewPresent(ctx, sharedDB, who)
	if err != nil || !present {
		return out, err
	}
	sel, cols := objectiveTeamRowsSelect()
	for start := 0; start < len(matchIDs); start += objectiveTeamRowsChunk {
		chunk := matchIDs[start:min(start+objectiveTeamRowsChunk, len(matchIDs))]
		rows, err := sharedDB.QueryContext(ctx,
			sel+"\n\t\tWHERE o.match_id IN ("+Placeholders(len(chunk))+")", ToAnySlice(chunk)...)
		if err != nil {
			return out, fmt.Errorf("%s: requête: %w", who, err)
		}
		err = scanObjectiveTeamRows(rows, cols, who, out)
		rows.Close()
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// LoadObjectiveTeamRows : LoadObjectiveTeamRowsForMatches sur le lecteur partagé du joueur. Implémente
// port.ObjectiveTeamRepository (câblé pour les seuls titres qui portent la capability
// match.objective.stats).
func (r *ObjectiveStatsRepo) LoadObjectiveTeamRows(ctx context.Context, matchIDs []string) (map[string][]narrative.ObjectiveTeamRow, error) {
	if len(matchIDs) == 0 {
		return map[string][]narrative.ObjectiveTeamRow{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, objectiveTeamRowsTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ObjectiveStatsRepo: shared reader: %w", err)
	}
	defer release()
	return LoadObjectiveTeamRowsForMatches(ctx, db, matchIDs)
}

// XUIDsByGamertag résout des gamertags en xuids par shared.xuid_aliases (les coéquipiers NON suivis
// n'ont pas de player DB). Un gamertag peut porter plusieurs xuids historiques ; un gamertag inconnu
// est absent du résultat.
func (r *ObjectiveStatsRepo) XUIDsByGamertag(ctx context.Context, gamertags []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(gamertags) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, objectiveTeamRowsTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("ObjectiveStatsRepo: shared reader: %w", err)
	}
	defer release()
	rows, err := db.QueryContext(ctx,
		"SELECT gamertag, xuid FROM xuid_aliases WHERE gamertag IN ("+Placeholders(len(gamertags))+")",
		ToAnySlice(gamertags)...)
	if err != nil {
		return out, fmt.Errorf("ObjectiveStatsRepo: alias: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var gt, xuid string
		if err := rows.Scan(&gt, &xuid); err != nil {
			return out, fmt.Errorf("ObjectiveStatsRepo: alias scan: %w", err)
		}
		out[gt] = append(out[gt], xuid)
	}
	return out, rows.Err()
}
