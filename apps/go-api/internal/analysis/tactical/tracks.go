package tactical

// tracks.go — L'OCCUPATION : « ou je passe mon temps », mesuree sur les pistes d'un
// artefact de rejeu.
//
// # POURQUOI UN REECHANTILLONNAGE, ET PAS LES POINTS BRUTS
//
// Les points d'une piste ne sont PAS repartis dans le temps : le film ne replique une
// position que lorsqu'elle change assez, si bien qu'un joueur immobile derriere un mur
// produit deux points en quinze secondes pendant qu'un joueur qui court en produit un
// par frame. Compter les points bruts mesurerait donc le MOUVEMENT, pas le temps passe —
// exactement l'inverse de la question posee. Le reechantillonnage a pas FIXE (250 ms) en
// tenant la derniere position connue rend chaque echantillon egal a tous les autres :
// un echantillon = 250 ms de presence, ou qu'on soit et quoi qu'on fasse.
//
// # LA FENETRE EST DEMI-OUVERTE, ET C'EST CE QUI REND LES COMPTES JUSTES
//
// Une vie qui couvre [t0, t1) rend exactement (t1 - t0) / pas echantillons : deux
// secondes de presence font huit quarts de seconde, pas neuf. Compter la borne haute
// ajouterait un echantillon par vie — soit, sur un match a une centaine de vies, une
// demi-minute de presence inventee.
//
// # PUR
//
// Aucune I/O, aucune dependance a `analysis/replay` : l'appelant PROJETTE ses pistes
// d'artefact vers les types de ce fichier (cf. la frontiere declaree dans doc.go et dans
// domain/tactical.go). C'est aussi lui qui resout l'echelle de l'axe de temps quand
// l'artefact ne la publie pas — ce paquet ne devine aucune cadence.

import (
	"sort"

	"levelup/go-api/internal/domain"
)

// PasOccupationMs est le pas d'echantillonnage de l'occupation, en millisecondes :
// 250 ms (plan tactique, phase 7 — avancee en phase 6 pour que les rasters cuits
// portent deja la bonne maille).
//
// POURQUOI 250 ms. C'est le compromis entre deux erreurs opposees : un pas plus large
// rate les traversees rapides d'une cellule de 0,5 m, un pas plus fin multiplie le
// volume du sidecar sans rien ajouter (le film lui-meme ne replique une position qu'au
// mieux toutes les 100 ms). A ce pas, un echantillon vaut un quart de seconde de
// presence — la conversion en secondes est exacte, jamais approchee.
const PasOccupationMs = 250

// MsParSeconde : la conversion des millisecondes en secondes, nommee plutot que
// recopiee sous forme de 1000 dans chaque division.
const MsParSeconde = 1000

// PointPiste est une position echantillonnee par le film, a la frame T.
//
// T est un INDEX DE FRAME, pas une duree : c'est l'axe de l'artefact, et c'est lui que
// le rejeu 2D sait rejouer (`?frame=`). Sa conversion en millisecondes passe par
// EntreeOccupation.IntervalleFrameMs.
type PointPiste struct {
	T    int
	X, Y float64
}

// Piste est UNE VIE (un slot de biped entre deux reapparitions), pas un joueur : le
// regroupement par joueur se fait par XUID, et un joueur en a autant que de morts.
type Piste struct {
	// XUID identifie le porteur de la vie. VIDE = vie non nommee par le film (le fil
	// des morts ne l'a jamais citee, ou c'est un bot) : elle est IGNOREE, jamais
	// rattachee a un joueur par position ou par slot — un slot est un ordre, pas une
	// identite.
	XUID string

	// Points est la trajectoire, CROISSANTE en T. L'artefact la produit dans cet ordre ;
	// une entree desordonnee est triee (copie locale) plutot que lue de travers.
	Points []PointPiste

	// StartFrame / EndFrame bornent la vie sur l'axe de temps. Une fenetre non declaree
	// (EndFrame <= StartFrame — les deux champs sont `omitempty` dans l'artefact) laisse
	// les points la definir eux-memes ; declaree, elle ECARTE tout point qui en sort.
	StartFrame int
	EndFrame   int
}

// EntreeOccupation regroupe les pistes d'UN match avec l'echelle de son axe de temps.
//
// Un struct plutot que trois parametres adjacents : deux entiers de suite (intervalle,
// pas) s'inversent sans que le compilateur le voie, et le resultat serait une occupation
// silencieusement fausse d'un facteur 2,5.
type EntreeOccupation struct {
	// MatchID est porte par chaque echantillon rendu : le plancher de rarete du
	// rasterisage se compte en matchs DISTINCTS par cellule.
	MatchID string

	// IntervalleFrameMs est la duree reelle d'une frame, en millisecondes. Elle vient de
	// l'artefact (`frameIntervalMs`) et l'appelant la resout quand il est absent —
	// aucune valeur par defaut ici. <= 0 : rien n'est mesurable, la sortie est vide.
	IntervalleFrameMs int

	Pistes []Piste
}

// SpawnPiste est le PREMIER point d'une vie : la reapparition.
type SpawnPiste struct {
	Frame int     `json:"frame"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
}

// EntreeCellule est la PREMIERE entree d'un joueur dans une cellule, sur l'ensemble de
// ses vies du match. C'est l'INSTANT CONTRIBUTEUR d'une lecture d'occupation (decision
// produit du plan tactique) : le clic sur une cellule chaude ouvre le rejeu la.
type EntreeCellule struct {
	Col   int `json:"col"`
	Lig   int `json:"lig"`
	Frame int `json:"frame"`
}

// OccupationJoueur est ce qu'un joueur NOMME a produit sur le match.
type OccupationJoueur struct {
	XUID string

	// Echantillons : les positions reechantillonnees a pas fixe, toutes vies confondues.
	// C'est l'entree du rasterisage — un echantillon vaut PasOccupationMs de presence.
	Echantillons []domain.PositionSample

	// Spawns : le premier point de chacune de ses vies, trie par frame.
	Spawns []SpawnPiste

	// PremieresEntrees : par cellule atteinte, la frame de la premiere fois. Trie par
	// colonne puis ligne, comme les cellules d'un raster.
	PremieresEntrees []EntreeCellule
}

// Occupation reechantillonne les pistes d'un match a pas fixe et rend, PAR JOUEUR NOMME,
// ses echantillons, ses reapparitions et sa premiere entree dans chaque cellule.
//
// `pasMs` <= 0, `IntervalleFrameMs` <= 0 : rien n'est mesurable, sortie vide (jamais une
// division par zero, jamais une cadence devinee).
//
// Les vies SANS XUID sont ecartees : le film ne les a pas nommees, et les rattacher a un
// joueur par leur slot serait prendre un ordre pour une identite.
func Occupation(g Grille, e EntreeOccupation, pasMs int) []OccupationJoueur {
	if pasMs <= 0 || e.IntervalleFrameMs <= 0 {
		return nil
	}
	ech := echantillonneur{grille: g, entree: e, pasMs: pasMs}
	parJoueur := make(map[string]*OccupationJoueur)
	vues := make(map[string]map[Cellule]int)
	for _, p := range e.Pistes {
		if p.XUID == "" {
			continue
		}
		points := pointsDansLaFenetre(p)
		if len(points) == 0 {
			continue
		}
		j := parJoueur[p.XUID]
		if j == nil {
			j = &OccupationJoueur{XUID: p.XUID}
			parJoueur[p.XUID] = j
			vues[p.XUID] = make(map[Cellule]int)
		}
		j.Spawns = append(j.Spawns, SpawnPiste{Frame: points[0].T, X: points[0].X, Y: points[0].Y})
		ech.vie(points, j, vues[p.XUID])
	}
	return assembler(parJoueur, vues)
}

// echantillonneur porte les reglages constants d'un match (grille, echelle de temps,
// pas) pour que la boucle d'une vie ne prenne que ce qui varie d'une vie a l'autre.
type echantillonneur struct {
	grille Grille
	entree EntreeOccupation
	pasMs  int
}

// vie parcourt la fenetre demi-ouverte [premier point, dernier point) au pas demande, en
// TENANT la derniere position connue entre deux points.
func (e echantillonneur) vie(points []PointPiste, j *OccupationJoueur, vues map[Cellule]int) {
	intervalle := e.entree.IntervalleFrameMs
	debutMs := points[0].T * intervalle
	finMs := points[len(points)-1].T * intervalle
	curseur := 0
	for tMs := debutMs; tMs < finMs; tMs += e.pasMs {
		// La derniere position CONNUE a cet instant : le curseur ne recule jamais, la
		// piste etant croissante en T.
		for curseur+1 < len(points) && points[curseur+1].T*intervalle <= tMs {
			curseur++
		}
		p := points[curseur]
		j.Echantillons = append(j.Echantillons,
			domain.PositionSample{MatchID: e.entree.MatchID, X: p.X, Y: p.Y})
		c, ok := e.grille.Cellule(p.X, p.Y)
		if !ok {
			// Position non finie : ecartee ici comme elle le sera au rasterisage
			// (Raster.PointsIgnores), jamais projetee sur une cellule arbitraire.
			continue
		}
		// LA FRAME EST CELLE DE L'ECHANTILLON, pas celle du point tenu : c'est l'instant
		// ou le joueur EST dans la cellule, donc celui que le rejeu doit ouvrir.
		frame := tMs / intervalle
		if deja, vu := vues[c]; !vu || frame < deja {
			vues[c] = frame
		}
	}
}

// pointsDansLaFenetre rend les points de la vie retenus par [StartFrame, EndFrame],
// dans l'ordre croissant des frames.
//
// Fenetre NON DECLAREE (EndFrame <= StartFrame) : les points la definissent eux-memes —
// les deux champs sont optionnels dans l'artefact, et un artefact ancien ne les porte pas.
func pointsDansLaFenetre(p Piste) []PointPiste {
	points := p.Points
	if p.EndFrame > p.StartFrame {
		retenus := make([]PointPiste, 0, len(points))
		for _, pt := range points {
			if pt.T < p.StartFrame || pt.T > p.EndFrame {
				continue
			}
			retenus = append(retenus, pt)
		}
		points = retenus
	}
	return trierParFrame(points)
}

// trierParFrame garantit l'ordre croissant sans allouer quand il est deja tenu (le cas
// de tout artefact). Une piste lue de travers rendrait une occupation silencieusement
// fausse : le curseur de `echantillonner` ne recule pas.
func trierParFrame(points []PointPiste) []PointPiste {
	trie := true
	for i := 1; i < len(points); i++ {
		if points[i].T < points[i-1].T {
			trie = false
			break
		}
	}
	if trie {
		return points
	}
	copie := make([]PointPiste, len(points))
	copy(copie, points)
	sort.SliceStable(copie, func(i, j int) bool { return copie[i].T < copie[j].T })
	return copie
}

// assembler ordonne la sortie : joueurs par xuid, spawns par frame, entrees par cellule.
// Un parcours de map est aleatoire, et un sidecar qui change d'octets sans que rien
// n'ait change est indebuggable (et se re-ecrirait a chaque passe de rattrapage).
func assembler(parJoueur map[string]*OccupationJoueur, vues map[string]map[Cellule]int) []OccupationJoueur {
	out := make([]OccupationJoueur, 0, len(parJoueur))
	for xuid, j := range parJoueur {
		sort.Slice(j.Spawns, func(a, b int) bool { return j.Spawns[a].Frame < j.Spawns[b].Frame })
		entrees := make([]EntreeCellule, 0, len(vues[xuid]))
		for c, frame := range vues[xuid] {
			entrees = append(entrees, EntreeCellule{Col: c.Col, Lig: c.Lig, Frame: frame})
		}
		sort.Slice(entrees, func(a, b int) bool {
			if entrees[a].Col != entrees[b].Col {
				return entrees[a].Col < entrees[b].Col
			}
			return entrees[a].Lig < entrees[b].Lig
		})
		j.PremieresEntrees = entrees
		out = append(out, *j)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].XUID < out[b].XUID })
	return out
}

// SecondesParEchantillon rend la duree que vaut UN echantillon, en secondes.
//
// Elle existe pour que la conversion « echantillons -> temps » ait UNE seule ecriture :
// le sidecar stocke des echantillons (des entiers, exacts et sommables), et la lecture
// les rend en secondes. Recopier `pas/1000` cote service en ferait une seconde
// definition, qui divergerait au premier changement de pas.
func SecondesParEchantillon(pasMs int) float64 {
	if pasMs <= 0 {
		return 0
	}
	return float64(pasMs) / MsParSeconde
}

// EnSecondes convertit la VALEUR PAR MATCH de cellules d'occupation — comptees en
// echantillons — en secondes par match. Le compte BRUT reste en echantillons : c'est la
// mesure, la seconde n'en est que l'unite de lecture (doctrine « jamais un taux seul »).
func EnSecondes(cellules []domain.CelluleTactique, pasMs int) []domain.CelluleTactique {
	s := SecondesParEchantillon(pasMs)
	out := make([]domain.CelluleTactique, len(cellules))
	copy(out, cellules)
	for i := range out {
		out[i].Valeur *= s
	}
	return out
}
