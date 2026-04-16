// Package chart — types de données pour les graphiques Halo Infinite.
//
// Toutes les structures sont renderer-agnostic (pas de dépendance Plotly).
// Le frontend consume ces types et assemble les graphiques côté client.
//
// Port Go des types de src/visualization/ et domain/chart/ Python.
package chart

import "time"

// ---------------------------------------------------------------------------
// Types de base
// ---------------------------------------------------------------------------

// DataPoint est un point de données générique (x numérique, y numérique).
type DataPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// LabeledValue associe un label à une valeur numérique.
type LabeledValue struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// MatchSeries représente une série temporelle indexée par match.
type MatchSeries struct {
	MatchID   string    `json:"match_id"`
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// SingleSeriesChartData encapsule une série avec son label.
type SingleSeriesChartData struct {
	Label  string        `json:"label"`
	Series []MatchSeries `json:"series"`
}

// MultiSeriesChartData regroupe plusieurs séries nommées.
type MultiSeriesChartData struct {
	Labels []string      `json:"labels"`
	Series []NamedSeries `json:"series"`
}

// NamedSeries est une série nommée.
type NamedSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
}

// ---------------------------------------------------------------------------
// Thème Halo
// ---------------------------------------------------------------------------

// HaloColors définit la palette d'outcomes Halo Infinite.
var HaloColors = struct {
	Win  string
	Loss string
	Tie  string
	DNF  string
	Perf struct{ High, Mid, Low, Bad string }
}{
	Win:  "#22c55e",
	Loss: "#ef4444",
	Tie:  "#8b5cf6",
	DNF:  "#8b5cf6",
	Perf: struct{ High, Mid, Low, Bad string }{
		High: "#22c55e",
		Mid:  "#3b82f6",
		Low:  "#f59e0b",
		Bad:  "#ef4444",
	},
}

// OkabeIto est la palette accessible Okabe-Ito (daltonisme-safe).
var OkabeIto = []string{
	"#E69F00", "#56B4E9", "#009E73", "#F0E442",
	"#0072B2", "#D55E00", "#CC79A7", "#000000",
}

// OutcomeColor retourne la couleur hex pour un code d'outcome Halo.
func OutcomeColor(code int) string {
	switch code {
	case 2:
		return HaloColors.Win
	case 3:
		return HaloColors.Loss
	case 1, 4:
		return HaloColors.Tie
	default:
		return "#94a3b8"
	}
}

// PerfColor retourne la couleur pour un score de performance.
func PerfColor(score float64) string {
	switch {
	case score >= 80:
		return HaloColors.Perf.High
	case score >= 60:
		return HaloColors.Perf.Mid
	case score >= 40:
		return HaloColors.Perf.Low
	default:
		return HaloColors.Perf.Bad
	}
}
