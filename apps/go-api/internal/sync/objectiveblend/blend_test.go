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
	if r := Compute(62.5, nil, history(30), p); r.hasObjective || r.Score != 62.5 {
		t.Errorf("sans faits : %+v", r)
	}
	cur := &Facts{Family: narrative.FamilyCTF, Index: 1, Share: 0.5}
	if r := Compute(62.5, cur, history(9), p); r.hasObjective || r.Score != 62.5 {
		t.Errorf("référence trop courte : %+v", r)
	}
	if r := Compute(62.5, cur, []*Facts{nil, nil}, p); r.hasObjective {
		t.Errorf("historique sans faits : %+v", r)
	}
}

// Référence de famille, puis repli sur la fenêtre sous le minimum.
func TestCompute_ReferenceFamillePuisRepli(t *testing.T) {
	p := DefaultParams(10)
	hist := history(30) // 30 Drapeau
	if r := Compute(60, &Facts{Family: narrative.FamilyCTF, Index: 1, Share: 0.5}, hist, p); !r.hasObjective || !r.familyScope {
		t.Errorf("famille : %+v", r)
	}
	if r := Compute(60, &Facts{Family: narrative.FamilyOddball, Index: 1, Share: 0.5}, hist, p); !r.hasObjective || r.familyScope {
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
	if math.Abs(r.percentile-(idx+share)/2) > 1e-9 || math.Abs(r.supportPercentile-(perMin+supShare)/2) > 1e-9 {
		t.Fatalf("O_obj %v O_sout %v ; rangs %v %v %v %v", r.percentile, r.supportPercentile, idx, share, perMin, supShare)
	}
	if math.Abs(r.alpha-(0.12+0.38*0.2)) > 1e-9 || math.Abs(r.supportAlpha-0.38*0.4) > 1e-9 {
		t.Fatalf("alphas %v / %v", r.alpha, r.supportAlpha)
	}
	want := (1-r.alpha-r.supportAlpha)*60 + r.alpha*r.percentile + r.supportAlpha*r.supportPercentile
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
	if r.supportAlpha != 0 {
		t.Errorf("alpha soutien = %v, want 0", r.supportAlpha)
	}
}

// alpha_obj + alpha_sout est plafonné à AlphaMax, les deux réduits en proportion : le combat garde
// au moins la moitié du poids.
func TestCompute_PlafonneLaSomme(t *testing.T) {
	p := DefaultParams(10)
	r := Compute(60, &Facts{Family: narrative.FamilyCTF, Index: 0.6, Share: 1, SupportShare: 1, SupportPerMin: 1.5}, history(40), p)
	if math.Abs(r.alpha+r.supportAlpha-p.AlphaMax) > 1e-9 || math.Abs(r.alpha/r.supportAlpha-0.50/0.38) > 1e-9 {
		t.Fatalf("alphas %v / %v", r.alpha, r.supportAlpha)
	}
	for i := 0; i < 60; i++ {
		c := &Facts{Family: narrative.FamilyCTF, Index: 0.05 * float64(i%30), Share: float64(i%11) / 10, SupportShare: float64(i%7) / 6, SupportPerMin: 0.1 * float64(i%25)}
		if r := Compute(60, c, history(40), p); r.alpha+r.supportAlpha > p.AlphaMax+1e-9 {
			t.Errorf("i=%d : somme %v au-dessus du plafond", i, r.alpha+r.supportAlpha)
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

// Les seuils sont EXACTS (>=) : au minimum de matchs de la famille, la référence est la famille ; un
// de moins, repli sur la fenêtre, qui exige le même minimum ; un de moins encore, métrique absente.
func TestCompute_SeuilsExactsDeLaReference(t *testing.T) {
	p := DefaultParams(10)
	cur := &Facts{Family: narrative.FamilyCTF, Index: 1, Share: 0.5, SupportShare: 0.5, SupportPerMin: 1}
	other := func(n int) []*Facts {
		out := history(n)
		for _, f := range out {
			f.Family = narrative.FamilyZonesKOTH
		}
		return out
	}
	if r := Compute(60, cur, history(10), p); !r.hasObjective || !r.familyScope {
		t.Errorf("10 matchs de la famille : %+v, want la référence de famille", r)
	}
	repli := append(history(9), other(1)...)
	if r := Compute(60, cur, repli, p); !r.hasObjective || r.familyScope {
		t.Errorf("9 de la famille + 1 autre : %+v, want le repli sur la fenêtre (10 matchs)", r)
	}
	if r := Compute(60, cur, history(9), p); r.hasObjective {
		t.Errorf("9 matchs au total : %+v, want la métrique absente", r)
	}
}

// En repli sur les autres familles, le débit de soutien par minute n'est pas classé : O_sout ne
// garde que le rang de la part du soutien. Dans la famille, le débit compte.
func TestCompute_RepliSoutienIgnoreLeDebitParMinute(t *testing.T) {
	p := DefaultParams(10)
	hist := history(40) // 40 Drapeau
	supShare := rank(0.4, hist, func(f *Facts) float64 { return f.SupportShare })
	rate := func(v float64) *Facts {
		return &Facts{Family: narrative.FamilyCTF, Index: 0.3, Share: 0.3, SupportShare: 0.4, SupportPerMin: v}
	}
	famLo, famHi := Compute(60, rate(0), hist, p), Compute(60, rate(9), hist, p)
	if !famLo.familyScope || famLo.supportPercentile == famHi.supportPercentile {
		t.Errorf("dans la famille le débit doit compter : %+v / %+v", famLo, famHi)
	}
	other := func(v float64) *Facts {
		f := rate(v)
		f.Family = narrative.FamilyOddball
		return f
	}
	repLo, repHi := Compute(60, other(0), hist, p), Compute(60, other(9), hist, p)
	if repLo.familyScope || repHi.familyScope {
		t.Fatalf("le repli était attendu : %+v / %+v", repLo, repHi)
	}
	if repLo.supportPercentile != repHi.supportPercentile || math.Abs(repLo.supportPercentile-supShare) > 1e-9 {
		t.Errorf("en repli O_sout = rang de la part seul : %v / %v, want %v", repLo.supportPercentile, repHi.supportPercentile, supShare)
	}
	if repLo.Score != repHi.Score {
		t.Errorf("le débit ne doit pas bouger la note en repli : %v / %v", repLo.Score, repHi.Score)
	}
}
