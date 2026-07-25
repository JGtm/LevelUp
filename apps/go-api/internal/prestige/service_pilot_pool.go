package prestige

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
)

// service_pilot_pool.go — méthodes Service pour le mode pilote (auto-attribution)
// et la génération de pool d'escouade. Séparé pour respecter le seuil 500 L
// de service.go.

// ─────── Mode pilote ───────

// EnablePilotMode active le mode pilote pour un joueur sur un titre.
//
// Référence : Axe 7 du plan conceptuel. Génère :
//   - 1 défi quotidien auto-attribué (cadence=daily, palier Heroic par défaut)
//   - 1 défi hebdomadaire forcé (cadence=weekly)
//   - 3 défis hebdomadaires proposés (le joueur en accepte au moins 1)
//
// L'activation est idempotente : si le joueur a déjà des défis pilotes
// actifs sur les cadences daily/weekly, ils sont conservés.
func (s *service) EnablePilotMode(ctx context.Context, userID, titleSlug string) (PilotModeAttribution, error) {
	if userID == "" || titleSlug == "" {
		return PilotModeAttribution{}, fmt.Errorf("%w: user_id/title_slug requis", ErrInvalidInput)
	}

	out := PilotModeAttribution{}

	// Quotidien : créer si pas déjà 1 actif sur la cadence daily.
	dailyCount, err := s.deps.Challenges.CountActiveByCadence(ctx, userID, titleSlug, CadenceDaily)
	if err != nil {
		return out, fmt.Errorf("count daily: %w", err)
	}
	if dailyCount == 0 {
		c, err := s.attributeFromCatalog(ctx, userID, titleSlug, CadenceDaily)
		if err != nil {
			slog.WarnContext(ctx, "prestige: daily auto-attribution failed", "err", err)
		} else {
			out.Daily = &c
		}
	}

	// Hebdomadaire forcé : créer si pas déjà 1 actif sur la cadence weekly.
	weeklyCount, err := s.deps.Challenges.CountActiveByCadence(ctx, userID, titleSlug, CadenceWeekly)
	if err != nil {
		return out, fmt.Errorf("count weekly: %w", err)
	}
	if weeklyCount == 0 {
		c, err := s.attributeFromCatalog(ctx, userID, titleSlug, CadenceWeekly)
		if err != nil {
			slog.WarnContext(ctx, "prestige: weekly forced attribution failed", "err", err)
		} else {
			out.WeeklyForced = &c
		}
	}

	// 3 propositions hebdomadaires (parmi les templates weekly que le joueur n'a pas).
	choices, err := s.suggestWeeklyChoices(ctx, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "prestige: weekly choices unavailable", "err", err)
	}
	out.WeeklyChoices = choices

	slog.InfoContext(ctx, "prestige: pilot mode enabled",
		"user_id", userID, "title_slug", titleSlug,
		"daily_attributed", out.Daily != nil,
		"weekly_forced_attributed", out.WeeklyForced != nil,
		"weekly_choices", len(out.WeeklyChoices))
	return out, nil
}

// DisablePilotMode désactive le mode pilote pour un joueur sur un titre.
//
// Effet : les défis pilote ACTIFS non complétés sont archivés (statut
// `archived` — retrait système, pas un abandon volontaire du joueur, distinct
// pour la télémétrie/PP). Les défis pilote déjà terminaux (completed/expired…)
// sont laissés intacts (ils restent dans l'historique/Réalisations). Aucune
// nouvelle auto-attribution ne se fera tant que le mode n'est pas réactivé —
// l'état ON/OFF est dérivé côté client de la présence de défis pilote actifs.
func (s *service) DisablePilotMode(ctx context.Context, userID, titleSlug string) error {
	if userID == "" || titleSlug == "" {
		return fmt.Errorf("%w: user_id/title_slug requis", ErrInvalidInput)
	}
	activeStatus := StatusActive
	pilotMode := ModePilote
	list, err := s.deps.Challenges.List(ctx, ChallengeFilter{
		UserID:    userID,
		TitleSlug: titleSlug,
		Status:    &activeStatus,
		Mode:      &pilotMode,
	})
	if err != nil {
		return fmt.Errorf("list active pilot challenges: %w", err)
	}
	now := s.deps.Now()
	archived := 0
	for _, c := range list {
		if err := s.deps.Challenges.UpdateStatus(ctx, c.ID, StatusArchived, now); err != nil {
			slog.WarnContext(ctx, "prestige: pilot challenge archive failed",
				"err", err, "challenge_id", c.ID)
			continue
		}
		// Trace la transition d'archivage (retrait système, distinct d'un abandon
		// volontaire) — observabilité du churn pilote (F5), comme AbandonChallenge.
		c.Status = StatusArchived
		s.emitter.EmitTransition(ctx, c, TelemetryArchived)
		archived++
	}
	slog.InfoContext(ctx, "prestige: pilot mode disabled",
		"user_id", userID, "title_slug", titleSlug, "archived", archived)
	return nil
}

// attributeFromCatalog crée un défi pilote depuis un template aléatoire de la cadence.
func (s *service) attributeFromCatalog(ctx context.Context, userID, titleSlug string, cadence Cadence) (Challenge, error) {
	templates, err := s.deps.Templates.ListByTitle(ctx, titleSlug)
	if err != nil {
		return Challenge{}, fmt.Errorf("list templates: %w", err)
	}
	pool := filterTemplatesByCadence(templates, cadence)
	if len(pool) == 0 {
		return Challenge{}, errors.New("no templates available for cadence " + string(cadence))
	}
	t := pool[rand.Intn(len(pool))]

	return s.CreateChallenge(ctx, CreateChallengeRequest{
		UserID:      userID,
		TitleSlug:   titleSlug,
		TemplateID:  t.ID,
		Metric:      t.Metric,
		Target:      t.HeroicTarget, // Mode pilote : palier Heroic par défaut
		WindowType:  t.WindowType,
		WindowValue: t.WindowValue,
		Cadence:     t.Cadence,
		EvalType:    t.EvalType,
		Mode:        ModePilote,
		Label:       t.LabelFR,
		// Défi auto-attribué par le mode pilote → origine pilot_mode (calage coach, ADR 0020).
		Source: ChallengeSourcePilotMode,
	})
}

// suggestWeeklyChoices retourne 3 templates weekly que le joueur n'a pas encore tentés.
func (s *service) suggestWeeklyChoices(ctx context.Context, titleSlug string) ([]Template, error) {
	all, err := s.deps.Templates.ListByTitle(ctx, titleSlug)
	if err != nil {
		return nil, err
	}
	weekly := filterTemplatesByCadence(all, CadenceWeekly)
	count := 3
	if len(weekly) < count {
		count = len(weekly)
	}
	return weekly[:count], nil
}

// filterTemplatesByCadence retourne les templates qui matchent la cadence.
func filterTemplatesByCadence(templates []Template, cadence Cadence) []Template {
	out := make([]Template, 0, len(templates))
	for _, t := range templates {
		if t.Cadence == cadence {
			out = append(out, t)
		}
	}
	return out
}

// ─────── Pool collectif d'escouade ───────

// RefreshSquadPool génère un pool de 6 à 9 défis thématiques pour une escouade.
//
// Référence : Axe 4 du plan conceptuel (modèle pool collectif).
// Le pool est stocké en mémoire/DB — Phase 4 simple : on retourne juste la
// liste des templates sélectionnés, l'escouade pourra ensuite proposer un
// défi depuis cette liste (les templates sont copiés dans squad_challenge
// au moment de la création par n'importe quel membre).
func (s *service) RefreshSquadPool(ctx context.Context, squadID, titleSlug, requestedBy string) ([]Template, error) {
	if squadID == "" || titleSlug == "" {
		return nil, fmt.Errorf("%w: squad_id/title_slug requis", ErrInvalidInput)
	}

	// requestedBy OBLIGATOIRE : sans lui, la garde d'appartenance ci-dessous
	// serait contournée (Lot 3, fermeture du bypass). Le handler fournit toujours
	// l'acteur du chemin ; un appel sans acteur est un bug/abus → 400.
	if requestedBy == "" {
		return nil, fmt.Errorf("%w: requested_by requis", ErrInvalidInput)
	}
	// Vérifier que requestedBy est membre de l'escouade (sinon rejeter).
	members, err := s.deps.Squads.ListMembers(ctx, squadID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	if !isMember(members, requestedBy) {
		return nil, fmt.Errorf("%w: not a member of squad", ErrInvalidInput)
	}

	all, err := s.deps.Templates.ListByTitle(ctx, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}

	// Écarte les templates que l'évaluateur d'escouade ne sait pas évaluer
	// honnêtement (cf. squadEligibleTemplates).
	eligible, discarded := squadEligibleTemplates(all)
	if discarded > 0 {
		slog.InfoContext(ctx, "prestige: squad pool discarded templates with unsupported eval_type",
			"squad_id", squadID, "title_slug", titleSlug,
			"catalog", len(all), "eligible", len(eligible), "discarded", discarded)
	}
	if len(eligible) == 0 {
		// Aucun template proposable : catalogue vide (titre sans
		// challenges/templates.toml comme halo_5, ou metadata non seedée) OU
		// catalogue entièrement écarté par le filtre eval_type. Dégradation
		// gracieuse — pool VIDE (200, []) plutôt qu'un 500 masqué. Title-agnostic
		// (aucune comparaison de slug). L'UI affiche « Aucun défi proposé ». Loggé
		// pour distinguer d'un vrai bug de seed en prod.
		slog.InfoContext(ctx, "prestige: squad pool empty (no eligible templates for title)",
			"squad_id", squadID, "title_slug", titleSlug, "requested_by", requestedBy,
			"catalog", len(all))
		return []Template{}, nil
	}

	// Génération aléatoire d'un pool entre size_min et size_max.
	size := s.deps.Tuning.SquadPool.SizeMin
	if s.deps.Tuning.SquadPool.SizeMax > size {
		size += rand.Intn(s.deps.Tuning.SquadPool.SizeMax - s.deps.Tuning.SquadPool.SizeMin + 1)
	}
	if size > len(eligible) {
		size = len(eligible)
	}

	// Mélange, puis biais coach : si un profil d'escouade est disponible, on place
	// en tête les templates ciblant l'axe le plus faible de l'escouade (au lieu du
	// shuffle pur). Sans provider/profil, focusAxis == "" → shuffle inchangé.
	shuffled := make([]Template, len(eligible))
	copy(shuffled, eligible)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	shuffled = BiasTemplatesByAxis(shuffled, s.squadFocusAxis(ctx, members, titleSlug))
	pool := shuffled[:size]

	slog.InfoContext(ctx, "prestige: squad pool refreshed",
		"squad_id", squadID, "title_slug", titleSlug,
		"requested_by", requestedBy, "size", size)
	return pool, nil
}

// squadEligibleTemplates ne garde que les templates dont la règle AFFICHÉE
// correspond à l'évaluation RÉELLE d'un défi d'escouade, et retourne le nombre
// d'écartés.
//
// L'évaluation d'escouade est CUMULATIVE pour tous les défis en V1
// (AggregateSquadProgress somme la métrique par membre sur les matchs de
// l'escouade ; le mode « threshold » — cible atteinte SUR UN match — n'est pas
// couvert, cf. squad_progress.go). Proposer un template `eval_type=threshold`
// revenait donc à afficher une règle que le back n'applique pas : le défi se
// complétait par cumul. On les retire du pool plutôt que d'exposer une règle
// mensongère.
//
// Title-agnostic : le filtre branche sur `eval_type` (donnée du catalogue), jamais
// sur le slug du titre. À retirer quand le threshold d'escouade sera implémenté
// (item « threshold squad par match » du backlog, PLAN_SQUAD_CHALLENGES).
func squadEligibleTemplates(templates []Template) (eligible []Template, discarded int) {
	eligible = make([]Template, 0, len(templates))
	for _, t := range templates {
		if t.EvalType == EvalThreshold {
			discarded++
			continue
		}
		eligible = append(eligible, t)
	}
	return eligible, discarded
}

// isMember vérifie qu'un user_id est dans la liste des membres.
func isMember(members []SquadMember, userID string) bool {
	for _, m := range members {
		if m.UserID == userID {
			return true
		}
	}
	return false
}

// squadFocusAxis dérive l'axe focal de l'escouade (le plus faible) depuis le
// profil 6-axes agrégé des membres, via le SquadProfile provider. Retourne ""
// (pas de biais) si le provider est absent ou si le profil est indisponible.
func (s *service) squadFocusAxis(ctx context.Context, members []SquadMember, titleSlug string) string {
	if s.deps.SquadProfile == nil {
		return ""
	}
	axes, err := s.deps.SquadProfile.SquadAxes(ctx, rosterXUIDs(members), titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "prestige: squad focus axis unavailable (squad axes failed), pool non biaisé",
			"err", err, "title_slug", titleSlug)
		return ""
	}
	if len(axes) == 0 {
		return ""
	}
	return SquadFocusAxis(AggregateSquadAxes(axes))
}
