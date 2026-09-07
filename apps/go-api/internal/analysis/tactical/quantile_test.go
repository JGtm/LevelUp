package tactical

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

// presque : les quantiles interpolent, et 0,95 n'a pas d'ecriture binaire exacte. Les
// ecarts que ces tests cherchent se comptent en unites, pas en 1e-9.
func presque(t *testing.T, nom string, got, attendu float64) {
	t.Helper()
	if math.Abs(got-attendu) > 1e-9 {
		t.Fatalf("%s = %v, attendu %v", nom, got, attendu)
	}
}

func cellulesDeValeurs(valeurs ...float64) []domain.CelluleTactique {
	out := make([]domain.CelluleTactique, 0, len(valeurs))
	for i, v := range valeurs {
		out = append(out, domain.CelluleTactique{Col: i, Valeur: v})
	}
	return out
}

// TestEchelle_QuantilesPoseesALaMain : onze valeurs de 1 a 11, dans le desordre.
// p50 -> rang 0,50*(11-1) = 5,0 -> la 6e valeur = 6.
// p95 -> rang 0,95*(11-1) = 9,5 -> entre 10 et 11 = 10,5.
func TestEchelle_QuantilesPoseesALaMain(t *testing.T) {
	e := Echelle(cellulesDeValeurs(7, 2, 11, 4, 1, 9, 3, 10, 5, 8, 6))

	if e.NCellules != 11 {
		t.Fatalf("NCellules = %d, attendu 11", e.NCellules)
	}
	if e.Symetrique {
		t.Fatalf("Symetrique = true, attendu false")
	}
	presque(t, "P50", e.P50, 6)
	presque(t, "P95", e.P95, 10.5)
	presque(t, "Borne", e.Borne, 10.5)
}

func TestEchelle_CasDegeneres(t *testing.T) {
	vide := Echelle(nil)
	if vide.NCellules != 0 || vide.P50 != 0 || vide.P95 != 0 || vide.Borne != 0 {
		t.Fatalf("Echelle(nil) = %+v, attendu la valeur zero", vide)
	}
	if sym := EchelleSymetrique(nil); !sym.Symetrique || sym.Borne != 0 {
		t.Fatalf("EchelleSymetrique(nil) = %+v, attendu Symetrique et Borne 0", sym)
	}

	une := Echelle(cellulesDeValeurs(4.2))
	presque(t, "P50", une.P50, 4.2)
	presque(t, "P95", une.P95, 4.2)
}

// TestEchelleSymetrique_BorneParLaValeurAbsolue : le cas qui distingue une echelle bornee
// par |valeur| d'un quantile pris sur le signe. Neuf cellules a -10, une a +1 : l'ecart
// mesure vaut 10. Un p95 des valeurs SIGNEES rendrait -3,95 — une borne negative, qui
// saturerait tout le cote victoire des la premiere cellule.
func TestEchelleSymetrique_BorneParLaValeurAbsolue(t *testing.T) {
	e := EchelleSymetrique(cellulesDeValeurs(-10, -10, -10, -10, -10, -10, -10, -10, -10, 1))

	if !e.Symetrique {
		t.Fatalf("Symetrique = false, attendu true")
	}
	if e.NCellules != 10 {
		t.Fatalf("NCellules = %d, attendu 10", e.NCellules)
	}
	presque(t, "P50", e.P50, 10)
	presque(t, "P95", e.P95, 10)
	presque(t, "Borne", e.Borne, 10)
	if e.Borne <= 0 {
		t.Fatalf("Borne = %v : une echelle symetrique bornee par un nombre negatif ou nul ne peint rien", e.Borne)
	}
}

// TestEchelleSymetrique_IndifferenteAuCote : deux lectures miroir (memes ecarts, signes
// inverses) doivent rendre la MEME borne — sans quoi le cote le plus resserre paraitrait
// plus intense a valeur egale.
func TestEchelleSymetrique_IndifferenteAuCote(t *testing.T) {
	a := EchelleSymetrique(cellulesDeValeurs(-3, -1, 2, 5, 8))
	b := EchelleSymetrique(cellulesDeValeurs(3, 1, -2, -5, -8))

	presque(t, "Borne (miroir)", a.Borne, b.Borne)
	presque(t, "P50 (miroir)", a.P50, b.P50)
	// |valeurs| triees = 1, 2, 3, 5, 8 -> p50 = 3 ; p95 -> rang 3,8 -> 5 + 0,8*3 = 7,4.
	presque(t, "P50", a.P50, 3)
	presque(t, "P95", a.P95, 7.4)
}
