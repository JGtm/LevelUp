// Package service — match_history_service_briefing_dimensions.go : cartes
// « Par carte / mode / sélection » du bandeau de briefing de l'Explorer
// (regroupement des raw rows par dimension, top / flop vs baseline).
package service

import (
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/breakdown"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// buildBriefingDimensions émet une carte par dimension LIBRE (≥ 2 valeurs
// distinctes dans le scope, DEC-8) parmi carte / mode / playlist. Par dimension :
// top 3 + flop 3 des groupes qualifiés (≥ MinDimensionGroupMatches), triés par
// delta winrate vs baseline ; moins de 2 groupes qualifiés → carte Insufficient.
func buildBriefingDimensions(scope, all []domain.MatchHistoryRawRow, groups domain.GroupesDeCartes) []domain.ExplorerBriefingDimension {
	if len(scope) == 0 {
		return nil
	}
	scopeRows := rawRowsToBreakdownRows(scope)
	allRows := rawRowsToBreakdownRows(all)
	// Plein historique = scope égal à l'historique complet (un filtre ne peut que
	// rétrécir → cardinalités égales ⟺ ensembles identiques). Les deltas vs baseline
	// sont alors tous nuls et le tri de CompareByKey dégénère : la sélection top/flop
	// bascule sur le taux de victoire (P-8, cf. buildDimension).
	fullHistory := len(scope) == len(all)
	out := make([]domain.ExplorerBriefingDimension, 0, 3)
	if d := buildDimension("map", mapKeyed(breakdown.ByMap(scopeRows, groups, libellesDeCartesFR)), mapKeyed(breakdown.ByMap(allRows, groups, libellesDeCartesFR)), fullHistory); d != nil {
		out = append(out, *d)
	}
	if d := buildDimension("mode", modeKeyed(breakdown.ByMode(scopeRows)), modeKeyed(breakdown.ByMode(allRows)), fullHistory); d != nil {
		out = append(out, *d)
	}
	if d := buildDimension("playlist", playlistKeyed(breakdown.ByPlaylist(scopeRows)), playlistKeyed(breakdown.ByPlaylist(allRows)), fullHistory); d != nil {
		out = append(out, *d)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// buildDimension construit une dimension à partir de ses agrégats scope / historique
// génériques par clé. Retourne nil si la dimension n'est pas libre (< 2 valeurs
// distinctes : axe filtré ou valeur unique) ; une dimension à 2 valeurs ou plus
// sans 2 groupes qualifiés (MinDimensionGroupMatches) est servie Insufficient.
func buildDimension(dim string, sessionKA, histKA []breakdown.KeyedAggregate, fullHistory bool) *domain.ExplorerBriefingDimension {
	distinct := 0
	for _, a := range sessionKA {
		if a.Played > 0 {
			distinct++
		}
	}
	if distinct < 2 {
		return nil
	}
	perfByKey := make(map[string]*float64, len(sessionKA))
	for _, a := range sessionKA {
		perfByKey[a.Key] = a.AvgPerformanceScore
	}
	deltas := breakdown.CompareByKey(sessionKA, histKA)
	qualified := make([]breakdown.KeyedDelta, 0, len(deltas))
	for _, d := range deltas {
		if d.Session.Played >= MinDimensionGroupMatches {
			qualified = append(qualified, d)
		}
	}
	// Une carte compare au moins deux groupes (D3) : avec 2 valeurs ou plus dans le
	// scope mais moins de 2 groupes qualifiés, elle est servie en état vide.
	if len(qualified) < 2 {
		return &domain.ExplorerBriefingDimension{
			Dimension:    dim,
			Entries:      []domain.ExplorerBriefingDimensionEntry{},
			Insufficient: true,
		}
	}
	// En plein historique, tous les WinRateDelta valent 0 (scope == historique) et
	// CompareByKey retombe sur un tri par clé (GUID de map → pseudo-aléatoire). On
	// re-trie par taux de victoire du groupe décroissant (tie-break libellé) pour
	// une sélection top/flop signifiante (P-8). Sous filtre : tri V1 (delta) conservé.
	if fullHistory {
		sort.SliceStable(qualified, func(i, j int) bool {
			if qualified[i].Session.WinRate != qualified[j].Session.WinRate {
				return qualified[i].Session.WinRate > qualified[j].Session.WinRate
			}
			return qualified[i].Label < qualified[j].Label
		})
	}
	qualified = selectTopFlop(qualified, dimensionTopFlopCount)
	entries := make([]domain.ExplorerBriefingDimensionEntry, 0, len(qualified))
	for _, d := range qualified {
		avgPerf := perfByKey[d.Key]
		e := domain.ExplorerBriefingDimensionEntry{
			Label:        d.Label,
			Matches:      d.Session.Played,
			WinRate:      d.Session.WinRate,
			DeltaWinRate: d.WinRateDelta,
			AvgPerf:      avgPerf,
		}
		if avgPerf != nil {
			tier := int(analysis.PerfTier(*avgPerf))
			e.NoteTier = &tier
		}
		entries = append(entries, e)
	}
	return &domain.ExplorerBriefingDimension{Dimension: dim, Entries: entries}
}

// selectTopFlop retourne les k premiers + k derniers éléments d'une liste triée.
// Si la liste tient en 2*k, elle est retournée telle quelle (pas de recouvrement).
func selectTopFlop[T any](items []T, k int) []T {
	if len(items) <= 2*k {
		return items
	}
	out := make([]T, 0, 2*k)
	out = append(out, items[:k]...)
	out = append(out, items[len(items)-k:]...)
	return out
}

// ─── conversion raw row → breakdown.Row (libellés FR) ────────────────────────

// rawRowsToBreakdownRows projette les raw rows vers breakdown.Row en résolvant
// les libellés FR (MapNameFR / PairNameFR normalisé / PlaylistName coalescé).
func rawRowsToBreakdownRows(rows []domain.MatchHistoryRawRow) []breakdown.Row {
	out := make([]breakdown.Row, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		br := breakdown.Row{Outcome: rawOutcomeToCanonical(r.Outcome)}
		if r.PerformanceScore != nil {
			v := *r.PerformanceScore
			br.PerformanceScore = &v
		}
		if r.MapID != nil {
			br.MapID = *r.MapID
		}
		br.MapLabel = coalesceStr(r.MapNameFR, r.MapName)
		// Mode : MÊME résolution que la colonne mode_ui du tableau — pair_name (FR)
		// puis fallback game_variant (titres/matchs sans pair, ex. H5). Sans le
		// fallback, les matchs dont le mode vient du game_variant sont écartés du
		// regroupement et la dimension « par mode » peut dégénérer / disparaître.
		// Convention centralisée — cf. analysis.ResolveModeUIWithVariant.
		if modeUI := analysis.ResolveModeUIWithVariant(r.PairName, r.PairNameFR, r.GameVariantName, r.GameVariantNameFR); modeUI != nil {
			br.ModeName = *modeUI
		}
		if r.PlaylistName != nil {
			br.PlaylistName = *r.PlaylistName
		}
		out = append(out, br)
	}
	return out
}

// rawOutcomeToCanonical mappe le code outcome int (domain.Outcome*) vers l'enum
// canonical.Outcome consommé par le package breakdown. "" si code inconnu.
func rawOutcomeToCanonical(code int) canonical.Outcome {
	switch code {
	case domain.OutcomeWin:
		return canonical.OutcomeWin
	case domain.OutcomeLoss:
		return canonical.OutcomeLoss
	case domain.OutcomeDraw:
		return canonical.OutcomeTie
	case domain.OutcomeDNF:
		return canonical.OutcomeDNF
	}
	return ""
}

// mapKeyed / modeKeyed / playlistKeyed projettent les agrégats de dimension vers
// la forme pivot KeyedAggregate (clé stable + libellé) pour breakdown.CompareByKey.
func mapKeyed(aggs []breakdown.MapAggregate) []breakdown.KeyedAggregate {
	out := make([]breakdown.KeyedAggregate, 0, len(aggs))
	for _, a := range aggs {
		label := a.MapLabel
		if label == "" {
			label = a.MapID
		}
		out = append(out, breakdown.KeyedAggregate{
			Key: a.MapID, Label: label, Counts: a.Counts, AvgPerformanceScore: a.AvgPerformanceScore,
		})
	}
	return out
}

func modeKeyed(aggs []breakdown.ModeAggregate) []breakdown.KeyedAggregate {
	out := make([]breakdown.KeyedAggregate, 0, len(aggs))
	for _, a := range aggs {
		out = append(out, breakdown.KeyedAggregate{
			Key: a.ModeName, Label: a.ModeName, Counts: a.Counts, AvgPerformanceScore: a.AvgPerformanceScore,
		})
	}
	return out
}

func playlistKeyed(aggs []breakdown.PlaylistAggregate) []breakdown.KeyedAggregate {
	out := make([]breakdown.KeyedAggregate, 0, len(aggs))
	for _, a := range aggs {
		out = append(out, breakdown.KeyedAggregate{
			Key: a.PlaylistName, Label: a.PlaylistName, Counts: a.Counts, AvgPerformanceScore: a.AvgPerformanceScore,
		})
	}
	return out
}
