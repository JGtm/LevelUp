package main

// controle_variantes.go — LE CONTROLE QUI DIT QU'UNE AGREGATION EST LEGITIME.
//
// La cle d'agregation rabote les suffixes de variante (« Live Fire » et « Live Fire -
// Ranked » sont une seule carte, cf. main.go). La justification de ce rabotage est que les
// deux assets partagent le niveau moteur, donc le repere monde. C'est une HYPOTHESE, et
// elle se CONTROLE : si les nuages de deux variantes ne sont pas au meme endroit, les
// sommer fabrique une carte qui n'existe pas.
//
// MESURE DU 2026-09-20 QUI A MOTIVE CE CONTROLE : sur « Live Fire », le nuage de kills de
// la variante « - Ranked » est decale et ETIRE en X (x median 21,3 m contre 8,2 m pour la
// carte de base, x maximal 36,9 m contre 27,3 m), alors que l'axe Y coincide au decimetre
// et que les ARTEFACTS DE REJEU des deux variantes, eux, s'accordent (x maximal 27,3 m des
// deux cotes). 13 des 30 matchs « - Ranked » debordent. C'est un defaut de decodage des
// positions de kill, hors du perimetre de ce chantier : le controle le CHIFFRE et le
// JOURNALISE, il ne le corrige pas et il n'ecarte rien.
//
// Le controle porte sur des MOYENNES et non des medianes : il n'a pas a etre robuste, il a
// a etre franc — un decalage systematique de plusieurs metres se voit sur une moyenne, et
// une moyenne se tient en deux accumulateurs au lieu d'une liste par variante.

import (
	"log/slog"
	"math"
	"sort"
)

// ecartVarianteM est l'ecart, en metres, au-dela duquel deux variantes d'une meme carte
// sont declarees incoherentes. Deux metres : quatre cellules de la grille, soit bien plus
// que le bruit d'echantillonnage entre deux corpus de plusieurs milliers de kills, et bien
// moins que les 13 m constates sur Live Fire.
const ecartVarianteM = 2.0

// statVariante accumule le barycentre des positions de tueur d'une variante.
type statVariante struct {
	n          int
	sumX, sumY float64
}

// moyennes rend le barycentre, ou (0, 0) si la variante est vide.
func (s statVariante) moyennes() (float64, float64) {
	if s.n == 0 {
		return 0, 0
	}
	return s.sumX / float64(s.n), s.sumY / float64(s.n)
}

// noteVariante enregistre une position de tueur sous le nom AFFICHE de sa carte.
func (c *Cible) noteVariante(mapName string, x, y float64) {
	if c.Variantes == nil {
		c.Variantes = map[string]*statVariante{}
	}
	s := c.Variantes[mapName]
	if s == nil {
		s = &statVariante{}
		c.Variantes[mapName] = s
	}
	s.n++
	s.sumX += x
	s.sumY += y
}

// ControleVariantes journalise l'ecart entre les barycentres des variantes d'une carte, et
// rend cet ecart (0 quand il n'y a qu'une variante).
func (c *Cible) ControleVariantes() float64 {
	if len(c.Variantes) < 2 {
		return 0
	}
	noms := make([]string, 0, len(c.Variantes))
	for nom := range c.Variantes {
		noms = append(noms, nom)
	}
	sort.Strings(noms)

	pire := 0.0
	for i := range noms {
		for j := i + 1; j < len(noms); j++ {
			xi, yi := c.Variantes[noms[i]].moyennes()
			xj, yj := c.Variantes[noms[j]].moyennes()
			if d := math.Hypot(xi-xj, yi-yj); d > pire {
				pire = d
			}
		}
	}
	args := []any{"carte", c.Carte, "variantes", noms,
		"ecart_barycentres_m", math.Round(pire*100) / 100, "seuil_m", ecartVarianteM}
	if pire > ecartVarianteM {
		slog.Warn("mappower: variantes d'une carte INCOHERENTES — agregation suspecte", args...)
		return pire
	}
	slog.Info("mappower: variantes d'une carte coherentes", args...)
	return pire
}
