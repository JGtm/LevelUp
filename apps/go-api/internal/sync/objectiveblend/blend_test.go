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

// Référence de famille, puis repli sur la fenêtre sous le minimum.
func TestCompute_ReferenceFamillePuisRepli(t *testing.T) {
	p := DefaultParams(10)
	hist := history(30) // 30 Drapeau
	if r := Compute(60, &Facts{Family: narrative.FamilyCTF, Index: 1, Share: 0.5}, hist, p); !r.HasObjective || !r.FamilyScope {
		t.Errorf("famille : %+v", r)
	}
	if r := Compute(60, &Facts{Family: narrative.FamilyOddball, Index: 1, Share: 0.5}, hist, p); !r.HasObjective || r.FamilyScope {
		t.Errorf("repli sur la fenêtre : %+v", r)
	}
}

// La note est la moyenne symétrique ; alpha_obj suit la part ; O_obj et O_sout sont les moyennes de
// deux rangs ; l'objectif peut monter OU baisser la note.
func TestCompute_Formule(t *testing.T) {
	p := DefaultParams(10)
	hist := history(40)
	cur := &Facts{Family: narrative.FamilyCTF, Index: 0.1, Share: 0.2, SupportShare: 0.4, SupportPerMin: 1.5}
	r := Compute(60, cur, hist, p)
	idx := rank(0.1, hist, func(f *Facts) float64 { return f.Index })
	share := rank(0.2, hist, func(f *Facts) float64 { return f.Share })
	perMin := rank(1.5, hist, func(f *Facts) float64 { return f.SupportPerMin })
	supShare := rank(0.4, hist, func(f *Facts) float64 { return f.SupportShare })
	if math.Abs(r.Percentile-(idx+share)/2) > 1e-9 || math.Abs(r.SupportPercentile-(perMin+supShare)/2) > 1e-9 {
		t.Fatalf("O_obj %v O_sout %v ; rangs %v %v %v %v", r.Percentile, r.SupportPercentile, idx, share, perMin, supShare)
	}
	if math.Abs(r.Alpha-(0.12+0.38*0.2)) > 1e-9 || math.Abs(r.SupportAlpha-0.38*0.4) > 1e-9 {
		t.Fatalf("alphas %v / %v", r.Alpha, r.SupportAlpha)
	}
	want := (1-r.Alpha-r.SupportAlpha)*60 + r.Alpha*r.Percentile + r.SupportAlpha*r.SupportPercentile
	if math.Abs(r.Score-want) > 1e-9 {
		t.Errorf("note %v, want %v", r.Score, want)
	}
	hi := Compute(30, &Facts{Family: narrative.FamilyCTF, Index: 9, Share: 1, SupportShare: 1, SupportPerMin: 9}, hist, p)
	lo := Compute(90, &Facts{Family: narrative.FamilyCTF, Index: 0, Share: 0, SupportShare: 0, SupportPerMin: 0}, hist, p)
	if !(hi.Score > 30) || !(lo.Score < 90) {
		t.Errorf("hausse %v, baisse %v : attendu les deux sens", hi.Score, lo.Score)
	}
}

// Sans action de soutien dans l'équipe, alpha_sout est nul : la note ne dépend que de l'objectif.
func TestCompute_SansSoutienPasDePoidsDuSoutien(t *testing.T) {
	r := Compute(60, &Facts{Family: narrative.FamilyCTF, Index: 0.6, Share: 0.3, SupportShare: 0, SupportPerMin: 0}, history(40), DefaultParams(10))
	if r.SupportAlpha != 0 {
		t.Errorf("alpha soutien = %v, want 0", r.SupportAlpha)
	}
}

// alpha_obj + alpha_sout est plafonné à AlphaMax, les deux réduits en proportion : le combat garde
// au moins la moitié du poids.
func TestCompute_PlafonneLaSomme(t *testing.T) {
	p := DefaultParams(10)
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
