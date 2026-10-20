package mapdecoupe

// oracle_test.go — LES MESURES DU DÉCOUPAGE, et les seuils qui les jugent.
//
// DEUX ORACLES, tous deux EXTÉRIEURS à la chaîne mesurée — c'est ce qui les empêche d'être
// tautologiques :
//
//  1. le découpé du POC sur Ridgeline (accord par zone, IoU). Il a été produit par une
//     autre chaîne, sur une autre donnée (les emprises de map_structure, avec l'altitude
//     donc le test d'ÉTAGE) et à un autre pas (5 cm contre 9,2). Un accord élevé dit que la
//     chaîne universelle retrouve, sans altitude, ce que le découpage par étage trouvait.
//  2. les positions RÉELLEMENT jouées (oracle_positions_test.go) : un joueur qui a couru
//     quelque part avait du sol sous les pieds, et il était dans une zone nommée.
//
// SEUILS FIXÉS AVANT LA MESURE (plan du 2026-08-16) : IoU médian >= 0,85 sur les
// 11 grandes zones de Ridgeline ; part des positions dans une zone qui ne baisse pas de plus
// de 2 points entre brut et découpé.

import (
	"sort"
	"testing"
)

// SeuilIoUMedian : l'accord minimal exigé avec le découpé du POC. Fixé par le plan AVANT la
// mesure — un seuil qu'on rabaisse après coup ne mesure plus rien.
const SeuilIoUMedian = 0.85

func TestOracleIoUContreLeDecoupePOC(t *testing.T) {
	c := ouvreCorpus(t)
	m := c.masque(t, moduleRidgeline, ToleranceParDefaut)
	if m == nil {
		t.Skip("fond de Ridgeline non publié")
	}
	if _, err := c.cat.Lookup(moduleRidgeline); err != nil {
		t.Skipf("Ridgeline absente du catalogue : %v", err)
	}
	ious := c.accordRidgeline(t, m, true)
	med := medianeF(ious)
	t.Logf("IoU médian = %.3f sur %d grandes zones — seuil %.2f", med, len(ious), SeuilIoUMedian)
	for _, rayon := range []float64{0, 0.25, 1.00, 2.00, 3.00, 4.00} {
		autre := c.masque(t, moduleRidgeline, rayon)
		t.Logf("  contrôle : rayon de fermeture %.2f m -> IoU médian %.3f, masque dur %.1f%% du cadre",
			rayon, medianeF(c.accordRidgeline(t, autre, false)), 100*autre.PartDure())
	}
	if len(ious) == 0 {
		t.Fatal("aucune grande zone mesurée : l'oracle ne dit rien")
	}
	if med < SeuilIoUMedian {
		t.Errorf("IoU médian %.3f sous le seuil %.2f", med, SeuilIoUMedian)
	}
}

// accordRidgeline mesure l'accord zone par zone avec le découpé du POC.
//
// La colonne « gardée » se lit à DEUX : la part que la chaîne universelle conserve, et celle
// que le POC conservait. C'est elle qui dit le SENS du désaccord — rogner moins que le
// découpage par étage n'est pas la même faute que rogner plus.
func (c corpus) accordRidgeline(t *testing.T, m *Masque, detail bool) []float64 {
	t.Helper()
	opts := OptionsParDefaut()
	var ious []float64
	if detail {
		t.Logf("%-24s %5s %-8s %8s %11s %9s %9s", "zone", "vol", "POC", "IoU", "aire brute", "gardée", "gardée POC")
	}
	for _, z := range c.cat.Maps[moduleRidgeline].Zones {
		if !z.Big {
			continue
		}
		d, ok := c.dump[z.VolumeIndex]
		if !ok || len(d.Brut.Polygone) < 3 {
			t.Fatalf("volume %d (%s) absent du dump du POC", z.VolumeIndex, z.EN)
		}
		r := Decoupe(d.Brut.Polygone, m, opts)
		v := iou(m, d.anneaux(), figureDe(r, d.Brut.Polygone))
		ious = append(ious, v)
		if !detail {
			continue
		}
		poc := aireFigure(m, d.anneaux())
		t.Logf("%-24s %5d %-8s %8.3f %9.1f m² %8.1f%% %8.1f%%",
			z.EN, z.VolumeIndex, d.Utiliser, v, r.AireBrutM2,
			100*r.PartGardee(), 100*poc/r.AireBrutM2)
	}
	return ious
}

// aireFigure rend l'aire d'une figure mesurée sur la grille du fond, en m².
func aireFigure(m *Masque, fig [][][2]float64) float64 {
	e, ok := m.emprise(fig)
	if !ok {
		return 0
	}
	return float64(compteVrai(m.rasterise(fig, e))) * m.CelluleM2()
}

// TestOracleZonesDegenerees publie les zones que le découpage ne peut pas servir : celles
// qui n'ont pas 1 m² de décor sous elles.
//
// Ce n'est pas une statistique de contrôle, c'est une LISTE À REGARDER : chaque nom qui sort
// ici est soit un trou du fond reconstruit, soit une zone que le jeu déclare au-dessus du
// vide. Aucune n'est écrasée — elles gardent leur pavé brut.
func TestOracleZonesDegenerees(t *testing.T) {
	c := ouvreCorpus(t)
	opts := OptionsParDefaut()
	totalZones, totalDecoupees, totalDegenerees, totalSommets := 0, 0, 0, 0
	for _, module := range c.modules {
		if module == moduleRidgeline {
			continue // servie par le dump du POC, pas par cette chaîne
		}
		m := c.masque(t, module, ToleranceParDefaut)
		zones, decoupees, sommets := 0, 0, 0
		var gardees []float64
		for _, z := range c.cat.Maps[module].Zones {
			brut := c.brutDe(module, z)
			if len(brut) < 3 {
				continue
			}
			zones++
			r := Decoupe(brut, m, opts)
			if r.Degenere {
				totalDegenerees++
				t.Logf("DÉGÉNÉRÉE %-18s %-26s vol %3d : %7.1f m² -> %5.1f m²",
					module, z.EN, z.VolumeIndex, r.AireBrutM2, r.AireM2)
				continue
			}
			decoupees++
			gardees = append(gardees, r.PartGardee())
			sommets += len(r.Contour)
			for _, p := range r.Parties {
				sommets += len(p)
			}
			for _, h := range r.Trous {
				sommets += len(h)
			}
		}
		totalZones += zones
		totalDecoupees += decoupees
		totalSommets += sommets
		t.Logf("%-18s zones=%3d découpées=%3d gardée(méd)=%5.1f%% sommets=%5d",
			module, zones, decoupees, 100*medianeF(gardees), sommets)
	}
	t.Logf("TOTAL : %d zones à forme, %d découpées, %d dégénérées (gardent le brut), %d sommets publiés",
		totalZones, totalDecoupees, totalDegenerees, totalSommets)
	if totalDecoupees == 0 {
		t.Fatal("aucune zone découpée : la chaîne ne produit rien")
	}
}

// figureDe rend la figure servie pour une zone : le découpage s'il tient, le brut sinon.
func figureDe(r Resultat, brut [][2]float64) [][][2]float64 {
	if r.Degenere {
		return [][][2]float64{brut}
	}
	out := [][][2]float64{r.Contour}
	out = append(out, r.Parties...)
	return append(out, r.Trous...)
}

// medianeF rend la médiane d'un échantillon (0 si vide), sans toucher à l'ordre d'entrée.
func medianeF(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[len(c)/2]
}
