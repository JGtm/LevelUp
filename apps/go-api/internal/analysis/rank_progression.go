// Package analysis — rank_progression.go : progression de paliers ranked du scope
// Explorer, segmentée PAR CHAÎNE de playlist (rating_type, playlist_group).
//
// Corrige le bug V3 du module « Classement » : le pt/match y était une moyenne de
// deltas tous playlist_group confondus au sein d'un type → flèche de paliers
// incohérente avec le sens du pt/match. Ici chaque CHAÎNE (type, playlist_group) est
// un segment autonome : paliers début/fin ET pt/match viennent des MÊMES matchs de
// la MÊME chaîne. CSR = chaîne unique « ranked » (csr_writes.go) ; LUSR se scinde en
// ses chaînes (arena_slayer/arena_objectif/btb/chaos, skillchain/classify.go). Algo
// pur title-agnostic (arch-rules) : aucun accès DB/HTTP, aucun effet de bord.
package analysis

import (
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/games/canonical"
)

// RankChainSample est un match rangé du scope, projeté depuis une raw row pour le
// calcul de progression par chaîne. RatingType = "CSR"/"LUSR" (casse DB tolérée) ;
// PlaylistGroup = groupe normalisé (nil/"" toléré). Les pointeurs nil signalent une
// donnée absente (match sans palier / sans valeur de rating / hors placement).
type RankChainSample struct {
	RatingType     string
	PlaylistGroup  *string
	StartTime      time.Time
	TierLabel      *string
	RatingValue    *float64
	RatingDelta    *float64
	PlacementDone  *int
	PlacementTotal *int
}

// RankChainProgression est la progression d'UNE chaîne (type, playlist_group) sur le
// scope : paliers début→fin (avec flags placement) + variation nette de rating
// ramenée au match. Une entrée par chaîne — jamais de flèche inter-chaînes.
type RankChainProgression struct {
	// RatingType : "csr" | "lusr" (normalisé minuscule, métrique connue du joueur).
	RatingType string
	// PlaylistGroup : groupe de la chaîne ("ranked" pour CSR, chaîne LUSR sinon ; ""
	// si la source ne renseigne pas de groupe).
	PlaylistGroup string
	// Matches : nombre de matchs de la chaîne dans le scope.
	Matches int
	// TierStartLabel / TierEndLabel : paliers du premier / dernier match noté de la
	// chaîne (nil si non résolu ou si placement — cf. flags).
	TierStartLabel *string
	TierEndLabel   *string
	// TierStartIsPlacement : le premier match noté de la chaîne est en placement.
	TierStartIsPlacement bool
	// TierEndPlacementRemaining : matchs de placement restants si le dernier match
	// noté est en placement. Nil hors placement en fin de chaîne.
	TierEndPlacementRemaining *int
	// DeltaPerMatch : variation nette du rating de la chaîne ramenée au match
	// (rating_value du dernier − du premier match noté) / nb de matchs notés. Garanti
	// co-signé avec la progression de paliers (paliers monotones dans le rating). Nil
	// si aucun match noté (pas de rating_value dans la chaîne).
	DeltaPerMatch *float64
}

// chainKey identifie une chaîne (type normalisé minuscule + groupe normalisé).
type chainKey struct {
	typ   string
	group string
}

// ComputeRankProgressionByChain groupe les samples par chaîne (rating_type,
// playlist_group) et calcule la progression de chacune. Ordre déterministe : type
// majoritaire d'abord (nb total de matchs du type décroissant, tie CSR d'abord),
// puis chaînes du type par nb de matchs décroissant (tie clé de chaîne croissante).
// Retourne une entrée PAR CHAÎNE (jamais fusionnées). nil si aucun sample.
func ComputeRankProgressionByChain(samples []RankChainSample) []RankChainProgression {
	if len(samples) == 0 {
		return nil
	}
	groups := map[chainKey][]RankChainSample{}
	order := make([]chainKey, 0)
	for _, s := range samples {
		k := chainKey{typ: normalizeRatingType(s.RatingType), group: normalizeGroup(s.PlaylistGroup)}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], s)
	}
	typeCounts := map[string]int{}
	for k, ss := range groups {
		typeCounts[k.typ] += len(ss)
	}
	progs := make([]RankChainProgression, 0, len(order))
	for _, k := range order {
		progs = append(progs, buildChainProgression(k, groups[k]))
	}
	sortChainProgressions(progs, typeCounts)
	return progs
}

// buildChainProgression calcule la progression d'une chaîne : Matches, paliers
// début/fin (flags placement) et variation nette de rating ramenée au match.
func buildChainProgression(k chainKey, samples []RankChainSample) RankChainProgression {
	sorted := make([]RankChainSample, len(samples))
	copy(sorted, samples)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].StartTime.Before(sorted[j].StartTime) })

	prog := RankChainProgression{RatingType: k.typ, PlaylistGroup: k.group, Matches: len(sorted)}
	if first := firstTieredSample(sorted); first != nil {
		applyTierStartSample(&prog, first)
	}
	if last := lastTieredSample(sorted); last != nil {
		applyTierEndSample(&prog, last)
	}
	prog.DeltaPerMatch = netRatingPerMatch(sorted)
	return prog
}

// netRatingPerMatch = (rating_value du dernier − du premier match noté) / nb notés.
// Bord : 1 seul match noté → son RatingDelta s'il existe, sinon 0. Aucun match noté
// → nil (pas de valeur de rating dans la chaîne).
func netRatingPerMatch(sorted []RankChainSample) *float64 {
	rated := make([]RankChainSample, 0, len(sorted))
	for _, s := range sorted {
		if s.RatingValue != nil {
			rated = append(rated, s)
		}
	}
	if len(rated) == 0 {
		return nil
	}
	if len(rated) == 1 {
		v := 0.0
		if rated[0].RatingDelta != nil {
			v = *rated[0].RatingDelta
		}
		return &v
	}
	v := (*rated[len(rated)-1].RatingValue - *rated[0].RatingValue) / float64(len(rated))
	return &v
}

// firstTieredSample / lastTieredSample : premier / dernier sample portant un palier
// (les samples sont déjà triés chronologiquement).
func firstTieredSample(sorted []RankChainSample) *RankChainSample {
	for i := range sorted {
		if sorted[i].TierLabel != nil {
			return &sorted[i]
		}
	}
	return nil
}

func lastTieredSample(sorted []RankChainSample) *RankChainSample {
	for i := len(sorted) - 1; i >= 0; i-- {
		if sorted[i].TierLabel != nil {
			return &sorted[i]
		}
	}
	return nil
}

// isPlacementSample : le match est en phase de placement (PlacementTotal renseigné).
func isPlacementSample(s *RankChainSample) bool { return s.PlacementTotal != nil }

// applyTierStartSample pose le palier de DÉBUT. En placement → flag sans compteur.
func applyTierStartSample(prog *RankChainProgression, s *RankChainSample) {
	if isPlacementSample(s) {
		prog.TierStartIsPlacement = true
		return
	}
	prog.TierStartLabel = s.TierLabel
}

// applyTierEndSample pose le palier de FIN. En placement → matchs restants.
func applyTierEndSample(prog *RankChainProgression, s *RankChainSample) {
	if isPlacementSample(s) {
		done := 0
		if s.PlacementDone != nil {
			done = *s.PlacementDone
		}
		remaining := *s.PlacementTotal - done
		if remaining < 0 {
			remaining = 0
		}
		prog.TierEndPlacementRemaining = &remaining
		return
	}
	prog.TierEndLabel = s.TierLabel
}

// sortChainProgressions ordonne : type majoritaire d'abord (typeCounts desc, tie CSR
// d'abord puis type asc), puis chaînes du type par Matches desc (tie PlaylistGroup
// asc). La clé de chaîne étant unique, l'ordre est total et déterministe.
func sortChainProgressions(progs []RankChainProgression, typeCounts map[string]int) {
	csr := string(canonical.RatingTypeCSR)
	sort.SliceStable(progs, func(i, j int) bool {
		a, b := progs[i], progs[j]
		if a.RatingType != b.RatingType {
			if ca, cb := typeCounts[a.RatingType], typeCounts[b.RatingType]; ca != cb {
				return ca > cb
			}
			if (a.RatingType == csr) != (b.RatingType == csr) {
				return a.RatingType == csr
			}
			return a.RatingType < b.RatingType
		}
		if a.Matches != b.Matches {
			return a.Matches > b.Matches
		}
		return a.PlaylistGroup < b.PlaylistGroup
	})
}

// normalizeRatingType réduit "CSR"/"csr"/" LUSR " → "csr"/"lusr" (métrique connue).
func normalizeRatingType(t string) string { return strings.ToLower(strings.TrimSpace(t)) }

// normalizeGroup réduit un playlist_group éventuellement nil/espacé à une clé stable.
func normalizeGroup(g *string) string {
	if g == nil {
		return ""
	}
	return strings.TrimSpace(*g)
}
