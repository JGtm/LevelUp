package objectiveblend

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
)

func history(n int) []*Facts {
	out := make([]*Facts, n)
	for i := range out {
		out[i] = &Facts{Family: narrative.FamilyCTF, Index: 0.05 * float64(i%20), Share: 0.3, SupportShare: 0.3, SupportPerMin: 0.1 * float64(i%20)}
	}
	return out
}

func TestRankMid_EgalitesAuMilieu(t *testing.T) {
	series := []float64{1, 1.25, 1.25, 1.5, 2, 2}
	if got := RankMid(1.25, series); math.Abs(got-(1+0.5*2)/6*100) > 1e-9 {
		t.Errorf("mi-rang de 1,25 = %v", got)
	}
	if got := RankMid(2, series); got >= 100 {
		t.Errorf("deux égalités au sommet ne valent pas 100 : %v", got)
	}
	if got := RankMid(3, []float64{1}); got != 50 {
		t.Errorf("référence d'une valeur : %v, want 50", got)
	}
}

// Sans faits ou sans référence suffisante, la métrique est absente : la note est le combat.
func TestCompute_MetriqueAbsente(t *testing.T) {
	p := DefaultParams(10)
	if r := Compute(62.5, nil, history(30), p); r.HasObjective || r.Score != 62.5 {
		t.Errorf("sans faits : %+v", r)
	}
	cur := &Facts{Family: narrative.FamilyCTF, Index: 1, Share: 0.5}
	if r := Compute(62.5, cur, history(9), p); r.HasObjective || r.Score != 62.5 {
		t.Errorf("référence trop courte : %+v", r)
	}
	if r := Compute(62.5, cur, []*Facts{nil, nil}, p); r.HasObjective {
		t.Errorf("historique sans faits : %+v", r)
	}
}

// Moyenne symétrique V2 : alpha suit la part, la note est (1 - alpha) C + alpha O.
func TestCompute_V2Symetrique(t *testing.T) {
	p := DefaultParams(10)
	hist := history(30)
	for _, c := range []struct{ share, alpha float64 }{{0, 0.12}, {0.5, 0.31}, {1, 0.50}, {7, 0.50}, {-1, 0.12}} {
		cur := &Facts{Family: narrative.FamilyCTF, Index: 0.4, Share: c.share}
		r := Compute(60, cur, hist, p)
		if !r.HasObjective || math.Abs(r.Alpha-c.alpha) > 1e-9 || r.SupportAlpha != 0 {
			t.Fatalf("part %v : %+v, want alpha %v", c.share, r, c.alpha)
		}
		if want := (1-r.Alpha)*60 + r.Alpha*r.Percentile; math.Abs(r.Score-want) > 1e-9 {
			t.Errorf("note %v, want %v", r.Score, want)
		}
	}
	// L'objectif peut monter OU baisser la note.
	hi := Compute(30, &Facts{Family: narrative.FamilyCTF, Index: 9, Share: 1}, hist, p)
	lo := Compute(90, &Facts{Family: narrative.FamilyCTF, Index: 0, Share: 1}, hist, p)
	if !(hi.Score > 30) || !(lo.Score < 90) {
		t.Errorf("hausse %v, baisse %v", hi.Score, lo.Score)
	}
}

// V5 : alpha_obj + alpha_sout plafonné à AlphaMax, les deux réduits en proportion.
func TestCompute_V5PlafonneLaSomme(t *testing.T) {
	p := DefaultParams(10)
	p.UseSupport = true
	r := Compute(60, &Facts{Family: narrative.FamilyCTF, Index: 0.6, Share: 1, SupportShare: 1, SupportPerMin: 1.5}, history(40), p)
	if math.Abs(r.Alpha+r.SupportAlpha-p.AlphaMax) > 1e-9 || math.Abs(r.Alpha/r.SupportAlpha-0.50/0.38) > 1e-9 {
		t.Fatalf("alphas %v / %v", r.Alpha, r.SupportAlpha)
	}
	for i := 0; i < 60; i++ {
		c := &Facts{Family: narrative.FamilyCTF, Index: 0.05 * float64(i%30), Share: float64(i%11) / 10, SupportShare: float64(i%7) / 6, SupportPerMin: 0.1 * float64(i%25)}
		if r := Compute(60, c, history(40), p); r.Alpha+r.SupportAlpha > p.AlphaMax+1e-9 {
			t.Errorf("i=%d : somme %v au-dessus du plafond", i, r.Alpha+r.SupportAlpha)
		}
	}
}

// V6 puis V7 : les deux rangs, moyennés.
func TestCompute_V6V7MoyennesDeRangs(t *testing.T) {
	hist := history(40)
	cur := &Facts{Family: narrative.FamilyCTF, Index: 0.1, Share: 0.9, SupportShare: 0.9, SupportPerMin: 0}
	v5 := DefaultParams(10)
	v5.UseSupport = true
	v6, v7 := v5, v5
	v6.ShareRank = true
	v7.ShareRank, v7.SupportShareRank = true, true
	idx := rank(0.1, hist, func(f *Facts) float64 { return f.Index })
	share := rank(0.9, hist, func(f *Facts) float64 { return f.Share })
	perMin := rank(0, hist, func(f *Facts) float64 { return f.SupportPerMin })
	supShare := rank(0.9, hist, func(f *Facts) float64 { return f.SupportShare })
	r5, r6, r7 := Compute(60, cur, hist, v5), Compute(60, cur, hist, v6), Compute(60, cur, hist, v7)
	if math.Abs(r5.Percentile-idx) > 1e-9 || math.Abs(r6.Percentile-(idx+share)/2) > 1e-9 {
		t.Errorf("O_obj V5 %v V6 %v ; rangs %v et %v", r5.Percentile, r6.Percentile, idx, share)
	}
	if math.Abs(r6.SupportPercentile-perMin) > 1e-9 || math.Abs(r7.SupportPercentile-(perMin+supShare)/2) > 1e-9 {
		t.Errorf("O_sout V6 %v V7 %v ; rangs %v et %v", r6.SupportPercentile, r7.SupportPercentile, perMin, supShare)
	}
}

func TestFactsOf_SoutienParMinute(t *testing.T) {
	share := narrative.ObjectiveTeamShare{Family: narrative.FamilyCTF, RawIndex: 1.4, Share: 0.3, SupportShare: 0.5, SupportPoints: 12}
	f := FactsOf(share, 360)
	if f.Index != 1.4 || f.Share != 0.3 || f.SupportShare != 0.5 || math.Abs(f.SupportPerMin-2) > 1e-9 {
		t.Fatalf("faits = %+v, want 12 points sur 6 minutes = 2 par minute", f)
	}
	if got := FactsOf(share, 0).SupportPerMin; got != 0 {
		t.Fatalf("durée inconnue : %v par minute, want 0", got)
	}
}
