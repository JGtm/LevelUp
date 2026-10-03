// Package duckdb — squad_life_placement_repo.go : le placement des vies du bloc « Groupés ou
// isolés » de l'Emprise (plan `.ai/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V3.1, décision V11).
//
// UN CHARGEMENT, UNE REQUÊTE, BORNÉE DEUX FOIS (ADR 0036) :
//
//	match_id  la liste des matchs du périmètre D2, liée en UNE constante `VARCHAR[]`
//	          (clauseListeMatchs) : c'est elle que DuckDB pousse sous la fenêtre de la vue
//	          `_latest` (I2) — la fenêtre ne voit que les lignes de ces matchs ;
//	xuid      les joueurs de la composition, en semi-jointure (clauseListeParJointure) : un
//	          filtre sur un joueur ne traverse pas la fenêtre, il retire seulement les vies du
//	          reste du lobby après elle.
//
// Vue `match_life_placement_latest` UNIQUEMENT (ADR 0026, garde TestNoRawAppendOnlyReads), jamais
// `v_gamertag_lookup` (I1 : les noms viennent des fiches de l'Emprise). La variante de chaque match
// vient de `match_registry` (table ordinaire), la même colonne que le collecteur lit pour écrire
// `radar_m` : le service en résout la portée COURANTE par `mappings.PorteeDuRadar`.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/games"
)

// SquadLifePlacementRepo lit le placement des vies sur le SharedReader du joueur.
type SquadLifePlacementRepo struct {
	pdb *PlayerDB
}

// NewSquadLifePlacementRepo construit le repo à partir de la player DB (SharedReader).
func NewSquadLifePlacementRepo(pdb *PlayerDB) *SquadLifePlacementRepo {
	return &SquadLifePlacementRepo{pdb: pdb}
}

const squadLifePlacementQueryTimeout = 15 * time.Second

// qSquadLifePlacement : les deux prédicats de liste sont posés par LoadLifePlacement (%s, %s).
// L'ordre est total (match, joueur, début de vie) : la clé d'une vie.
const qSquadLifePlacement = `
SELECT p.match_id, p.xuid, p.start_ms, p.end_ms, p.duration_ms, p.measured_ms, p.median_m,
       p.beyond_ms, p.radar_m, p.carrier_ms, p.team_down_ms, p.unplaced_ms,
       p.teammate_unplaced_ms, p.kills, COALESCE(mr.game_variant_name, '') AS game_variant_name
FROM match_life_placement_latest p
LEFT JOIN match_registry mr ON mr.match_id = p.match_id
WHERE %s AND %s
ORDER BY p.match_id, p.xuid, p.start_ms`

// LoadLifePlacement rend les vies des `xuids` sur les `matchIDs`, et la variante de chaque match
// qui en porte. Liste vide : rien à lire. Table absente : games.ErrCapabilityNotSupported.
func (r *SquadLifePlacementRepo) LoadLifePlacement(
	ctx context.Context, matchIDs, xuids []string,
) (squademprise.PlacementRead, error) {
	out := squademprise.PlacementRead{Variants: map[string]string{}}
	if len(matchIDs) == 0 || len(xuids) == 0 {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, squadLifePlacementQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return out, fmt.Errorf("SquadLifePlacementRepo: shared reader: %w", err)
	}
	defer release()

	parMatch, argMatchs := clauseListeMatchs("p.match_id", matchIDs)
	parJoueur, argJoueurs := clauseListeParJointure("p.xuid", xuids)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(qSquadLifePlacement, parMatch, parJoueur), argMatchs, argJoueurs)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "SquadLifePlacementRepo: table du placement absente",
				"table", "match_life_placement_latest", "err", err)
			return out, games.ErrCapabilityNotSupported
		}
		return out, fmt.Errorf("SquadLifePlacementRepo: query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		row, variante, err := scanLifePlacement(rows)
		if err != nil {
			return out, err
		}
		out.Rows = append(out.Rows, row)
		out.Variants[row.MatchID] = variante
	}
	return out, rows.Err()
}

// scanLifePlacement lit une ligne ; un NULL reste nil (vie non mesurée, portée inconnue).
func scanLifePlacement(rows *sql.Rows) (squademprise.PlacementRow, string, error) {
	var row squademprise.PlacementRow
	var median, radar sql.NullFloat64
	var beyond sql.NullInt64
	var variante string
	if err := rows.Scan(&row.MatchID, &row.XUID, &row.StartMS, &row.EndMS, &row.DurationMS,
		&row.MeasuredMS, &median, &beyond, &radar, &row.CarrierMS, &row.TeamDownMS,
		&row.UnplacedMS, &row.TeammateUnplacedMS, &row.Kills, &variante); err != nil {
		return row, "", fmt.Errorf("SquadLifePlacementRepo: scan: %w", err)
	}
	if median.Valid {
		v := median.Float64
		row.MedianM = &v
	}
	if beyond.Valid {
		v := beyond.Int64
		row.BeyondMS = &v
	}
	if radar.Valid {
		v := radar.Float64
		row.RadarM = &v
	}
	return row, variante, nil
}
