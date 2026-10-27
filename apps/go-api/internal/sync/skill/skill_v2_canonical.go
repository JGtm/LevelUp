package skill

// skill_v2_canonical.go — Stratégie C (write-through aliasing) : v2 écrit
// dans match_skill_rank avec rating_type='LUSR' (slot historique lu par
// l'UI) + rating_type='LUSR_V2' (audit trail). Cf. ADR 0024 +
// .ai/archive/V7/LUSR v2/LUSR_V2_HANDOFF.md. La v2 est le seul modèle : le mode
// « canonique » est inconditionnel dès qu'une base joueur est fournie (aucun
// drapeau d'environnement ; ADR 0024, amendement du 2026-10-17).
//
// Extrait de skill_v2_shadow.go (2026-05-27) — concern unique : convertir
// un SkillV2State posterior en row(s) match_skill_rank et écrire de façon
// atomique via AppendOnlyLUSRPersister.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/persist"
)

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
func writeCanonicalLUSRRow(ctx context.Context, c shadowRunContext, matchID string,
	state domain.SkillV2State, expectedWinProb *float64, startTime time.Time) error {
	playerDB, boundaries := c.playerDB, c.tierBoundaries
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
	prev := previousQuery{group: state.PlaylistGroup, matchID: matchID, start: startTime, excluded: c.excluded}
	prevOrd := loadPreviousDisplayedOrdinal(ctx, playerDB, prev, boundaries)
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
	subPtr := &sub
	if tier.SubTiers <= 1 {
		// Onyx : pas de sub-tier, on garde sub=0 dans le slot
		zero := 0
		subPtr = &zero
	}
	// Sprint 3.B : delta vs le rating LUSR précédent du groupe, hors matchs exclus
	// ("+12 LUSR ce match"). nil au premier match. Calculé AVANT l'insertion (la row courante
	// n'existe pas encore → la query renvoie bien le match précédent).
	var ratingDelta *float64
	if prevRating := loadPreviousLUSRRating(ctx, playerDB, prev); prevRating != nil {
		d := rating - *prevRating
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

// previousQuery borne la recherche du match précédent d'un groupe de playlist.
type previousQuery struct {
	group string
	// matchID est exclu (retry sur table append-only : sa propre ligne peut exister).
	matchID string
	// start borne strictement (<) : ni soi-même ni un match futur déjà écrit.
	start time.Time
	// excluded : matchs is_excluded du joueur. Leur ancienne ligne reste dans la table
	// append-only après l'exclusion : elle ne peut pas servir de précédent.
	excluded map[string]bool
}

// previousLUSRSQL liste les lignes LUSR candidates au rôle de précédent, du plus récent
// au plus ancien, par la vue match_skill_rank_latest_by_type (règle ART n°2 : une ligne
// par match et par type de note, jamais une version périmée). %s : prédicat de non-nullité
// de la colonne dont l'appelant a besoin.
//
// Ordre par start_time (date réelle du match), PAS written_at (= ordre d'ÉCRITURE,
// fragile : NULL si DEFAULT manquant, ou désordonné après ré-écriture/live re-sync
// — cause du bug delta +75 sur Choco, 2026-06-11). Fallback written_at uniquement
// pour les rows pas encore re-backfillées (start_time NULL), dominées dès qu'une
// row avec start_time existe (NULLS LAST).
const previousLUSRSQL = `
	SELECT match_id, rating_value, tier, sub_tier FROM match_skill_rank_latest_by_type
	WHERE rating_type = 'LUSR' AND playlist_group = ? AND %s
	  AND match_id != ?
	  AND (start_time IS NULL OR start_time < ?)
	ORDER BY start_time DESC NULLS LAST, written_at DESC, id DESC`

// previousRow est une ligne candidate au rôle de précédent.
type previousRow struct {
	rating sql.NullFloat64
	tier   sql.NullString
	sub    sql.NullInt64
}

// loadPreviousRow rend la ligne LUSR du match CHRONOLOGIQUEMENT précédent du groupe qui
// n'est pas exclu (start_time juste avant q.start). nil si aucune. notNull : colonne que la
// ligne doit porter (rating_value ou tier).
func loadPreviousRow(ctx context.Context, playerDB *sql.DB, q previousQuery, notNull string) (*previousRow, error) {
	rows, err := playerDB.QueryContext(ctx, fmt.Sprintf(previousLUSRSQL, notNull+" IS NOT NULL"),
		q.group, q.matchID, q.start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var r previousRow
		if err := rows.Scan(&id, &r.rating, &r.tier, &r.sub); err != nil {
			return nil, err
		}
		if q.excluded[id] {
			continue
		}
		return &r, nil
	}
	return nil, rows.Err()
}

// loadPreviousLUSRRating retourne le rating_value LUSR du match CHRONOLOGIQUEMENT
// précédent du groupe, hors matchs exclus — base du delta. nil si aucun (premier match
// du groupe).
//
// Best-effort : toute erreur → nil + warn (delta simplement absent, pas de blocage).
func loadPreviousLUSRRating(ctx context.Context, playerDB *sql.DB, q previousQuery) *float64 {
	r, err := loadPreviousRow(ctx, playerDB, q, "rating_value")
	if err != nil {
		slog.WarnContext(ctx, "LUSR v2: lecture rating précédent échouée — delta absent",
			"group", q.group, "err", err)
		return nil
	}
	if r == nil || !r.rating.Valid {
		return nil
	}
	return &r.rating.Float64
}

// loadPreviousDisplayedOrdinal retourne l'ordinal de palier AFFICHÉ au match
// précédent du groupe (slot rating_type='LUSR', hors matchs exclus), pour alimenter
// l'hystérésis de descente. Retourne -1 si aucun précédent ou en cas d'erreur (→ pas de
// lissage, on affiche la cible). Lit le tier/sub_tier lissés du match précédent, donc
// l'hystérésis se chaîne correctement de match en match.
func loadPreviousDisplayedOrdinal(ctx context.Context, playerDB *sql.DB, q previousQuery, boundaries []skillv2.TierBoundary) int {
	if playerDB == nil {
		return -1
	}
	r, err := loadPreviousRow(ctx, playerDB, q, "tier")
	if err != nil {
		slog.WarnContext(ctx, "LUSR v2: lecture palier précédent échouée — hystérésis désactivée ce match",
			"group", q.group, "err", err)
		return -1
	}
	if r == nil || !r.tier.Valid {
		return -1
	}
	sub := 0
	if r.sub.Valid {
		sub = int(r.sub.Int64)
	}
	return skillv2.TierOrdinal(boundaries, r.tier.String, sub)
}
