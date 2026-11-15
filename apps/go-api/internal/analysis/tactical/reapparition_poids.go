package tactical

import "levelup/go-api/internal/domain"

// PoidsDeFamille convertit les poids nommes d'un fichier de poids en tableau ordonne : un nom
// inconnu est ignore (le fichier peut venir d'une version qui avait une variable de plus), et une
// variable absente vaut zero.
func PoidsDeFamille(f domain.PoidsFamille) [NbVariables]float64 {
	var w [NbVariables]float64
	for j, nom := range NomsVariables {
		w[j] = f.Poids[nom]
	}
	return w
}

// PoidsNommes est la conversion inverse, pour ecrire un fichier de poids.
func PoidsNommes(w [NbVariables]float64) map[string]float64 {
	out := make(map[string]float64, NbVariables)
	for j, nom := range NomsVariables {
		out[nom] = w[j]
	}
	return out
}
