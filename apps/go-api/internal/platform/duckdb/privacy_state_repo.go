// Package duckdb — privacy_state_repo.go : persistance de l'état de privacy joueur.
//
// Sprint 55 E4 : PrivacyStateRepo implémente port.PrivacyStateRepository.
// Table : player_privacy_state (créée par migration "add_player_privacy_state").
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// PrivacyStateRepo persiste/charge l'état de privacy depuis stats.duckdb.
type PrivacyStateRepo struct {
	pdb *PlayerDB
}

// NewPrivacyStateRepo crée un PrivacyStateRepo depuis un PlayerDB.
func NewPrivacyStateRepo(pdb *PlayerDB) *PrivacyStateRepo {
	return &PrivacyStateRepo{pdb: pdb}
}

// UpsertPrivacyState persiste l'état observé depuis Waypoint.
// Remplace silencieusement une entrée existante pour le même xuid.
func (r *PrivacyStateRepo) UpsertPrivacyState(ctx context.Context, state domain.PlayerPrivacyState) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return fmt.Errorf("PrivacyStateRepo.UpsertPrivacyState: open rw: %w", err)
	}
	defer rwDB.Close()

	_, err = rwDB.Exec(ctx,
		`INSERT INTO player_privacy_state (xuid, is_private, observed_at, source)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT (xuid) DO UPDATE SET
		   is_private  = EXCLUDED.is_private,
		   observed_at = EXCLUDED.observed_at,
		   source      = EXCLUDED.source`,
		state.XUID, state.IsPrivate, state.ObservedAt.UTC(), state.Source,
	)
	if err != nil {
		return fmt.Errorf("PrivacyStateRepo.UpsertPrivacyState: exec: %w", err)
	}
	return nil
}

// LoadPrivacyState charge l'état de privacy persisté. Retourne nil si absent.
func (r *PrivacyStateRepo) LoadPrivacyState(ctx context.Context, xuid string) (*domain.PlayerPrivacyState, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var s domain.PlayerPrivacyState
	err := r.pdb.ReadDB().QueryRow(ctx,
		`SELECT xuid, is_private, observed_at, source
		   FROM player_privacy_state
		  WHERE xuid = ?`,
		xuid,
	).Scan(&s.XUID, &s.IsPrivate, &s.ObservedAt, &s.Source)

	switch err {
	case nil:
		return &s, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, fmt.Errorf("PrivacyStateRepo.LoadPrivacyState: scan: %w", err)
	}
}
