package domain

import (
	"math"
	"time"
)

// FirstBloodMatchPoint porte les premiers evenements d'UN joueur sur UN match,
// en secondes depuis le debut du gameplay (referentiel T0 corrige), plus les
// metadonnees d'affichage du match (carte, mode, date de debut) — DEC-4
// (retours utilisateur 2026-08-29) : le tooltip du chart doit dire « carte .
// mode . date », plus jamais l'uuid du match.
//
// nil = evenement absent du match (le joueur n'a pas fragge / n'est pas tombe).
// Le match reste compte dans le denominateur cote front (« 11/12 matchs »), d'ou
// l'absence d'`omitempty` sur FirstKillSec/FirstDeathSec : la valeur nulle est
// une INFORMATION, pas un champ manquant.
//
// MapUI/ModeUI restent `omitempty` : un titre ou un match peut ne pas exposer
// de libelle resolu (bots, metadata absente) — degradation cote front, jamais
// de cle brute ni d'uuid affiche (cf. features/_shared/firstBlood.ts).
type FirstBloodMatchPoint struct {
	MatchID       string    `json:"match_id"`
	FirstKillSec  *float64  `json:"first_kill_sec"`
	FirstDeathSec *float64  `json:"first_death_sec"`
	MapUI         string    `json:"map_ui,omitempty"`
	ModeUI        string    `json:"mode_ui,omitempty"`
	StartTime     time.Time `json:"start_time"`
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

// FirstBloodMatchMeta porte les metadonnees d'affichage (carte, mode, date de
// debut) d'UN match, resolues par l'appelant — les builders solo (StatsMatchRow)
// et squad (SquadMatchRow) ont deja cette info en scope — et injectees dans le
// point construit par NewFirstBloodPoint. Regroupees en struct pour ne pas
// depasser 5 parametres sur le constructeur (seuil CLAUDE.md). StartTime est
// deja la valeur canonique de la ligne source : ne JAMAIS la recalculer ici
// (regle 8, timezone canonique).
type FirstBloodMatchMeta struct {
	MapUI     string
	ModeUI    string
	StartTime time.Time
}

// NewFirstBloodPoint construit un point a partir des premiers timestamps en
// MILLISECONDES (nil = evenement absent) et des metadonnees d'affichage du
// match. Point de conversion UNIQUE ms -> secondes du chart : arrondi au
// dixieme de seconde, jamais duplique ailleurs.
func NewFirstBloodPoint(matchID string, firstKillMS, firstDeathMS *int64, meta FirstBloodMatchMeta) FirstBloodMatchPoint {
	p := FirstBloodMatchPoint{
		MatchID:   matchID,
		MapUI:     meta.MapUI,
		ModeUI:    meta.ModeUI,
		StartTime: meta.StartTime,
	}
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
