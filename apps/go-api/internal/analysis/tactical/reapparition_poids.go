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

// PoidsPourFamille choisit le jeu de poids d'un match dans un fichier de poids : celui de sa
// famille s'il en a un, sinon celui de la famille « tous ». Rend le nom du jeu retenu, ses poids
// et vrai ; faux quand le fichier n'a ni l'un ni l'autre (fichier vide ou absent : l'appelant sert
// alors la prevision sans modele).
//
// Une famille vide (match sans famille), inconnue du fichier, ou sans poids propres (effectif
// trop faible pour avoir ete ajustee) se replie sur « tous ».
func PoidsPourFamille(f domain.PoidsReapparition, famille string) (string, [NbVariables]float64, bool) {
	if p, ok := f.Familles[famille]; ok && famille != "" && len(p.Poids) > 0 {
		return famille, PoidsDeFamille(p), true
	}
	if p, ok := f.Familles[domain.FamilleToutesModes]; ok && len(p.Poids) > 0 {
		return domain.FamilleToutesModes, PoidsDeFamille(p), true
	}
	return "", [NbVariables]float64{}, false
}
