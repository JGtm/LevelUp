package prestigetuning

import (
	"fmt"
	"sort"
	"time"
)

// analyze.go — coeur PUR de l'analyseur : agrégation par métrique, application
// des seuils, génération des recommandations. Aucune I/O, entièrement testable.

// rate retourne num/den, ou -1 (sentinelle) si le dénominateur est nul —
// distingue "taux 0" (aucun succès) de "non calculable" (aucun défi).
func rate(num, den int) float64 {
	if den <= 0 {
		return -1
	}
	return float64(num) / float64(den)
}

// Analyze applique la règle de tuning et produit un rapport. Fonction pure :
//   - counts : comptes par (source, métrique, fenêtre) issus de la jointure.
//   - accept : acceptation par source (contexte, inclut rejets non persistés).
//   - grammar : vue de synthesis_grammar.toml.
//   - thr : seuils (complétion, échantillon, source analysée).
//
// now est injecté pour des tests déterministes.
func Analyze(
	counts []MetricWindowCount,
	accept []SourceAcceptance,
	grammar GrammarView,
	thr Thresholds,
	now time.Time,
) Report {
	rep := Report{
		GeneratedAt:      now,
		Thresholds:       thr,
		SourceAcceptance: sortedAcceptance(accept),
		Metrics:          []MetricRecommendation{},
		Orphans:          []MetricRecommendation{},
	}

	// Agrège par métrique pour la source analysée + total événements (toutes sources).
	byMetric := map[string][]MetricWindowCount{}
	for _, c := range counts {
		rep.TotalEvents += c.Created + c.Completed + c.Expired + c.Abandoned
		if c.Source == thr.Source {
			byMetric[c.Metric] = append(byMetric[c.Metric], c)
		}
	}

	seen := map[string]bool{}
	for _, metric := range sortedKeys(byMetric) {
		reco := analyzeMetric(metric, byMetric[metric], grammar, thr)
		seen[metric] = true
		if reco.InGrammar {
			rep.Metrics = append(rep.Metrics, reco)
		} else {
			rep.Orphans = append(rep.Orphans, reco)
		}
	}

	// Métriques de grammaire SANS aucune télémétrie sur la source analysée :
	// données insuffisantes (sample 0), listées pour visibilité.
	for _, metric := range grammar.Metrics() {
		if seen[metric] {
			continue
		}
		rep.Metrics = append(rep.Metrics, MetricRecommendation{
			Metric:         metric,
			InGrammar:      true,
			GrammarWindows: grammar.Windows(metric),
			Sample:         0,
			CompletionRate: -1,
			Status:         StatusInsufficientData,
			Message: fmt.Sprintf("Données insuffisantes : aucun défi %q accepté (échantillon 0 < %d requis).",
				thr.Source, thr.MinSample),
		})
	}
	sortRecos(rep.Metrics)
	sortRecos(rep.Orphans)
	return rep
}

// analyzeMetric agrège les fenêtres d'une métrique et statue.
func analyzeMetric(metric string, rows []MetricWindowCount, grammar GrammarView, thr Thresholds) MetricRecommendation {
	reco := MetricRecommendation{
		Metric:         metric,
		InGrammar:      grammar.HasMetric(metric),
		GrammarWindows: grammar.Windows(metric),
	}
	for _, r := range rows {
		reco.Sample += r.Created
		reco.Completed += r.Completed
		reco.Expired += r.Expired
		reco.Abandoned += r.Abandoned
		reco.Windows = append(reco.Windows, WindowBreakdown{
			Window:         r.WindowSpec(),
			Created:        r.Created,
			Completed:      r.Completed,
			CompletionRate: rate(r.Completed, r.Created),
			InGrammar:      grammar.HasWindow(metric, r.WindowSpec()),
		})
	}
	sort.Slice(reco.Windows, func(i, j int) bool { return reco.Windows[i].Window < reco.Windows[j].Window })
	reco.CompletionRate = rate(reco.Completed, reco.Sample)
	reco.Status, reco.Message = verdict(reco, thr)
	return reco
}

// verdict applique les seuils et compose le message de recommandation.
func verdict(reco MetricRecommendation, thr Thresholds) (Status, string) {
	if !reco.InGrammar {
		return StatusInsufficientData, fmt.Sprintf(
			"Métrique orpheline : %d défi(s) %q observé(s) dans la télémétrie mais métrique ABSENTE de synthesis_grammar.toml (dérive de nommage ou défi legacy). Non actionnable sur le TOML.",
			reco.Sample, thr.Source)
	}
	if reco.Sample < thr.MinSample {
		return StatusInsufficientData, fmt.Sprintf(
			"Données insuffisantes : %d défi(s) %q accepté(s) < %d requis. Aucune recommandation (pas de reco sur du bruit).",
			reco.Sample, thr.Source, thr.MinSample)
	}
	if reco.CompletionRate < thr.MinCompletionRate {
		return StatusRecommendAdjust, fmt.Sprintf(
			"Taux de complétion %.0f%% (< seuil %.0f%%) sur %d défis %q acceptés. Recommandation : retirer la métrique %q de synthesis_grammar.toml, OU réduire ses fenêtres (déclarées : %v).%s",
			reco.CompletionRate*100, thr.MinCompletionRate*100, reco.Sample, thr.Source,
			reco.Metric, reco.GrammarWindows, worstWindowsHint(reco.Windows, thr))
	}
	return StatusHealthy, fmt.Sprintf(
		"Taux de complétion %.0f%% (>= seuil %.0f%%) sur %d défis %q acceptés. Aucun ajustement nécessaire.",
		reco.CompletionRate*100, thr.MinCompletionRate*100, reco.Sample, thr.Source)
}

// worstWindowsHint pointe la/les fenêtre(s) à échantillon significatif dont la
// complétion est la plus basse — pistes concrètes de réduction.
func worstWindowsHint(windows []WindowBreakdown, thr Thresholds) string {
	var candidates []WindowBreakdown
	for _, w := range windows {
		if w.Created > 0 && w.CompletionRate >= 0 && w.CompletionRate < thr.MinCompletionRate {
			candidates = append(candidates, w)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].CompletionRate < candidates[j].CompletionRate })
	hint := " Fenêtres les plus faibles : "
	for i, w := range candidates {
		if i > 0 {
			hint += ", "
		}
		hint += fmt.Sprintf("%s=%.0f%% (n=%d)", w.Window, w.CompletionRate*100, w.Created)
	}
	return hint + "."
}

// sortedAcceptance renvoie l'acceptation par source triée par source.
func sortedAcceptance(in []SourceAcceptance) []SourceAcceptance {
	out := make([]SourceAcceptance, len(in))
	copy(out, in)
	sort.Slice(out, func(i, j int) bool { return out[i].Source < out[j].Source })
	if out == nil {
		return []SourceAcceptance{}
	}
	return out
}

func sortedKeys(m map[string][]MetricWindowCount) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortRecos(recos []MetricRecommendation) {
	sort.Slice(recos, func(i, j int) bool { return recos[i].Metric < recos[j].Metric })
}
