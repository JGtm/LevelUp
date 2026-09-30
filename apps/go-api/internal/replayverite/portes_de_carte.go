package replayverite

// portes_de_carte.go — LES PORTES DE CARTE (ascenseurs, canons, largages) EXEMPTEES DE V-1.
//
// LISTE NOMMEE, DATEE, ECRITE A LA MAIN — ET POURQUOI. La regle du superviseur (2026-09-30) veut
// que toute exemption de saut vienne de DONNEES DE CARTE existantes. Elles n'existent pas :
// `map_objectives.json` ne porte ni ascenseur ni canon positionne (seulement des noms de script
// `Teleporter.*` sans coordonnees), `map_callouts.json` porte des zones nommees « Lift » qui sont des
// libelles d'appel, pas des points d'arrivee, et `map_positions_jouees.json` ne couvre qu'une carte.
// La liste ci-dessous est donc une MESURE, faite le 2026-09-30 sur les artefacts `b452391f7` :
// les arrivees de sauts > 100 m/s partagees par au moins trois vies distinctes, regroupees par
// cellule de 2 m, puis centrees.
//
// Critere de retrait : un catalogue de carte qui positionne ascenseurs, canons et largages (par
// exemple une extension de `map_objectives.json`) ; ces entrees en seront alors lues, et cette liste
// supprimee. Un saut sur une carte absente de la liste n'est JAMAIS exempte : il compte.

// porteDeCarte est le centre mesure d'une porte, sur une carte designee par son asset (`mapId`).
type porteDeCarte struct {
	mapID  string
	centre [3]float64
	sauts  int // arrivees mesurees le 2026-09-30, pour memoire
}

// Les assets (`mapId`) des cartes mesurees.
const (
	mapThunderhead = "28a3ac28-f69d-4fa9-9ebf-a0449c89c8da" // Thunderhead (BTB)
	mapDredge      = "e4bb06db-065f-4902-b93b-d8dac315eac4" // Dredge
)

// portesDeCarteMesurees : mesure du 2026-09-30 (`b452391f7`, temoins `11de8353` et `d9781168`).
var portesDeCarteMesurees = []porteDeCarte{
	{mapThunderhead, [3]float64{13.7, 119.8, 98.0}, 16},
	{mapThunderhead, [3]float64{-80.6, 48.8, 98.0}, 8},
	{mapThunderhead, [3]float64{6.1, 96.8, 52.6}, 13},
	{mapThunderhead, [3]float64{-74.7, 73.4, 52.7}, 5},
	{mapDredge, [3]float64{-18.3, 0.9, 82.5}, 38},
}

// portesDe rend les portes mesurees d'une carte.
func portesDe(mapID string) [][3]float64 {
	var out [][3]float64
	for _, p := range portesDeCarteMesurees {
		if p.mapID == mapID {
			out = append(out, p.centre)
		}
	}
	return out
}
