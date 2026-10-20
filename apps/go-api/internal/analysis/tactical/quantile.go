package tactical

import (
	"math"
	"sort"

	"levelup/go-api/internal/domain"
)

// OrdreP50 et OrdreP95 sont les deux reperes de coloration arretes par le plan tactique :
// l'echelle des densites va du p50 vers le p95 des cellules alimentees. Le haut est un p95
// et non le maximum parce qu'une seule cellule extreme (un couloir ou tout le monde meurt)
// aplatit toutes les autres si elle borne l'echelle.
const (
	OrdreP50 = 0.50
	OrdreP95 = 0.95
)

// Echelle rend les reperes de coloration d'une lecture NON signee.
//
// Les quantiles portent sur les cellules ALIMENTEES uniquement. Il n'y a pas de zeros
// implicites a inclure : une cellule jamais atteinte n'existe pas (cf. raster.go), et
// l'ajouter au calcul tirerait tous les quantiles vers le bas jusqu'a rendre l'echelle
// illisible sur les grandes cartes, ou le terrain non joue domine en surface.
func Echelle(cellules []domain.CelluleTactique) domain.EchelleTactique {
	valeurs := valeursTriees(cellules, false)
	e := domain.EchelleTactique{NCellules: len(valeurs)}
	if len(valeurs) == 0 {
		return e
	}
	e.P50 = quantile(valeurs, OrdreP50)
	e.P95 = quantile(valeurs, OrdreP95)
	e.Borne = e.P95
	return e
}

// EchelleSymetrique rend les reperes d'une lecture SIGNEE (« ou je gagne ») : l'echelle va
// de -Borne a +Borne, avec Borne = p95 de la VALEUR ABSOLUE.
//
// POURQUOI JAMAIS UN QUANTILE SUR LE SIGNE : un p95 des valeurs signees mesure la
// proportion de cellules favorables, pas l'intensite de l'ecart. Sur une lecture ou 80 %
// des cellules penchent du cote defaite, ce p95 tomberait dans les valeurs faiblement
// positives et saturerait tout le cote victoire des la premiere cellule. La valeur absolue
// mesure l'ecart, quel que soit le cote ou il penche.
//
// POURQUOI L'ECHELLE EST SYMETRIQUE : deux ecarts de meme ampleur doivent se peindre avec
// la meme intensite. Une echelle bornee separement de chaque cote ferait paraitre le cote
// le plus resserre plus intense a valeur egale — une illusion de lecture, pas une mesure.
func EchelleSymetrique(cellules []domain.CelluleTactique) domain.EchelleTactique {
	valeurs := valeursTriees(cellules, true)
	e := domain.EchelleTactique{Symetrique: true, NCellules: len(valeurs)}
	if len(valeurs) == 0 {
		return e
	}
	e.P50 = quantile(valeurs, OrdreP50)
	e.P95 = quantile(valeurs, OrdreP95)
	e.Borne = e.P95
	return e
}

// valeursTriees extrait les valeurs des cellules, en absolu si demande, et les trie.
func valeursTriees(cellules []domain.CelluleTactique, absolu bool) []float64 {
	valeurs := make([]float64, 0, len(cellules))
	for _, c := range cellules {
		v := c.Valeur
		if absolu {
			v = math.Abs(v)
		}
		valeurs = append(valeurs, v)
	}
	sort.Float64s(valeurs)
	return valeurs
}

// quantile rend le quantile d'ordre q d'une serie TRIEE non vide, par interpolation
// lineaire entre les deux rangs encadrants — meme convention que
// `analysis/temporal.quantileSorted`, pour que deux mesures du depot qui disent « p95 »
// disent la meme chose.
func quantile(triees []float64, q float64) float64 {
	n := len(triees)
	if n == 1 {
		return triees[0]
	}
	pos := q * float64(n-1)
	bas := int(pos)
	if bas >= n-1 {
		return triees[n-1]
	}
	return triees[bas] + (pos-float64(bas))*(triees[bas+1]-triees[bas])
}
