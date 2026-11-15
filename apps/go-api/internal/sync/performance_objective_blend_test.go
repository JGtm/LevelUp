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

// Une seule durée de repli pour tous les calculs par minute : le même match ne compte jamais 600 s dans
// l'historique et 0 dans les faits d'objectif (tenue de l'indice, soutien par minute).
func TestEffectivePlayedSeconds(t *testing.T) {
	for _, c := range []struct{ in, want float64 }{{0, perfFallbackDurationSeconds}, {-3, perfFallbackDurationSeconds}, {937, 937}} {
		if got := effectivePlayedSeconds(c.in); got != c.want {
			t.Errorf("effectivePlayedSeconds(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestObjectiveFactsOf_DureeNulleSuitLeRepliDesCadences(t *testing.T) {
	// Drapeau : le joueur porte le drapeau 26 s et soutient (retours, porteurs abattus) ; son camp aussi.
	rows := []narrative.ObjectiveTeamRow{
		{XUID: "me", TeamID: 0, Family: narrative.FamilyCTF, TimePlayedSeconds: 0, Values: map[string]float64{
			"flag_captures": 2, "time_as_flag_carrier_seconds": 26, "flag_returns": 3, "flag_carriers_killed": 2,
		}},
		{XUID: "mate", TeamID: 0, Family: narrative.FamilyCTF, TimePlayedSeconds: 0, Values: map[string]float64{"flag_returns": 1}},
	}
	repli := objectiveFactsOf(rows, "me", 0)
	explicite := objectiveFactsOf(rows, "me", perfFallbackDurationSeconds)
	negative := objectiveFactsOf(rows, "me", -1)
	if repli == nil || explicite == nil || negative == nil {
		t.Fatal("faits absents")
	}
	if *repli != *explicite || *negative != *explicite {
		t.Fatalf("durée nulle %+v, négative %+v, repli explicite %+v : le même repli attendu", *repli, *negative, *explicite)
	}
	if repli.SupportPerMin <= 0 {
		t.Errorf("soutien par minute = %v avec une durée nulle : 0 serait compté au lieu de 600 s", repli.SupportPerMin)
	}
	// La tenue de l'indice (26 s sur la durée de repli) compte : sans elle l'indice serait celui des actions seules.
	sansTenue := rows
	sansTenue[0].Values = map[string]float64{"flag_captures": 2, "flag_returns": 3, "flag_carriers_killed": 2}
	if hors := objectiveFactsOf(sansTenue, "me", 0); hors.Index >= repli.Index {
		t.Errorf("indice sans tenue %v >= indice avec tenue %v : la tenue lue sur la durée de repli doit compter", hors.Index, repli.Index)
	}
	// Les lignes d'origine ne sont pas modifiées.
	if rows[0].TimePlayedSeconds != 0 {
		t.Errorf("durée de la ligne modifiée : %v", rows[0].TimePlayedSeconds)
	}
}

// Le seuil est EXACT : avec 10 matchs d'historique la note existe et le mélange s'applique ; avec 9,
// il n'y a pas de note.
func TestComputeChainPerformanceScore_SeuilExactDeDixMatchs(t *testing.T) {
	chain := skill.LUSRChainArenaObjectif
	facts := &objectiveblend.Facts{Family: narrative.FamilyCTF, Index: 5, Share: 1, SupportShare: 1, SupportPerMin: 9}
	cur := matchOf(12, 6, facts)
	if got := computeChainPerformanceScore(chain, &cur, blendHistory(MinMatchesForRelative-1)); got != nil {
		t.Fatalf("9 matchs : note %v, want nil", *got)
	}
	hist := blendHistory(MinMatchesForRelative)
	got := computeChainPerformanceScore(chain, &cur, hist)
	if got == nil {
		t.Fatal("10 matchs : pas de note")
	}
	combat, _ := weightedPercentileMean(&cur, hist, skill.WeightsForChain(chain))
	if *got == roundPerfScore(combat) {
		t.Errorf("10 matchs : la note %v est le combat seul, le mélange doit s'appliquer", *got)
	}
}
