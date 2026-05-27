package sync

// skill_v2_canonical.go — Stratégie C (write-through aliasing) : v2 écrit
// dans match_skill_rank avec rating_type='LUSR' (slot historique lu par
// l'UI) + rating_type='LUSR_V2' (audit trail). Cf. ADR 0024 +
// .ai/LUSR_V2_HANDOFF.md.
//
// Extrait de skill_v2_shadow.go (2026-05-27) — concern unique : convertir
// un SkillV2State posterior en row(s) match_skill_rank et écrire de façon
// atomique via AppendOnlyLUSRPersister.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/persist"
)

// lusrCanonicalEnvFlag décide quel modèle est le "canonical" (= ce qui s'écrit
// dans rating_type='LUSR' du match_skill_rank, donc ce que voit l'UI).
//   - "LUSR" (défaut) : v1 batchComputeLUSR reste le writer canonical, v2
//     tourne en shadow et écrit dans ses propres tables.
//   - "LUSR_V2"        : v1 est court-circuité, v2 écrit dans rating_type='LUSR'
//     via Stratégie C (write-through aliasing — cf. ADR 0024).
const lusrCanonicalEnvFlag = "LEVELUP_LUSR_CANONICAL"

// IsLUSRV2Canonical retourne true si v2 est le writer canonical. Quand vrai :
//   - engine_postsync doit SKIPPER batchComputeLUSR (v1)
//   - le shadow runner doit écrire dans match_skill_rank avec rating_type='LUSR'
//     en plus de player_skill_state_v2 (Stratégie C, ADR 0024)
//
// La transition est gated par env var : tant que `LEVELUP_LUSR_CANONICAL=LUSR_V2`
// n'est pas posée, comportement = identique à avant (v1 canonical, v2 shadow).
func IsLUSRV2Canonical() bool {
	v := strings.ToUpper(strings.TrimSpace(os.Getenv(lusrCanonicalEnvFlag)))
	return v == "LUSR_V2"
}

// writeCanonicalLUSRRow écrit l'état v2 du owner dans match_skill_rank, en
// mode dual-row Stratégie C (ADR 0024) :
//   - rating_type='LUSR'    : slot historique lu par les readers UI
//   - rating_type='LUSR_V2' : audit trail pour analyse / sentinelle
//
// Les deux rows sont écrites dans la MÊME transaction (atomicité garantie par
// AppendOnlyLUSRPersister.Persist) — si l'INSERT v2 échoue, le LUSR rollback
// aussi. C'est l'invariant central de la sentinelle dual-row.
//
// Incrémente les compteurs expvar `canonicalWritesTotal` / `canonicalWriteErrors`.
func writeCanonicalLUSRRow(ctx context.Context, playerDB *sql.DB, matchID string,
	state domain.SkillV2State, boundaries []skillv2.TierBoundary) error {
	if playerDB == nil {
		return fmt.Errorf("writeCanonicalLUSRRow: playerDB nil")
	}
	rating := skillv2.MapMuToLegacyRating(state.Mu, boundaries)
	deviation := skillv2.MapSigmaToLegacyDeviation(state.Sigma)
	tier, sub := skillv2.InferTier(state.Mu, boundaries)
	label := skillv2.FormatTierLabel(state.Mu, boundaries)

	tierName := tier.Name
	tierFR := tier.NameFR
	subPtr := &sub
	if tier.SubTiers <= 1 {
		// Onyx : pas de sub-tier, on garde sub=0 dans le slot
		zero := 0
		subPtr = &zero
	}
	baseRow := persist.LUSRRatingInsert{
		MatchID:         matchID,
		RatingValue:     rating,
		RatingDeviation: deviation,
		Tier:            &tierName,
		TierFR:          &tierFR,
		SubTier:         subPtr,
		TierLabel:       &label,
		RatingDelta:     nil,
		PlaylistGroup:   state.PlaylistGroup,
	}
	rowLUSR := baseRow
	rowLUSR.RatingType = "LUSR"
	rowV2 := baseRow
	rowV2.RatingType = "LUSR_V2"

	if err := persist.NewAppendOnlyLUSRPersister(playerDB).Persist(ctx,
		[]persist.LUSRRatingInsert{rowLUSR, rowV2}); err != nil {
		canonicalWriteErrors.Add(1)
		return err
	}
	canonicalWritesTotal.Add(1)
	return nil
}
