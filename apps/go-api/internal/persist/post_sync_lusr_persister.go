// Package persist — post_sync_lusr_persister.go : Phase 4 du refactor
// Collect→Persist — site #7 (LUSR ratings).
//
// **Problème résolu** : `sync.upsertLUSRRatings` faisait des UPDATE row-by-row
// concurrents sur `match_skill_rank` (sous mutex dblease mais multi-joueur
// en parallèle pool_size=4). Le pattern UPDATE-then-INSERT contourne pas
// le bug ART DuckDB (cf. INCIDENT_ART_CORRUPTION_DUCKDB.md verdict empirique).
//
// **Stratégie Option B (du PLAN_PHASE4_POSTSYNC_REFACTOR.md)** :
//
//  1. Compute tous les LUSR en RAM (déjà fait par batchComputeLUSR)
//  2. BEGIN TX
//     a. DELETE FROM match_skill_rank WHERE rating_type='LUSR' (1 batch)
//     b. INSERT batch tous les nouveaux LUSR
//  3. COMMIT
//
// 1 seul DELETE + 1 INSERT batch dans une TX = pas de UPDATE row-by-row =
// pas de stress ART sous concurrence. Les rows CSR (rating_type='CSR') sont
// préservées par le filtre WHERE.

package persist

import (
	"context"
	"errors"
	"fmt"
)

// LUSRRatingInsert — row LUSR prête à INSERT dans match_skill_rank.
// Structure identique à SkillRankInsert mais simplifiée pour clarté
// (LUSR uniquement, pas de CSR ici).
type LUSRRatingInsert struct {
	MatchID         string
	RatingValue     float64
	RatingDeviation float64
	Tier            *string
	TierFR          *string
	SubTier         *int
	TierLabel       *string
	RatingDelta     *float64
	PlaylistGroup   string
}

// PostSyncLUSRPersister écrit un batch complet de LUSR ratings en mode
// INSERT-only via DELETE batch + INSERT batch en 1 transaction.
type PostSyncLUSRPersister struct {
	db txBeginner
}

// NewPostSyncLUSRPersister construit un persister LUSR.
func NewPostSyncLUSRPersister(db txBeginner) *PostSyncLUSRPersister {
	return &PostSyncLUSRPersister{db: db}
}

// Persist remplace toutes les rows LUSR par les rows fournies, en une
// transaction atomique. Les rows CSR (rating_type='CSR') sont préservées
// (filtre WHERE rating_type='LUSR' sur le DELETE).
//
// Si rows est vide → no-op (pas de DELETE, on préserve l'existant).
//
// Atomique : si l'INSERT échoue, le DELETE est rollback.
func (p *PostSyncLUSRPersister) Persist(ctx context.Context, rows []LUSRRatingInsert) error {
	if len(rows) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx LUSR: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 1. DELETE batch unique des rows LUSR existantes.
	// Filtre WHERE rating_type='LUSR' → préserve les CSR.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM match_skill_rank WHERE rating_type = 'LUSR'`,
	); err != nil {
		return fmt.Errorf("persist: DELETE LUSR rows: %w", err)
	}

	// 2. INSERT batch — chaque row 1 par 1 (DuckDB ne supporte pas le
	// VALUES multi-row efficient comme PostgreSQL, mais la TX unique
	// suffit à la performance et la cohérence).
	for _, r := range rows {
		if r.MatchID == "" {
			return errors.New("persist: LUSRRatingInsert.MatchID vide")
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO match_skill_rank
				(match_id, rating_type, rating_value, rating_deviation,
				 tier, tier_fr, sub_tier, tier_label,
				 rating_delta, playlist_group)
			VALUES (?, 'LUSR', ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.MatchID, r.RatingValue, r.RatingDeviation,
			r.Tier, r.TierFR, r.SubTier, r.TierLabel,
			r.RatingDelta, r.PlaylistGroup,
		); err != nil {
			return fmt.Errorf("persist: INSERT LUSR %s: %w", r.MatchID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit LUSR: %w", err)
	}
	return nil
}
