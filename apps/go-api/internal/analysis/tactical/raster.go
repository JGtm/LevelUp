package tactical

import (
	"errors"
	"fmt"
	"sort"

	"levelup/go-api/internal/domain"
)

// ErrMatchHorsUnivers : un point porte un match qui n'appartient pas a l'univers declare.
// C'est un bug d'appelant (un filtre qui ne dit pas la meme chose que la lecture), pas une
// donnee a arbitrer : ni avale, ni compte en silence.
var ErrMatchHorsUnivers = errors.New("tactical: point d'un match absent de l'univers")

// Raster est le comptage d'un ensemble de positions sur une grille : par cellule, le
// nombre de passages et les matchs qui les ont produits.
//
// UNE CELLULE JAMAIS ATTEINTE N'EXISTE PAS. Il n'y a pas de zero explicite dans un raster,
// et c'est une decision produit : une cellule sans passage est du terrain non observe (ou
// pas du terrain du tout), pas une cellule froide. La peindre en froid inventerait une
// mesure. La table est donc creuse, et sa taille suit ce qui a ete joue.
//
// POURQUOI LE COMPTE EST GARDE PAR MATCH (`cellules[c][matchID]`) plutot qu'en un seul
// entier : le plancher de rarete se compte en matchs DISTINCTS, et la lecture signee a
// besoin de savoir quels passages viennent des victoires et lesquels des defaites. Un
// compteur global perdrait les deux, et il faudrait alors trois jeux de compteurs paralleles
// tenus en phase a la main.
type Raster struct {
	grille Grille

	// cellules : cellule -> matchID -> nombre de passages. Creuse (cf. ci-dessus).
	cellules map[Cellule]map[string]int

	// resultats est L'UNIVERS : tous les matchs RETENUS par le filtre, avec leur code de
	// resultat (domain.OutcomeWin / OutcomeLoss / OutcomeUnknown / ...). Il vient de
	// l'appelant et n'est JAMAIS deduit des points — cf. Rasterise.
	resultats map[string]int

	// ignores : les points ecartes parce que leur position n'etait pas finie. Compte
	// expose plutot qu'avale : un decodage qui derape se voit ici.
	ignores int
}

// Rasterise compte les points sur la grille pour l'univers `matchs` — la liste des matchs
// RETENUS par le filtre. Le resultat de chacun reste inconnu : c'est la forme employee par
// les lectures non signees (ou je meurs, ou je tue).
func Rasterise(g Grille, matchs []string, points []domain.PositionSample) (*Raster, error) {
	resultats := make(map[string]int, len(matchs))
	for _, m := range matchs {
		resultats[m] = domain.OutcomeUnknown
	}
	return RasteriseAvecResultats(g, resultats, points)
}

// RasteriseAvecResultats compte les points sur la grille ; la table `resultats` (matchID ->
// domain.OutcomeWin / OutcomeLoss / ...) EST l'univers des matchs retenus. Un match retenu
// dont le resultat n'est pas connu y figure avec domain.OutcomeUnknown : il compte au
// denominateur, dans aucun des deux cotes.
//
// L'UNIVERS EST UNE ENTREE, JAMAIS UNE DEDUCTION DES POINTS, et c'est le coeur de la
// mesure. Un match retenu par le filtre peut n'avoir AUCUN point sur la lecture courante —
// un match sans kill pour « ou je tue », un match sans mort pour « ou je meurs » : c'est un
// zero LEGITIME, qui doit compter au denominateur « par match ». Le deduire des points
// l'effacerait, et la lecture signee peindrait alors une zone gagnante la ou il n'y a que
// des matchs muets du cote adverse (12 victoires dont 2 sans kill : 6/10 - 4/8 = +0,10 au
// lieu de 6/12 - 4/8 = 0,00).
//
// A ne pas confondre avec les points ILLISIBLES (position non finie) : ceux-la ne sont pas
// un zero, ils sont un decodage rate, et ils se comptent a part dans PointsIgnores.
//
// Un point dont le match n'est pas dans l'univers rend ErrMatchHorsUnivers.
func RasteriseAvecResultats(g Grille, resultats map[string]int, points []domain.PositionSample) (*Raster, error) {
	r := &Raster{
		grille:    g,
		cellules:  make(map[Cellule]map[string]int),
		resultats: make(map[string]int, len(resultats)),
	}
	for matchID, res := range resultats {
		r.resultats[matchID] = res
	}
	for _, p := range points {
		if _, retenu := r.resultats[p.MatchID]; !retenu {
			return nil, fmt.Errorf("%w (match %q)", ErrMatchHorsUnivers, p.MatchID)
		}
		c, ok := g.Cellule(p.X, p.Y)
		if !ok {
			r.ignores++
			continue
		}
		parMatch := r.cellules[c]
		if parMatch == nil {
			parMatch = make(map[string]int)
			r.cellules[c] = parMatch
		}
		parMatch[p.MatchID]++
	}
	return r, nil
}

// PasM rend le pas de la grille du raster, en metres.
func (r *Raster) PasM() float64 { return r.grille.PasM() }

// NbCellules rend le nombre de cellules ALIMENTEES (avant tout plancher).
func (r *Raster) NbCellules() int { return len(r.cellules) }

// NbMatchs rend la taille de l'univers : tous les matchs retenus, y compris ceux qui n'ont
// alimente aucune cellule. C'est le denominateur de la valeur « par match » d'une lecture
// non signee.
func (r *Raster) NbMatchs() int { return len(r.resultats) }

// NbMatchsResultat rend le nombre de matchs de l'univers dont le resultat vaut `code`
// (domain.OutcomeWin, domain.OutcomeLoss, ...).
func (r *Raster) NbMatchsResultat(code int) int {
	n := 0
	for _, res := range r.resultats {
		if res == code {
			n++
		}
	}
	return n
}

// Occurrences rend le nombre de passages comptes dans une cellule. Zero pour une cellule
// jamais atteinte — qui n'existe pas dans la table.
func (r *Raster) Occurrences(c Cellule) int {
	n := 0
	for _, occ := range r.cellules[c] {
		n += occ
	}
	return n
}

// MatchsDistincts rend le nombre de matchs distincts ayant alimente une cellule. C'est la
// grandeur du plancher de rarete : deux matchs differents sont deux observations
// independantes, alors qu'un joueur immobile ne fait que gonfler les passages.
func (r *Raster) MatchsDistincts(c Cellule) int { return len(r.cellules[c]) }

// PointsIgnores rend le nombre de points ecartes faute de position finie.
func (r *Raster) PointsIgnores() int { return r.ignores }

// Bornes rend le rectangle englobant les cellules LISIBLES — celles qui passent le plancher
// de rarete, donc celles que le peintre pose. Non valide tant qu'aucune cellule n'est
// lisible.
//
// POURQUOI LE PLANCHER EST APPLIQUE ICI AUSSI : sur Dredge, des cellules vues une seule fois
// dans un seul match s'etendent a 268 m du centre de l'arene, la ou les cellules a trois
// matchs tiennent dans 19,4 m (mesure du 2026-08-30, cf. doc.go). Un cadrage sur les
// cellules alimentees rendrait donc une carte quatorze fois trop large pour ce qui y est
// peint. Les cellules de la lecture SIGNEE sont un sous-ensemble de celles-ci (trois
// victoires ET trois defaites font au moins six matchs distincts) : ce cadre les englobe.
func (r *Raster) Bornes() domain.BornesMonde {
	var b domain.BornesMonde
	for c, parMatch := range r.cellules {
		if len(parMatch) < PlancherMatchsParCellule {
			continue
		}
		b = UnionBornes(b, r.grille.BornesDe(c))
	}
	return b
}

// CelluleBrute est une cellule ALIMENTEE d'un raster, avec son compte NON NORMALISE et
// SANS aucun plancher.
type CelluleBrute struct {
	Cellule
	Occurrences int
}

// CellulesBrutes rend toutes les cellules alimentees, sans plancher et sans division —
// la forme qu'on STOCKE, pas celle qu'on affiche.
//
// # POURQUOI ELLE EXISTE A COTE DE Cellules()
//
// `Cellules()` est la forme de LECTURE : elle applique le plancher de rarete (trois
// matchs distincts) et la valeur par match. Sur un raster d'UN SEUL match, ce plancher
// vide tout par construction — une cellule d'un match compte un match distinct. Un
// sidecar par match ecrit avec `Cellules()` serait donc TOUJOURS vide, et le plancher se
// serait applique deux fois : une fois a l'ecriture, ou il n'a aucun sens, et une fois a
// la somme, ou il en a un.
//
// LE PLANCHER APPARTIENT A L'AGREGAT, JAMAIS AU MATCH. C'est toute la doctrine du
// stockage par match : chaque fichier porte une mesure BRUTE et complete, et les regles
// de lecture (rarete, normalisation, echelle) s'appliquent une seule fois, sur la somme.
//
// Trie par colonne puis ligne : la sortie d'un parcours de map est aleatoire, et un
// sidecar qui change d'octets sans que rien n'ait change se re-ecrirait a chaque passe de
// rattrapage.
func (r *Raster) CellulesBrutes() []CelluleBrute {
	out := make([]CelluleBrute, 0, len(r.cellules))
	for c, parMatch := range r.cellules {
		occ := 0
		for _, n := range parMatch {
			occ += n
		}
		out = append(out, CelluleBrute{Cellule: c, Occurrences: occ})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Col != out[j].Col {
			return out[i].Col < out[j].Col
		}
		return out[i].Lig < out[j].Lig
	})
	return out
}
