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
	"levelup/go-api/internal/platform/dblease"
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

	// P2 (revue 2026-06-01) : sérialise l'UPSERT ON CONFLICT (xuid) DO UPDATE avec
	// le post-sync / CLI qui écrivent la même player DB (cf. SetExclusion). Le lease
	// KindPlayer remplace l'effet de bord fragile MaxOpenConns(1). Best-effort caller
	// (bootstrap) : ErrDBLocked est simplement remonté (ignoré côté bootstrap).
	w, err := r.pdb.AcquirePlayerWriterTimeout(dblease.PlayerLeaseTimeout)
	if err != nil {
		return fmt.Errorf("PrivacyStateRepo.UpsertPrivacyState: lease: %w", err)
	}
	defer w.Release()

	rwDB, err := OpenReadWrite(r.pdb.Player.Path())
	if err != nil {
		return fmt.Errorf("PrivacyStateRepo.UpsertPrivacyState: open rw: %w", err)
	}
	defer rwDB.Close()

	// ART-safe : SELECT-then-UPDATE-or-INSERT (pas d'ON CONFLICT, qui réécrit via
	// l'index ART de la PK xuid). État "dernière valeur observée", aucun index
	// secondaire → UPDATE non-indexé sûr. Sérialisé par le lease KindPlayer.
	if err := rwDB.UpsertNoConflict(ctx,
		`SELECT 1 FROM player_privacy_state WHERE xuid = ?`,
		[]any{state.XUID},
		`UPDATE player_privacy_state SET is_private = ?, observed_at = ?, source = ? WHERE xuid = ?`,
		[]any{state.IsPrivate, state.ObservedAt.UTC(), state.Source, state.XUID},
		`INSERT INTO player_privacy_state (xuid, is_private, observed_at, source) VALUES (?, ?, ?, ?)`,
		[]any{state.XUID, state.IsPrivate, state.ObservedAt.UTC(), state.Source},
	); err != nil {
		return fmt.Errorf("PrivacyStateRepo.UpsertPrivacyState: exec: %w", err)
	}
	return nil
}

// LoadPrivacyState charge l'état de privacy persisté. Retourne nil si absent.
func (r *PrivacyStateRepo) LoadPrivacyState(ctx context.Context, xuid string) (*domain.PlayerPrivacyState, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var s domain.PlayerPrivacyState
	rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx,
		`SELECT xuid, is_private, observed_at, source
		   FROM player_privacy_state
		  WHERE xuid = ?`,
		xuid,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("PrivacyStateRepo.LoadPrivacyState: scan: %w", err)
	}
	defer rows.Close()
	if err := rows.Scan(&s.XUID, &s.IsPrivate, &s.ObservedAt, &s.Source); err != nil {
		return nil, fmt.Errorf("PrivacyStateRepo.LoadPrivacyState: scan: %w", err)
	}
	return &s, nil
}
