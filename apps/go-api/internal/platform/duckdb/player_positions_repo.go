// Package duckdb — player_positions_repo.go : repo de LECTURE des positions joueurs
// (shared.match_player_positions), pour la carte de chaleur de la fiche de match.
//
// LECTURE SEULE DEPUIS LE 2026-09-06 (décision utilisateur 1 du plan v2). `WriteMatch`
// (DELETE-then-INSERT par match, sur le handle de LECTURE du pool) a été SUPPRIMÉE : la table
// est désormais une PROJECTION DE L'ARTEFACT de rejeu, écrite en INSERT purs par
// `persist.PlayerPositionsPersister` sous le lease RW, depuis les dérivations post-rangement
// (`sync/replayartifacts/positions.go`). Un DELETE indexé sur une table écrite dans le cycle de
// sync est le déclencheur direct du bug ART DuckDB #23046 — la doctrine (ADR 0019/0026) l'exclut,
// et `match_player_positions` figure maintenant dans les listes de `no_art_patterns_test.go` et
// `append_only_state_guard_test.go`.
//
// LA LECTURE PASSE PAR LA VUE `_latest`, ET C'EST OBLIGATOIRE (règle ART n°2) : la table est
// append-only par PASSE, et une lecture brute servirait les positions de toutes les projections
// précédentes empilées — une carte de chaleur qui compterait chaque point deux ou trois fois.
//
// Connexion : comme les autres readers shared, on passe par SharedReadDB().Get(ctx) → connexion
// DIRECTE à shared_matches_v2.duckdb (tables à la racine, PAS de préfixe `shared.`).
//
// Capability gating : si la vue n'existe pas (titre sans capability film / migration non
// appliquée), DuckDB remonte "Table ... does not exist", intercepté via isTableNotFoundErr ->
// games.ErrCapabilityNotSupported.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis/positions"
	"levelup/go-api/internal/games"
)

// PlayerPositionsRepo persiste et relit les positions keyframe dans shared.
type PlayerPositionsRepo struct {
	pdb *PlayerDB
}

// NewPlayerPositionsRepo crée un PlayerPositionsRepo lié à un PlayerDB.
func NewPlayerPositionsRepo(pdb *PlayerDB) *PlayerPositionsRepo {
	return &PlayerPositionsRepo{pdb: pdb}
}

// LoadMatch relit toutes les positions d'un match, ordonnées par time_ms puis
// par ordre d'insertion (rowid). Retourne games.ErrCapabilityNotSupported si la
// table n'existe pas.
func (r *PlayerPositionsRepo) LoadMatch(ctx context.Context, matchID string) ([]positions.PlayerPosition, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("PlayerPositionsRepo.LoadMatch: shared reader: %w", err)
	}
	defer release()

	out, err := loadPlayerPositionRows(ctx, db, matchID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, games.ErrCapabilityNotSupported
		}
		return nil, fmt.Errorf("PlayerPositionsRepo.LoadMatch(%s): %w", matchID, err)
	}
	return out, nil
}

// loadPlayerPositionRows lit les positions d'un match, ordonnées par time_ms ASC.
//
// ⚠ LA VUE `_latest`, JAMAIS LA TABLE (ADR 0026, règle ART n°2) : la table empile une génération
// par projection d'artefact, et une lecture brute servirait toutes les passes superposées.
func loadPlayerPositionRows(ctx context.Context, db *sql.DB, matchID string) ([]positions.PlayerPosition, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT time_ms, x, y, z, team
		FROM match_player_positions_latest
		WHERE match_id = ?
		ORDER BY time_ms ASC`, matchID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var out []positions.PlayerPosition
	for rows.Next() {
		var (
			p            positions.PlayerPosition
			timeMS, team sql.NullInt64
			x, y, z      sql.NullFloat64
		)
		if err := rows.Scan(&timeMS, &x, &y, &z, &team); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		p.TimeMS = int(timeMS.Int64)
		p.X = float32(x.Float64)
		p.Y = float32(y.Float64)
		p.Z = float32(z.Float64)
		if team.Valid {
			p.Team = int(team.Int64)
		} else {
			p.Team = positions.TeamUnknown
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}
