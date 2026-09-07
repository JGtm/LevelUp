package coordination

import "levelup/go-api/internal/domain"

// sansFenetre desactive la borne de temps de chercheVengeur (cf. suivreMorts).
const sansFenetre int64 = -1

// Ripostes suit chaque mort et rend le delai jusqu'a la riposte SANS AUCUNE BORNE DE TEMPS.
//
// A QUOI CA SERT, ET A QUOI CA NE SERT PAS.
//
// La page Escouade montre la DISTRIBUTION du delai d'echange. Une distribution qui
// s'arreterait net a 5 s ne dirait pas si la fenetre coupe une population dense ou du vide :
// « 40 % de morts vengees » se lit tres differemment selon que les ripostes manquees
// arrivent a 5,2 s ou a 40 s. Ce sont les deux dernieres barres de l'histogramme, montrees
// HORS FENETRE et jamais comptees dans le taux.
//
// CE N'EST DONC PAS UNE MESURE D'ECHANGE. Une riposte a 40 s n'est pas une vengeance : le
// tueur a eu le temps de sortir du combat, d'en gagner un autre, de rentrer. Le seul chiffre
// qui vaut echange est celui de Echanges, borne a FenetreEchangeMs. C'est aussi pourquoi
// cette fonction rend les MORTS et pas un domain.BilanEchanges : un bilan porte
// NbVengees/NbVengeables, deux compteurs qui, ici, ne veulent pas dire ce que leur nom dit.
//
// Sous la fenetre, les deux lectures coincident EXACTEMENT (meme noyau suivreMorts, meme
// « premier vengeur valide ») : une mort vengee a 3 s sort avec DelaiMs = 3 000 des deux
// cotes. C'est verifie par riposte_test.go, et c'est ce qui autorise l'histogramme a se
// construire sur cette seule lecture.
//
// Memes definitions et memes cas limites que Echanges : seule la borne change.
func Ripostes(kills []domain.KillEvent, equipes domain.EquipesParMatch) []domain.MortSuivie {
	return suivreMorts(kills, equipes, sansFenetre)
}
