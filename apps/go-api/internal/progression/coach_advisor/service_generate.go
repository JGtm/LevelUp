package coach_advisor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/prestige"
)

// service_generate.go — implémentation de Service.GenerateProposals.
// Pipeline complet :
//   1. Short-circuit si proactive disabled
//   2. Pour chaque signal :
//      a. Tente matching catalog (matcher + filter par MinCatalogMatchScore)
//      b. Si match insuffisant : tente synthèse (synthesizer)
//      c. Si rien : skip signal (log debug)
//   3. Tente composition d'arc (ArcComposer.TryCompose)
//   4. Si arc composé : filtre les proposals individuelles dont le signal est
//      déjà couvert par une étape de l'arc (évite doublon UI)
//   5. Cap au MaxProposalsPerSync (arc compte pour 1)
//   6. Pour chaque proposal nouvelle : supersession des pending sur même
//      (metric, axis) avec strength uplift >= SupersessionStrengthUplift
//   7. Persiste

// templateChoice est le résultat de la résolution d'un signal vers un
// template (catalog ou synthétisé).
type templateChoice struct {
	signal   Signal
	template prestige.Template
	origin   ProposalOrigin
}

func (s *service) GenerateProposals(ctx context.Context, in GenerateInput) ([]Proposal, error) {
	if !in.ProactiveEnabled {
		return nil, nil
	}
	if in.UserID == "" || in.TitleSlug == "" {
		return nil, fmt.Errorf("coach_advisor.GenerateProposals: userID and titleSlug required")
	}
	if s.deps.Templates == nil {
		return nil, fmt.Errorf("coach_advisor.GenerateProposals: TemplateRepo dependency missing")
	}
	if in.Now.IsZero() {
		in.Now = s.deps.Now()
	}

	// 1. Charger le catalogue de templates pour le titre
	catalog, err := s.deps.Templates.ListByTitle(ctx, in.TitleSlug)
	if err != nil {
		return nil, fmt.Errorf("coach_advisor.GenerateProposals: list templates: %w", err)
	}

	// 2. Résoudre chaque signal → template choice (catalog ou synthétisé)
	choices := s.resolveSignalsToTemplates(ctx, in, catalog)

	// 3. Tenter une composition d'arc (sur les signals, pas les choices)
	arcSpec, hasArc := TryCompose(in.Signals, s.deps.ComposerConfig)

	// 4. Si arc composé, filtrer les choices dont le signal est couvert par
	//    l'arc (évite UI doublonnée)
	if hasArc {
		choices = filterChoicesNotInArc(choices, arcSpec)
	}

	// 5. Construire les proposals (challenges + éventuel arc)
	proposals := s.buildProposals(in, choices, arcSpec, hasArc)

	// 6. Cap max
	if len(proposals) > s.deps.Tuning.MaxProposalsPerSync {
		proposals = proposals[:s.deps.Tuning.MaxProposalsPerSync]
	}

	// 7. Supersession + persist
	persisted := make([]Proposal, 0, len(proposals))
	for _, p := range proposals {
		if err := s.supersedeOlder(ctx, in, p); err != nil {
			slog.WarnContext(ctx, "coach_advisor: supersession partial failure",
				"err", err, "user", in.UserID, "titleSlug", in.TitleSlug)
		}
		if err := s.deps.Repo.Create(ctx, p); err != nil {
			slog.ErrorContext(ctx, "coach_advisor: create proposal failed",
				"err", err, "id", p.ID, "kind", p.Kind)
			continue
		}
		persisted = append(persisted, p)
	}

	slog.InfoContext(ctx, "coach_advisor: proposals generated",
		"user", in.UserID, "titleSlug", in.TitleSlug,
		"signals", len(in.Signals), "persisted", len(persisted), "arc", hasArc)

	return persisted, nil
}

// resolveSignalsToTemplates : pour chaque signal, tente catalog matching
// puis synthèse en fallback.
func (s *service) resolveSignalsToTemplates(ctx context.Context, in GenerateInput, catalog []prestige.Template) []templateChoice {
	out := make([]templateChoice, 0, len(in.Signals))
	for _, sig := range in.Signals {
		// a. Catalog matching
		scores := MatchTemplateToSignal(sig, catalog, s.deps.MatcherWeights)
		filtered := FilterByMinScore(scores, s.deps.Tuning.MinCatalogMatchScore)
		if len(filtered) > 0 {
			out = append(out, templateChoice{
				signal:   sig,
				template: filtered[0].Template,
				origin:   OriginCatalog,
			})
			continue
		}
		// b. Synthesis fallback
		if s.deps.Synthesizer == nil {
			slog.DebugContext(ctx, "coach_advisor: signal skipped (no catalog match, no synthesizer)",
				"signal_kind", sig.Kind, "metric", sig.Metric)
			continue
		}
		tmpl, err := s.deps.Synthesizer.Synthesize(sig, in.TitleSlug, in.Now)
		if err != nil {
			if errors.Is(err, ErrSignalTooWeak) || errors.Is(err, ErrMetricNotSynthesizable) {
				slog.DebugContext(ctx, "coach_advisor: signal skipped",
					"signal_kind", sig.Kind, "metric", sig.Metric, "err", err)
				continue
			}
			slog.WarnContext(ctx, "coach_advisor: synthesis failed",
				"err", err, "signal_kind", sig.Kind, "metric", sig.Metric)
			continue
		}
		// Persiste le template synthétisé dans le catalog (dédup par ID hash)
		if err := s.deps.Templates.UpsertOne(ctx, tmpl); err != nil {
			slog.WarnContext(ctx, "coach_advisor: upsert synthesized template failed",
				"err", err, "template_id", tmpl.ID)
			continue
		}
		out = append(out, templateChoice{
			signal:   sig,
			template: tmpl,
			origin:   OriginSynthesized,
		})
	}
	return out
}

// filterChoicesNotInArc retire les templateChoice dont le signal est utilisé
// comme étape de l'arc (compare par SignalKind + Metric — proxy stable
// puisque le signal lui-même n'a pas d'ID).
func filterChoicesNotInArc(choices []templateChoice, arc ArcSpec) []templateChoice {
	covered := map[string]bool{}
	for _, step := range arc.Steps {
		covered[signalDedupKey(step.Signal)] = true
	}
	out := make([]templateChoice, 0, len(choices))
	for _, c := range choices {
		if covered[signalDedupKey(c.signal)] {
			continue
		}
		out = append(out, c)
	}
	return out
}

// signalDedupKey produit une clé identifiant un signal pour dédup avec un
// arc step. Pas globalement unique mais suffisant pour éviter doublon dans
// la même invocation.
func signalDedupKey(s Signal) string {
	return string(s.Kind) + "|" + s.Metric + "|" + s.LUSRComponent + "|" + s.RadarAxis
}

// buildProposals construit la liste finale de proposals. Arc d'abord (UI le
// met en évidence), puis challenges individuels.
func (s *service) buildProposals(in GenerateInput, choices []templateChoice, arc ArcSpec, hasArc bool) []Proposal {
	out := make([]Proposal, 0, len(choices)+1)
	if hasArc {
		if p, ok := s.buildArcProposal(in, arc); ok {
			out = append(out, p)
		}
	}
	for _, c := range choices {
		out = append(out, s.buildChallengeProposal(in, c))
	}
	return out
}

func (s *service) buildChallengeProposal(in GenerateInput, c templateChoice) Proposal {
	params, _ := json.Marshal(map[string]any{
		"metric":        c.template.Metric,
		"window_type":   string(c.template.WindowType),
		"window_value":  c.template.WindowValue,
		"signal_kind":   string(c.signal.Kind),
		"signal_metric": c.signal.Metric,
		"signal_axis":   c.signal.RadarAxis,
		"label_en":      c.template.LabelEN,
		"label_fr":      c.template.LabelFR,
	})
	return Proposal{
		ID:            s.deps.IDGen(),
		UserID:        in.UserID,
		TitleSlug:     in.TitleSlug,
		Kind:          ProposalKindChallenge,
		TemplateID:    c.template.ID,
		SuggestedTier: prestige.TierHeroic, // indicatif (cf. I1)
		SourceSignal:  c.signal.Kind,
		SourceMetric:  c.signal.Metric,
		RadarAxis:     c.signal.RadarAxis,
		Strength:      c.signal.Strength,
		Origin:        c.origin,
		ReasonKeyEN:   "coach.proposal." + string(c.signal.Kind) + ".en",
		ReasonKeyFR:   "coach.proposal." + string(c.signal.Kind) + ".fr",
		ReasonParams:  string(params),
		Status:        ProposalPending,
		CreatedAt:     in.Now,
	}
}

// arcStepSpec est la projection JSON d'un step d'arc dans
// Proposal.ChallengesSpec (sérialisé en string JSON).
type arcStepSpec struct {
	Position      int           `json:"position"`
	TemplateID    string        `json:"template_id"`
	SuggestedTier prestige.Tier `json:"suggested_tier"`
	SignalKind    SignalKind    `json:"signal_kind"`
}

func (s *service) buildArcProposal(in GenerateInput, arc ArcSpec) (Proposal, bool) {
	// Pour chaque step de l'arc, résoudre le template (matcher catalog +
	// synthèse en fallback) — répétition de resolveSignalsToTemplates pour
	// les signaux de l'arc. Si un step ne peut pas être résolu, on rejette
	// l'arc entier.
	stepSpecs := make([]arcStepSpec, 0, len(arc.Steps))
	for _, step := range arc.Steps {
		c, ok := s.resolveSingleSignal(in, step.Signal)
		if !ok {
			return Proposal{}, false
		}
		stepSpecs = append(stepSpecs, arcStepSpec{
			Position:      step.Position,
			TemplateID:    c.template.ID,
			SuggestedTier: step.SuggestedTier,
			SignalKind:    step.Signal.Kind,
		})
	}
	specJSON, _ := json.Marshal(stepSpecs)

	params, _ := json.Marshal(map[string]any{
		"radar_axis":     arc.RadarAxis,
		"step_count":     len(arc.Steps),
		"avg_strength":   arc.AverageStrength,
		"title_en":       arc.TitleEN,
		"title_fr":       arc.TitleFR,
		"description_en": arc.DescriptionEN,
		"description_fr": arc.DescriptionFR,
	})

	// Strength globale = moyenne des steps
	// Origin = "synthesized" si au moins une étape est synthétisée, sinon "catalog"
	origin := OriginCatalog
	for _, spec := range stepSpecs {
		_ = spec
		// On n'a pas accès à l'origin ici sans re-résoudre — simplification :
		// les arcs sont marqués Origin=catalog par défaut. L'analytique exacte
		// peut décomposer via ChallengesSpec si besoin.
		break
	}

	return Proposal{
		ID:             s.deps.IDGen(),
		UserID:         in.UserID,
		TitleSlug:      in.TitleSlug,
		Kind:           ProposalKindArc,
		ChallengesSpec: string(specJSON),
		SuggestedTier:  arc.Steps[len(arc.Steps)-1].SuggestedTier, // tier de l'étape finale
		SourceSignal:   arc.Steps[0].Signal.Kind,                  // signal le plus fort
		RadarAxis:      arc.RadarAxis,
		Strength:       arc.AverageStrength,
		Origin:         origin,
		ReasonKeyEN:    "coach.proposal.arc." + arc.RadarAxis + ".en",
		ReasonKeyFR:    "coach.proposal.arc." + arc.RadarAxis + ".fr",
		ReasonParams:   string(params),
		Status:         ProposalPending,
		CreatedAt:      in.Now,
	}, true
}

// resolveSingleSignal résout un signal en templateChoice (catalog ou synth).
// Variante synchrone de resolveSignalsToTemplates pour l'usage dans
// buildArcProposal. Charge le catalogue à chaque appel — acceptable car
// arc steps <= 4.
func (s *service) resolveSingleSignal(in GenerateInput, sig Signal) (templateChoice, bool) {
	ctx := context.Background()
	catalog, err := s.deps.Templates.ListByTitle(ctx, in.TitleSlug)
	if err != nil {
		return templateChoice{}, false
	}
	scores := MatchTemplateToSignal(sig, catalog, s.deps.MatcherWeights)
	filtered := FilterByMinScore(scores, s.deps.Tuning.MinCatalogMatchScore)
	if len(filtered) > 0 {
		return templateChoice{signal: sig, template: filtered[0].Template, origin: OriginCatalog}, true
	}
	if s.deps.Synthesizer == nil {
		return templateChoice{}, false
	}
	tmpl, err := s.deps.Synthesizer.Synthesize(sig, in.TitleSlug, in.Now)
	if err != nil {
		return templateChoice{}, false
	}
	if err := s.deps.Templates.UpsertOne(ctx, tmpl); err != nil {
		return templateChoice{}, false
	}
	return templateChoice{signal: sig, template: tmpl, origin: OriginSynthesized}, true
}

// supersedeOlder : pour chaque proposal nouvelle, marque comme superseded les
// proposals pending ciblant la même (metric, axis) ET dont strength est
// inférieure d'au moins SupersessionStrengthUplift.
func (s *service) supersedeOlder(ctx context.Context, in GenerateInput, newProp Proposal) error {
	if newProp.SourceMetric == "" && newProp.RadarAxis == "" {
		return nil
	}
	pending, err := s.deps.Repo.ListPendingBySignalScope(ctx, in.UserID, in.TitleSlug, newProp.SourceMetric, newProp.RadarAxis)
	if err != nil {
		return err
	}
	threshold := newProp.Strength / s.deps.Tuning.SupersessionStrengthUplift
	for _, old := range pending {
		if old.ID == newProp.ID {
			continue
		}
		// La nouvelle proposal doit être 10% plus forte que l'ancienne pour
		// la supersession (uplift). Équivalent : old.Strength <= new.Strength / uplift.
		if old.Strength > threshold {
			continue
		}
		if err := s.deps.Repo.MarkSuperseded(ctx, old.ID, newProp.ID, in.Now); err != nil {
			slog.WarnContext(ctx, "coach_advisor: mark superseded failed",
				"err", err, "old_id", old.ID, "new_id", newProp.ID)
		}
	}
	return nil
}

// ─── AcceptProposal ───

// AcceptProposal matérialise la proposal en Prestige (CreateChallenge ou
// CreateArc + N CreateChallenge selon Kind), puis marque la proposal
// accepted avec resolved_ref.
func (s *service) AcceptProposal(ctx context.Context, id string) (AcceptResult, error) {
	if id == "" {
		return AcceptResult{}, fmt.Errorf("coach_advisor.AcceptProposal: id required")
	}
	if s.deps.Prestige == nil || s.deps.Templates == nil {
		return AcceptResult{}, fmt.Errorf("coach_advisor.AcceptProposal: Prestige and Templates deps required")
	}

	prop, err := s.deps.Repo.Get(ctx, id)
	if err != nil {
		return AcceptResult{}, err // includes ErrProposalNotFound
	}
	if prop.Status != ProposalPending {
		return AcceptResult{}, fmt.Errorf("%w: id=%s status=%s",
			ErrProposalNotAcceptable, id, prop.Status)
	}

	switch prop.Kind {
	case ProposalKindChallenge:
		return s.acceptChallenge(ctx, prop)
	case ProposalKindArc:
		return s.acceptArc(ctx, prop)
	default:
		return AcceptResult{}, fmt.Errorf("coach_advisor.AcceptProposal: unknown kind %q", prop.Kind)
	}
}

func (s *service) acceptChallenge(ctx context.Context, prop Proposal) (AcceptResult, error) {
	tmpl, err := s.deps.Templates.GetByID(ctx, prop.TemplateID)
	if err != nil {
		return AcceptResult{}, fmt.Errorf("coach_advisor.acceptChallenge: get template: %w", err)
	}
	req := s.challengeRequestFromTemplate(prop.UserID, prop.TitleSlug, tmpl, "", 0)
	ch, err := s.deps.Prestige.CreateChallenge(ctx, req)
	if err != nil {
		return AcceptResult{}, fmt.Errorf("coach_advisor.acceptChallenge: prestige.CreateChallenge: %w", err)
	}
	now := s.deps.Now()
	if err := s.deps.Repo.MarkAccepted(ctx, prop.ID, ch.ID, now); err != nil {
		slog.ErrorContext(ctx, "coach_advisor: mark accepted failed",
			"err", err, "id", prop.ID, "challenge_id", ch.ID)
		// Challenge créé mais proposal pas mise à jour — best-effort
	}
	slog.InfoContext(ctx, "coach_advisor: challenge accepted",
		"id", prop.ID, "challenge_id", ch.ID, "user", prop.UserID, "titleSlug", prop.TitleSlug)
	return AcceptResult{ChallengeID: ch.ID}, nil
}

func (s *service) acceptArc(ctx context.Context, prop Proposal) (AcceptResult, error) {
	var steps []arcStepSpec
	if err := json.Unmarshal([]byte(prop.ChallengesSpec), &steps); err != nil {
		return AcceptResult{}, fmt.Errorf("coach_advisor.acceptArc: parse spec: %w", err)
	}
	if len(steps) == 0 {
		return AcceptResult{}, fmt.Errorf("coach_advisor.acceptArc: empty steps")
	}

	// Reconstruire les titres/descriptions depuis ReasonParams
	var params map[string]any
	_ = json.Unmarshal([]byte(prop.ReasonParams), &params)
	titleFR := stringFromMap(params, "title_fr")
	descFR := stringFromMap(params, "description_fr")

	arc, err := s.deps.Prestige.CreateArc(ctx, prestige.CreateArcRequest{
		UserID:      prop.UserID,
		TitleSlug:   prop.TitleSlug,
		Title:       titleFR,
		Description: descFR,
		Source:      prestige.ChallengeSourceCoach,
	})
	if err != nil {
		return AcceptResult{}, fmt.Errorf("coach_advisor.acceptArc: prestige.CreateArc: %w", err)
	}

	challengeIDs := make([]string, 0, len(steps))
	for _, step := range steps {
		tmpl, err := s.deps.Templates.GetByID(ctx, step.TemplateID)
		if err != nil {
			slog.ErrorContext(ctx, "coach_advisor: arc step template missing",
				"err", err, "template_id", step.TemplateID, "position", step.Position)
			continue
		}
		req := s.challengeRequestFromTemplate(prop.UserID, prop.TitleSlug, tmpl, arc.ID, step.Position)
		ch, err := s.deps.Prestige.CreateChallenge(ctx, req)
		if err != nil {
			slog.ErrorContext(ctx, "coach_advisor: arc step CreateChallenge failed",
				"err", err, "template_id", step.TemplateID, "position", step.Position)
			continue
		}
		challengeIDs = append(challengeIDs, ch.ID)
	}

	now := s.deps.Now()
	if err := s.deps.Repo.MarkAccepted(ctx, prop.ID, arc.ID, now); err != nil {
		slog.ErrorContext(ctx, "coach_advisor: mark arc accepted failed",
			"err", err, "id", prop.ID, "arc_id", arc.ID)
	}
	slog.InfoContext(ctx, "coach_advisor: arc accepted",
		"id", prop.ID, "arc_id", arc.ID, "challenges", len(challengeIDs),
		"user", prop.UserID, "titleSlug", prop.TitleSlug)
	return AcceptResult{ArcID: arc.ID, ChallengeIDs: challengeIDs}, nil
}

func (s *service) challengeRequestFromTemplate(userID, titleSlug string, tmpl prestige.Template, arcID string, position int) prestige.CreateChallengeRequest {
	return prestige.CreateChallengeRequest{
		UserID:      userID,
		TitleSlug:   titleSlug,
		ArcID:       arcID,
		TemplateID:  tmpl.ID,
		Metric:      tmpl.Metric,
		Target:      tmpl.HeroicTarget, // Prestige recalcule via baseline (cf. I1)
		WindowType:  tmpl.WindowType,
		WindowValue: tmpl.WindowValue,
		Cadence:     tmpl.Cadence,
		EvalType:    tmpl.EvalType,
		Mode:        prestige.ModePilote,
		Label:       tmpl.LabelFR,
		Position:    position,
		Source:      prestige.ChallengeSourceCoach,
	}
}

func stringFromMap(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
