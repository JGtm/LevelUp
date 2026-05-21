// Package duckdb — BootstrapRepo implémente port.BootstrapRepository.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// versionUnknown est le placeholder retourné quand la version DuckDB n'a pas
// pu être lue (DB indisponible / pragma indisponible) — utilisé par /health.
const versionUnknown = "unknown"

// SharedReader est l'interface minimale dont BootstrapRepo a besoin pour
// lire `shared_matches_v2.duckdb`. Elle est volontairement structurelle —
// satisfaite par sharedprovider.Provider (sous-package) sans import croisé
// et par un wrapper local autour de *DB pour les tests / mode legacy.
//
// Contrat : Get retourne un *sql.DB prêt à lire + une fonction release que
// le caller DOIT appeler (typiquement via defer). Si err != nil, release
// est nil. Voir docs/adr/0014-shared-db-provider-b-swap.md (à venir).
type SharedReader interface {
	Get(ctx context.Context) (*sql.DB, func(), error)
}

// BootstrapRepo lit les données nécessaires à l'endpoint /bootstrap
// depuis shared_matches_v2.duckdb et metadata.duckdb.
type BootstrapRepo struct {
	shared   SharedReader
	metadata *DB
}

// NewBootstrapRepo crée un BootstrapRepo.
//
// shared : SharedReader (sharedprovider.Provider en prod, wrapper *DB en tests).
// metadataDB : *DB direct (hors scope du sprint sharedprovider).
func NewBootstrapRepo(shared SharedReader, metadataDB *DB) *BootstrapRepo {
	return &BootstrapRepo{shared: shared, metadata: metadataDB}
}

// GetMatchCount retourne le nombre de matchs dans match_registry.
func (r *BootstrapRepo) GetMatchCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetMatchCount: %w", err)
	}
	defer release()

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM match_registry").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("GetMatchCount: %w", err)
	}
	return count, nil
}

// GetDBVersion retourne la version DuckDB via pragma.
func (r *BootstrapRepo) GetDBVersion(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return versionUnknown, nil //nolint:nilerr // dégradation acceptable pour /health
	}
	defer release()

	var version string
	err = db.QueryRowContext(ctx, "PRAGMA version").Scan(&version)
	if err != nil {
		// fallback : essayer SELECT version()
		err2 := db.QueryRowContext(ctx, "SELECT version()").Scan(&version)
		if err2 != nil {
			return versionUnknown, nil
		}
	}
	return version, nil
}

// GetPlayerCount retourne le nombre de joueurs distincts dans match_participants.
func (r *BootstrapRepo) GetPlayerCount(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetPlayerCount: %w", err)
	}
	defer release()

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(DISTINCT xuid) FROM match_participants").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("GetPlayerCount: %w", err)
	}
	return count, nil
}

// GetLastSyncAt retourne le timestamp de la dernière modification dans match_registry.
func (r *BootstrapRepo) GetLastSyncAt(ctx context.Context) (*time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return nil, nil //nolint:nilerr // dégradation acceptable
	}
	defer release()

	var t *time.Time
	err = db.QueryRowContext(ctx, "SELECT MAX(last_updated_at) FROM match_registry").Scan(&t)
	if err != nil {
		return nil, nil //nolint:nilerr // table vide ou absente
	}
	return t, nil
}

// ValidateTypes vérifie que les types critiques sont correctement mappés.
// Utilisé dans le Sprint 0 comme test de sanité — pas exposé en prod.
func (r *BootstrapRepo) ValidateTypes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return fmt.Errorf("ValidateTypes: %w", err)
	}
	defer release()

	// Test 1 : UBIGINT → uint64 (weapon_id dans weapon_kills)
	var uval uint64
	err = db.QueryRowContext(ctx,
		"SELECT CAST(1234567890123456789 AS UBIGINT)").Scan(&uval)
	if err != nil {
		return fmt.Errorf("UBIGINT mapping: %w", err)
	}
	if uval != 1234567890123456789 {
		return fmt.Errorf("UBIGINT mapping: got %d, want 1234567890123456789", uval)
	}

	// Test 2 : TIMESTAMP WITH TIME ZONE → time.Time
	var tval time.Time
	err = db.QueryRowContext(ctx,
		"SELECT TIMESTAMPTZ '2024-01-15 10:30:00+00'").Scan(&tval)
	if err != nil {
		return fmt.Errorf("TIMESTAMPTZ mapping: %w", err)
	}
	if tval.IsZero() {
		return fmt.Errorf("TIMESTAMPTZ mapping: got zero time")
	}

	// Test 3 : BOOLEAN → bool
	var bval bool
	err = db.QueryRowContext(ctx, "SELECT TRUE").Scan(&bval)
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
