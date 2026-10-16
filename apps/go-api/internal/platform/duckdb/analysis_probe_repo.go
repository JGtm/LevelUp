// Package duckdb — analysis_probe_repo.go : avancement de l'analyse du film d'un lot de
// matchs (signal `analysis_pending` des tuiles de l'accueil et de l'en-tête de la vue match).
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/matchflags"
)

// analysisProbeTimeout borne les deux lectures : deux `IN (...)` sur au plus quelques
// dizaines d'identifiants.
const analysisProbeTimeout = 10 * time.Second

// Q29AnalysisProbeRegistryTpl : par match du lot, sa première synchro, les bits du registre
// et la révision sous laquelle son film a été lu sans kill-feed. Lecture de `match_registry`
// (non append-only). Placeholders ? : les match_id.
const Q29AnalysisProbeRegistryTpl = `
SELECT match_id,
       first_sync_at,
       COALESCE(backfill_completed, 0),
       killsource_sans_killfeed_rev IS NOT NULL
FROM match_registry
WHERE match_id IN (%s)`

// Q29AnalysisProbeCitationsTpl : les matchs du lot qui portent au moins une ligne de
// citations. « Calculées » = AUCUNE condition sur la sentinelle `_processed` : elle n'est
// écrite que pour un match à zéro citation, un match avec citations porte ses deltas seuls
// (writeCitations, sync/citations.go) ; la sélection des candidats lit la même absence de
// ligne. Vue `_latest` uniquement (ADR 0026).
const Q29AnalysisProbeCitationsTpl = `
SELECT DISTINCT match_id
FROM match_citations_latest
WHERE match_id IN (%s)`

// AnalysisProbeRepo implémente port.AnalysisProbeRepository.
type AnalysisProbeRepo struct {
	pdb *PlayerDB
}

// NewAnalysisProbeRepo construit le dépôt sur la base du joueur.
func NewAnalysisProbeRepo(pdb *PlayerDB) *AnalysisProbeRepo {
	return &AnalysisProbeRepo{pdb: pdb}
}

// LoadAnalysisProbes : cf. port.AnalysisProbeRepository. Le registre se lit sur le lecteur
// shared LIVE (jamais le snapshot immuable de la vue match : un bit posé après le dernier cut
// y resterait invisible et l'indicateur ne s'éteindrait pas).
func (r *AnalysisProbeRepo) LoadAnalysisProbes(ctx context.Context, matchIDs []string) (map[string]domain.AnalysisProbe, error) {
	out := make(map[string]domain.AnalysisProbe, len(matchIDs))
	if len(matchIDs) == 0 || r == nil || r.pdb == nil {
		return out, nil
	}
	ctx, cancel := context.WithTimeout(ctx, analysisProbeTimeout)
	defer cancel()
	if err := r.loadRegistry(ctx, matchIDs, out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}
	return out, r.loadCitations(ctx, matchIDs, out)
}

func (r *AnalysisProbeRepo) loadRegistry(ctx context.Context, matchIDs []string, out map[string]domain.AnalysisProbe) error {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return fmt.Errorf("AnalysisProbeRepo.LoadAnalysisProbes: shared reader: %w", err)
	}
	defer release()
	rows, err := db.QueryContext(ctx, fmt.Sprintf(Q29AnalysisProbeRegistryTpl, Placeholders(len(matchIDs))), ToAnySlice(matchIDs)...)
	if err != nil {
		return fmt.Errorf("AnalysisProbeRepo.LoadAnalysisProbes (registre): %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			id          string
			firstSyncAt *time.Time
			bits        int64
			readNoFeed  bool
		)
		if err := rows.Scan(&id, &firstSyncAt, &bits, &readNoFeed); err != nil {
			return fmt.Errorf("AnalysisProbeRepo.LoadAnalysisProbes (registre) scan: %w", err)
		}
		out[id] = domain.AnalysisProbe{
			FirstSyncAt:    firstSyncAt,
			FilmAbsent:     bits&matchflags.MBitFilmAbsent != 0,
			KillSourceDone: bits&(matchflags.MBitWeaponKills|matchflags.MBitFilmAbsent) != 0 || readNoFeed,
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("AnalysisProbeRepo.LoadAnalysisProbes (registre) rows: %w", err)
	}
	return nil
}

func (r *AnalysisProbeRepo) loadCitations(ctx context.Context, matchIDs []string, out map[string]domain.AnalysisProbe) error {
	rows, err := r.pdb.ReadDB().QueryRecovered(ctx, fmt.Sprintf(Q29AnalysisProbeCitationsTpl, Placeholders(len(matchIDs))), ToAnySlice(matchIDs)...)
	if err != nil {
		return fmt.Errorf("AnalysisProbeRepo.LoadAnalysisProbes (citations): %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("AnalysisProbeRepo.LoadAnalysisProbes (citations) scan: %w", err)
		}
		if p, ok := out[id]; ok {
			p.HasCitationRows = true
			out[id] = p
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("AnalysisProbeRepo.LoadAnalysisProbes (citations) rows: %w", err)
	}
	return nil
}
