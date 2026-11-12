package profile

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/prestige"
	syncpkg "levelup/go-api/internal/sync"
)

// service.go — orchestre les lectures de port.ProfileStore pour bâtir un PlayerProfile.
//
// Un seul constructeur : NewService(store). Le hook post-sync (post_sync_progression.go) n'appelle que
// Load() (LUSR + tier + tendance) ; l'endpoint HTTP appelle BuildProfile(). Le magasin porte le joueur ;
// ses modèles de défis (référentiels) sont facultatifs, sans eux aucune suggestion n'est faite.

// Service expose les lectures du profil de progression.
type Service struct {
	store  port.ProfileStore
	awards *mappings.AwardMappingSet // optionnel — Section A1 radar fidèle (V2 §2)
}

// NewService construit le service sur un magasin de profil. store nil : Load() ne peut pas servir et
// BuildProfile() rend une erreur.
func NewService(store port.ProfileStore) *Service {
	return &Service{store: store}
}

// WithAwardMapping injecte un AwardMappingSet pour la Section A1 radar (V2 §2).
// Sans mapping, computeRadarAxes retombe sur l'heuristique V1
// (match_participants agrégats — Objective renvoie 0).
func (s *Service) WithAwardMapping(set *mappings.AwardMappingSet) *Service {
	s.awards = set
	return s
}

// Load construit le PlayerProfile minimal (V2-compat) avec LUSR + tier + MuTrend.
//
// `now` est le timestamp de référence (cutoff de la fenêtre). En prod = time.Now() ;
// en test = fixé pour déterminisme.
func (s *Service) Load(ctx context.Context, userID, titleSlug string, lowessWindowDays int, now time.Time) (*PlayerProfile, error) {
	profile := &PlayerProfile{
		UserID:    userID,
		TitleSlug: titleSlug,
		UpdatedAt: now,
	}

	lusr, err := s.loadLUSRSnapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("loadLUSRSnapshot: %w", err)
	}
	profile.LUSR = lusr

	if lusr.MatchesCount < MinMatchesForRating {
		slog.DebugContext(ctx, "profile: skip tier+trend, insufficient matches",
			"user_id", userID, "matches", lusr.MatchesCount, "min", MinMatchesForRating)
		return profile, nil
	}

	profile.Tier = TierFromMu(lusr.Mu)
	profile.NextTier = NextTierFromMu(lusr.Mu)

	muSeries, err := s.loadMuSeries(ctx, now.AddDate(0, 0, -lowessWindowDays), now)
	if err != nil {
		return nil, fmt.Errorf("loadMuSeries: %w", err)
	}
	if len(muSeries) >= 3 {
		profile.MuTrend = ComputeMuTrend(muSeries)
	}
	return profile, nil
}

// BuildProfile construit le PlayerProfile V1 complet (Sections A1/A2/B/C).
//
// Pré-requis : pdb non nil (NewServiceFromPlayerDB). Sinon retourne erreur.
// Si moins de MinMatchesForProfile matchs sont disponibles, HasEnoughData=false
// et les sections A2/B/C sont vides ou partielles ; A1 (radar) dégrade
// gracieusement (axes calculables uniquement).
func (s *Service) BuildProfile(ctx context.Context, userID, titleSlug string, lowessWindowDays int, now time.Time) (*PlayerProfile, error) {
	if s.store == nil {
		return nil, errors.New("BuildProfile: service constructed without PlayerDB")
	}
	profile := &PlayerProfile{
		UserID:    userID,
		TitleSlug: titleSlug,
		UpdatedAt: now,
	}

	matchesCount, err := s.countMatchesInWindow(ctx, userID, now.AddDate(0, 0, -lowessWindowDays), now)
	if err != nil {
		return nil, fmt.Errorf("countMatchesInWindow: %w", err)
	}
	profile.MatchesAnalyzed = matchesCount
	profile.HasEnoughData = matchesCount >= MinMatchesForProfile

	// ── Section B : LUSR + tier + tendance (toujours calculé, même partiel)
	if err := s.fillSectionB(ctx, profile, lowessWindowDays, now); err != nil {
		slog.WarnContext(ctx, "profile: section B partial", "err", err)
	}

	// ── Sparkline tendance LUSR 90 j (DEC-5/D2) — fenêtre FIXE, indépendante de
	// window_days. Best-effort : une erreur laisse SkillTrend vide (front masque).
	if pts, err := s.loadMuSeriesPoints(ctx, now.AddDate(0, 0, -SkillTrendWindowDays), now); err != nil {
		slog.WarnContext(ctx, "profile: skill trend partial", "err", err)
	} else {
		profile.SkillTrend = buildSkillTrend(pts)
	}

	// ── Section A1 : radar narrative 6 axes
	if err := s.aggregateNarrative(ctx, profile, userID, now.AddDate(0, 0, -lowessWindowDays), now); err != nil {
		slog.WarnContext(ctx, "profile: section A1 partial", "err", err)
	}

	// ── Section A2 : style + engagement (nécessite HasEnoughData)
	if profile.HasEnoughData {
		if err := s.computeStyleSignature(ctx, profile, userID, now.AddDate(0, 0, -lowessWindowDays), now); err != nil {
			slog.WarnContext(ctx, "profile: style signature partial", "err", err)
		}
		if err := s.computeEngagement(ctx, profile, userID, now.AddDate(0, 0, -lowessWindowDays), now); err != nil {
			slog.WarnContext(ctx, "profile: engagement partial", "err", err)
		}
	}

	// ── Section C : leviers + suggestions (nécessite Section B remplie)
	if profile.HasEnoughData && !profile.Tier.IsEmpty() {
		if err := s.computeLUSRComponents(ctx, profile, userID); err != nil {
			slog.WarnContext(ctx, "profile: LUSR components partial", "err", err)
		}
		profile.Leverages = identifyLeverages(profile.LUSRComponents)
		if err := s.selectSuggestedChallenges(ctx, profile, titleSlug); err != nil {
			slog.WarnContext(ctx, "profile: suggestions partial", "err", err)
		}
	}

	return profile, nil
}

// fillSectionB charge LUSR + tier + tendance + SkillRatingSnapshot.
func (s *Service) fillSectionB(ctx context.Context, profile *PlayerProfile, lowessWindowDays int, now time.Time) error {
	lusr, err := s.loadLUSRSnapshot(ctx)
	if err != nil {
		return err
	}
	profile.LUSR = lusr
	if lusr.MatchesCount < MinMatchesForRating {
		return nil
	}
	profile.Tier = TierFromMu(lusr.Mu)
	profile.NextTier = NextTierFromMu(lusr.Mu)
	profile.SkillRating = buildSkillRatingSnapshot(lusr, profile.Tier, profile.NextTier, i18n.FromContext(ctx))

	muSeries, err := s.loadMuSeries(ctx, now.AddDate(0, 0, -lowessWindowDays), now)
	if err != nil {
		return err
	}
	if len(muSeries) >= 3 {
		profile.MuTrend = ComputeMuTrend(muSeries)
	}
	return nil
}

// aggregateNarrative calcule les axes radar via narrative.ComputeParticipationProfile.
// Axes combat/survival/support/score/impact : heuristiques match_participants (V1)
// + enrichissement awards.toml (V2 §2). Axe objective : index par opportunité
// (computeObjectiveIndexAxis, match_objective_stats_latest) avec seuil spécifique
// ObjectiveIndexThreshold — RETIRÉ si la fenêtre n'a aucun match à objectif ou si
// le titre ne porte pas la capability match.objective.stats (plan
// PLAN_AXE_OBJECTIFS_INDEX).
func (s *Service) aggregateNarrative(ctx context.Context, profile *PlayerProfile, userID string, since, until time.Time) error {
	rawByAxis, err := s.computeRadarAxes(ctx, userID, since, until)
	if err != nil {
		return err
	}
	if len(rawByAxis) == 0 {
		return nil
	}
	objRaw, objN := s.computeObjectiveIndexAxis(ctx, userID, since, until)
	if objN > 0 {
		rawByAxis[narrative.AxisObjective] = objRaw
	} else {
		delete(rawByAxis, narrative.AxisObjective)
	}
	thresholds := narrative.DefaultThresholds("custom")
	thresholds.Objective = narrative.ObjectiveIndexThreshold
	scores := narrative.ComputeParticipationProfile(rawByAxis, thresholds)
	profile.RadarAxes = make([]ParticipationAxisValue, 0, len(scores))
	for _, sc := range scores {
		if sc.Axis == narrative.AxisObjective && objN == 0 {
			continue // aucun match à objectif sur la fenêtre → axe retiré
		}
		profile.RadarAxes = append(profile.RadarAxes, ParticipationAxisValue{
			Axis:  string(sc.Axis),
			Value: sc.Value,
			Raw:   sc.Raw,
		})
	}

	// Strengths : 3 axes avec Value le plus haut. ImprovementAreas : 3 plus bas.
	sortedAxes := append([]ParticipationAxisValue(nil), profile.RadarAxes...)
	sort.SliceStable(sortedAxes, func(i, j int) bool { return sortedAxes[i].Value > sortedAxes[j].Value })
	topN := 3
	if len(sortedAxes) < topN {
		topN = len(sortedAxes)
	}
	profile.Strengths = make([]RadarAxisInsight, 0, topN)
	for i := 0; i < topN; i++ {
		profile.Strengths = append(profile.Strengths, RadarAxisInsight{
			Axis:    sortedAxes[i].Axis,
			Value:   sortedAxes[i].Value,
			Message: "profile.strength." + sortedAxes[i].Axis,
		})
	}
	// Bottom 3 (ne pas répéter ceux du top si len < 6)
	if len(sortedAxes) >= 6 {
		profile.ImprovementAreas = make([]RadarAxisInsight, 0, 3)
		for i := len(sortedAxes) - 3; i < len(sortedAxes); i++ {
			profile.ImprovementAreas = append(profile.ImprovementAreas, RadarAxisInsight{
				Axis:    sortedAxes[i].Axis,
				Value:   sortedAxes[i].Value,
				Message: "profile.improvement." + sortedAxes[i].Axis,
			})
		}
	}

	// Dominant + secondary role : pour V1 dérivés du top axis. La logique fine
	// (clutch_finisher / silent_hero…) requiert highlight_events ; deferré.
	if len(sortedAxes) > 0 {
		profile.DominantRole = roleFromAxis(sortedAxes[0].Axis)
		if len(sortedAxes) > 1 {
			profile.SecondaryRole = roleFromAxis(sortedAxes[1].Axis)
		}
	}
	return nil
}

// computeStyleSignature calcule FK / FD (premiers frags et premières morts du match, cf. computeFKFD).
// Une erreur de lecture remonte à BuildProfile, qui la journalise ; la signature reste à zéro.
// Sans événement pour la fenêtre, 0/0.
func (s *Service) computeStyleSignature(ctx context.Context, profile *PlayerProfile, userID string, since, until time.Time) error {
	fk, fd, err := s.computeFKFD(ctx, userID, since, until)
	if err != nil {
		return err
	}
	ratio := 0.0
	if fd > 0 {
		ratio = float64(fk) / float64(fd)
	} else if fk > 0 {
		ratio = float64(fk)
	}
	profile.StyleSignature = StyleSignature{
		FirstKillCount:  fk,
		FirstDeathCount: fd,
		FKFDRatio:       ratio,
		StyleKey:        classifyStyle(fk, fd, ratio),
	}
	return nil
}

// computeEngagement remplit la section A2 engagement.
func (s *Service) computeEngagement(ctx context.Context, profile *PlayerProfile, userID string, since, until time.Time) error {
	snap, err := s.computeEngagementSimple(ctx, userID, since, until)
	if err != nil {
		return err
	}
	profile.EngagementSnap = snap
	return nil
}

// computeLUSRComponents calcule les 8 composantes avec current/top20/target.
//
// target_for_tier : composite cible (entrée du tier suivant) via
// analysis.RequiredCompositeForTier. La même cible s'applique aux 8
// composantes (le composite est leur moyenne pondérée — affichage compact).
func (s *Service) computeLUSRComponents(ctx context.Context, profile *PlayerProfile, userID string) error {
	rows, err := s.loadLUSRComponentsBreakdown(ctx)
	if err != nil {
		return err
	}
	target := requiredCompositeFor(profile.LUSR.Mu, profile.NextTier.LowerMu, profile.LUSR.Sigma)
	weights := compositeWeightsExternal()
	out := make([]LUSRComponentBreakdown, 0, len(weights))
	for _, name := range orderedComponentNames(weights) {
		row := rows[name]
		out = append(out, LUSRComponentBreakdown{
			Name:          name,
			Weight:        weights[name],
			CurrentAvg:    row.currentAvg,
			PersonalTop20: row.top20,
			TargetForTier: target,
			Trend:         row.trend,
		})
	}
	profile.LUSRComponents = out
	return nil
}

// identifyLeverages sélectionne les 2 composantes avec le plus fort levier =
// (1 - current) × weight. Ce sont les axes où le joueur a le plus à gagner.
func identifyLeverages(components []LUSRComponentBreakdown) []ProgressionLeverage {
	type leverage struct {
		Component string
		Value     float64
	}
	levers := make([]leverage, 0, len(components))
	for _, c := range components {
		v := (1.0 - c.CurrentAvg) * c.Weight
		if v < 0 {
			v = 0
		}
		levers = append(levers, leverage{Component: c.Name, Value: v})
	}
	sort.SliceStable(levers, func(i, j int) bool { return levers[i].Value > levers[j].Value })
	maxN := 2
	if len(levers) < maxN {
		maxN = len(levers)
	}
	out := make([]ProgressionLeverage, 0, maxN)
	for i := 0; i < maxN; i++ {
		out = append(out, ProgressionLeverage{
			Component:       levers[i].Component,
			LeverageValue:   levers[i].Value,
			NarrativeAxes:   narrativeAxesForComponent(levers[i].Component),
			CoachingMessage: "profile.coach." + levers[i].Component,
		})
	}
	return out
}

// selectSuggestedChallenges sélectionne 3 templates dont les lusr_components
// recoupent les leviers identifiés. Triés par : long-terme prioritaires.
func (s *Service) selectSuggestedChallenges(ctx context.Context, profile *PlayerProfile, titleSlug string) error {
	leverageSet := make(map[string]struct{}, len(profile.Leverages))
	for _, lv := range profile.Leverages {
		leverageSet[lv.Component] = struct{}{}
	}
	templates, err := s.templatesByLUSRComponents(ctx, titleSlug, leverageSet)
	if err != nil {
		return err
	}
	// Tri : IsLongTerm en premier, puis ID.
	sort.SliceStable(templates, func(i, j int) bool {
		if templates[i].IsLongTerm != templates[j].IsLongTerm {
			return templates[i].IsLongTerm
		}
		return templates[i].ID < templates[j].ID
	})
	maxN := 3
	if len(templates) < maxN {
		maxN = len(templates)
	}
	loc := i18n.FromContext(ctx)
	out := make([]SuggestedChallenge, 0, maxN)
	for i := 0; i < maxN; i++ {
		out = append(out, suggestedChallengeFromTemplate(templates[i], loc))
	}
	profile.SuggestedChallenges = out
	return nil
}

// templatesByLUSRComponents retourne les modèles du titre dont les lusr_components recoupent l'ensemble
// fourni (filtre côté Go : au moins une composante du modèle est dans l'ensemble des leviers). Sans levier,
// ou sans base de référentiels, la liste est vide et aucune lecture n'a lieu.
func (s *Service) templatesByLUSRComponents(ctx context.Context, titleSlug string, components map[string]struct{}) ([]prestige.Template, error) {
	if len(components) == 0 {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	all, available, err := s.store.SuggestionTemplates(ctx, titleSlug)
	if err != nil {
		return nil, fmt.Errorf("listTemplatesByLUSRComponents: %w", err)
	}
	if !available {
		return nil, nil
	}
	var out []prestige.Template
	for _, t := range all {
		for _, c := range t.LUSRComponents {
			if _, ok := components[c]; ok {
				out = append(out, t)
				break
			}
		}
	}
	return out, nil
}

// suggestedChallengeFromTemplate construit la suggestion d'un template. V2 §3 : hydrate le libellé et la
// description depuis le template, dans la langue `loc` de la requête (repli `en`, ADR 0040 D-3), pour
// que l'UI affiche un libellé humain sans round-trip au catalogue.
func suggestedChallengeFromTemplate(t prestige.Template, loc i18n.Locale) SuggestedChallenge {
	return SuggestedChallenge{
		TemplateID:       t.ID,
		TargetTier:       defaultTargetTierForTemplate(t),
		HistoricalStreak: 0, // V1 placeholder ; V2 lira l'historique de complétions
		IsArcStep:        false,
		Label:            t.Labels.Pick(loc),
		Description:      t.Descriptions.PickOptional(loc),
	}
}

// ─── Helpers ────────────────────────────────────────────────────────────────

// requiredCompositeFor wrappe analysis.RequiredCompositeForTier.
// Si tergetMu invalide ou égal à current, retourne 0 (pas d'objectif).
func requiredCompositeFor(currentMu, targetMu, sigma float64) float64 {
	if targetMu <= currentMu {
		return 0
	}
	if sigma <= 0 {
		sigma = 1.0
	}
	return analysis.RequiredCompositeForTier(currentMu, targetMu, sigma)
}

// narrativeAxesForComponent mappe une composante LUSR → axes narrative ciblés.
// Réf : analyse heuristique narrative ↔ composantes (cf. plan §4.3).
func narrativeAxesForComponent(comp string) []string {
	switch comp {
	case analysis.PerfMetricKillsVsExpected, "offensive_conversion":
		return []string{string(narrative.AxisCombat), string(narrative.AxisImpact)}
	case analysis.PerfMetricDeathsVsExpected, "defensive_resistance":
		return []string{string(narrative.AxisSurvival)}
	case "win_factor":
		return []string{string(narrative.AxisObjective)}
	case "damage_efficiency":
		return []string{string(narrative.AxisCombat), string(narrative.AxisSurvival)}
	case "accuracy_delta":
		return []string{string(narrative.AxisCombat)}
	case "medal_exploit":
		return []string{string(narrative.AxisImpact), string(narrative.AxisSupport)}
	}
	return nil
}

// buildSkillRatingSnapshot construit le SkillRatingSnapshot d'affichage UI. `TierName` est le nom du
// tier dans la langue `loc` (repli `en`), à défaut le nom canonique du tier.
func buildSkillRatingSnapshot(lusr LUSRState, tier, nextTier TierState, loc i18n.Locale) SkillRatingSnapshot {
	tierName := tier.Names.PickOptional(loc)
	if tierName == "" {
		tierName = tier.Name
	}
	out := SkillRatingSnapshot{
		TierName: tierName,
		SubTier:  tier.SubTier,
		Label:    tier.Label,
		Mu:       lusr.Mu,
		Sigma:    lusr.Sigma,
	}
	if !nextTier.IsEmpty() {
		out.NextTierLabel = nextTier.Label
		out.NextTierMu = nextTier.LowerMu
		out.GapToNext = nextTier.LowerMu - lusr.Mu
	}
	if tier.UpperMu > tier.LowerMu {
		ratio := (lusr.Mu - tier.LowerMu) / (tier.UpperMu - tier.LowerMu)
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		out.ProgressRatio = ratio
	}
	return out
}

// defaultTargetTierForTemplate choisit un palier cible raisonnable selon le
// niveau LUSR du joueur. Heuristique V1 : "normal" pour tous (validation
// utilisateur > complexité).
func defaultTargetTierForTemplate(_ prestige.Template) string {
	return string(prestige.TierNormal)
}

// compositeWeightsExternal retourne sync.CompositeWeights (re-export pour éviter
// l'import direct de internal/sync depuis les callers de profile).
func compositeWeightsExternal() map[string]float64 {
	out := make(map[string]float64, len(syncpkg.CompositeWeights))
	for k, v := range syncpkg.CompositeWeights {
		out[k] = v
	}
	return out
}

// orderedComponentNames retourne les noms des composantes triés par weight desc
// (les plus impactantes en premier).
func orderedComponentNames(weights map[string]float64) []string {
	names := make([]string, 0, len(weights))
	for k := range weights {
		names = append(names, k)
	}
	sort.SliceStable(names, func(i, j int) bool {
		if weights[names[i]] != weights[names[j]] {
			return weights[names[i]] > weights[names[j]]
		}
		return names[i] < names[j]
	})
	return names
}

// roleFromAxis dérive un dominant_role i18n-key approximatif depuis l'axe top.
// Volontairement simple en V1 — l'attribution fine via highlight_events est
// déjà disponible dans narrative.IdentifyImpactRoles mais hors-scope ici.
func roleFromAxis(axis string) string {
	switch axis {
	case string(narrative.AxisCombat):
		return "top_killer"
	case string(narrative.AxisSurvival):
		return "survivor"
	case string(narrative.AxisSupport):
		return "silent_hero"
	case string(narrative.AxisScore):
		return "scorer"
	case string(narrative.AxisObjective):
		return "objective_runner"
	case string(narrative.AxisImpact):
		return "first_blood"
	}
	return ""
}

// classifyStyle déduit un StyleKey de FK/FD ratio.
// Réf : plan §5.2 (style_signature).
func classifyStyle(fk, fd int, ratio float64) string {
	if fk == 0 && fd == 0 {
		return ""
	}
	if fk >= 5 && ratio >= 1.5 {
		return "opportunistic_finisher"
	}
	if fd >= 5 && ratio < 0.5 {
		return "overextended"
	}
	if fk >= 5 && fd >= 5 {
		return "hyper_engaged"
	}
	return "passive"
}

// ─── Existing helpers (V2 compat) ───────────────────────────────────────────

// loadLUSRSnapshot lit le dernier rating LUSR + count total.
func (s *Service) loadLUSRSnapshot(ctx context.Context) (LUSRState, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	l, err := s.store.LatestLUSR(ctx)
	return LUSRState{Mu: l.Mu, Sigma: l.Sigma, MatchesCount: l.MatchesCount, LastMatchAt: l.LastMatchAt}, err
}

// loadMuSeriesPoints retourne les ratings LUSR DATÉS (ASC) sur la fenêtre, pour dater chaque point de la
// sparkline (DEC-5/D2).
func (s *Service) loadMuSeriesPoints(ctx context.Context, since, until time.Time) ([]muPoint, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	pts, err := s.store.LUSRSeriesPoints(ctx, since, until)
	if err != nil {
		return nil, err
	}
	var out []muPoint
	for _, p := range pts {
		out = append(out, muPoint{at: p.At, value: p.Value})
	}
	return out, nil
}

// loadMuSeries retourne la série de μ ordonnée chronologiquement (ASC).
func (s *Service) loadMuSeries(ctx context.Context, since, until time.Time) ([]float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return s.store.LUSRSeries(ctx, since, until)
}
