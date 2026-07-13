// Package domain — types du bandeau de briefing de l'Explorer (mode Matchs).
//
// Le bandeau est une lecture compacte du RÉSULTAT DE RECHERCHE (sous-ensemble
// filtré) : KPIs agrégés, comparaison à la baseline personnelle (historique
// complet du titre), frise des résultats, et modules conditionnels (notes par
// dimension, tendance, classé) qui s'activent selon les filtres, les
// capabilities du titre et la taille d'échantillon. Ce N'EST PAS une page
// Synthèse (cf. PLAN_EXPLORER_BRIEFING_CARDS_2026-07.md).
//
// Tous les blocs (hors socle KPIs/frise/période) sont optionnels : un bloc nil
// = module non émis (dégradation par omission, jamais de N/A affiché). Les
// unités suivent ADR 0006 : WinRate en ratio 0..1 (le front formate en %).
package domain

import "time"

// ExplorerBriefing est le bloc de briefing servi au-dessus du tableau Explorer.
// Émis uniquement quand la requête le demande (include_briefing=true) et que le
// sous-ensemble filtré est non vide. Les modules conditionnels sont nil quand
// leur seuil d'échantillon / capability n'est pas atteint.
type ExplorerBriefing struct {
	// KPIs : indicateurs personnels agrégés sur le sous-ensemble filtré.
	KPIs *KPIStats `json:"kpis,omitempty"`
	// LowSample : vrai quand le sous-ensemble est sous MinBriefingModulesMatches.
	// Dans ce cas seuls KPIs + frise + période sont émis (les autres blocs nil).
	LowSample bool `json:"low_sample,omitempty"`
	// PeriodStart/PeriodEnd : bornes temporelles (min/max start_time) du scope.
	PeriodStart *time.Time `json:"period_start,omitempty"`
	PeriodEnd   *time.Time `json:"period_end,omitempty"`
	// OutcomeSequence : frise des derniers résultats du scope (cap
	// MaxOutcomeSequencePoints), ordre chronologique ascendant.
	OutcomeSequence []ExplorerBriefingOutcome `json:"outcome_sequence,omitempty"`
	// Baseline : comparaison à l'historique complet (post-exclusions). Nil si
	// low sample.
	Baseline *ExplorerBriefingBaseline `json:"baseline,omitempty"`
	// Dimensions : notes par dimension libre (carte/mode/playlist). Vide/nil si
	// low sample ou aucune dimension libre.
	Dimensions []ExplorerBriefingDimension `json:"dimensions,omitempty"`
	// Trend : tendance temporelle bucketée. Nil si seuils non atteints.
	Trend *ExplorerBriefingTrend `json:"trend,omitempty"`
	// Ranked : module classé (delta rating cumulé + attendu vs réel). Nil si le
	// titre n'expose pas la capability ranked ou si aucun match rangé dans le scope.
	Ranked *ExplorerBriefingRanked `json:"ranked,omitempty"`
}

// ExplorerBriefingOutcome est un point de la frise des résultats.
type ExplorerBriefingOutcome struct {
	MatchID string `json:"match_id"`
	// OutcomeCode : 1=Égalité, 2=Victoire, 3=Défaite, 4=Abandon (convention produit).
	OutcomeCode int       `json:"outcome_code"`
	StartTime   time.Time `json:"start_time"`
}

// ExplorerBriefingBaseline compare le scope à la baseline personnelle
// (historique complet du titre, post-exclusions manuelles, AVANT filtres de
// recherche). Les champs WinRate/KDA/AvgPerf portent les valeurs de LA BASELINE ;
// les Delta* sont signés (scope − baseline).
type ExplorerBriefingBaseline struct {
	Matches      int      `json:"matches"`              // nombre de matchs de la baseline
	WinRate      float64  `json:"win_rate"`             // ratio 0..1 (baseline)
	KDA          float64  `json:"kda"`                  // KDA agrégat ADR 0006 (baseline)
	AvgPerf      *float64 `json:"avg_perf,omitempty"`   // perf moyenne 0..100 (baseline)
	DeltaWinRate float64  `json:"delta_win_rate"`       // scope − baseline (points de ratio 0..1)
	DeltaKDA     float64  `json:"delta_kda"`            // scope − baseline
	DeltaPerf    *float64 `json:"delta_perf,omitempty"` // scope − baseline (nil si perf absente)
}

// ExplorerBriefingDimension porte les entrées d'une dimension libre.
type ExplorerBriefingDimension struct {
	// Dimension : "map" | "mode" | "playlist".
	Dimension string                           `json:"dimension"`
	Entries   []ExplorerBriefingDimensionEntry `json:"entries"`
}

// ExplorerBriefingDimensionEntry est un groupe (carte/mode/playlist) qualifié.
type ExplorerBriefingDimensionEntry struct {
	Label        string   `json:"label"`
	Matches      int      `json:"matches"`
	WinRate      float64  `json:"win_rate"`       // ratio 0..1 (scope)
	DeltaWinRate float64  `json:"delta_win_rate"` // scope − baseline pour ce groupe
	AvgPerf      *float64 `json:"avg_perf,omitempty"`
	// NoteTier : palier de performance 1..5 (1=Excellent). Nil si le groupe est
	// sous MinDimensionGroupMatches (pas de note fiable).
	NoteTier *int `json:"note_tier,omitempty"`
}

// ExplorerBriefingTrend porte la série temporelle bucketée du scope.
type ExplorerBriefingTrend struct {
	// Granularity : "1d" | "1w" | "1m" (largeur de bucket résolue serveur).
	Granularity string                       `json:"granularity"`
	Points      []ExplorerBriefingTrendPoint `json:"points"`
}

// ExplorerBriefingTrendPoint est un bucket de la tendance.
type ExplorerBriefingTrendPoint struct {
	BucketStart time.Time `json:"bucket_start"`
	Matches     int       `json:"matches"`
	WinRate     float64   `json:"win_rate"` // ratio 0..1
	AvgPerf     *float64  `json:"avg_perf,omitempty"`
}

// ExplorerBriefingRanked est le module classé (delta rating + attendu vs réel).
type ExplorerBriefingRanked struct {
	// RatingKind : "csr" | "lusr" (type majoritaire du scope).
	RatingKind string `json:"rating_kind"`
	// DeltaSum : somme signée des deltas de rating du scope.
	DeltaSum float64 `json:"delta_sum"`
	// ExpectedWinRate : moyenne des expected_win_prob disponibles (ratio 0..1).
	// Nil si aucun match du scope ne porte de prédiction.
	ExpectedWinRate *float64 `json:"expected_win_rate,omitempty"`
	// ActualWinRate : taux de victoire réel du scope (ratio 0..1).
	ActualWinRate float64 `json:"actual_win_rate"`
	// MatchesWithPrediction : nombre de matchs du scope portant un expected_win_prob.
	MatchesWithPrediction int `json:"matches_with_prediction"`
}
