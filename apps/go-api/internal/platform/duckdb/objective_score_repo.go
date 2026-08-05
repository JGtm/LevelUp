// Package duckdb — objective_score_repo.go : write+read repo de la timeline de
// score per-équipe v3 (shared.match_objective_score_timeline).
//
// Source : internal/analysis/objectivescore.DecodeScoreTimeline produit des
// []objectivescore.ScoreFrame par match (Strongholds/KOTH). Ce repo persiste /
// relit ces frames.
//
// Connexion : comme ObjectiveEventsRepo / weapon_kills_repo, on passe par
// SharedReadDB().Get(ctx) (ADR 0016 : handle DIRECT à shared_matches_v2.duckdb,
// tables à la racine, PAS de préfixe `shared.`). En legacy/CLI ce handle est RW.
//
// Écriture par match : DELETE-then-INSERT par match dans une transaction
// (idempotent + atomique), miroir d'ObjectiveEventsRepo.WriteMatch. frames
// nil/[] = no-op total (ne purge pas l'existant).
//
// Capability gating : tables absentes → isTableNotFoundErr ->
// games.ErrCapabilityNotSupported (comme ObjectiveEventsRepo).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/analysis/objectivescore"
	"levelup/go-api/internal/games"
)

// ObjectiveScoreRepo persiste et relit la timeline de score objectif v3 (shared).
type ObjectiveScoreRepo struct {
	pdb *PlayerDB
}

// NewObjectiveScoreRepo crée un ObjectiveScoreRepo lié à un PlayerDB.
func NewObjectiveScoreRepo(pdb *PlayerDB) *ObjectiveScoreRepo {
	return &ObjectiveScoreRepo{pdb: pdb}
}

// WriteMatch remplace atomiquement TOUTE la timeline de score d'un match
// (DELETE-then-INSERT par match). frames nil/[] = no-op (garde-fou anti-perte).
// Retourne games.ErrCapabilityNotSupported si la table n'existe pas.
func (r *ObjectiveScoreRepo) WriteMatch(ctx context.Context, matchID string, frames []objectivescore.ScoreFrame) error {
	if len(frames) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return fmt.Errorf("ObjectiveScoreRepo.WriteMatch: shared reader: %w", err)
	}
	defer release()

	if err := writeObjectiveScoreTx(ctx, db, matchID, frames); err != nil {
		if isTableNotFoundErr(err) {
			return games.ErrCapabilityNotSupported
		}
		return fmt.Errorf("ObjectiveScoreRepo.WriteMatch(%s): %w", matchID, err)
	}
	return nil
}

// writeObjectiveScoreTx exécute le DELETE+INSERT dans une transaction.
func writeObjectiveScoreTx(ctx context.Context, db *sql.DB, matchID string, frames []objectivescore.ScoreFrame) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM match_objective_score_timeline WHERE match_id = ?`, matchID); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	for _, f := range frames {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO match_objective_score_timeline (
				match_id, time_ms, team_0_score, team_1_score, source, confidence
			) VALUES (?, ?, ?, ?, ?, ?)`,
			matchID, f.TimeMS, f.Team0, f.Team1, f.Source, f.Confidence,
		); err != nil {
			return fmt.Errorf("insert time_ms=%d: %w", f.TimeMS, err)
		}
	}
	return tx.Commit()
}

// LoadMatch relit toute la timeline de score d'un match, ordonnée par time_ms.
// Retourne games.ErrCapabilityNotSupported si la table n'existe pas.
func (r *ObjectiveScoreRepo) LoadMatch(ctx context.Context, matchID string) ([]objectivescore.ScoreFrame, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("ObjectiveScoreRepo.LoadMatch: shared reader: %w", err)
	}
	defer release()

	frames, err := loadObjectiveScoreRows(ctx, db, matchID)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, games.ErrCapabilityNotSupported
		}
		return nil, fmt.Errorf("ObjectiveScoreRepo.LoadMatch(%s): %w", matchID, err)
	}
	return frames, nil
}

// loadObjectiveScoreRows lit la timeline ordonnée par time_ms.
func loadObjectiveScoreRows(ctx context.Context, db *sql.DB, matchID string) ([]objectivescore.ScoreFrame, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT time_ms, team_0_score, team_1_score, source, confidence
		FROM match_objective_score_timeline
		WHERE match_id = ?
		ORDER BY time_ms ASC`, matchID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var out []objectivescore.ScoreFrame
	for rows.Next() {
		var (
			f            objectivescore.ScoreFrame
			t0, t1       sql.NullInt64
			source, conf sql.NullString
		)
		if err := rows.Scan(&f.TimeMS, &t0, &t1, &source, &conf); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		f.Team0 = int(t0.Int64)
		f.Team1 = int(t1.Int64)
		f.Source = source.String
		f.Confidence = conf.String
		out = append(out, f)
	}
	return out, rows.Err()
}
