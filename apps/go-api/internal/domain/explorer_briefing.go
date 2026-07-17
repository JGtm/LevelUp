// Package domain — types du bandeau de briefing de l'Explorer (mode Matchs).
//
// Le bandeau est une lecture compacte du RÉSULTAT DE RECHERCHE (sous-ensemble
// filtré) : KPIs agrégés, comparaison à la baseline personnelle (historique
// complet du titre) et modules conditionnels (notes par dimension, classé) qui
// s'activent selon les filtres, les capabilities du titre et la taille
// d'échantillon. Ce N'EST PAS une page Synthèse (cf.
// PLAN_EXPLORER_BRIEFING_CARDS_2026-07.md).
//
// Tous les blocs (hors socle KPIs/période) sont optionnels : un bloc nil
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
	// Dans ce cas seuls KPIs + période sont émis (les autres blocs nil).
	LowSample bool `json:"low_sample,omitempty"`
	// PeriodStart/PeriodEnd : bornes temporelles (min/max start_time) du scope.
	PeriodStart *time.Time `json:"period_start,omitempty"`
	PeriodEnd   *time.Time `json:"period_end,omitempty"`
	// Baseline : comparaison à l'historique complet (post-exclusions). Nil si
	// low sample.
	Baseline *ExplorerBriefingBaseline `json:"baseline,omitempty"`
	// Dimensions : notes par dimension libre (carte/mode/playlist). Vide/nil si
	// low sample ou aucune dimension libre.
	Dimensions []ExplorerBriefingDimension `json:"dimensions,omitempty"`
	// Ranked : module « Classement » (progression de paliers PAR CHAÎNE de playlist).
	// Nil si le titre n'expose pas la capability ranked ou si aucun match rangé
	// dans le scope.
	Ranked *ExplorerBriefingRanked `json:"ranked,omitempty"`
	// ContextSplit : comparaison solo vs escouade du scope. Nil si non pertinent
	// (un sous-groupe sous le seuil, ou scope déjà réduit à un seul contexte).
	ContextSplit *ExplorerBriefingContextSplit `json:"context_split,omitempty"`
	// Streaks : séries extrêmes du scope (meilleure série de victoires / pire série
	// de défaites), calculées sur TOUT le scope filtré (P-9). Nil si non pertinent
	// (low sample, ou aucune row datée). Un segment à zéro est omis côté front.
	Streaks *ExplorerBriefingStreaks `json:"streaks,omitempty"`
	// Dominance : compteurs de moments forts (DominanceFlag 1..5) du scope. Nil si
	// non pertinent (low sample, ou tous les compteurs à zéro).
	Dominance *ExplorerBriefingDominance `json:"dominance,omitempty"`
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
	// TotalDurationSeconds : somme des durées (r.duration_seconds) des matchs du
	// scope. Nil si aucune durée disponible. Formaté « h min » côté front.
	TotalDurationSeconds *int `json:"total_duration_seconds,omitempty"`
	// PeakKDA : meilleur KDA (valeur API native par match) d'un seul match du scope.
	// Toujours tenté ; nil si aucun match du scope ne porte de KDA.
	PeakKDA *float64 `json:"peak_kda,omitempty"`
	// PeakTeamMMR : meilleur team_mmr d'un match du scope. Nil si aucun team_mmr
	// (métrique brute, soumise au masquage MMR côté front).
	PeakTeamMMR *float64 `json:"peak_team_mmr,omitempty"`
	// PeakRanks : meilleur palier ATTEINT par système de rating sur le scope (0, 1 ou
	// 2 entrées ; ordre déterministe LUSR puis CSR). Ce n'est PAS le palier final : le
	// pic peut dépasser le palier d'arrivée du module Classement. Nil si aucun palier.
	PeakRanks []ExplorerBriefingPeakRank `json:"peak_ranks,omitempty"`
}

// ExplorerBriefingPeakRank porte le meilleur palier atteint pour UN système de
// rating sur le scope (Pic rang). RatingType = "csr" | "lusr" ; TierLabel = label FR
// résolu (ex. « Diamant IV ») de la row au palier maximal, sélectionnée par
// (analysis.CSRTierOrdinal(tier EN), sub_tier).
type ExplorerBriefingPeakRank struct {
	RatingType string `json:"rating_type"`
	TierLabel  string `json:"tier_label"`
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

// ExplorerBriefingRanked est le module « Classement » : progression de paliers PAR
// CHAÎNE de playlist (rating_type, playlist_group) sur le scope. Une entrée par
// chaîne — paliers début/fin ET pt/match viennent des MÊMES matchs de la MÊME chaîne,
// jamais de flèche inter-chaînes (DEC-RANK-BE). CSR = chaîne unique « ranked » (P-3
// préservé) ; LUSR se scinde en ses chaînes. Le bloc « attendu vs réel »
// (expected_win_prob) a été retiré (décision produit 2026-07-16 : donnée non fiable).
type ExplorerBriefingRanked struct {
	// Kinds : une entrée par CHAÎNE (rating_type, playlist_group). Ordre déterministe
	// (type majoritaire du scope d'abord, puis chaînes du type par nb de matchs). Au
	// moins une entrée quand le bloc est émis (sinon Ranked est nil).
	Kinds []ExplorerBriefingRankedKind `json:"kinds"`
}

// ExplorerBriefingRankedKind est la progression du scope pour UNE chaîne
// (rating_type, playlist_group).
type ExplorerBriefingRankedKind struct {
	// Kind : "csr" | "lusr" — la métrique connue du joueur (affichée telle quelle).
	Kind string `json:"kind"`
	// PlaylistGroup : groupe de playlist de la chaîne ("ranked" pour CSR — chaîne
	// unique ; une des 4 chaînes LUSR sinon : arena_slayer/arena_objectif/btb/chaos).
	// Une entrée = UNE chaîne (kind, playlist_group). "" si la source ne renseigne pas
	// de groupe. Le front n'affiche le libellé de chaîne que si le type a ≥ 2 chaînes.
	PlaylistGroup string `json:"playlist_group,omitempty"`
	// Matches : nombre de matchs du scope portant cette chaîne.
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
	// DeltaPerMatch : variation nette du rating de la chaîne ramenée au match
	// (rating_value du dernier − du premier match noté, / nb de matchs notés). Garantie
	// co-signée avec la progression de paliers (paliers monotones dans le rating). Nil
	// si aucun match noté. Unité = points de rating natifs du type (points CSR / mu LUSR).
	DeltaPerMatch *float64 `json:"delta_per_match,omitempty"`
}

// ExplorerBriefingContextSplit compare les performances du scope selon le
// contexte social (solo vs escouade, signal IsWithFriends). Émis UNIQUEMENT
// quand les deux sous-groupes atteignent minContextSplitMatches (scope
// réellement multi-contexte ET fiabilité minimale) ; nil sinon (dégradation par
// omission). Sans capability rang requise (IsWithFriends dispo tous titres).
type ExplorerBriefingContextSplit struct {
	// Solo : agrégats des matchs joués en solo (IsWithFriends = false).
	Solo ExplorerBriefingContextGroup `json:"solo"`
	// Squad : agrégats des matchs joués en escouade (IsWithFriends = true).
	Squad ExplorerBriefingContextGroup `json:"squad"`
}

// ExplorerBriefingContextGroup porte les agrégats socle d'un contexte social
// (solo ou escouade). Symétrique du socle (ExplorerBriefingScope). Unités
// ADR 0006 : WinRate en ratio 0..1, KDA = net agrégat ((frags + assists/3) −
// morts)/matchs, AvgPerf en 0..100.
type ExplorerBriefingContextGroup struct {
	Matches int     `json:"matches"`
	WinRate float64 `json:"win_rate"` // ratio 0..1
	KDA     float64 `json:"kda"`      // KDA agrégat ADR 0006
	// AvgPerf : perf moyenne 0..100. Nil si aucun match du groupe n'a de score.
	AvgPerf *float64 `json:"avg_perf,omitempty"`
}

// ExplorerBriefingStreaks porte les séries extrêmes du scope, calculées sur TOUT
// le scope filtré (jamais depuis la frise cappée) après tri chronologique. Une
// série est rompue par TOUT autre outcome (P-9). Émis hors low_sample ; nil si
// aucune row datée. Un segment à zéro (aucune victoire / aucune défaite) reste à
// 0 (omitempty) et le front l'omet.
type ExplorerBriefingStreaks struct {
	// BestWinStreak : plus longue série de victoires consécutives. 0 si aucune.
	BestWinStreak int `json:"best_win_streak,omitempty"`
	// WorstLossStreak : plus longue série de défaites consécutives. 0 si aucune.
	WorstLossStreak int `json:"worst_loss_streak,omitempty"`
}

// ExplorerBriefingDominance compte les moments forts (DominanceFlag 1..5,
// cf. analysis.DominanceFlag*) du scope. Émis hors low_sample ; nil si tous les
// compteurs sont à zéro (dégradation par omission). Les catégories à zéro sont
// omises côté front (les libellés réutilisent narrative.dominance.*).
type ExplorerBriefingDominance struct {
	Dominations      int `json:"dominations,omitempty"`
	Humiliations     int `json:"humiliations,omitempty"`
	Remontadas       int `json:"remontadas,omitempty"`
	Debandades       int `json:"debandades,omitempty"`
	ContreRemontadas int `json:"contre_remontadas,omitempty"`
}
