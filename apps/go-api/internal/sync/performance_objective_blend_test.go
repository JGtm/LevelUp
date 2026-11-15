package sync

// performance_objective_blend_test.go — la note à objectif branchée dans la note de performance
// relative (computeChainPerformanceScore) : V7 sur les chaînes de famille objectif, le score
// relatif tel quel ailleurs, la métrique absente sans faits (Halo 5, match sans
// match_objective_stats). Les variantes elles-mêmes sont testées dans le paquet objectiveblend.

import (
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/sync/objectiveblend"
	"levelup/go-api/internal/sync/skill"
)

// blendHistory fabrique n matchs d'historique à objectif : le combat varie de façon déterministe,
// l'indice, la part et le soutien montent avec i, la famille alterne Drapeau / Bases.
func blendHistory(n int) []historyRow {
	out := make([]historyRow, 0, n)
	for i := 0; i < n; i++ {
		fam := narrative.FamilyCTF
		if i%2 == 1 {
			fam = narrative.FamilyZonesStrongholds
		}
		out = append(out, historyRow{
			Kills: float64(6 + (i*7)%14), Deaths: float64(5 + (i*3)%9), Assists: float64(i % 5),
			TimePlayedSeconds: 600, PersonalScore: float64(1500 + 40*(i%11)), DamageDealt: float64(3000 + 90*(i%13)),
			ObjectiveBlend: &objectiveblend.Facts{Family: fam, Index: 0.05 * float64(i%20), Share: 0.02 * float64(i%20),
				SupportShare: 0.03 * float64(i%10), SupportPerMin: 0.1 * float64(i%20)},
			Chain: skill.LUSRChainArenaObjectif,
		})
		fullCoverage(&out[len(out)-1], i)
	}
	return out
}

// fullCoverage pose toutes les métriques du profil (précision, attendus, rang et MMR, rendement,
// médailles) pour que la somme des poids retenus soit celle du profil.
func fullCoverage(r *historyRow, i int) {
	r.Accuracy = 0.35 + 0.01*float64(i%9)
	r.KillsExpected = 9 + float64(i%4)
	r.DeathsExpected = 8 + float64(i%3)
	r.Rank = float64(1 + i%8)
	r.TeamMMR = 1500 + float64(10*(i%7))
	r.EnemyMMR = 1500 + float64(10*(i%5))
	r.OffensiveConversion = 0.8 + 0.03*float64(i%10)
	r.DefensiveResistance = 0.9 + 0.02*float64(i%8)
	r.MedalExploitScore = 1 + float64(i%6)
}

func matchOf(kills, deaths float64, facts *objectiveblend.Facts) historyRow {
	r := historyRow{Kills: kills, Deaths: deaths, Assists: 2, TimePlayedSeconds: 600, PersonalScore: 1600, DamageDealt: 3300,
		ObjectiveBlend: facts, Chain: skill.LUSRChainArenaObjectif}
	fullCoverage(&r, 3)
	return r
}

// Sans faits (Halo 5, match sans match_objective_stats) : la métrique est absente et la note est le
// combat renormalisé du profil de la chaîne, sans panique.
func TestComputeChainPerformanceScore_SansFaitsEstLeCombat(t *testing.T) {
	hist := blendHistory(30)
	cur := matchOf(12, 6, nil)
	got := computeChainPerformanceScore(skill.LUSRChainArenaObjectif, &cur, hist)
	combat, ok := weightedPercentileMean(&cur, hist, skill.WeightsForChain(skill.LUSRChainArenaObjectif))
	if got == nil || !ok || *got != roundPerfScore(combat) {
		t.Fatalf("note %v, want le combat renormalisé %v", got, combat)
	}
	// Historique sans aucun fait : même résultat.
	for i := range hist {
		hist[i].ObjectiveBlend = nil
	}
	cur.ObjectiveBlend = &objectiveblend.Facts{Family: narrative.FamilyCTF, Index: 1, Share: 1}
	if got := computeChainPerformanceScore(skill.LUSRChainArenaObjectif, &cur, hist); got == nil || *got != roundPerfScore(combat) {
		t.Fatalf("référence vide : %v, want le combat %v", got, combat)
	}
}

// Hors des chaînes de famille objectif, la note est le score relatif du profil de la chaîne, que
// les faits soient présents ou non.
func TestComputeChainPerformanceScore_ChaineDeCombatInchangee(t *testing.T) {
	hist := blendHistory(30)
	cur := matchOf(12, 6, &objectiveblend.Facts{Family: narrative.FamilyCTF, Index: 2, Share: 1, SupportShare: 1, SupportPerMin: 9})
	for _, chain := range []string{skill.LUSRChainArenaSlayer, skill.LUSRChainBTB, skill.PerfChainRankedSlayer, skill.PerfChainFirefight} {
		got := computeChainPerformanceScore(chain, &cur, hist)
		want := computeRelativePerformanceScore(&cur, hist, skill.WeightsForChain(chain))
		if got == nil || want == nil || *got != *want {
			t.Errorf("%s : %v, want %v", chain, got, want)
		}
	}
}

// Historique trop court : pas de note.
func TestComputeChainPerformanceScore_HistoriqueTropCourt(t *testing.T) {
	cur := matchOf(12, 6, &objectiveblend.Facts{Family: narrative.FamilyCTF, Index: 1, Share: 0.5})
	if got := computeChainPerformanceScore(skill.LUSRChainArenaObjectif, &cur, blendHistory(MinMatchesForRelative-1)); got != nil {
		t.Fatalf("note %v sous le minimum d'historique, want nil", *got)
	}
}

// Le cas qui motive le lot : un joueur à 0 frag qui porte seul son équipe monte, un tueur sans
// objectif baisse (moyenne symétrique) ; l'objectif et le soutien jouent dans les deux sens.
func TestComputeChainPerformanceScore_PorteurMonteTueurBaisse(t *testing.T) {
	hist := blendHistory(40)
	chain := skill.LUSRChainArenaObjectif
	combatOf := func(c *historyRow) float64 {
		v, _ := weightedPercentileMean(c, hist, skill.WeightsForChain(chain))
		return roundPerfScore(v)
	}
	porteur := matchOf(0, 8, &objectiveblend.Facts{Family: narrative.FamilyCTF, Index: 3, Share: 0.8, SupportShare: 0.1, SupportPerMin: 0.5})
	tueur := matchOf(20, 4, &objectiveblend.Facts{Family: narrative.FamilyCTF, Index: 0, Share: 0, SupportShare: 0, SupportPerMin: 0})
	if got := computeChainPerformanceScore(chain, &porteur, hist); got == nil || *got <= combatOf(&porteur)+0.5 {
		t.Errorf("porteur : %v, combat %v : attendu un gain", got, combatOf(&porteur))
	}
	if got := computeChainPerformanceScore(chain, &tueur, hist); got == nil || *got >= combatOf(&tueur) {
		t.Errorf("tueur sans objectif : %v, combat %v : attendu une baisse", got, combatOf(&tueur))
	}
}

// Les réglages de la note à objectif : fixés avec l'utilisateur, couverts par l'empreinte de
// scoring_revision_test.go ; ce test dit leurs valeurs.
func TestObjectiveBlendParams_ValeursRetenues(t *testing.T) {
	p := objectiveBlendParams()
	if p.AlphaMin != 0.12 || p.AlphaMax != 0.50 || p.SupportSlope != 0.38 || p.MinFamilyMatches != MinMatchesForRelative {
		t.Fatalf("réglages = %+v", p)
	}
}
