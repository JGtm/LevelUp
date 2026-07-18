package prestigetuning

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// render.go — sérialisation du rapport en texte lisible (FR) ou JSON structuré.

// RenderJSON sérialise le rapport en JSON indenté.
func RenderJSON(rep Report) ([]byte, error) {
	return json.MarshalIndent(rep, "", "  ")
}

// RenderText produit un rapport texte lisible en français.
func RenderText(rep Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "== Analyse de tuning — grammaire de synthèse coach ==\n")
	fmt.Fprintf(&b, "Généré        : %s\n", rep.GeneratedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(&b, "Titre         : %s\n", rep.TitleSlug)
	fmt.Fprintf(&b, "Portée joueurs: %s (%d DB scannée(s))\n", rep.PlayerScope, rep.PlayersScanned)
	fmt.Fprintf(&b, "Seuils        : complétion < %.0f%% sur >= %d défis %q acceptés\n",
		rep.Thresholds.MinCompletionRate*100, rep.Thresholds.MinSample, rep.Thresholds.Source)
	fmt.Fprintf(&b, "Événements    : %d (jointés à une métrique)\n\n", rep.TotalEvents)

	renderAcceptance(&b, rep)
	renderMetrics(&b, rep)
	renderOrphans(&b, rep)
	renderSummary(&b, rep)
	return b.String()
}

func renderAcceptance(b *strings.Builder, rep Report) {
	fmt.Fprintf(b, "-- Acceptation par origine (toute télémétrie) --\n")
	if len(rep.SourceAcceptance) == 0 {
		fmt.Fprintf(b, "  (aucune donnée)\n\n")
		return
	}
	for _, s := range rep.SourceAcceptance {
		fmt.Fprintf(b, "  %-12s created=%-5d rejected=%-5d acceptance=%s\n",
			s.Source, s.Created, s.Rejected, pct(s.AcceptanceRate))
	}
	fmt.Fprintln(b)
}

func renderMetrics(b *strings.Builder, rep Report) {
	fmt.Fprintf(b, "-- Métriques de grammaire (source analysée : %q) --\n", rep.Thresholds.Source)
	if len(rep.Metrics) == 0 {
		fmt.Fprintf(b, "  (aucune métrique de grammaire)\n\n")
		return
	}
	for _, m := range rep.Metrics {
		fmt.Fprintf(b, "  [%s] %s\n", statusTag(m.Status), m.Metric)
		fmt.Fprintf(b, "      échantillon=%d complétion=%s (complétés=%d expirés=%d abandonnés=%d)\n",
			m.Sample, pct(m.CompletionRate), m.Completed, m.Expired, m.Abandoned)
		fmt.Fprintf(b, "      %s\n", m.Message)
		renderWindows(b, m)
	}
	fmt.Fprintln(b)
}

func renderWindows(b *strings.Builder, m MetricRecommendation) {
	if len(m.Windows) == 0 {
		return
	}
	for _, w := range m.Windows {
		flag := ""
		if !w.InGrammar {
			flag = " [hors grammaire]"
		}
		fmt.Fprintf(b, "        · %-22s n=%-4d complétion=%s%s\n",
			w.Window, w.Created, pct(w.CompletionRate), flag)
	}
}

func renderOrphans(b *strings.Builder, rep Report) {
	if len(rep.Orphans) == 0 {
		return
	}
	fmt.Fprintf(b, "-- Métriques orphelines (télémétrie sans entrée de grammaire) --\n")
	for _, o := range rep.Orphans {
		fmt.Fprintf(b, "  [orphelin] %s (échantillon=%d)\n      %s\n", o.Metric, o.Sample, o.Message)
	}
	fmt.Fprintln(b)
}

func renderSummary(b *strings.Builder, rep Report) {
	var adjust []string
	for _, m := range rep.Metrics {
		if m.Status == StatusRecommendAdjust {
			adjust = append(adjust, m.Metric)
		}
	}
	sort.Strings(adjust)
	fmt.Fprintf(b, "-- Synthèse --\n")
	if len(adjust) == 0 {
		fmt.Fprintf(b, "  Aucun ajustement recommandé (échantillon insuffisant ou complétion saine).\n")
		return
	}
	fmt.Fprintf(b, "  %d métrique(s) à ajuster manuellement dans synthesis_grammar.toml : %s\n",
		len(adjust), strings.Join(adjust, ", "))
}

// pct formate un taux 0..1 en pourcentage, ou "n/a" pour la sentinelle -1.
func pct(r float64) string {
	if r < 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.0f%%", r*100)
}

func statusTag(s Status) string {
	switch s {
	case StatusRecommendAdjust:
		return "AJUSTER"
	case StatusHealthy:
		return "OK"
	default:
		return "INSUFFISANT"
	}
}
