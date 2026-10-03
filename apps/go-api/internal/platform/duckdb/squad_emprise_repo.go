// Package duckdb — squad_emprise_repo.go : la feuille de match du bloc « Emprise » de l'Escouade
// (lot L4 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) — les frags aux armes spéciales
// de chaque participant, les deux camps.
//
// UNE LECTURE SUR UN SCOPE FERMÉ de match_id (aucun filtre temporel : le fragment timezone
// canonique ne s'applique pas). `match_participants` n'est pas une table append-only : pas de vue
// `_latest` à lire.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis/squademprise"
)

// SquadEmpriseRepo lit la feuille de match sur le SharedReader du joueur.
type SquadEmpriseRepo struct {
	pdb *PlayerDB
}

// NewSquadEmpriseRepo construit le repo à partir de la player DB (SharedReader).
func NewSquadEmpriseRepo(pdb *PlayerDB) *SquadEmpriseRepo {
	return &SquadEmpriseRepo{pdb: pdb}
}

const squadEmpriseQueryTimeout = 15 * time.Second

// LoadPowerWeaponKills rend une ligne par (match, participant). Une valeur NULL reste nil : la
// feuille ne le dit pas, ce n'est pas un zéro.
func (r *SquadEmpriseRepo) LoadPowerWeaponKills(ctx context.Context, matchIDs []string) ([]squademprise.PowerKillRow, error) {
	if len(matchIDs) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, squadEmpriseQueryTimeout)
	defer cancel()
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("SquadEmpriseRepo: shared reader: %w", err)
	}
	defer release()

	q := `SELECT match_id, xuid, team_id, power_weapon_kills
	      FROM match_participants
	      WHERE match_id IN (` + Placeholders(len(matchIDs)) + `)`
	rows, err := db.QueryContext(ctx, q, ToAnySlice(matchIDs)...)
	if err != nil {
		return nil, fmt.Errorf("SquadEmpriseRepo: power_weapon_kills query: %w", err)
	}
	defer rows.Close()
	var out []squademprise.PowerKillRow
	for rows.Next() {
		var row squademprise.PowerKillRow
		var team, kills sql.NullInt64
		if err := rows.Scan(&row.MatchID, &row.XUID, &team, &kills); err != nil {
			return nil, fmt.Errorf("SquadEmpriseRepo: power_weapon_kills scan: %w", err)
		}
		if team.Valid {
			t := int(team.Int64)
			row.TeamID = &t
		}
		if kills.Valid {
			k := int(kills.Int64)
			row.Kills = &k
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
