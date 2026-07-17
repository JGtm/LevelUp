package prestigetuning

import "sort"

// grammar.go — vue introspectable de la grammaire de synthèse, découplée du
// format TOML et du package coach_advisor. Permet d'écrire l'analyse (analyze.go)
// et ses tests sans dépendre du filesystem : les tests construisent une
// GrammarView directement, le cmd la dérive de coach_advisor.SynthesisGrammar.

// GrammarView expose les métriques et fenêtres autorisées de la grammaire, en
// lecture seule.
type GrammarView struct {
	// metricWindows : métrique → fenêtres au format "type:value" (ou "type").
	metricWindows map[string][]string
}

// NewGrammarView construit une vue à partir d'une map métrique → fenêtres.
// La map est copiée en défensive (les slices sont partagées, non mutées).
func NewGrammarView(metricWindows map[string][]string) GrammarView {
	cp := make(map[string][]string, len(metricWindows))
	for k, v := range metricWindows {
		cp[k] = v
	}
	return GrammarView{metricWindows: cp}
}

// HasMetric indique si la métrique figure dans la grammaire.
func (g GrammarView) HasMetric(metric string) bool {
	_, ok := g.metricWindows[metric]
	return ok
}

// Windows retourne les fenêtres autorisées pour une métrique (nil si absente).
func (g GrammarView) Windows(metric string) []string {
	return g.metricWindows[metric]
}

// HasWindow indique si une fenêtre (format "type:value" ou "type") est déclarée
// pour la métrique dans la grammaire.
func (g GrammarView) HasWindow(metric, window string) bool {
	for _, w := range g.metricWindows[metric] {
		if w == window {
			return true
		}
	}
	return false
}

// Metrics retourne les métriques de la grammaire, triées.
func (g GrammarView) Metrics() []string {
	out := make([]string, 0, len(g.metricWindows))
	for k := range g.metricWindows {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
