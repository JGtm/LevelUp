package tactical

// placement_test.go — le placement d'une mort (seul / près) : appariement du contexte à
// ± TolerancePlacementMs, puis comparaison à la portée du radar du match.

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func metres(v float64) *float64 { return &v }

func ctxMort(match, victime string, t int64, d *float64) domain.ContexteDeMort {
	return domain.ContexteDeMort{MatchID: match, VictimXUID: victime, TimeMs: t, PlusProcheM: d, Visibles: 1}
}

func TestContexteLePlusProche_ToleranceEtVictime(t *testing.T) {
	ctxs := []domain.ContexteDeMort{
		ctxMort("m1", "v1", 10_000, metres(4)),
		ctxMort("m1", "v2", 11_000, metres(9)), // autre victime, plus proche dans le temps
		ctxMort("m2", "v1", 11_000, metres(7)), // autre match
	}
	if c := ContexteLePlusProche(ctxs, "m1", "v1", 11_500); c == nil || *c.PlusProcheM != 4 {
		t.Fatalf("à 1 500 ms (borne comprise) : attendu le contexte de v1 dans m1, obtenu %+v", c)
	}
	if c := ContexteLePlusProche(ctxs, "m1", "v1", 11_501); c != nil {
		t.Fatalf("à 1 501 ms : attendu aucun contexte, obtenu %+v", c)
	}
	if c := ContexteLePlusProche(ctxs, "m1", "v1", 8_500); c == nil {
		t.Fatal("à −1 500 ms (borne comprise) : attendu le contexte")
	}
}

func TestContexteLePlusProche_LePlusProcheDansLeTemps(t *testing.T) {
	ctxs := []domain.ContexteDeMort{
		ctxMort("m1", "v1", 10_000, metres(4)),
		ctxMort("m1", "v1", 10_900, metres(12)),
	}
	if c := ContexteLePlusProche(ctxs, "m1", "v1", 10_600); c == nil || *c.PlusProcheM != 12 {
		t.Fatalf("attendu le contexte à 300 ms, obtenu %+v", c)
	}
	// Égalité d'écart : le plus ancien.
	if c := ContexteLePlusProche(ctxs, "m1", "v1", 10_450); c == nil || *c.PlusProcheM != 4 {
		t.Fatalf("égalité d'écart : attendu le plus ancien, obtenu %+v", c)
	}
}

func TestPlacementDeLaMort(t *testing.T) {
	sansVisible := ctxMort("m1", "v1", 0, nil)
	cases := []struct {
		nom      string
		ctx      *domain.ContexteDeMort
		rayon    float64
		aRayon   bool
		nil_     bool
		seul     bool
		distance *float64
	}{
		{nom: "sans contexte", ctx: nil, rayon: 18, aRayon: true, nil_: true},
		{nom: "aucun coéquipier visible", ctx: &sansVisible, rayon: 18, aRayon: true, seul: true},
		{nom: "aucun visible, portée inconnue", ctx: &sansVisible, aRayon: false, seul: true},
		{nom: "à la portée = près", ctx: ptrCtx(ctxMort("m1", "v1", 0, metres(18))), rayon: 18, aRayon: true, distance: metres(18)},
		{nom: "au-delà = seul", ctx: ptrCtx(ctxMort("m1", "v1", 0, metres(18.01))), rayon: 18, aRayon: true, seul: true, distance: metres(18.01)},
		{nom: "distance sans portée connue", ctx: ptrCtx(ctxMort("m1", "v1", 0, metres(5))), aRayon: false, nil_: true},
	}
	for _, c := range cases {
		got := PlacementDeLaMort(c.ctx, c.rayon, c.aRayon)
		if c.nil_ {
			if got != nil {
				t.Errorf("%s : attendu aucun badge, obtenu %+v", c.nom, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("%s : attendu un badge", c.nom)
			continue
		}
		if got.Seul != c.seul {
			t.Errorf("%s : Seul = %v, attendu %v", c.nom, got.Seul, c.seul)
		}
		if (got.DistanceM == nil) != (c.distance == nil) || (got.DistanceM != nil && *got.DistanceM != *c.distance) {
			t.Errorf("%s : DistanceM = %v, attendu %v", c.nom, got.DistanceM, c.distance)
		}
	}
}

func ptrCtx(c domain.ContexteDeMort) *domain.ContexteDeMort { return &c }
