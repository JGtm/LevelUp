package skill

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
	"log/slog"
	"os"
	"strings"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/persist"
)

// lusrCanonicalEnvFlag décide quel modèle est le "canonical" (= ce qui s'écrit
// dans rating_type='LUSR' du match_skill_rank, donc ce que voit l'UI).
//   - "LUSR" (défaut) : v1 BatchComputeLUSRWithMedals reste le writer canonical, v2
//     tourne en shadow et écrit dans ses propres tables.
//   - "LUSR_V2"        : v1 est court-circuité, v2 écrit dans rating_type='LUSR'
//     via Stratégie C (write-through aliasing — cf. ADR 0024).
const lusrCanonicalEnvFlag = "LEVELUP_LUSR_CANONICAL"

// IsLUSRV2Canonical retourne true si v2 est le writer canonical. Quand vrai :
//   - engine_postsync doit SKIPPER BatchComputeLUSRWithMedals (v1)
//   - le shadow runner doit écrire dans match_skill_rank avec rating_type='LUSR'
//     en plus de player_skill_state_v2 (Stratégie C, ADR 0024)
//
// La transition est gated par env var : tant que `LEVELUP_LUSR_CANONICAL=LUSR_V2`
// n'est pas posée, comportement = identique à avant (v1 canonical, v2 shadow).
func IsLUSRV2Canonical() bool {
	v := strings.ToUpper(strings.TrimSpace(os.Getenv(lusrCanonicalEnvFlag)))
	return v == "LUSR_V2"
}

// DefaultLUSRModeIfUnset pose les valeurs par défaut LUSR v2 canonical (ADR 0024)
// quand les flags d'environnement sont ABSENTS du process. Rend la bascule v2
// permanente côté serveur : elle survit à un reset de .air.toml / .env.local,
// sans dépendre d'une variable à ne pas oublier.
//
// Invariant : à appeler UNIQUEMENT au boot serveur (cmd/server), AVANT toute
// lecture des flags par la pipeline. Les tests et les CLI ne passent pas par là
// et restent déterministes (défaut v1, opt-in explicite) — c'est voulu.
//
// os.LookupEnv (et non Getenv) : un flag explicitement posé, MÊME à "" ou à une
// valeur d'opt-out (LEVELUP_LUSR_CANONICAL=LUSR pour revenir en v1), n'est jamais
// écrasé. Seul l'absence totale déclenche le défaut.
func DefaultLUSRModeIfUnset(ctx context.Context) {
	if _, set := os.LookupEnv(lusrV2EnvFlag); !set {
		if err := os.Setenv(lusrV2EnvFlag, "1"); err != nil {
			slog.WarnContext(ctx, "LUSR boot: défaut LEVELUP_LUSR_V2_ENABLED échoué", "err", err)
		} else {
			slog.InfoContext(ctx, "LUSR boot: LEVELUP_LUSR_V2_ENABLED absent → défaut 1 (v2 actif)")
		}
	}
	if _, set := os.LookupEnv(lusrCanonicalEnvFlag); !set {
		if err := os.Setenv(lusrCanonicalEnvFlag, "LUSR_V2"); err != nil {
			slog.WarnContext(ctx, "LUSR boot: défaut LEVELUP_LUSR_CANONICAL échoué", "err", err)
		} else {
			slog.InfoContext(ctx, "LUSR boot: LEVELUP_LUSR_CANONICAL absent → défaut LUSR_V2 (v2 canonical). Opt-out: =LUSR")
		}
	}
}

// LogLUSRModeAtBoot émet au démarrage un log du mode LUSR actif (routé vers
// logs/sync.log via la détection de module par package). Confirme la config au
// boot et ALERTE sur la misconfig dangereuse `canonical sans enabled` (v1 serait
// skippé ET le shadow v2 ne tournerait pas → aucun rating écrit).
func LogLUSRModeAtBoot(ctx context.Context) {
	enabled, canonical := IsLUSRV2Enabled(), IsLUSRV2Canonical()
	switch {
	case canonical && !enabled:
		slog.WarnContext(ctx, "LUSR mode: MISCONFIG — canonical=LUSR_V2 mais LUSR_V2_ENABLED absent : "+
			"v1 skippé ET shadow v2 inactif → aucun rating écrit. Poser LEVELUP_LUSR_V2_ENABLED=1.",
			"enabled", enabled, "canonical", canonical)
	case canonical:
		slog.InfoContext(ctx, "LUSR mode: v2 CANONICAL (écrit match_skill_rank rating_type=LUSR, v1 skippé)",
			"enabled", enabled, "canonical", canonical)
	case enabled:
		slog.InfoContext(ctx, "LUSR mode: v2 shadow (player_skill_state_v2 seul, UI lit v1)",
			"enabled", enabled, "canonical", canonical)
	default:
		slog.InfoContext(ctx, "LUSR mode: v1 (canonical historique)",
			"enabled", enabled, "canonical", canonical)
	}
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
// expectedWinProb (Sprint 1.A) : proba de victoire pré-match de l'équipe du
// owner, calculée par le caller à partir des ratings AVANT-match. nil si non
// disponible (match dégénéré) → colonne NULL.
//
// Incrémente les compteurs expvar `canonicalWritesTotal` / `canonicalWriteErrors`.
func writeCanonicalLUSRRow(ctx context.Context, playerDB *sql.DB, matchID string,
	state domain.SkillV2State, expectedWinProb *float64, startTime time.Time, boundaries []skillv2.TierBoundary) error {
	if playerDB == nil {
		return fmt.Errorf("writeCanonicalLUSRRow: playerDB nil")
	}
	deviation := skillv2.MapSigmaToLegacyDeviation(state.Sigma)

	// Palier CIBLE (μ brut) puis lissage d'affichage : montée immédiate, descente
	// ≤1 sous-palier/match (hystérésis / demotion protection), désactivé pendant
	// la phase de placement. μ interne reste honnête ; seul l'AFFICHÉ est lissé.
	// Cf. skill_v2/display_smoothing.go + étude .ai/thought_log.md [2026-05-31].
	tgtTier, tgtSub := skillv2.InferTier(state.Mu, boundaries)
	targetOrd := skillv2.TierOrdinal(boundaries, tgtTier.Name, tgtSub)
	prevOrd := loadPreviousDisplayedOrdinal(ctx, playerDB, state.PlaylistGroup, matchID, startTime, boundaries)
	dispOrd := skillv2.SmoothDisplayedOrdinal(prevOrd, targetOrd, state.Experience)
	tier, sub := skillv2.TierSubFromOrdinal(boundaries, dispOrd)

	// rating_value = position CONTINUE de μ (≠ bas du sous-palier), clampée dans
	// la plage du sous-palier AFFICHÉ (lissé). Le clamp préserve la cohérence
	// libellé↔valeur quand l'hystérésis bride une descente ; en régime normal μ
	// tombe dans le sous-palier affiché → pas de clamp → la valeur bouge à chaque
	// match, donc rating_delta = vrai gain de skill (cf. thought_log [2026-06-10]).
	rating := skillv2.MapMuToContinuousRating(state.Mu, boundaries)
	subMin, subMax := skillv2.LegacySubTierRange(tier, sub)
	if rating < subMin || rating > subMax {
		// μ continu hors du sous-palier AFFICHÉ : l'hystérésis bride l'affichage
		// (descente protégée). On clampe pour garder valeur↔badge cohérents ; trace
		// Debug pour observer un découplage prolongé (logs/sync.log).
		slog.DebugContext(ctx, "LUSR v2: valeur continue clampée au sous-palier affiché (hystérésis)",
			"match_id", matchID, "group", state.PlaylistGroup, "mu", state.Mu,
			"raw_rating", rating, "tier", tier.Name, "sub", sub, "sub_min", subMin, "sub_max", subMax)
		if rating < subMin {
			rating = subMin
		} else {
			rating = subMax
		}
	}
	label := skillv2.FormatTierSubLabel(tier, sub)

	tierName := tier.Name
	tierFR := tier.NameFR
	subPtr := &sub
	if tier.SubTiers <= 1 {
		// Onyx : pas de sub-tier, on garde sub=0 dans le slot
		zero := 0
		subPtr = &zero
	}
	// Sprint 3.B : delta vs le rating LUSR précédent du groupe ("+12 LUSR ce
	// match"). nil au premier match. Calculé AVANT l'insertion (la row courante
	// n'existe pas encore → la query renvoie bien le match précédent).
	var ratingDelta *float64
	if prev := loadPreviousLUSRRating(ctx, playerDB, state.PlaylistGroup, matchID, startTime); prev != nil {
		d := rating - *prev
		ratingDelta = &d
	}
	startPtr := &startTime
	if startTime.IsZero() {
		startPtr = nil // pas de start_time fiable → colonne NULL plutôt que l'époque zéro
	}
	baseRow := persist.LUSRRatingInsert{
		MatchID:         matchID,
		RatingValue:     rating,
		RatingDeviation: deviation,
		Tier:            &tierName,
		TierFR:          &tierFR,
		SubTier:         subPtr,
		TierLabel:       &label,
		RatingDelta:     ratingDelta,
		PlaylistGroup:   state.PlaylistGroup,
		ExpectedWinProb: expectedWinProb,
		StartTime:       startPtr,
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

// loadPreviousLUSRRating retourne le rating_value LUSR du match CHRONOLOGIQUEMENT
// précédent du groupe (start_time juste avant currentStart) — base du delta.
// nil si aucun (premier match du groupe).
//
// Ordre par start_time (date réelle du match), PAS written_at (= ordre d'ÉCRITURE,
// fragile : NULL si DEFAULT manquant, ou désordonné après ré-écriture/live re-sync
// — cause du bug delta +75 sur Choco, 2026-06-11). Fallback written_at uniquement
// pour les rows pas encore re-backfillées (start_time NULL), dominées dès qu'une
// row avec start_time existe (NULLS LAST).
//
// currentMatchID est EXCLU (retry sur table append-only → sa propre ligne pourrait
// exister). currentStart borne strictement (< ) pour ne pas se sélectionner soi-même
// ni un match futur déjà écrit par un run antérieur.
//
// Best-effort : toute erreur → nil + warn (delta simplement absent, pas de blocage).
func loadPreviousLUSRRating(ctx context.Context, playerDB *sql.DB, playlistGroup, currentMatchID string, currentStart time.Time) *float64 {
	var v sql.NullFloat64
	err := playerDB.QueryRowContext(ctx, `
		SELECT rating_value FROM match_skill_rank
		WHERE rating_type = 'LUSR' AND playlist_group = ? AND rating_value IS NOT NULL
		  AND match_id != ?
		  AND (start_time IS NULL OR start_time < ?)
		ORDER BY start_time DESC NULLS LAST, written_at DESC, id DESC
		LIMIT 1`, playlistGroup, currentMatchID, currentStart).Scan(&v)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		slog.WarnContext(ctx, "LUSR v2: lecture rating précédent échouée — delta absent",
			"group", playlistGroup, "err", err)
		return nil
	}
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

// loadPreviousDisplayedOrdinal retourne l'ordinal de palier AFFICHÉ au match
// précédent du groupe (slot rating_type='LUSR'), pour alimenter l'hystérésis de
// descente. Retourne -1 si aucun précédent ou en cas d'erreur (→ pas de lissage,
// on affiche la cible). Lit le tier/sub_tier lissés du match précédent, donc
// l'hystérésis se chaîne correctement de match en match.
//
// currentMatchID est EXCLU (retry sur table append-only). Ordre par start_time
// chronologique (même robustesse que loadPreviousLUSRRating), fallback written_at.
func loadPreviousDisplayedOrdinal(ctx context.Context, playerDB *sql.DB, playlistGroup, currentMatchID string, currentStart time.Time, boundaries []skillv2.TierBoundary) int {
	if playerDB == nil {
		return -1
	}
	var tierName sql.NullString
	var subTier sql.NullInt64
	err := playerDB.QueryRowContext(ctx, `
		SELECT tier, sub_tier FROM match_skill_rank
		WHERE rating_type = 'LUSR' AND playlist_group = ? AND tier IS NOT NULL
		  AND match_id != ?
		  AND (start_time IS NULL OR start_time < ?)
		ORDER BY start_time DESC NULLS LAST, written_at DESC, id DESC
		LIMIT 1`, playlistGroup, currentMatchID, currentStart).Scan(&tierName, &subTier)
	if err == sql.ErrNoRows || !tierName.Valid {
		return -1
	}
	if err != nil {
		slog.WarnContext(ctx, "LUSR v2: lecture palier précédent échouée — hystérésis désactivée ce match",
			"group", playlistGroup, "err", err)
		return -1
	}
	sub := 0
	if subTier.Valid {
		sub = int(subTier.Int64)
	}
	return skillv2.TierOrdinal(boundaries, tierName.String, sub)
}
