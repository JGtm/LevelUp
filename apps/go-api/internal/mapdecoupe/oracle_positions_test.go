package mapdecoupe

// oracle_positions_test.go — L'ORACLE QUI NE PEUT PAS ÊTRE TAUTOLOGIQUE.
//
// Toutes les mesures internes au découpage (aire retirée, accord avec un autre découpage)
// jugent la chaîne avec ses propres outils. Une seule vient de DEHORS : les positions
// réellement jouées, décodées des films. Un joueur qui a couru quelque part était dans une
// zone nommée — si le découpage lui retire son sol, le découpage a tort, quelle que soit la
// beauté de son contour.
//
// LE SEUIL EST FIXÉ D'AVANCE (plan du 2026-08-16) : la part des positions qui tombent dans
// une zone ne doit pas baisser de plus de 2 POINTS entre le pavé brut et le découpé.

import (
	"sort"
	"testing"
)

// SeuilPerteMaxPoints : perte de rétention tolérée, en points de pourcentage.
const SeuilPerteMaxPoints = 2.0

// filmsMinimum / cartesMinimum : la taille d'échantillon exigée par le plan.
const (
	filmsMinimum  = 5
	cartesMinimum = 3
)

func TestOraclePositionsJouees(t *testing.T) {
	c := ouvreCorpus(t)
	films := chargeFilms(t, c)
	identifie(t, c, films)
	parCarte := groupeParCarte(films)
	if len(parCarte) < cartesMinimum {
		t.Skipf("seulement %d carte(s) reconnue(s), le plan en exige %d", len(parCarte), cartesMinimum)
	}

	opts := OptionsParDefaut()
	strict := opts
	strict.RendLesEnclaves = false
	mesures, cartes := 0, 0
	t.Logf("%-10s %-18s %8s %9s %9s %9s %9s %9s",
		"film", "carte", "points", "brut", "strict", "enclaves", "écart", "s/matière")
	for _, module := range triCles(parCarte) {
		m := c.masque(t, module, ToleranceParDefaut)
		if m == nil {
			continue
		}
		gBrut := unionRaster(m, c.figureBrute(module))
		gStrict := unionRaster(m, figuresDecoupees(c, module, m, strict))
		gServi := unionRaster(m, figuresDecoupees(c, module, m, opts))
		cartes++
		for _, f := range parCarte[module] {
			brut := 100 * partDansGrille(m, gBrut, f.pts)
			str := 100 * partDansGrille(m, gStrict, f.pts)
			servi := 100 * partDansGrille(m, gServi, f.pts)
			mesures++
			// « s/matière » : la part des positions posées sur du décor publié. C'est le
			// PLAFOND de ce que le masquage STRICT peut garder — un écart qui s'en approche
			// dit que le fond a des trous, pas que le découpage est trop mordant.
			t.Logf("%-10s %-18s %8d %8.2f%% %8.2f%% %8.2f%% %+7.2f pt %8.2f%%",
				f.id, module, len(f.pts), brut, str, servi, servi-brut, 100*partSurMatiere(m, f.pts))
			if brut-servi > SeuilPerteMaxPoints {
				t.Errorf("%s (%s) : le découpage coûte %.2f points de positions jouées (max %.1f)",
					f.id, module, brut-servi, SeuilPerteMaxPoints)
			}
		}
	}
	for _, f := range films {
		if f.module == "" {
			t.Logf("écarté   %-10s %d points — %s", f.id, len(f.pts), f.ecart)
		}
	}
	t.Logf("%d films mesurés sur %d cartes", mesures, cartes)
	if mesures < filmsMinimum {
		t.Errorf("%d films mesurés, le plan en exige %d", mesures, filmsMinimum)
	}
}

// TestOracleTolerance balaie le rayon de fermeture du masque : c'est LUI qui étalonne
// ToleranceParDefaut, et le tableau qu'il imprime est la seule justification de ce nombre.
func TestOracleTolerance(t *testing.T) {
	if testing.Short() {
		t.Skip("balayage long : quatre fermetures par carte")
	}
	c := ouvreCorpus(t)
	films := chargeFilms(t, c)
	identifie(t, c, films)
	parCarte := groupeParCarte(films)
	if len(parCarte) == 0 {
		t.Skip("aucune carte reconnue")
	}
	opts := OptionsParDefaut()
	t.Logf("%-18s %9s %10s %10s %9s %12s", "carte", "rayon (m)", "brut", "découpé", "écart", "aire retirée")
	for _, module := range triCles(parCarte) {
		for _, rayon := range []float64{0, 1.00, 2.00, 3.00, 4.00, 5.00} {
			m := c.masque(t, module, rayon)
			if m == nil {
				continue
			}
			gBrut := unionRaster(m, c.figureBrute(module))
			gDec := unionRaster(m, figuresDecoupees(c, module, m, opts))
			aBrut := float64(compteVrai(gBrut)) * m.CelluleM2()
			aDec := float64(compteVrai(gDec)) * m.CelluleM2()
			var sommeBrut, sommeDec, poids float64
			for _, f := range parCarte[module] {
				n := float64(len(f.pts))
				sommeBrut += n * partDansGrille(m, gBrut, f.pts)
				sommeDec += n * partDansGrille(m, gDec, f.pts)
				poids += n
			}
			if poids == 0 {
				continue
			}
			b, d := 100*sommeBrut/poids, 100*sommeDec/poids
			t.Logf("%-18s %9.2f %9.2f%% %9.2f%% %+8.2f pt %11.1f%%",
				module, rayon, b, d, d-b, 100*(1-aDec/aBrut))
		}
	}
}

// figuresDecoupees applique la chaîne à toutes les zones d'une carte.
func figuresDecoupees(c corpus, module string, m *Masque, opts Options) [][][][2]float64 {
	var out [][][][2]float64
	for _, z := range c.cat.Maps[module].Zones {
		brut := c.brutDe(module, z)
		if len(brut) < 3 {
			continue
		}
		out = append(out, figureDe(Decoupe(brut, m, opts), brut))
	}
	return out
}

func groupeParCarte(films []film) map[string][]film {
	out := map[string][]film{}
	for _, f := range films {
		if f.module == "" {
			continue
		}
		out[f.module] = append(out[f.module], f)
	}
	return out
}

func triCles(m map[string][]film) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// partSurMatiere rend la part des positions posées sur du décor publié (fermeture comprise).
func partSurMatiere(m *Masque, pts [][3]float64) float64 {
	if len(pts) == 0 {
		return 0
	}
	n := 0
	for _, p := range pts {
		if m.Praticable(p[0], p[1]) {
			n++
		}
	}
	return float64(n) / float64(len(pts))
}

func compteVrai(g []bool) int {
	n := 0
	for _, v := range g {
		if v {
			n++
		}
	}
	return n
}
