// Package prestigetuning — analyseur de tuning de la grammaire de synthèse du
// coach_advisor (config/coach_advisor/synthesis_grammar.toml, ADR 0020/0028).
//
// Le package produit des RECOMMANDATIONS d'ajustement de la grammaire à partir
// de la télémétrie Prestige (table append-only prestige_telemetry, jointe à
// challenge pour la métrique/fenêtre). L'application reste MANUELLE : un humain
// lit le rapport et édite le TOML. Aucun mécanisme de PR automatique, aucun
// override runtime (cadrage superviseur).
//
// Découpage : la logique d'analyse (seuils, agrégation, génération de
// recommandations) est constituée de fonctions PURES sans I/O (analyze.go), la
// lecture SQL en LECTURE SEULE est isolée dans collect.go, le rendu dans
// render.go. Le cmd (cmd/prestige-tuning-analyze) n'est qu'un point d'entrée fin.
package prestigetuning

import "time"

// Status classe le verdict d'analyse d'une métrique de grammaire.
type Status string

const (
	// StatusInsufficientData : échantillon sous le minimum → aucune recommandation
	// (ne jamais recommander sur du bruit).
	StatusInsufficientData Status = "insufficient_data"
	// StatusHealthy : échantillon suffisant ET taux de complétion au-dessus du seuil.
	StatusHealthy Status = "healthy"
	// StatusRecommendAdjust : échantillon suffisant ET taux de complétion sous le
	// seuil → recommander de retirer la métrique ou de réduire ses fenêtres.
	StatusRecommendAdjust Status = "recommend_adjust"
)

// DefaultThresholds retourne les seuils par défaut de la règle d'analyse de
// référence (backlog) : complétion < 30 % sur >= 50 défis coach acceptés.
func DefaultThresholds() Thresholds {
	return Thresholds{
		MinCompletionRate: 0.30,
		MinSample:         50,
		Source:            "coach",
	}
}

// Thresholds paramètre la règle d'analyse.
type Thresholds struct {
	// MinCompletionRate : sous ce taux (0..1), une métrique à échantillon suffisant
	// déclenche une recommandation d'ajustement.
	MinCompletionRate float64 `json:"min_completion_rate"`
	// MinSample : nombre minimal de défis acceptés (created) requis pour statuer.
	// En dessous : "données insuffisantes".
	MinSample int `json:"min_sample"`
	// Source : origine des défis analysée pour les recommandations ("coach" par
	// défaut). Les autres origines sont agrégées à titre contextuel uniquement.
	Source string `json:"source"`
}

// MetricWindowCount est le compte d'événements pour un triplet
// (source, métrique, fenêtre), issu de la jointure prestige_telemetry ⋈ challenge.
//
// Note de conception : un défi auto-rejeté à la création (RejectTooEasy) n'est
// PAS persisté dans la table challenge (cf. prestige/service.go) — ses événements
// "rejected" n'ont donc pas de ligne challenge et sont exclus de cette jointure.
// L'acceptance par métrique n'est pas calculable ici ; elle est fournie au niveau
// source par SourceAcceptance (requête sans jointure). La complétion (created +
// completed, tous deux persistés) est, elle, pleinement attribuable à la métrique.
type MetricWindowCount struct {
	Source      string `json:"source"`
	Metric      string `json:"metric"`
	WindowType  string `json:"window_type"`
	WindowValue string `json:"window_value"`
	Created     int    `json:"created"`
	Completed   int    `json:"completed"`
	Expired     int    `json:"expired"`
	Abandoned   int    `json:"abandoned"`
}

// WindowSpec reconstruit la fenêtre au format grammaire ("type:value" ou "type"
// pour une fenêtre sans valeur, ex. "session").
func (c MetricWindowCount) WindowSpec() string {
	if c.WindowValue == "" {
		return c.WindowType
	}
	return c.WindowType + ":" + c.WindowValue
}

// SourceAcceptance est le compte d'acceptation d'une origine, agrégé sur TOUTE la
// télémétrie (sans jointure challenge), afin d'inclure les rejets non persistés.
type SourceAcceptance struct {
	Source   string `json:"source"`
	Created  int    `json:"created"`
	Rejected int    `json:"rejected"`
	// AcceptanceRate = created / (created + rejected). -1 si dénominateur nul.
	AcceptanceRate float64 `json:"acceptance_rate"`
}

// WindowBreakdown détaille la complétion d'une fenêtre d'une métrique — permet
// de cibler quelle(s) fenêtre(s) réduire dans une recommandation.
type WindowBreakdown struct {
	Window    string `json:"window"`
	Created   int    `json:"created"`
	Completed int    `json:"completed"`
	// CompletionRate = completed / created. -1 si created nul.
	CompletionRate float64 `json:"completion_rate"`
	// InGrammar : la fenêtre observée figure-t-elle dans la grammaire pour cette
	// métrique ? Une fenêtre télémétrie absente de la grammaire signale une dérive.
	InGrammar bool `json:"in_grammar"`
}

// MetricRecommendation est le verdict d'analyse d'une métrique.
type MetricRecommendation struct {
	Metric string `json:"metric"`
	// InGrammar : la métrique figure-t-elle dans synthesis_grammar.toml ?
	InGrammar bool `json:"in_grammar"`
	// GrammarWindows : fenêtres déclarées dans la grammaire pour cette métrique
	// (vide si orpheline). Relie la recommandation aux entrées réelles du TOML.
	GrammarWindows []string `json:"grammar_windows,omitempty"`
	// Sample = nombre de défis acceptés (created) pour la source analysée.
	Sample    int `json:"sample"`
	Completed int `json:"completed"`
	Expired   int `json:"expired"`
	Abandoned int `json:"abandoned"`
	// CompletionRate = completed / sample. -1 si sample nul.
	CompletionRate float64           `json:"completion_rate"`
	Status         Status            `json:"status"`
	Message        string            `json:"message"`
	Windows        []WindowBreakdown `json:"windows,omitempty"`
}

// Report est le rapport complet de l'analyseur.
type Report struct {
	GeneratedAt    time.Time  `json:"generated_at"`
	TitleSlug      string     `json:"title_slug"`
	Thresholds     Thresholds `json:"thresholds"`
	PlayersScanned int        `json:"players_scanned"`
	PlayerScope    string     `json:"player_scope"` // "all" ou un player_slug
	// TotalEvents : nombre d'événements de télémétrie considérés (jointure metric).
	TotalEvents int `json:"total_events"`
	// SourceAcceptance : contexte multi-origines (created/rejected par source).
	SourceAcceptance []SourceAcceptance `json:"source_acceptance"`
	// Metrics : verdict par métrique de grammaire ayant de la télémétrie sur la
	// source analysée (plus les métriques de grammaire sans donnée).
	Metrics []MetricRecommendation `json:"metrics"`
	// Orphans : métriques observées dans la télémétrie (source analysée) mais
	// ABSENTES de la grammaire — signalées comme orphelines (non actionnables sur
	// le TOML, mais révèlent une dérive de nommage ou un défi legacy).
	Orphans []MetricRecommendation `json:"orphans"`
}
