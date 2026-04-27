// Package domain — charts.go : types generiques de payload chart server-side.
//
// Conformement au PLAN_META_FOUNDATIONS_GO § 5.2.1 : toutes les donnees
// chart transitent en series brutes JSON, le frontend (ECharts) reconstruit
// l'option a partir de ces series via des wrappers specialises (<Lollipop>,
// <Heatmap2D>, etc.).
//
// Generic via Go 1.18+ : `ChartSeries[T]` permet aux wrappers Go de typer
// strictement leurs datapoints (ChartPoint2D, ChartPointHeatmap,
// ChartPointStacked).
package domain

// ChartSeries est une serie de datapoints typee, sortie d'un service.
//
//	Key        : identifiant stable (ex. "self.kills_per_match", "main.win_rate")
//	LabelKey   : cle i18n du libelle de la serie (resolu cote front)
//	ColorToken : token semantique (ex. "narrative.dominance.win.strong"),
//	             jamais un hex. Resolu via tokenCssVar() cote front.
//	Datapoints : payload type (ChartPoint2D, ChartPointStacked, etc.)
//	Meta       : indications d'axe / threshold / hover format
type ChartSeries[T any] struct {
	Key        string         `json:"key"`
	LabelKey   string         `json:"label_key,omitempty"`
	ColorToken *string        `json:"color_token,omitempty"`
	Datapoints []T            `json:"datapoints"`
	Meta       map[string]any `json:"meta,omitempty"`
}

// ChartPoint2D : datapoint cartesien generique.
//
//	X : peut etre time.Time (timeseries), float64 (numerique) ou string (categorie).
//	    Encode en JSON avec sa representation native.
//	Y : valeur numerique principale.
//	Label : override du tooltip / hover, optionnel.
type ChartPoint2D struct {
	X     any     `json:"x"`
	Y     float64 `json:"y"`
	Label *string `json:"label,omitempty"`
}

// ChartPointHeatmap : datapoint pour heatmap 2D (joueur x carte, match x phase).
//
//	X, Y   : axes (deja resolus en labels lisibles cote service).
//	Value  : valeur de la cellule (winrate, perf score, intensite normalisee).
//	Detail : payload du tooltip (KDA, outcome, etc.) consomme par le wrapper.
type ChartPointHeatmap struct {
	X      string         `json:"x"`
	Y      string         `json:"y"`
	Value  float64        `json:"value"`
	Detail map[string]any `json:"detail,omitempty"`
}

// ChartPointStacked : datapoint pour barres empilees (W/L/T/DNF, frags+morts+assists).
//
//	Category   : axe principal (ex. nom de carte, joueur, periode).
//	Components : map de sous-cles → valeur. Ex. {"win": 12, "loss": 5, "tie": 2}.
type ChartPointStacked struct {
	Category   string             `json:"category"`
	Components map[string]float64 `json:"components"`
}
