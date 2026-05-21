// Package duckdb — csr_thresholds_repo.go : lookup season_id → seuil placement CSR.
//
// Halo Infinite a baissé le seuil de placement de 10 → 5 à partir de Season 3
// (2023-03-07). Cette table de mapping permet au display d'afficher "(X/N)"
// avec le bon N selon la saison du match/snapshot consulté.
//
// Source de vérité : metadata.csr_placement_thresholds (créée par la migration
// add_csr_placement_thresholds). Fallback CSRPlacementThresholdDefault (=5) si
// la saison est inconnue ou si la connexion metadata est indisponible.
package duckdb

import (
	"context"
	"database/sql"
	"sync"
)

// CSRPlacementThresholdDefault est le seuil par défaut quand la saison est
// inconnue (S3+ depuis 2023-03-07). Dupliqué depuis sync.CSRPlacementThresholdDefault
// pour éviter un import cycle (sync importe duckdb). Les deux DOIVENT rester
// synchronisés ; test cross-package dans csr_thresholds_repo_test.go.
const CSRPlacementThresholdDefault = 5

// CSRThresholdsRepo expose un lookup season_id → threshold avec cache mémoire.
// La table change rarement (insert manuel par saison) ; invalidation au boot
// uniquement (acceptable pour cette donnée quasi-statique).
type CSRThresholdsRepo struct {
	metadata *DB
	mu       sync.RWMutex
	cache    map[string]int
}

// NewCSRThresholdsRepo construit un repo. metadata peut être nil (fallback
// systématique au default — comportement de dégradation, non bloquant).
func NewCSRThresholdsRepo(metadata *DB) *CSRThresholdsRepo {
	return &CSRThresholdsRepo{
		metadata: metadata,
		cache:    make(map[string]int),
	}
}

// Get retourne le threshold pour une saison. Fallback :
//   - seasonID vide → default
//   - metadata indisponible → default
//   - saison absente de la table → default
//   - erreur SQL → default (log au debug, pas warn — pas bloquant)
//
// Cache hit après premier lookup réussi.
func (r *CSRThresholdsRepo) Get(ctx context.Context, seasonID string) int {
	if seasonID == "" || r == nil || r.metadata == nil {
		return CSRPlacementThresholdDefault
	}
	r.mu.RLock()
	if v, ok := r.cache[seasonID]; ok {
		r.mu.RUnlock()
		return v
	}
	r.mu.RUnlock()

	var threshold int
	err := r.metadata.QueryRow(ctx,
		`SELECT threshold FROM csr_placement_thresholds WHERE season_id = ?`,
		seasonID,
	).Scan(&threshold)
	if err != nil {
		if err == sql.ErrNoRows {
			// Saison inconnue : fallback default + cache pour éviter requery.
			r.cacheSet(seasonID, CSRPlacementThresholdDefault)
		}
		return CSRPlacementThresholdDefault
	}
	r.cacheSet(seasonID, threshold)
	return threshold
}

func (r *CSRThresholdsRepo) cacheSet(seasonID string, threshold int) {
	r.mu.Lock()
	r.cache[seasonID] = threshold
	r.mu.Unlock()
}
