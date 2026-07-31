// Package duckdb — player_positions_repo.go : write+read repo des positions
// joueurs keyframe v3 (shared.match_player_positions).
//
// Source : pipeline diagnostic v3 (décodage film) qui produit des
// []positions.PlayerPosition par match (match-level, pas d'attribution xuid —
// cf. §N de .ai/RESEARCH_THEATER_RE.md). Ce repo persiste / relit ces positions.
//
// Connexion : comme les autres writers/readers shared (ObjectiveEventsRepo,
// weapon_kills), on passe par SharedReadDB().Get(ctx) → connexion DIRECTE à
// shared_matches_v2.duckdb (tables à la racine, PAS de préfixe `shared.`). En
// mode legacy / CLI backfill, ce handle est RW : WriteMatch peut écrire dessus.
// Le backfill v3 tourne HORS chemin live (sérialisé) — pas de pression ART.
//
// Écriture par match : DELETE FROM match_player_positions WHERE match_id = ?,
// puis INSERT de toutes les positions, dans une seule transaction (idempotent +
// atomique) — même pattern que ObjectiveEventsRepo.
//
// Capability gating : si la table n'existe pas (titre sans capability film /
// migration non appliquée), DuckDB remonte "Table ... does not exist",
// intercepté via isTableNotFoundErr -> games.ErrCapabilityNotSupported.
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

// WriteMatch remplace atomiquement TOUTES les positions d'un match.
//
// Pattern DELETE-then-INSERT-par-match : DELETE FROM match_player_positions
// WHERE match_id = ?, puis INSERT de toutes les positions, dans une seule
// transaction. Ré-exécuter pour le même matchID écrase l'existant (idempotent).
//
// positions == nil/[] est un no-op total (ni DELETE ni INSERT) : un décodage qui
// n'a rien produit ne doit pas effacer les positions déjà persistées (garde-fou
// anti-perte de données, aligné sur ObjectiveEventsRepo).
//
// Retourne games.ErrCapabilityNotSupported si la table n'existe pas.
func (r *PlayerPositionsRepo) WriteMatch(ctx context.Context, matchID string, pos []positions.PlayerPosition) error {
	if len(pos) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return fmt.Errorf("PlayerPositionsRepo.WriteMatch: shared reader: %w", err)
	}
	defer release()

	if err := writePlayerPositionsTx(ctx, db, matchID, pos); err != nil {
		if isTableNotFoundErr(err) {
			return games.ErrCapabilityNotSupported
		}
		return fmt.Errorf("PlayerPositionsRepo.WriteMatch(%s): %w", matchID, err)
	}
	return nil
}

// writePlayerPositionsTx exécute le DELETE+INSERT dans une transaction.
func writePlayerPositionsTx(ctx context.Context, db *sql.DB, matchID string, pos []positions.PlayerPosition) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM match_player_positions WHERE match_id = ?`, matchID); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	for i, p := range pos {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO match_player_positions (match_id, time_ms, x, y, z, team)
			VALUES (?, ?, ?, ?, ?, ?)`,
			matchID, p.TimeMS, p.X, p.Y, p.Z, p.Team,
		); err != nil {
			return fmt.Errorf("insert #%d: %w", i, err)
		}
	}
	return tx.Commit()
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
func loadPlayerPositionRows(ctx context.Context, db *sql.DB, matchID string) ([]positions.PlayerPosition, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT time_ms, x, y, z, team
		FROM match_player_positions
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
