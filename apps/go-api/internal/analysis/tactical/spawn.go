package tactical

// spawn.go — LES GRAPPES DE REAPPARITION : ou le jeu fait naitre les joueurs.
//
// # AUCUN CATALOGUE MANUEL (decision produit du plan)
//
// Les points de spawn ne sont declares nulle part : ni le film, ni le registre, ni un
// fichier de reference ne disent « cette carte a quatre spawns la ». Ils se DEDUISENT du
// jeu observe — la densite des premiers points de vie sur la grille de 0,5 m — et se
// nomment par le callout le plus proche de leur barycentre. Une carte inconnue, une carte
// Forge, une rotation nouvelle : la mesure marche pareil, parce qu'elle ne suppose rien.
//
// # LE PLANCHER PORTE SUR LA GRAPPE, PAS SUR LA CELLULE, ET C'EST DELIBERE
//
// « Composantes connexes au-dessus du plancher de 3 matchs distincts » se lit de deux
// facons, et une seule mesure quelque chose. Appliquer le plancher CELLULE PAR CELLULE
// avant de connecter effacerait presque tous les spawns : sur une grille de 0,5 m, trois
// reapparitions de trois matchs differents tombent dans trois cellules VOISINES, jamais
// dans la meme — chaque cellule compterait un seul match, et il ne resterait rien a
// connecter. Le plancher s'applique donc a la COMPOSANTE : on connecte tout ce qui est
// alimente, puis on garde les amas vus dans au moins trois matchs distincts. C'est aussi
// la lecture qui rend vraie la regle produit « sous le plancher, la grappe n'existe pas » :
// ce qui est rare est un amas rare, pas une cellule rare.
//
// # L'IDENTIFIANT EST UNE POSITION, PAS UN RANG
//
// Un index de tableau change des qu'un match entre dans le filtre — et le lien
// `?spawn=<id>` d'un utilisateur designerait alors une autre grappe. L'identifiant est donc
// derive du BARYCENTRE arrondi au decimetre : deux lectures du meme amas rendent le meme
// identifiant, et deux amas distincts ne peuvent pas le partager (ils sont separes par au
// moins une cellule vide de 0,5 m, soit cinq decimetres).
//
// PUR : aucune I/O. Les zones nommees sont une ENTREE — c'est le service qui les resout
// depuis le catalogue de callouts versionne (`analysis/replay`, que ce paquet n'importe
// pas : cf. le ratchet de purete).

import (
	"fmt"
	"math"
	"sort"
)

// PointSpawn est une reapparition observee : ou, et dans quel match.
//
// Le match est porte par le point parce que le plancher se compte en matchs DISTINCTS —
// quatre reapparitions d'une meme partie ne sont qu'une observation.
type PointSpawn struct {
	MatchID string
	X, Y    float64
}

// ZoneNommee est un callout : un nom de lieu et son point de reference, en metres monde.
type ZoneNommee struct {
	Nom  string
	X, Y float64
}

// GrappeSpawn est un amas de reapparitions : la ou le jeu fait naitre les joueurs.
type GrappeSpawn struct {
	// ID est stable entre deux lectures du meme amas (cf. l'en-tete). C'est lui que le
	// filtre `?spawn=` transporte.
	ID string
	// Nom est le callout le plus proche du barycentre. VIDE quand la carte n'a aucune zone
	// nommee au catalogue : on ne fabrique pas de nom de repli — le jeu ne le prononcerait
	// pas (meme regle que le rendu des zones du rejeu).
	Nom string
	// X, Y : le barycentre de l'amas, en metres monde.
	X, Y float64
	// Matchs est le nombre de matchs DISTINCTS ayant alimente l'amas — sa solidite.
	Matchs int
	// Cellules sont les cellules de l'amas, triees. Elles disent son EMPRISE, et c'est par
	// elles que le filtre decide si une reapparition appartient a la grappe.
	Cellules []Cellule
}

// GrappesDeSpawn rend les amas de reapparition d'un ensemble de points, tries par nombre de
// matchs decroissant puis par identifiant (ordre stable, independant de l'ordre d'entree).
//
// `zones` peut etre vide : les grappes sortent alors sans nom.
func GrappesDeSpawn(g Grille, points []PointSpawn, zones []ZoneNommee) []GrappeSpawn {
	parCellule := make(map[Cellule]map[string]bool)
	for _, p := range points {
		c, ok := g.Cellule(p.X, p.Y)
		if !ok {
			// Position non finie : ecartee, jamais projetee sur une cellule arbitraire.
			continue
		}
		if parCellule[c] == nil {
			parCellule[c] = make(map[string]bool)
		}
		parCellule[c][p.MatchID] = true
	}
	out := make([]GrappeSpawn, 0, 8)
	for _, comp := range composantes(parCellule) {
		matchs := make(map[string]bool)
		for _, c := range comp {
			for m := range parCellule[c] {
				matchs[m] = true
			}
		}
		if len(matchs) < PlancherMatchsParCellule {
			// SOUS LE PLANCHER, LA GRAPPE N'EXISTE PAS — elle n'est pas rendue « faible »,
			// elle est absente : un amas vu dans deux parties ne dit pas ou le jeu fait
			// naitre, il dit ou il a fait naitre deux fois.
			continue
		}
		x, y := barycentre(g, comp)
		out = append(out, GrappeSpawn{
			ID:       identifiantDeGrappe(x, y),
			Nom:      calloutLePlusProche(zones, x, y),
			X:        x,
			Y:        y,
			Matchs:   len(matchs),
			Cellules: comp,
		})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Matchs != out[b].Matchs {
			return out[a].Matchs > out[b].Matchs
		}
		return out[a].ID < out[b].ID
	})
	return out
}

// composantes regroupe les cellules alimentees en composantes connexes (8-voisinage),
// chacune triee. Le parcours part des cellules TRIEES : un parcours de map est aleatoire,
// et deux lectures du meme jeu doivent rendre les memes amas dans le meme ordre.
func composantes(parCellule map[Cellule]map[string]bool) [][]Cellule {
	restantes := make(map[Cellule]bool, len(parCellule))
	ordre := make([]Cellule, 0, len(parCellule))
	for c := range parCellule {
		restantes[c] = true
		ordre = append(ordre, c)
	}
	trierCellulesBrutes(ordre)

	var out [][]Cellule
	for _, depart := range ordre {
		if !restantes[depart] {
			continue
		}
		var comp []Cellule
		pile := []Cellule{depart}
		delete(restantes, depart)
		for len(pile) > 0 {
			c := pile[len(pile)-1]
			pile = pile[:len(pile)-1]
			comp = append(comp, c)
			for _, v := range voisines(c) {
				if restantes[v] {
					delete(restantes, v)
					pile = append(pile, v)
				}
			}
		}
		trierCellulesBrutes(comp)
		out = append(out, comp)
	}
	return out
}

// voisines rend les huit cellules adjacentes (diagonales comprises).
//
// LE 8-VOISINAGE EST LA REGLE DU PLAN, et il compte : en 4-voisinage, un amas pose en
// diagonale — le cas d'un couloir de spawn oblique — se scinderait en autant de grappes que
// de cellules.
func voisines(c Cellule) []Cellule {
	out := make([]Cellule, 0, 8)
	for dc := -1; dc <= 1; dc++ {
		for dl := -1; dl <= 1; dl++ {
			if dc == 0 && dl == 0 {
				continue
			}
			out = append(out, Cellule{Col: c.Col + dc, Lig: c.Lig + dl})
		}
	}
	return out
}

// barycentre rend le centre de masse des cellules d'un amas, en metres monde.
func barycentre(g Grille, comp []Cellule) (x, y float64) {
	if len(comp) == 0 {
		return 0, 0
	}
	var sx, sy float64
	for _, c := range comp {
		cx, cy := g.Centre(c)
		sx += cx
		sy += cy
	}
	n := float64(len(comp))
	return sx / n, sy / n
}

// identifiantDeGrappe derive un identifiant STABLE du barycentre, arrondi au decimetre.
//
// Deux amas distincts ne peuvent pas le partager : ils sont separes par au moins une
// cellule vide de 0,5 m, donc leurs barycentres different d'au moins cinq decimetres.
func identifiantDeGrappe(x, y float64) string {
	return fmt.Sprintf("s%+06d%+06d", int(math.Round(x*10)), int(math.Round(y*10)))
}

// calloutLePlusProche rend le nom de la zone la plus proche d'un point, en distance 2D.
// Aucune zone, ou aucune zone nommee : chaine vide — jamais un nom invente.
func calloutLePlusProche(zones []ZoneNommee, x, y float64) string {
	meilleur := ""
	best := math.Inf(1)
	for _, z := range zones {
		if z.Nom == "" {
			continue
		}
		if d := math.Hypot(z.X-x, z.Y-y); d < best {
			best, meilleur = d, z.Nom
		}
	}
	return meilleur
}

// trierCellulesBrutes ordonne des cellules par colonne puis ligne.
func trierCellulesBrutes(cs []Cellule) {
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].Col != cs[j].Col {
			return cs[i].Col < cs[j].Col
		}
		return cs[i].Lig < cs[j].Lig
	})
}
