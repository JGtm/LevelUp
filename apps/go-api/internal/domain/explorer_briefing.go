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
	// Scope : agrégats du sous-ensemble filtré (socle) calculés sur les MÊMES raw
	// rows que le tableau — le compteur, le bilan et les indicateurs sont donc
	// cohérents avec « N matchs trouvés » et avec les modules baseline/dimensions.
	Scope *ExplorerBriefingScope `json:"scope,omitempty"`
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
	// Ranked : module « Classement » (progression de paliers PAR TYPE de rating).
	// Nil si le titre n'expose pas la capability ranked ou si aucun match rangé
	// dans le scope.
	Ranked *ExplorerBriefingRanked `json:"ranked,omitempty"`
}

// ExplorerBriefingScope porte les agrégats socle du sous-ensemble filtré,
// calculés sur les raw rows (même source que le tableau et les modules). Unités
// ADR 0006 : WinRate en ratio 0..1, KDA = net agrégat ((frags + assists/3) −
// morts)/matchs, AvgPerf en 0..100.
type ExplorerBriefingScope struct {
	Matches int     `json:"matches"`
	Wins    int     `json:"wins"`
	Losses  int     `json:"losses"`
	Ties    int     `json:"ties"`
	DNF     int     `json:"dnf"`
	WinRate float64 `json:"win_rate"`
	KDA     float64 `json:"kda"`
	// AvgPerf : perf moyenne 0..100. Nil si aucun match du scope n'a de score.
	AvgPerf *float64 `json:"avg_perf,omitempty"`
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

// ExplorerBriefingRanked est le module « Classement » : progression de paliers
// PAR TYPE de rating (CSR, LUSR) sur le scope. Une entrée par type suffisamment
// représenté (P-3) — jamais de paliers de deux systèmes mélangés sur une même
// ligne. Le bloc « attendu vs réel » (expected_win_prob) a été retiré (décision
// produit 2026-07-16 : donnée jugée non fiable).
type ExplorerBriefingRanked struct {
	// Kinds : une entrée par type de rating retenu. Ordre déterministe (type
	// majoritaire du scope d'abord). Toujours au moins une entrée quand le bloc
	// est émis (sinon Ranked est nil).
	Kinds []ExplorerBriefingRankedKind `json:"kinds"`
}

// ExplorerBriefingRankedKind est la progression du scope pour UN type de rating.
type ExplorerBriefingRankedKind struct {
	// Kind : "csr" | "lusr" — la métrique connue du joueur (affichée telle quelle).
	Kind string `json:"kind"`
	// Matches : nombre de matchs du scope portant ce type de rating.
	Matches int `json:"matches"`
	// TierStartLabel : palier du premier match chronologique portant un palier
	// (label FR déjà résolu en base, ex. « Bronze I »). Nil si non résolvable ou
	// si le début est en placement (voir TierStartIsPlacement).
	TierStartLabel *string `json:"tier_start_label,omitempty"`
	// TierEndLabel : palier du dernier match chronologique portant un palier.
	// Nil si non résolvable ou si la fin est en placement (voir
	// TierEndPlacementRemaining).
	TierEndLabel *string `json:"tier_end_label,omitempty"`
	// TierStartIsPlacement : vrai si le premier match du scope est en phase de
	// placement (rendu « Placement » sans compteur côté front — D-D).
	TierStartIsPlacement bool `json:"tier_start_is_placement,omitempty"`
	// TierEndPlacementRemaining : nombre de matchs de placement restants si le
	// dernier match du scope est encore en placement (rendu « Placement (N
	// restants) » côté front — D-D). Nil hors placement en fin de scope.
	TierEndPlacementRemaining *int `json:"tier_end_placement_remaining,omitempty"`
	// DeltaPerMatch : moyenne signée du delta de rating par match (Value/Count du
	// bucket). Nil si aucun match compté. Unité = points de rating natifs du type.
	DeltaPerMatch *float64 `json:"delta_per_match,omitempty"`
}
