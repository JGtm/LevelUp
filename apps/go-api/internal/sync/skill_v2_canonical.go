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
	"log/slog"
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
	state domain.SkillV2State, expectedWinProb *float64, boundaries []skillv2.TierBoundary) error {
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
	// Sprint 3.B : delta vs le rating LUSR précédent du groupe ("+12 LUSR ce
	// match"). nil au premier match. Calculé AVANT l'insertion (la row courante
	// n'existe pas encore → la query renvoie bien le match précédent).
	var ratingDelta *float64
	if prev := loadPreviousLUSRRating(ctx, playerDB, state.PlaylistGroup); prev != nil {
		d := rating - *prev
		ratingDelta = &d
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

// loadPreviousLUSRRating retourne le rating_value LUSR le plus récemment écrit
// pour ce groupe — la "version courante" AVANT l'insertion du match en cours
// (qui n'a pas encore été persisté quand cette fonction est appelée). Comme le
// shadow runner traite les matchs en ordre chronologique, c'est le rating du
// match précédent. nil si aucun (premier match du groupe).
//
// Best-effort : toute erreur (table non migrée, etc.) → nil + warn (le delta
// reste simplement absent, pas de blocage de l'écriture).
func loadPreviousLUSRRating(ctx context.Context, playerDB *sql.DB, playlistGroup string) *float64 {
	var v sql.NullFloat64
	err := playerDB.QueryRowContext(ctx, `
		SELECT rating_value FROM match_skill_rank
		WHERE rating_type = 'LUSR' AND playlist_group = ? AND rating_value IS NOT NULL
		ORDER BY written_at DESC, id DESC
		LIMIT 1`, playlistGroup).Scan(&v)
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
