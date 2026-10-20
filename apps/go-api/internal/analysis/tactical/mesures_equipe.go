package tactical

// mesures_equipe.go — LES PETITES MESURES DU DETAIL D'UNE ZONE : distance entre deux points du plan et
// mediane d'une serie de distances. Pures ; deux decimales, la precision des coordonnees publiees.

import (
	"math"
	"slices"

	"levelup/go-api/internal/domain"
)

// Arrondi2 arrondit a deux decimales.
func Arrondi2(v float64) float64 { return math.Round(v*100) / 100 }

// DistanceM est la distance horizontale entre deux points du plan, en metres, a deux decimales.
func DistanceM(a, b domain.TacticalPoint) float64 {
	return Arrondi2(math.Hypot(a.X-b.X, a.Y-b.Y))
}

// MedianeM rend la mediane des distances (moyenne des deux centrales si leur nombre est pair), a deux
// decimales ; nil sans valeur — une absence de mesure, jamais zero.
func MedianeM(valeurs []float64) *float64 {
	if len(valeurs) == 0 {
		return nil
	}
	v := slices.Clone(valeurs)
	slices.Sort(v)
	m := v[len(v)/2]
	if len(v)%2 == 0 {
		m = (v[len(v)/2-1] + v[len(v)/2]) / 2
	}
	r := Arrondi2(m)
	return &r
}
