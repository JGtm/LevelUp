package prestige

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// service_evaluate.go — partie évaluation du Service Prestige.
//
// Séparée de service.go pour respecter le seuil 500 L de CLAUDE.md
// et concentrer la logique de transition Status → Status sur un seul fichier.

// EvaluateForUser ré-évalue tous les défis actifs d'un joueur.
//
// Pour chaque défi actif :
//  1. Charger les matchs/médailles de la fenêtre
//  2. Évaluer (threshold ou cumulative)
//  3. Si transition → persister + émettre PP + télémétrie
//
// Cette méthode est volontairement minimaliste : la sélection précise des
// matchs (par mode) est déléguée au BaselineProvider en Phase 3 via une
// méthode dédiée. Phase 2 implémente le squelette ; Phase 3 branchera l'API.
func (s *service) EvaluateForUser(ctx context.Context, userID, titleSlug string) ([]EvaluationOutcome, error) {
	active, err := s.ListActiveChallenges(ctx, userID, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("list active: %w", err)
	}
	now := s.deps.Now()
	outcomes := make([]EvaluationOutcome, 0, len(active))

	for _, c := range active {
		outcome := s.evaluateOne(ctx, c, now)
		outcomes = append(outcomes, outcome)
	}
	return outcomes, nil
}

// evaluateOne traite un défi actif (chargement matchs + évaluation + persistance).
func (s *service) evaluateOne(ctx context.Context, c Challenge, now time.Time) EvaluationOutcome {
	out := EvaluationOutcome{
		ChallengeID: c.ID,
		OldStatus:   c.Status,
		NewStatus:   c.Status,
	}

	// Phase 2 : on récupère les matchs récents (proxy raisonnable pour Phase 3).
	// Le provider est LIÉ au joueur du service (cf. BaselineProvider) : ne pas
	// lui repasser c.UserID — c'est un player_slug, pas un xuid, et c'est
	// exactement l'inversion qui figeait tous les objectifs à zéro (2026-07-26).
	matches, err := s.deps.BaselineProvider.RecentMatches(
		ctx, c.TitleSlug, c.Metric, s.deps.Tuning.Baseline.WindowMatches,
	)
	if err != nil {
		slog.WarnContext(ctx, "prestige: evaluate fetch matches failed",
			"challenge_id", c.ID, "err", err)
		return out
	}

	samples := make([]MatchSample, len(matches))
	for i, m := range matches {
		samples[i] = MatchSample{StartedAt: m.StartedAt, MetricValue: m.MetricValue}
	}

	var result EvaluationResult
	switch c.EvalType {
	case EvalThreshold:
		result = EvaluateThreshold(s.deps.Tuning, c, samples, now)
	case EvalCumulative:
		// Phase 3 : brancher MedalEvent. Pour l'instant fallback sur threshold.
		result = EvaluateThreshold(s.deps.Tuning, c, samples, now)
	}

	out.NewValue = result.NewValue
	out.Reason = result.Reason

	if !result.StatusChanged {
		return out
	}

	if err := s.applyTransition(ctx, c, result, now, &out); err != nil {
		slog.WarnContext(ctx, "prestige: evaluate apply transition failed",
			"challenge_id", c.ID, "err", err)
	}
	return out
}

// applyTransition persiste la nouvelle status + crédite les PP en cas de complétion.
func (s *service) applyTransition(ctx context.Context, c Challenge, r EvaluationResult, now time.Time, out *EvaluationOutcome) error {
	if err := s.deps.Challenges.UpdateStatus(ctx, c.ID, r.NewStatus, now); err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	out.NewStatus = r.NewStatus

	switch r.NewStatus {
	case StatusCompleted:
		s.creditCompletion(ctx, c, now, out)
	case StatusExpired:
		updated := c
		updated.Status = StatusExpired
		updated.ExpiredAt = &now
		s.emitter.EmitTransition(ctx, updated, TelemetryExpired)
	}
	return nil
}

// creditCompletion calcule et émet les PP pour un défi completed.
func (s *service) creditCompletion(ctx context.Context, c Challenge, now time.Time, out *EvaluationOutcome) {
	updated := c
	updated.Status = StatusCompleted
	updated.CompletedAt = &now
	pp := PPForCompletion(s.deps.Tuning, c.Tier, false, c.DataTier)
	out.PPCredited = pp

	if pp > 0 {
		// Crédit STRICTEMENT mono-titre : les PP d'un défi vont au titre du défi et
		// à lui seul (décision produit 2026-07-18 — arcs, défis et points de
		// progression indépendants par titre). Le seam creditTitlesFor(), qui
		// permettait de créditer N titres, a été retiré le 2026-07-25 ; le garde-rail
		// no_cross_title_aggregation_test.go interdit sa réintroduction.
		ev := PrestigeEvent{
			ID:         newID("pe"),
			UserID:     c.UserID,
			TitleSlug:  c.TitleSlug,
			SourceType: SourceChallenge,
			SourceID:   c.ID,
			PPAmount:   pp,
			Tier:       c.Tier,
			CreatedAt:  now,
		}
		if err := s.deps.Prestige.EmitEvent(ctx, ev); err != nil {
			slog.WarnContext(ctx, "prestige: emit event failed", "err", err)
		}
	}
	s.emitter.EmitTransition(ctx, updated, TelemetryCompleted)

	// Si ce défi était la dernière étape d'un arc, clôturer l'arc et créditer
	// son bonus de complétion (distinct des PP du défi déjà crédités ci-dessus).
	if c.ArcID != "" {
		s.maybeCompleteArc(ctx, c.ArcID, c.UserID, c.TitleSlug, now, out)
	}
}

// maybeCompleteArc clôture un arc et crédite son bonus si toutes ses étapes
// sont terminées. No-op si l'arc est déjà clôturé ou s'il reste des objectifs
// non complétés. Best-effort : toute erreur de persistance est loggée sans
// faire échouer l'évaluation du défi déclencheur.
func (s *service) maybeCompleteArc(ctx context.Context, arcID, userID, titleSlug string, now time.Time, out *EvaluationOutcome) {
	arc, err := s.deps.Arcs.Get(ctx, arcID)
	if err != nil {
		slog.WarnContext(ctx, "prestige: arc fetch for completion failed", "arc_id", arcID, "err", err)
		return
	}
	if arc.CompletedAt != nil {
		return // déjà clôturé → bonus déjà crédité, idempotent
	}
	objectives, err := s.deps.Challenges.List(ctx, ChallengeFilter{
		UserID: userID, TitleSlug: titleSlug, ArcID: &arcID,
	})
	if err != nil || len(objectives) == 0 {
		return
	}
	objectivesPP := 0
	for _, o := range objectives {
		if o.Status != StatusCompleted {
			return // au moins une étape reste à faire
		}
		objectivesPP += PPForCompletion(s.deps.Tuning, o.Tier, false, o.DataTier)
	}

	if err := s.deps.Arcs.MarkCompleted(ctx, arcID, now); err != nil {
		slog.WarnContext(ctx, "prestige: mark arc completed failed", "arc_id", arcID, "err", err)
		return
	}
	bonus := PPForArcCompletion(s.deps.Tuning, objectivesPP)
	out.ArcCompletedID = arcID
	out.ArcPPCredited = bonus
	if bonus > 0 {
		ev := PrestigeEvent{
			ID:         newID("pe"),
			UserID:     userID,
			TitleSlug:  titleSlug,
			SourceType: SourceArc,
			SourceID:   arcID,
			PPAmount:   bonus,
			CreatedAt:  now,
		}
		if err := s.deps.Prestige.EmitEvent(ctx, ev); err != nil {
			slog.WarnContext(ctx, "prestige: emit arc event failed", "arc_id", arcID, "err", err)
		}
	}
	slog.InfoContext(ctx, "prestige: arc completed",
		"arc_id", arcID, "user_id", userID, "objectives_pp", objectivesPP, "bonus_pp", bonus)
}
