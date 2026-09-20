package fusion

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/analysis/tactical"
)

func TestAgregeCellulesPrendLeMeilleurEtage(t *testing.T) {
	noeuds := []NoeudGeo{
		{Col: 1, Lig: 2, Z: 0, Score: 0.3},
		{Col: 1, Lig: 2, Z: 4.4, Score: 0.6},
		{Col: 0, Lig: 0, Z: 1, Score: 0.1},
	}
	out := AgregeCellules(noeuds)
	if len(out) != 2 {
		t.Fatalf("2 cellules attendues, %d", len(out))
	}
	if out[0].Col != 0 || out[1].Col != 1 {
		t.Fatalf("tri par adresse attendu : %+v", out)
	}
	c := out[1]
	if c.Score != 0.6 || c.NbNoeuds != 2 || c.ZMin != 0 || c.ZMax != 4.4 {
		t.Fatalf("agregation fausse : %+v", c)
	}
}

func scoree(col, lig int, score float64) powerpos.CelluleScoree {
	g := tactical.GrilleParDefaut()
	x, y := g.Centre(tactical.Cellule{Col: col, Lig: lig})
	return powerpos.CelluleScoree{
		Cellule: powerpos.Cellule{Col: col, Lig: lig, CentreX: x, CentreY: y, KillsDepuis: 3, MatchsKills: 4},
		Score:   score,
	}
}

func reglageTest(a float64) Reglage {
	return Reglage{PoidsGeo: a, PoidsEmp: 1 - a, EmpAbsent: 0.5, GeoAbsent: 0.25}
}

// Vingt et une cellules geometriques a scores etales (la normalisation p5..p95 a besoin
// d'une serie), dont une seule est aussi empirique.
func serieGeo() []CelluleGeo {
	var out []CelluleGeo
	for i := 0; i <= 20; i++ {
		out = append(out, CelluleGeo{Col: i, Lig: 0, Score: float64(i) / 20, ZMin: 1, ZMax: 1})
	}
	return out
}

func serieEmp() []powerpos.CelluleScoree {
	var out []powerpos.CelluleScoree
	for i := 0; i <= 20; i++ {
		out = append(out, scoree(i, 5, 0.5+float64(i)/100))
	}
	return append(out, scoree(20, 0, 0.7))
}

func TestFusionneAbsencesExplicites(t *testing.T) {
	g := tactical.GrilleParDefaut()
	cellules := Fusionne(g, serieGeo(), serieEmp(), reglageTest(0.6))
	if len(cellules) != 21+21 {
		t.Fatalf("21 geo + 21 emp seules attendues (une commune), %d", len(cellules))
	}
	index := map[tactical.Cellule]Cellule{}
	for _, c := range cellules {
		index[tactical.Cellule{Col: c.Col, Lig: c.Lig}] = c
	}
	// Geometrie seule : emp_norm vaut EmpAbsent, le centre vient de la grille.
	seule := index[tactical.Cellule{Col: 10, Lig: 0}]
	if !seule.GeoPresent || seule.EmpPresent || seule.EmpNorm != 0.5 {
		t.Fatalf("cellule geo seule mal fusionnee : %+v", seule)
	}
	if seule.CentreX != 5.25 || seule.CentreY != 0.25 {
		t.Fatalf("centre attendu (5.25, 0.25), obtenu (%v, %v)", seule.CentreX, seule.CentreY)
	}
	if math.Abs(seule.Score-(0.6*seule.GeoNorm+0.4*0.5)) > 1e-9 {
		t.Fatalf("score geo seule faux : %+v", seule)
	}
	// Empirique seule : geo_norm vaut GeoAbsent, les comptes suivent.
	emp := index[tactical.Cellule{Col: 3, Lig: 5}]
	if emp.GeoPresent || !emp.EmpPresent || emp.GeoNorm != 0.25 || emp.KillsDepuis != 3 {
		t.Fatalf("cellule emp seule mal fusionnee : %+v", emp)
	}
	// Commune : les deux presentes, comptes empiriques portes, Z geometrique porte.
	commune := index[tactical.Cellule{Col: 20, Lig: 0}]
	if !commune.GeoPresent || !commune.EmpPresent || commune.KillsDepuis != 3 || commune.ZMax != 1 {
		t.Fatalf("cellule commune mal fusionnee : %+v", commune)
	}
	if commune.GeoNorm != 1 {
		t.Fatalf("le p95 et au-dela doit valoir 1 apres normalisation, obtenu %v", commune.GeoNorm)
	}
	if cellules[0].Score < cellules[len(cellules)-1].Score {
		t.Fatal("tri par score decroissant attendu")
	}
}

func TestFusionnePoidsExtremes(t *testing.T) {
	g := tactical.GrilleParDefaut()
	geoSeul := Fusionne(g, serieGeo(), serieEmp(), reglageTest(1))
	for _, c := range geoSeul {
		if math.Abs(c.Score-c.GeoNorm) > 1e-9 {
			t.Fatalf("a = 1 : le score doit etre geo_norm, %+v", c)
		}
	}
	empSeul := Fusionne(g, serieGeo(), serieEmp(), reglageTest(0))
	for _, c := range empSeul {
		if math.Abs(c.Score-c.EmpNorm) > 1e-9 {
			t.Fatalf("a = 0 : le score doit etre emp_norm, %+v", c)
		}
	}
}

func TestSelectionneEtBilan(t *testing.T) {
	g := tactical.GrilleParDefaut()
	// Un pave de 4 x 4 cellules geometriques fortes, le reste faible ; aucune empirique.
	var geoC []CelluleGeo
	for col := 0; col < 12; col++ {
		for lig := 0; lig < 12; lig++ {
			s := 0.1
			if col < 4 && lig < 4 {
				s = 0.9
			}
			geoC = append(geoC, CelluleGeo{Col: col, Lig: lig, Score: s, ZMin: 2, ZMax: 2})
		}
	}
	r := Reglage{PoidsGeo: 0.5, PoidsEmp: 0.5, EmpAbsent: 0.5, GeoAbsent: 0,
		Selection: powerpos.Reglage{QuantileAmorce: 0.95, QuantileCroissance: 0.90,
			FermetureRayonCellules: 1, Connexite8: true, TailleMiniComposante: 10, MaxComposantes: 8}}
	cellules := Fusionne(g, geoC, nil, r)
	positions := Selectionne(g, cellules, r)
	if len(positions) != 1 {
		t.Fatalf("une position attendue, %d", len(positions))
	}
	p := positions[0]
	if p.NbCellulesMesurees != 16 {
		t.Fatalf("16 cellules mesurees attendues, %d", p.NbCellulesMesurees)
	}
	b := BilanDe(p, cellules)
	if b.SansEmp != 16 || b.SansGeo != 0 || b.ZMin != 2 || b.ZMax != 2 {
		t.Fatalf("bilan faux : %+v", b)
	}
	if math.Abs(b.EmpMoyen-0.5) > 1e-9 || b.GeoMoyen != 1 {
		t.Fatalf("moyennes fausses : %+v", b)
	}
}

func TestReglageFusionV1EstCoherent(t *testing.T) {
	r := ReglageFusionV1()
	if math.Abs(r.PoidsGeo+r.PoidsEmp-1) > 1e-9 {
		t.Fatalf("les poids doivent sommer a 1 : %+v", r)
	}
	if r.Selection.QuantileAmorce <= 0 || r.Selection.QuantileCroissance > r.Selection.QuantileAmorce {
		t.Fatalf("hysteresis incoherente : %+v", r.Selection)
	}
	if r.Selection.TailleMiniComposante <= 0 || r.Selection.MaxComposantes <= 0 {
		t.Fatalf("selection sans borne : %+v", r.Selection)
	}
}
