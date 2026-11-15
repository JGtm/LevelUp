package sync

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/sync/skill"
)

// blendHistory fabrique n matchs d'historique à objectif : le combat varie de façon
// déterministe, l'indice d'objectif et les points d'awards montent ensemble (même ordre
// de classement), la famille alterne CTF / Strongholds.
func blendHistory(n int) []historyRow {
	out := make([]historyRow, 0, n)
	for i := 0; i < n; i++ {
		fam := narrative.FamilyCTF
		if i%2 == 1 {
			fam = narrative.FamilyZonesStrongholds
		}
		idx := 0.05 * float64(i%20)
		pts := 10 * float64(i%20)
		out = append(out, historyRow{
			Kills: float64(6 + (i*7)%14), Deaths: float64(5 + (i*3)%9), Assists: float64(i % 5),
			TimePlayedSeconds: 600, PersonalScore: float64(1500 + 40*(i%11)), DamageDealt: float64(3000 + 90*(i%13)),
			ObjectiveScore: &pts,
			ObjectiveBlend: &objectiveBlendFacts{Family: fam, Index: idx, Share: 0.3},
			Chain:          skill.LUSRChainArenaObjectif,
		})
		fullCoverage(&out[len(out)-1], i)
	}
	return out
}

// fullCoverage pose toutes les métriques du profil (précision, attendus, rang et MMR,
// rendement, médailles) : sans elles, la somme des poids RETENUS est plus petite que 1,04 et
// le poids relatif d'ospm dans la note actuelle monte au-dessus de 0,12 / 1,04.
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

func objectiveWeights() map[string]float64 {
	return skill.WeightsForChain(skill.LUSRChainArenaObjectif)
}

// Sur un profil sans la métrique d'objectif (chaînes de combat), la variante ne s'applique
// pas : l'appelant garde la note actuelle.
func TestObjectiveBlend_ProfilSansObjectifNonApplicable(t *testing.T) {
	cur := historyRow{Kills: 10, Deaths: 5, TimePlayedSeconds: 600}
	if _, ok := computeObjectiveBlend(&cur, blendHistory(30), skill.WeightsForChain(skill.LUSRChainArenaSlayer), defaultObjectiveBlendParams()); ok {
		t.Fatal("ok = true sur le profil de combat, want false")
	}
	if got := computeObjectiveBlendedScore(&cur, blendHistory(30), skill.RelativeWeights, defaultObjectiveBlendParams()); got != nil {
		t.Fatalf("score = %v, want nil", *got)
	}
}

func TestObjectiveBlend_HistoriqueTropCourt(t *testing.T) {
	cur := historyRow{Kills: 10, Deaths: 5, TimePlayedSeconds: 600,
		ObjectiveBlend: &objectiveBlendFacts{Family: narrative.FamilyCTF, Index: 0.5, Share: 0.5}}
	if got := computeObjectiveBlendedScore(&cur, blendHistory(MinMatchesForRelative-1), objectiveWeights(), defaultObjectiveBlendParams()); got != nil {
		t.Fatalf("score = %v sous le minimum d'historique, want nil", *got)
	}
}

// Métrique absente (Halo 5, match sans bloc d'objectif) : la note est le combat renormalisé,
// sans panique, et rien n'est compté à zéro d'objectif.
func TestObjectiveBlend_MetriqueAbsenteEstLeCombatRenormalise(t *testing.T) {
	hist := blendHistory(30)
	cur := historyRow{Kills: 12, Deaths: 6, Assists: 2, TimePlayedSeconds: 600, PersonalScore: 1700, DamageDealt: 3400}
	res, ok := computeObjectiveBlend(&cur, hist, objectiveWeights(), defaultObjectiveBlendParams())
	if !ok || res.HasObjective {
		t.Fatalf("res = %+v ok=%v, want métrique absente", res, ok)
	}
	want, _ := weightedPercentileMean(&cur, hist, profileWithoutObjective(objectiveWeights()))
	if math.Abs(res.Combat-want) > 1e-9 || res.Score != roundPerfScore(want) {
		t.Fatalf("Score = %v Combat = %v, want combat renormalisé %v", res.Score, res.Combat, want)
	}
	// Historique sans aucun fait d'objectif : même résultat, sans panique.
	for i := range hist {
		hist[i].ObjectiveBlend = nil
	}
	cur.ObjectiveBlend = &objectiveBlendFacts{Family: narrative.FamilyCTF, Index: 1, Share: 1}
	if res, ok := computeObjectiveBlend(&cur, hist, objectiveWeights(), defaultObjectiveBlendParams()); !ok || res.HasObjective {
		t.Fatalf("référence vide : %+v ok=%v, want métrique absente", res, ok)
	}
}

// Le profil sans objectif retire bien la clé et rien d'autre.
func TestProfileWithoutObjective(t *testing.T) {
	w := objectiveWeights()
	got := profileWithoutObjective(w)
	if _, in := got[MetricKeyObjectiveParticipation]; in {
		t.Fatal("la métrique d'objectif est restée au profil")
	}
	if len(got) != len(w)-1 {
		t.Fatalf("len = %d, want %d", len(got), len(w)-1)
	}
	if _, in := w[MetricKeyObjectiveParticipation]; !in {
		t.Fatal("le profil d'origine a été modifié")
	}
}

// α = AlphaMin + (AlphaMax − AlphaMin) × part ; la part est bornée à [0, 1].
func TestObjectiveBlend_AlphaSuitLaPart(t *testing.T) {
	hist := blendHistory(30)
	p := defaultObjectiveBlendParams()
	for _, c := range []struct{ share, alpha float64 }{{0, 0.12}, {0.5, 0.31}, {1, 0.50}, {7, 0.50}, {-1, 0.12}} {
		cur := historyRow{Kills: 10, Deaths: 6, TimePlayedSeconds: 600, PersonalScore: 1600, DamageDealt: 3300,
			ObjectiveBlend: &objectiveBlendFacts{Family: narrative.FamilyCTF, Index: 0.4, Share: c.share}}
		res, ok := computeObjectiveBlend(&cur, hist, objectiveWeights(), p)
		if !ok || !res.HasObjective || math.Abs(res.Alpha-c.alpha) > 1e-9 {
			t.Errorf("part %v : %+v ok=%v, want alpha %v", c.share, res, ok, c.alpha)
		}
	}
	// Sensibilité : AlphaMax 0,65.
	p.AlphaMax = 0.65
	cur := historyRow{Kills: 10, Deaths: 6, TimePlayedSeconds: 600, PersonalScore: 1600, DamageDealt: 3300,
		ObjectiveBlend: &objectiveBlendFacts{Family: narrative.FamilyCTF, Index: 0.4, Share: 1}}
	if res, _ := computeObjectiveBlend(&cur, hist, objectiveWeights(), p); math.Abs(res.Alpha-0.65) > 1e-9 {
		t.Errorf("AlphaMax 0,65 : alpha = %v", res.Alpha)
	}
}

// Référence par famille, puis repli sur la fenêtre sous le minimum, puis absence.
func TestObjectiveBlend_ReferenceFamilleRepliAbsence(t *testing.T) {
	p := defaultObjectiveBlendParams()
	hist := blendHistory(30) // 15 CTF, 15 Strongholds
	cur := historyRow{Kills: 10, Deaths: 6, TimePlayedSeconds: 600, PersonalScore: 1600, DamageDealt: 3300,
		ObjectiveBlend: &objectiveBlendFacts{Family: narrative.FamilyCTF, Index: 0.4, Share: 0.4}}
	if res, _ := computeObjectiveBlend(&cur, hist, objectiveWeights(), p); !res.FamilyScope || !res.HasObjective {
		t.Fatalf("15 matchs de famille : %+v, want référence de famille", res)
	}
	// Une famille absente de la fenêtre : repli sur la chaîne.
	cur.ObjectiveBlend.Family = narrative.FamilyOddball
	if res, _ := computeObjectiveBlend(&cur, hist, objectiveWeights(), p); res.FamilyScope || !res.HasObjective {
		t.Fatalf("famille sans match : %+v, want repli sur la fenêtre", res)
	}
	// Fenêtre trop courte même en repli : métrique absente.
	short := blendHistory(MinMatchesForRelative)
	for i := 3; i < len(short); i++ {
		short[i].ObjectiveBlend = nil
	}
	if res, ok := computeObjectiveBlend(&cur, short, objectiveWeights(), p); !ok || res.HasObjective {
		t.Fatalf("3 faits d'objectif : %+v ok=%v, want absent", res, ok)
	}
}

// Le cas qui motive le lot : un porteur à 0 frag, grosse part d'objectif, gagne des points.
// Un tueur sans objectif n'en perd jamais : l'objectif ne descend pas sous le combat seul.
func TestObjectiveBlend_PorteurMonteTueurNePerdRien(t *testing.T) {
	hist := blendHistory(40)
	p := defaultObjectiveBlendParams()
	porteur := historyRow{Kills: 0, Deaths: 8, Assists: 0, TimePlayedSeconds: 600, PersonalScore: 1500, DamageDealt: 3000,
		ObjectiveScore: ptrF(190),
		ObjectiveBlend: &objectiveBlendFacts{Family: narrative.FamilyCTF, Index: 0.95, Share: 0.8}}
	fullCoverage(&porteur, 1)
	old := computeRelativePerformanceScore(&porteur, hist, objectiveWeights())
	res, ok := computeObjectiveBlend(&porteur, hist, objectiveWeights(), p)
	if old == nil || !ok || !res.HasObjective {
		t.Fatalf("old=%v res=%+v ok=%v", old, res, ok)
	}
	if res.Score <= *old+0.5 {
		t.Errorf("porteur à 0 frag : nouvelle %v, ancienne %v : attendu un gain", res.Score, *old)
	}
	tueur := historyRow{Kills: 20, Deaths: 4, Assists: 3, TimePlayedSeconds: 600, PersonalScore: 2000, DamageDealt: 4000,
		ObjectiveBlend: &objectiveBlendFacts{Family: narrative.FamilyCTF, Index: 0, Share: 0}}
	fullCoverage(&tueur, 2)
	resT, _ := computeObjectiveBlend(&tueur, hist, objectiveWeights(), p)
	if resT.Score != roundPerfScore(resT.Combat) || !resT.Floored {
		t.Errorf("tueur sans objectif : %+v, want la note du combat seul (plancher)", resT)
	}
}

// Part nulle : alpha vaut AlphaMin, et la note est celle du combat seul dès que le percentile
// objectif ne dépasse pas le combat ; sinon elle ne le dépasse que de AlphaMin x l'écart.
func TestObjectiveBlend_PartNulleDonneLeCombatSeul(t *testing.T) {
	hist := blendHistory(50)
	p := defaultObjectiveBlendParams()
	floored, above := 0, 0
	for i := 0; i < 20; i++ {
		cur := historyRow{Kills: float64(2 + i), Deaths: float64(12 - i/2), Assists: float64(i % 4), TimePlayedSeconds: 600,
			PersonalScore: float64(1200 + 60*i), DamageDealt: float64(2500 + 150*i)}
		fullCoverage(&cur, i+3)
		cur.ObjectiveBlend = &objectiveBlendFacts{Family: narrative.FamilyCTF, Index: 0.05 * float64(i), Share: 0}
		res, ok := computeObjectiveBlend(&cur, hist, objectiveWeights(), p)
		if !ok || !res.HasObjective || res.Alpha != p.AlphaMin {
			t.Fatalf("i=%d : %+v ok=%v, want alpha = AlphaMin", i, res, ok)
		}
		if res.Floored {
			floored++
			if res.Score != roundPerfScore(res.Combat) {
				t.Errorf("i=%d : plancher, note %v != combat %v", i, res.Score, res.Combat)
			}
			continue
		}
		above++
		want := roundPerfScore(res.Combat + p.AlphaMin*(res.Percentile-res.Combat))
		if res.Score != want || res.Score < roundPerfScore(res.Combat) {
			t.Errorf("i=%d : note %v, want %v (combat %v)", i, res.Score, want, res.Combat)
		}
	}
	if floored == 0 || above == 0 {
		t.Errorf("le jeu d'essai doit couvrir les deux cas : plancher %d, au-dessus %d", floored, above)
	}
}

// L'objectif n'enlève jamais rien : la note n'est jamais sous le combat seul, pour toute part,
// tout indice et les deux modes de part.
func TestObjectiveBlend_JamaisSousLeCombat(t *testing.T) {
	hist := blendHistory(50)
	for _, support := range []bool{false, true} {
		p := defaultObjectiveBlendParams()
		p.UseSupportShare = support
		for i := 0; i < 60; i++ {
			cur := historyRow{Kills: float64(i % 17), Deaths: float64(3 + i%9), TimePlayedSeconds: 600,
				PersonalScore: float64(900 + 40*i), DamageDealt: float64(2000 + 70*i)}
			fullCoverage(&cur, i)
			cur.ObjectiveBlend = &objectiveBlendFacts{Family: narrative.FamilyCTF, Index: 0.07 * float64(i%25),
				Share: float64(i%11) / 10, SupportShare: float64(i%7) / 6}
			res, ok := computeObjectiveBlend(&cur, hist, objectiveWeights(), p)
			if !ok {
				t.Fatalf("i=%d : non calculable", i)
			}
			if res.Score < roundPerfScore(res.Combat) {
				t.Errorf("i=%d support=%v : note %v sous le combat %v", i, support, res.Score, res.Combat)
			}
		}
	}
}

// Indice brut : au-delà de 1,25 les matchs restent classés entre eux, et les égalités se
// partagent le milieu de leur plage (mi-rang).
func TestObjectiveBlend_IndiceBrutEtMiRang(t *testing.T) {
	series := []float64{1, 1.25, 1.25, 1.5, 2, 2}
	if got := percentileRankMid(1.25, series); math.Abs(got-(1+0.5*2)/6*100) > 1e-9 {
		t.Errorf("mi-rang de 1,25 = %v", got)
	}
	if lo, hi := percentileRankMid(1.5, series), percentileRankMid(2, series); !(lo < hi) {
		t.Errorf("1,5 doit se classer sous 2 (%v, %v) : plus d'égalité au plafond", lo, hi)
	}
	if got := percentileRankMid(2, series); got >= 100 {
		t.Errorf("deux égalités au sommet ne valent pas 100 : %v", got)
	}
	if got := percentileRankMid(3, []float64{1}); got != 50 {
		t.Errorf("référence d'une valeur : %v, want 50", got)
	}
	// Même rang, plus d'égalités : l'ancien rang (<=) donnait 100 à toutes les égalités du sommet.
	if percentileRank(2, series) != 100 {
		t.Fatal("percentileRank doit rester le rang <= des autres métriques")
	}
}

// Variante d'escorte : alpha suit max(part de l'objectif, part du soutien).
func TestObjectiveBlend_PartDuSoutien(t *testing.T) {
	hist := blendHistory(30)
	cur := historyRow{Kills: 8, Deaths: 6, TimePlayedSeconds: 600, PersonalScore: 1600, DamageDealt: 3300,
		ObjectiveBlend: &objectiveBlendFacts{Family: narrative.FamilyCTF, Index: 0.6, Share: 0.1, SupportShare: 0.5}}
	fullCoverage(&cur, 3)
	base := defaultObjectiveBlendParams()
	sup := base
	sup.UseSupportShare = true
	rb, _ := computeObjectiveBlend(&cur, hist, objectiveWeights(), base)
	rs, _ := computeObjectiveBlend(&cur, hist, objectiveWeights(), sup)
	if math.Abs(rb.Alpha-(0.12+0.38*0.1)) > 1e-9 || math.Abs(rs.Alpha-(0.12+0.38*0.5)) > 1e-9 {
		t.Fatalf("alpha base %v, soutien %v", rb.Alpha, rs.Alpha)
	}
	if rs.Score < rb.Score {
		t.Errorf("la part du soutien ne peut que monter la note : %v < %v", rs.Score, rb.Score)
	}
}
