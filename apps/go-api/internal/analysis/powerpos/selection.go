package powerpos

// selection.go — DE LA CARTE DE SCORES AUX POSITIONS PUBLIABLES.
//
// Trois filtres, dans cet ordre, et l'ordre compte :
//
//  1. LE SEUIL EST DOUBLE — un quantile PAR CARTE et un plancher ABSOLU. Le quantile seul
//     retiendrait toujours 10 % des cellules, y compris sur une carte ou rien ne ressort :
//     « pas de calque sans preuve » (D8) exige de pouvoir ne rien publier. Le plancher seul
//     serait un etalonnage global sur des cartes qui n'ont ni les memes volumes ni les
//     memes distances.
//  2. LA TAILLE MINIMALE DE COMPOSANTE. Une position de force est un LIEU : a 0,5 m de pas,
//     douze cellules font 3 m2, soit le pas de tir d'un joueur et sa marge. En dessous, ce
//     sont des cellules chanceuses isolees, exactement ce que le bruit binomial produit.
//  3. LE NOMBRE MAXIMAL DE POSITIONS PAR CARTE. Une carte d'arene en compte trois a six
//     dans les guides ; en publier trente serait colorier la carte, pas la lire.
//
// Le NOMMAGE (zone nommee dominante) n'est PAS ici : il demande le catalogue de zones, donc
// une lecture de fichier, et ce paquet est pur. L'appelant nomme ce qu'il publie.

import (
	"sort"

	"levelup/go-api/internal/analysis/tactical"
)

// Position est une position de force candidate : une composante connexe de cellules
// retenues, son enveloppe et ses comptes.
type Position struct {
	// Cellules : les adresses retenues, triees. Elles peuvent inclure des cellules
	// ajoutees par la fermeture morphologique, qui n'ont pas de score.
	Cellules []tactical.Cellule
	// NbCellulesMesurees : parmi elles, celles qui portent reellement un score. C'est le
	// denominateur de `ScoreMoyen`.
	NbCellulesMesurees int
	// Polygone : l'enveloppe monde (cf. Enveloppe — convexe et dilatee en v1).
	Polygone [][2]float64
	// AireM2 : l'aire de l'enveloppe. Sert aux invariants de relecture.
	AireM2 float64
	// CentreX, CentreY : le barycentre des cellules, en metres monde.
	CentreX, CentreY float64

	ScoreMoyen float64
	ScoreMax   float64

	// Kills / Morts : les comptes des CELLULES de la composante (pas de leurs disques) —
	// c'est ce qu'on peut affirmer de la position elle-meme.
	Kills int
	Morts int
	// Matchs : matchs distincts qui ont alimente au moins une cellule de la composante.
	Matchs int
}

// Selectionne rend les positions de force d'une carte, triees par score moyen decroissant.
// Rend nil quand rien ne passe les filtres — et c'est un resultat, pas un echec.
//
// DEUX REGLES DE RETENUE COHABITENT, et le reglage tranche : seuil UNIQUE (v1, quand
// `QuantileAmorce` vaut zero) ou HYSTERESIS amorce/croissance suivie d'une fermeture
// morphologique (v2). La v1 n'est pas une branche de compatibilite a retirer : c'est la
// regle sur laquelle le verdict date du 2026-09-20 a ete rendu, et elle doit rester
// rejouable a l'identique (D12 du plan).
func Selectionne(g tactical.Grille, scorees []CelluleScoree, r Reglage) []Position {
	parAdresse := make(map[tactical.Cellule]CelluleScoree, len(scorees))
	for _, c := range scorees {
		parAdresse[tactical.Cellule{Col: c.Col, Lig: c.Lig}] = c
	}
	avantFermeture := retenuesDe(scorees, r)
	// Les cellules qui MESURENT sont celles retenues AVANT la fermeture : une cellule que la
	// fermeture ajoute peut fort bien etre scorable (et donc dans `parAdresse`) avec un
	// score sous le seuil — c'est precisement pour cela qu'elle n'y etait pas. La regarder
	// dans `parAdresse` ne suffit donc pas a l'ecarter des moyennes (cf. construis).
	mesurables := make(map[tactical.Cellule]bool, len(avantFermeture))
	for _, adr := range avantFermeture {
		mesurables[adr] = true
	}
	retenues := Fermeture(avantFermeture, r.FermetureRayonCellules)

	var out []Position
	for _, comp := range ComposantesConnexite(retenues, r.Connexite8) {
		if len(comp) < r.TailleMiniComposante {
			continue
		}
		out = append(out, construis(g, comp, parAdresse, mesurables))
	}
	trieParScore(out)
	if r.MaxComposantes > 0 && len(out) > r.MaxComposantes {
		out = out[:r.MaxComposantes]
	}
	return out
}

// retenuesDe rend les adresses des cellules retenues, avant fermeture.
func retenuesDe(scorees []CelluleScoree, r Reglage) []tactical.Cellule {
	if r.QuantileAmorce > 0 {
		return retenuesParHysteresis(scorees, r)
	}
	seuil := seuilDeCarte(scorees, r)
	var out []tactical.Cellule
	for _, c := range scorees {
		if c.Score >= seuil {
			out = append(out, tactical.Cellule{Col: c.Col, Lig: c.Lig})
		}
	}
	return out
}

// retenuesParHysteresis applique la regle v2 : une cellule est retenue si son score passe
// le seuil de CROISSANCE et si elle est reliee, de proche en proche, a une cellule qui
// passe le seuil d'AMORCE.
//
// # POURQUOI DEUX SEUILS PLUTOT QU'UN QUANTILE
//
// Mesure du verdict v1 (section 6) : deux zones fortes ont ete manquees a MOINS DE 0,01 du
// seuil (Orange Pipes a 0,678 contre 0,680 ; Platform a 0,661 contre 0,667). Un seuil
// unique pose sur un quantile est une falaise : la cellule a 0,6799 et celle a 0,6801 ne
// different par rien de mesurable, et l'une entre quand l'autre est jetee. L'hysteresis —
// le seuil de Canny, celui de tous les detecteurs de contour — dit autre chose : un lieu
// commence a un MAXIMUM franc et s'etend tant que le score reste eleve. Une cellule a 0,679
// entre alors si elle touche un maximum, et n'entre pas si elle est seule dans son coin.
// C'est exactement la distinction que le seuil unique ne sait pas faire.
//
// La croissance se compte en 4-CONNEXITE : c'est avant la fermeture, le semis est encore
// clairseme, et laisser passer les diagonales a ce stade ferait couler la croissance d'un
// lieu vers son voisin par un coin. La 8-connexite ne vient qu'apres la fermeture.
func retenuesParHysteresis(scorees []CelluleScoree, r Reglage) []tactical.Cellule {
	amorce, croissance := seuilsHysteresis(scorees, r)
	germes := make(map[tactical.Cellule]bool)
	var candidates []tactical.Cellule
	for _, c := range scorees {
		adr := tactical.Cellule{Col: c.Col, Lig: c.Lig}
		if c.Score >= amorce {
			germes[adr] = true
		}
		if c.Score >= croissance {
			candidates = append(candidates, adr)
		}
	}
	var out []tactical.Cellule
	for _, comp := range Composantes(candidates) {
		if contientUnGerme(comp, germes) {
			out = append(out, comp...)
		}
	}
	return out
}

// seuilsHysteresis rend le seuil d'amorce (jamais sous le plancher absolu) et celui de
// croissance.
func seuilsHysteresis(scorees []CelluleScoree, r Reglage) (float64, float64) {
	scores := make([]float64, 0, len(scorees))
	for _, c := range scorees {
		scores = append(scores, c.Score)
	}
	amorce := Quantile(scores, r.QuantileAmorce)
	if amorce < r.SeuilScoreMin {
		amorce = r.SeuilScoreMin
	}
	croissance := Quantile(scores, r.QuantileCroissance)
	if croissance > amorce {
		croissance = amorce
	}
	return amorce, croissance
}

// contientUnGerme dit si une composante touche au moins une cellule d'amorce.
func contientUnGerme(comp []tactical.Cellule, germes map[tactical.Cellule]bool) bool {
	for _, c := range comp {
		if germes[c] {
			return true
		}
	}
	return false
}

// seuilDeCarte rend le plus exigeant des deux seuils (regle v1).
func seuilDeCarte(scorees []CelluleScoree, r Reglage) float64 {
	scores := make([]float64, 0, len(scorees))
	for _, c := range scorees {
		scores = append(scores, c.Score)
	}
	quantile := Quantile(scores, r.QuantileSeuil)
	if quantile > r.SeuilScoreMin {
		return quantile
	}
	return r.SeuilScoreMin
}

// construis assemble une position a partir de sa composante.
//
// LES CELLULES AJOUTEES PAR LA FERMETURE NE COMPTENT PAS DANS LES MOYENNES. Une fermeture
// morphologique ajoute des cellules qui n'ont pas passe le seuil — soit parce qu'elles
// n'etaient pas scorables, soit parce que leur score etait sous le seuil de croissance.
// Elles font partie du LIEU (donc du polygone, qui doit etre d'un seul tenant), pas de la
// MESURE : compter leur score sous le seuil tirerait le score moyen vers le bas, et une
// cellule non scorable n'a rien a apporter. Elles sont donc dans `Cellules` et hors des
// agregats ; `mesurables` est l'ensemble des cellules retenues AVANT fermeture, et
// `NbCellulesMesurees` dit combien de cellules portent reellement une mesure.
func construis(g tactical.Grille, comp []tactical.Cellule,
	parAdresse map[tactical.Cellule]CelluleScoree, mesurables map[tactical.Cellule]bool) Position {
	p := Position{Cellules: comp, Polygone: Enveloppe(g, comp)}
	p.AireM2 = AirePolygone(p.Polygone)
	var sx, sy, somme float64
	matchs, mesurees := 0, 0
	for _, adr := range comp {
		c, ok := parAdresse[adr]
		if !ok || !mesurables[adr] {
			continue
		}
		mesurees++
		sx += c.CentreX
		sy += c.CentreY
		somme += c.Score
		if c.Score > p.ScoreMax {
			p.ScoreMax = c.Score
		}
		p.Kills += c.KillsDepuis
		p.Morts += c.MortsDedans
		if c.MatchsKills > matchs {
			matchs = c.MatchsKills
		}
	}
	p.NbCellulesMesurees = mesurees
	if mesurees > 0 {
		n := float64(mesurees)
		p.CentreX, p.CentreY = sx/n, sy/n
		p.ScoreMoyen = somme / n
	}
	// Matchs : le MAXIMUM des matchs distincts des cellules, et non leur somme (le meme
	// match alimente plusieurs cellules voisines) ni leur minimum (qui vaudrait toujours le
	// plancher). C'est la borne basse honnete du corpus derriere la position.
	p.Matchs = matchs
	return p
}

// trieParScore ordonne les positions par score moyen decroissant, puis par adresse.
func trieParScore(positions []Position) {
	sort.SliceStable(positions, func(i, j int) bool {
		return avantPosition(positions[i], positions[j])
	})
}

// avantPosition est l'ordre total sur les positions.
func avantPosition(a, b Position) bool {
	if a.ScoreMoyen != b.ScoreMoyen {
		return a.ScoreMoyen > b.ScoreMoyen
	}
	if len(a.Cellules) == 0 || len(b.Cellules) == 0 {
		return len(a.Cellules) > len(b.Cellules)
	}
	return avantCellule(a.Cellules[0], b.Cellules[0])
}
