package duckdb

import (
	"context"
	"time"

	"levelup/go-api/internal/analysis/narrative"
)

// campaign_objective_samples.go — échantillons par match de la fenêtre d'un joueur pour les défis de
// campagne : requête chronologique commune (windowSamplesQuery) et mesure de l'axe « Objectif »
// (loadObjectiveSamples), lue depuis les lignes d'objectif des deux camps.

// windowSamplesQuery assemble la requête d'échantillons par match du joueur dans la fenêtre,
// chronologique, `selectExpr` étant l'expression lue. Temps canonique (règle n°8) : filtre de
// fenêtre ET tri, sinon un import à start_time NULL sort de la fenêtre et l'ordre des échantillons
// dérive. Campagne exclue (D-5, backlog B4) ; titre = celui de la base partagée lue.
func (p *CampaignSampleProvider) windowSamplesQuery(
	selectExpr, userID, playlistGroup string, since, until time.Time,
) (string, []any) {
	tsMr := StartTimeCanonicalSQL("mr")
	q := "SELECT " + selectExpr + ` AS val
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.xuid = ?
		  AND ` + tsMr + ` >= ? AND ` + tsMr + ` <= ?` +
		excludeCampaignClause(pdbTitleSlug(p.pdb), "mr")
	args := []any{userID, since, until}
	if playlistGroup != "" && playlistGroup != "all" {
		q += ` AND mr.playlist_id = ?`
		args = append(args, playlistGroup)
	}
	q += " ORDER BY " + tsMr + " ASC"
	return q, args
}

// loadObjectiveSamples rend, dans l'ordre chronologique, la note d'objectif de chaque match MESURÉ du
// joueur dans la fenêtre (narrative.ObjectiveScoresByMatch : part du joueur dans l'objectif de son
// équipe, 0-100, la même mesure que le radar de la vue Match). Les lignes des deux camps de tous les
// matchs se lisent en UNE requête. Un match sans donnée d'objectif (Slayer, Halo 5, ligne absente)
// n'est pas un échantillon : un titre sans match.objective.stats rend aucun échantillon, jamais des
// zéros qui feraient croire à une absence de progrès.
func (p *CampaignSampleProvider) loadObjectiveSamples(
	ctx context.Context, userID, playlistGroup string, since, until time.Time,
) ([]float64, error) {
	q, args := p.windowSamplesQuery("mp.match_id", userID, playlistGroup, since, until)
	db, release, err := p.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	teamRows, err := LoadObjectiveTeamRowsForMatches(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	scores := narrative.ObjectiveScoresByMatch(teamRows, userID)
	var out []float64
	for _, id := range ids {
		if s, ok := scores[id]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}
