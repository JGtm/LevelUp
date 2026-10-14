// Package duckdb — squad_emprise_repo.go : les frags aux armes spéciales du bloc « Emprise » (lot L4
// du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) — la feuille de match de chaque participant,
// les deux camps, et (squad_emprise_journal_repo.go) les frags par arme du journal des morts.
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
	"levelup/go-api/internal/port"
)

// SquadEmpriseRepo lit la feuille de match et le journal des morts sur le SharedReader du joueur.
type SquadEmpriseRepo struct {
	pdb *PlayerDB
	// classifier traduit une source de dégât en clé de registre ; nil (titre sans
	// `film.kill_source`) : le journal n'est pas lu.
	classifier port.KillSourceClassifier
}

// NewSquadEmpriseRepo construit le repo à partir de la player DB (SharedReader). classifier peut
// être nil.
func NewSquadEmpriseRepo(pdb *PlayerDB, classifier port.KillSourceClassifier) *SquadEmpriseRepo {
	return &SquadEmpriseRepo{pdb: pdb, classifier: classifier}
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
