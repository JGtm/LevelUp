package domain

import "math"

// FirstBloodMatchPoint porte les premiers evenements d'UN joueur sur UN match,
// en secondes depuis le debut du gameplay (referentiel T0 corrige).
//
// nil = evenement absent du match (le joueur n'a pas fragge / n'est pas tombe).
// Le match reste compte dans le denominateur cote front (« 11/12 matchs »), d'ou
// l'absence d'`omitempty` : la valeur nulle est une INFORMATION, pas un champ
// manquant.
type FirstBloodMatchPoint struct {
	MatchID       string   `json:"match_id"`
	FirstKillSec  *float64 `json:"first_kill_sec"`
	FirstDeathSec *float64 `json:"first_death_sec"`
}

// FirstBloodPlayerSeries est la serie d'un joueur pour le chart
// « Premier frag / premiere mort » (FirstBloodLanes). Une serie = une bande.
//
// Contrat PAR MATCH (aucun bucketing serveur) : le front derive lui-meme les
// medianes, l'ecart et la fenetre de l'axe. Meme forme sur les trois surfaces
// produit (Escouade/Dynamique 2-4 joueurs, Timeseries solo, Session solo).
type FirstBloodPlayerSeries struct {
	Player  string                 `json:"player"`
	Matches []FirstBloodMatchPoint `json:"matches"`
}

// NewFirstBloodPoint construit un point a partir des premiers timestamps en
// MILLISECONDES (nil = evenement absent). Point de conversion UNIQUE ms ->
// secondes du chart : arrondi au dixieme de seconde, jamais duplique ailleurs.
func NewFirstBloodPoint(matchID string, firstKillMS, firstDeathMS *int64) FirstBloodMatchPoint {
	p := FirstBloodMatchPoint{MatchID: matchID}
	if firstKillMS != nil {
		p.FirstKillSec = msToSecondsPtr(*firstKillMS)
	}
	if firstDeathMS != nil {
		p.FirstDeathSec = msToSecondsPtr(*firstDeathMS)
	}
	return p
}

// HasEvents indique qu'au moins un point porte un premier frag ou une premiere
// mort. Une serie sans aucun evenement ne dessine rien : les services l'ecartent
// de la reponse (le front rend alors son etat vide).
func (s FirstBloodPlayerSeries) HasEvents() bool {
	for i := range s.Matches {
		if s.Matches[i].FirstKillSec != nil || s.Matches[i].FirstDeathSec != nil {
			return true
		}
	}
	return false
}

// msToSecondsPtr convertit des millisecondes en secondes arrondies au dixieme.
func msToSecondsPtr(ms int64) *float64 {
	v := math.Round(float64(ms)/100) / 10
	return &v
}
