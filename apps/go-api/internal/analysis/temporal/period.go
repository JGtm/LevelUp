// Package temporal fournit les helpers de filtrage temporel et de bucketing
// utilises transversalement par les services qui agregent des donnees joueur.
//
// Ce package n'a aucune dependance DB ni HTTP : ce sont des fonctions pures
// que l'on peut tester en isolation. Voir docs/FOUNDATIONS_GUIDE.md pour les
// cas d'usage.
package temporal

import "time"

// Period represente une fenetre temporelle relative a un instant courant.
// Les valeurs admises sont les 5 paliers utilises par tous les selecteurs UI :
// all / 2y / 1y / 1m / 1w.
type Period string

// Constantes des periodes admises.
const (
	PeriodAll Period = "all"
	Period2Y  Period = "2y"
	Period1Y  Period = "1y"
	Period1M  Period = "1m"
	Period1W  Period = "1w"
)

// IsValid retourne true si la valeur est l'une des constantes connues.
func (p Period) IsValid() bool {
	switch p {
	case PeriodAll, Period2Y, Period1Y, Period1M, Period1W:
		return true
	}
	return false
}

// Since retourne le timestamp de reference sous lequel les rows sont exclues.
// Retourne nil si la periode vaut PeriodAll (pas de filtrage temporel) ou si
// la valeur est inconnue (defensive default).
func (p Period) Since(now time.Time) *time.Time {
	var t time.Time
	switch p {
	case PeriodAll:
		return nil
	case Period2Y:
		t = now.AddDate(-2, 0, 0)
	case Period1Y:
		t = now.AddDate(-1, 0, 0)
	case Period1M:
		t = now.AddDate(0, -1, 0)
	case Period1W:
		t = now.AddDate(0, 0, -7)
	default:
		return nil
	}
	return &t
}

// HasStartTime est l'interface contrainte par les helpers temporal.
// Toute structure ayant une methode GetStartTime() peut etre filtree et bucketee.
type HasStartTime interface {
	GetStartTime() time.Time
}

// FilterByPeriod retourne les rows dont GetStartTime() >= Since(now).
// Si period == PeriodAll, retourne le slice d'origine inchange (pas de copie).
// Le filtre est inclusif sur la borne inferieure (>=).
func FilterByPeriod[T HasStartTime](rows []T, period Period, now time.Time) []T {
	since := period.Since(now)
	if since == nil {
		return rows
	}
	out := make([]T, 0, len(rows))
	for _, r := range rows {
		if !r.GetStartTime().Before(*since) {
			out = append(out, r)
		}
	}
	return out
}
