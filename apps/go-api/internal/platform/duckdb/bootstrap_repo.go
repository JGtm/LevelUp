// Package duckdb — BootstrapRepo implémente port.BootstrapRepository.
package duckdb

import (
	"context"
	"fmt"
	"time"
)

// BootstrapRepo lit les données nécessaires à l'endpoint /bootstrap
// depuis shared_matches_v2.duckdb et metadata.duckdb.
type BootstrapRepo struct {
	shared   *DB
	metadata *DB
}

// NewBootstrapRepo crée un BootstrapRepo à partir des bases partagées.
func NewBootstrapRepo(sharedDB, metadataDB *DB) *BootstrapRepo {
	return &BootstrapRepo{shared: sharedDB, metadata: metadataDB}
}

// GetMatchCount retourne le nombre de matchs dans match_registry.
func (r *BootstrapRepo) GetMatchCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	err := r.shared.QueryRow(ctx, "SELECT COUNT(*) FROM match_registry").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("GetMatchCount: %w", err)
	}
	return count, nil
}

// GetDBVersion retourne la version DuckDB via pragma.
func (r *BootstrapRepo) GetDBVersion(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var version string
	err := r.shared.QueryRow(ctx, "PRAGMA version").Scan(&version)
	if err != nil {
		// fallback : essayer SELECT version()
		err2 := r.shared.QueryRow(ctx, "SELECT version()").Scan(&version)
		if err2 != nil {
			return "unknown", nil
		}
	}
	return version, nil
}

// ValidateTypes vérifie que les types critiques sont correctement mappés.
// Utilisé dans le Sprint 0 comme test de sanité — pas exposé en prod.
func (r *BootstrapRepo) ValidateTypes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Test 1 : UBIGINT → uint64 (weapon_id dans weapon_kills)
	var uval uint64
	err := r.shared.QueryRow(ctx,
		"SELECT CAST(1234567890123456789 AS UBIGINT)").Scan(&uval)
	if err != nil {
		return fmt.Errorf("UBIGINT mapping: %w", err)
	}
	if uval != 1234567890123456789 {
		return fmt.Errorf("UBIGINT mapping: got %d, want 1234567890123456789", uval)
	}

	// Test 2 : TIMESTAMP WITH TIME ZONE → time.Time
	var tval time.Time
	err = r.shared.QueryRow(ctx,
		"SELECT TIMESTAMPTZ '2024-01-15 10:30:00+00'").Scan(&tval)
	if err != nil {
		return fmt.Errorf("TIMESTAMPTZ mapping: %w", err)
	}
	if tval.IsZero() {
		return fmt.Errorf("TIMESTAMPTZ mapping: got zero time")
	}

	// Test 3 : BOOLEAN → bool
	var bval bool
	err = r.shared.QueryRow(ctx, "SELECT TRUE").Scan(&bval)
	if err != nil {
		return fmt.Errorf("BOOLEAN mapping: %w", err)
	}
	if !bval {
		return fmt.Errorf("BOOLEAN mapping: got false, want true")
	}

	return nil
}

// GetCareerRanksSample vérifie que metadata.duckdb est lisible (test Sprint 0).
func (r *BootstrapRepo) GetCareerRanksSample(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	err := r.metadata.QueryRow(ctx, "SELECT COUNT(*) FROM career_ranks").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("GetCareerRanksSample: %w", err)
	}
	return count, nil
}
