// Package persist — lusr_append_only_persister.go : Phase 2 du plan
// d'éradication ART (cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md).
//
// **Prototype TDD** : nouvelle version de PostSyncLUSRPersister qui ne fait
// JAMAIS de DELETE. Toute écriture est un INSERT pur dans une table
// append-only (PK élargie à (match_id, rating_type, written_at)) où la
// version "courante" est obtenue via la vue match_skill_rank_latest.
//
// Coexiste avec l'ancien PostSyncLUSRPersister (à supprimer en Phase 2.D
// quand la bascule du code de prod sera complétée).
//
// **Invariant** : un INSERT ne déclenche jamais le bug ART DuckDB (le bug
// se déclenche sur DELETE/UPDATE qui touchent l'index). Donc cette
// implémentation est immune par construction.

package persist

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// LUSRRatingInsert — row LUSR prête à INSERT dans match_skill_rank.
// Simplifié pour le path append-only (LUSR uniquement ; CSR a son propre
// chemin via UpsertCSRRow).
//
// RatingType : permet la Stratégie C (ADR 0024) où v2 écrit DEUX rows pour le
// même match — une avec rating_type='LUSR' (slot historique pour readers UI)
// et une avec rating_type='LUSR_V2' (audit trail). Si vide, défaut 'LUSR'.
type LUSRRatingInsert struct {
	MatchID         string
	RatingType      string // défaut 'LUSR' si vide
	RatingValue     float64
	RatingDeviation float64
	Tier            *string
	TierFR          *string
	SubTier         *int
	TierLabel       *string
	RatingDelta     *float64
	PlaylistGroup   string
	// ExpectedWinProb : proba de victoire pré-match de l'équipe du joueur
	// (LUSR v2 Sprint 1.A). nil pour les chemins qui ne la calculent pas
	// (CSR, héritage v1) → colonne NULL.
	ExpectedWinProb *float64
	// StartTime : timestamp réel du match (match_registry, UTC). Peuplé pour
	// donner un ordre CHRONOLOGIQUE fiable aux readers du delta (loadPrevious*),
	// indépendant de written_at (= ordre d'ÉCRITURE, fragile aux ré-écritures).
	// nil → colonne NULL (chemins legacy qui ne le fournissent pas).
	StartTime *time.Time
}

// AppendOnlyLUSRPersister écrit des ratings LUSR en mode strictement
// append-only — chaque appel ajoute des rows, sans jamais effacer les
// anciennes. La "version courante" d'un match est obtenue côté lecture
// via la vue match_skill_rank_latest (PARTITION BY match_id, rating_type
// ORDER BY written_at DESC).
//
// Hypothèse de schéma cible (Phase 2.B — non encore appliquée en prod) :
//
//	CREATE TABLE match_skill_rank (
//	  match_id          VARCHAR NOT NULL,
//	  rating_type       VARCHAR NOT NULL,
//	  rating_value      FLOAT,
//	  rating_deviation  FLOAT,
//	  tier              VARCHAR,
//	  tier_fr           VARCHAR,
//	  sub_tier          SMALLINT,
//	  tier_label        VARCHAR,
//	  rating_delta      FLOAT,
//	  playlist_group    VARCHAR,
//	  start_time        TIMESTAMP,
//	  written_at        TIMESTAMP NOT NULL DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
//	  created_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
//	  updated_at        TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP),
//	  PRIMARY KEY (match_id, rating_type, written_at)
//	);
//	CREATE VIEW match_skill_rank_latest AS
//	  SELECT * FROM match_skill_rank
//	  QUALIFY ROW_NUMBER() OVER (PARTITION BY match_id, rating_type
//	                             ORDER BY written_at DESC) = 1;
type AppendOnlyLUSRPersister struct {
	db txBeginner
}

// NewAppendOnlyLUSRPersister construit un persister LUSR append-only.
func NewAppendOnlyLUSRPersister(db txBeginner) *AppendOnlyLUSRPersister {
	return &AppendOnlyLUSRPersister{db: db}
}

// Persist ajoute les rows LUSR fournies à la table append-only. Aucun
// DELETE, aucun UPDATE — uniquement des INSERT. Idempotent du point de
// vue de la vue match_skill_rank_latest (la dernière version d'un
// (match_id, rating_type) est celle qui sort).
//
// Atomique : tout le batch est wrappé dans une seule transaction.
// Si un INSERT échoue, le batch entier est rollback.
//
// Sémantique :
//   - rows vide → no-op
//   - PK collision sur (match_id, rating_type, written_at) → géré par le
//     DEFAULT now() qui produit un timestamp unique en pratique. En cas
//     extrême de collision (même nanoseconde), l'INSERT échoue et la TX
//     rollback. C'est le seul cas d'échec d'écriture.
func (p *AppendOnlyLUSRPersister) Persist(ctx context.Context, rows []LUSRRatingInsert) error {
	if len(rows) == 0 {
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx LUSR append-only: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, r := range rows {
		if r.MatchID == "" {
			return errors.New("persist: LUSRRatingInsert.MatchID vide")
		}
		// INSERT pur. Pas de ON CONFLICT, pas de WHERE, pas de DELETE.
		// La PK (match_id, rating_type, written_at) tolère N versions
		// par (match_id, rating_type). written_at = horloge UTC côté DB.
		rt := r.RatingType
		if rt == "" {
			rt = "LUSR"
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO match_skill_rank
				(match_id, rating_type, rating_value, rating_deviation,
				 tier, tier_fr, sub_tier, tier_label,
				 rating_delta, playlist_group, expected_win_prob, start_time)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.MatchID, rt, r.RatingValue, r.RatingDeviation,
			r.Tier, r.TierFR, r.SubTier, r.TierLabel,
			r.RatingDelta, r.PlaylistGroup, r.ExpectedWinProb, r.StartTime,
		); err != nil {
			return fmt.Errorf("persist: INSERT LUSR append-only %s/%s: %w", r.MatchID, rt, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit LUSR append-only: %w", err)
	}
	return nil
}
