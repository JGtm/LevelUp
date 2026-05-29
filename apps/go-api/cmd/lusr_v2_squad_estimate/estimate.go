//go:build cgo

package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// pairResult : offset estimé + nombre de matchs ensemble (diagnostic).
type pairResult struct {
	offset float64
	n      int
}

// estimateOffsets accumule, pour chaque paire de coéquipiers, les couples
// (issue réelle, proba solo prédite) puis calcule l'offset des paires ayant
// atteint le seuil minMatches.
func estimateOffsets(matches []matchRoster, ratings map[string]map[string]skillv2.Gaussian,
	minMatches int, gain float64) map[pairKey]pairResult {
	accum := make(map[pairKey][]skillv2.SquadCoMatch)
	for i := range matches {
		mr := &matches[i]
		teamIDs := twoTeamIDs(mr.teams)
		for ti, teamID := range teamIDs {
			members := mr.teams[teamID]
			if len(members) < 2 {
				continue // pas de paire sur cette équipe
			}
			oppID := teamIDs[1-ti]
			co := skillv2.SquadCoMatch{
				Won:         mr.wonByTeam[teamID],
				SoloWinProb: predictTeamWin(mr.group, members, mr.teams[oppID], ratings),
			}
			for _, k := range pairsOf(members, mr.group) {
				accum[k] = append(accum[k], co)
			}
		}
	}
	out := make(map[pairKey]pairResult)
	for k, co := range accum {
		if len(co) < minMatches {
			continue
		}
		out[k] = pairResult{offset: skillv2.ComputeSquadOffset(co, gain), n: len(co)}
	}
	return out
}

// twoTeamIDs retourne les 2 team_id triés (l'appelant a déjà filtré len==2).
func twoTeamIDs(teams map[int][]string) []int {
	ids := make([]int, 0, len(teams))
	for id := range teams {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// predictTeamWin retourne la proba de victoire de `team` vs `opp` selon les
// ratings SOLO (sans offset). Joueurs inconnus → prior par défaut.
func predictTeamWin(group string, team, opp []string,
	ratings map[string]map[string]skillv2.Gaussian) float64 {
	return skillv2.PredictWinProbability(
		teamGaussians(group, team, ratings),
		teamGaussians(group, opp, ratings),
		skillv2.DefaultPriors())
}

// teamGaussians résout les gaussiennes solo d'une liste de xuids (fallback prior).
func teamGaussians(group string, xuids []string,
	ratings map[string]map[string]skillv2.Gaussian) []skillv2.Gaussian {
	prior := skillv2.DefaultPriors().NewPlayerState()
	out := make([]skillv2.Gaussian, len(xuids))
	for i, x := range xuids {
		if g, ok := ratings[group][x]; ok {
			out[i] = g
		} else {
			out[i] = prior
		}
	}
	return out
}

// pairsOf retourne toutes les paires non ordonnées (a<b) d'une équipe.
func pairsOf(members []string, group string) []pairKey {
	var out []pairKey
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			a, b := members[i], members[j]
			if a > b {
				a, b = b, a
			}
			out = append(out, pairKey{a: a, b: b, group: group})
		}
	}
	return out
}

// writeOffsets persiste chaque paire dans les DEUX sens (synergie symétrique).
func writeOffsets(ctx context.Context, repo *duckdb.SquadOffsetRepo,
	offsets map[pairKey]pairResult, source string) error {
	for k, r := range offsets {
		for _, pair := range [][2]string{{k.a, k.b}, {k.b, k.a}} {
			if err := repo.UpsertSquadOffset(ctx, domain.SquadOffset{
				XUID:          pair[0],
				PartnerXUID:   pair[1],
				PlaylistGroup: k.group,
				OffsetValue:   r.offset,
				MatchCount:    r.n,
				Source:        source,
			}); err != nil {
				return fmt.Errorf("upsert %s↔%s/%s: %w", k.a, k.b, k.group, err)
			}
		}
	}
	return nil
}

func printReport(offsets map[pairKey]pairResult, since time.Time, minMatches int, gain float64) {
	keys := make([]pairKey, 0, len(offsets))
	for k := range offsets {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].group != keys[j].group {
			return keys[i].group < keys[j].group
		}
		if keys[i].a != keys[j].a {
			return keys[i].a < keys[j].a
		}
		return keys[i].b < keys[j].b
	})
	var sb strings.Builder
	sb.WriteString("\n=== LUSR v2 Sprint 1.C — Squad offset estimate ===\n")
	sb.WriteString(fmt.Sprintf("Depuis: %s · min-matches: %d · gain: %.2f · paires éligibles: %d\n\n",
		since.Format("2006-01-02"), minMatches, gain, len(offsets)))
	for _, k := range keys {
		r := offsets[k]
		sb.WriteString(fmt.Sprintf("  [%s] %s ↔ %s : offset=%+.3f (n=%d)\n",
			k.group, k.a, k.b, r.offset, r.n))
	}
	fmt.Fprintln(os.Stdout, sb.String())
}
