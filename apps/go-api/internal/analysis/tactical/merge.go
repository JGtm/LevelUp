package tactical

import (
	"errors"
	"fmt"
	"sort"

	"levelup/go-api/internal/domain"
)

// PlancherMatchsParCellule est le nombre minimal de matchs DISTINCTS qu'une cellule doit
// compter pour etre lue. Trois, valeur mesuree par cmd/mappos-build le 2026-08-30 : sans
// filtre le nuage des positions s'etend a 268 m du centre de l'arene, a deux matchs il
// retombe a 27 m, a trois a 19,4 m (le 99e centile du nuage lui-meme). En dessous, ce sont
// des cellules vues une fois dans un seul match, qui tirent des bras hors de la carte.
const PlancherMatchsParCellule = 3

// PlancherMatchsParCote est le meme plancher, applique A CHACUN DES DEUX COTES d'une lecture
// signee : au moins trois victoires ET trois defaites dans la cellule. Un cote seul ne
// mesure pas un ecart, il mesure une absence — trois victoires et zero defaite ne disent pas
// que la cellule fait gagner, elles disent qu'on n'y a jamais perdu ASSEZ pour le savoir.
const PlancherMatchsParCote = 3

var (
	// ErrAucunRaster : sommer zero raster ne rend pas un raster vide, ca rend une erreur —
	// un appelant qui somme une liste vide s'est trompe de filtre.
	ErrAucunRaster = errors.New("tactical: aucun raster a sommer")
	// ErrPasIncompatible : deux rasters de pas differents n'adressent pas les memes
	// cellules ; les sommer melangerait deux resolutions.
	ErrPasIncompatible = errors.New("tactical: rasters de pas differents")
	// ErrResultatIncoherent : le meme match porte deux resultats selon le raster. C'est une
	// erreur d'appel, pas une donnee a arbitrer en silence.
	ErrResultatIncoherent = errors.New("tactical: resultats contradictoires pour un meme match")
)

// Somme agrege des rasters en un seul. Le cas nominal est un raster PAR MATCH (calcule une
// fois a la cuisson) que la page somme a l'affichage.
//
// Le meme match peut apparaitre dans plusieurs rasters (deux vues partielles d'un match,
// par exemple ses morts et ses kills) : ses passages s'additionnent, et il ne compte qu'UNE
// FOIS dans les matchs distincts d'une cellule comme dans le denominateur par match.
func Somme(rasters ...*Raster) (*Raster, error) {
	if len(rasters) == 0 {
		return nil, ErrAucunRaster
	}
	out := &Raster{
		grille:    Grille{pasM: rasters[0].PasM()},
		cellules:  make(map[Cellule]map[string]int),
		resultats: make(map[string]int),
	}
	for _, r := range rasters {
		if r.PasM() != out.PasM() {
			return nil, fmt.Errorf("%w (%v et %v)", ErrPasIncompatible, out.PasM(), r.PasM())
		}
		for matchID, res := range r.resultats {
			connu, deja := out.resultats[matchID]
			if deja && connu != res {
				return nil, fmt.Errorf("%w (match %s : %d et %d)", ErrResultatIncoherent, matchID, connu, res)
			}
			out.resultats[matchID] = res
		}
		for c, parMatch := range r.cellules {
			cible := out.cellules[c]
			if cible == nil {
				cible = make(map[string]int, len(parMatch))
				out.cellules[c] = cible
			}
			for matchID, occ := range parMatch {
				cible[matchID] += occ
			}
		}
		out.ignores += r.ignores
	}
	return out, nil
}

// Cellules rend les cellules lisibles d'une lecture NON signee (ou je meurs, ou je tue, ou
// je passe mon temps), triees pour un rendu stable.
//
// Deux regles y sont appliquees, et elles sont indissociables :
//   - le plancher de rarete (PlancherMatchsParCellule matchs distincts) ;
//   - la valeur PAR MATCH, divisee par le nombre total de matchs du raster et non par les
//     seuls matchs de la cellule. Diviser par les matchs de la cellule rendrait 1,0 aussi
//     bien pour trois passages sur trois matchs que pour trente sur trente, et effacerait
//     exactement ce que la lecture cherche : l'intensite.
func (r *Raster) Cellules() []domain.CelluleTactique {
	total := float64(r.NbMatchs())
	out := make([]domain.CelluleTactique, 0, len(r.cellules))
	for c, parMatch := range r.cellules {
		if len(parMatch) < PlancherMatchsParCellule {
			continue
		}
		brut := 0
		for _, occ := range parMatch {
			brut += occ
		}
		cell := r.celluleDeBase(c)
		cell.Brut = float64(brut)
		cell.Matchs = len(parMatch)
		if total > 0 {
			cell.Valeur = float64(brut) / total
		}
		out = append(out, cell)
	}
	trierCellules(out)
	return out
}

// CellulesSignees rend les cellules lisibles de la lecture SIGNEE (« ou je gagne ») :
// difference entre le cote victoire et le cote defaite, chacun ramene a SON propre nombre
// de matchs.
//
// POURQUOI CHAQUE COTE EST NORMALISE SEPAREMENT : avec 20 victoires et 5 defaites, une
// cellule traversee 20 fois en victoire et 5 fois en defaite est NEUTRE (meme rythme des
// deux cotes) ; une difference brute y lirait +15 et peindrait une zone gagnante qui n'est
// que le reflet du taux de victoire global. La valeur est donc un ecart de taux par match.
//
// Le plancher est applique PAR COTE (PlancherMatchsParCote) : une cellule vue dans trois
// victoires et aucune defaite est RETIREE, pas peinte en positif.
func (r *Raster) CellulesSignees() []domain.CelluleTactique {
	nbV := float64(r.NbMatchsResultat(domain.OutcomeWin))
	nbD := float64(r.NbMatchsResultat(domain.OutcomeLoss))
	out := make([]domain.CelluleTactique, 0, len(r.cellules))
	for c, parMatch := range r.cellules {
		occV, matchsV := r.cumulCote(parMatch, domain.OutcomeWin)
		occD, matchsD := r.cumulCote(parMatch, domain.OutcomeLoss)
		if matchsV < PlancherMatchsParCote || matchsD < PlancherMatchsParCote {
			continue
		}
		cell := r.celluleDeBase(c)
		cell.Brut = float64(occV - occD)
		// Matchs porte ici les seuls matchs qui alimentent la valeur signee : les nuls et
		// les matchs de resultat inconnu ne participent a aucun des deux cotes.
		cell.Matchs = matchsV + matchsD
		cell.MatchsVictoire = matchsV
		cell.MatchsDefaite = matchsD
		cell.Valeur = float64(occV)/nbV - float64(occD)/nbD
		out = append(out, cell)
	}
	trierCellules(out)
	return out
}

// cumulCote rend les passages et les matchs distincts d'une cellule pour un cote donne.
func (r *Raster) cumulCote(parMatch map[string]int, code int) (occurrences, matchs int) {
	for matchID, occ := range parMatch {
		if r.resultats[matchID] != code {
			continue
		}
		occurrences += occ
		matchs++
	}
	return occurrences, matchs
}

// celluleDeBase pose l'adresse et le centre ; les comptes sont remplis par l'appelant.
func (r *Raster) celluleDeBase(c Cellule) domain.CelluleTactique {
	x, y := r.grille.Centre(c)
	return domain.CelluleTactique{Col: c.Col, Lig: c.Lig, CentreX: x, CentreY: y}
}

// trierCellules ordonne par colonne puis par ligne : la sortie d'un parcours de map est
// aleatoire, et un rendu qui bouge sans que rien n'ait change est indebuggable.
func trierCellules(cellules []domain.CelluleTactique) {
	sort.Slice(cellules, func(i, j int) bool {
		if cellules[i].Col != cellules[j].Col {
			return cellules[i].Col < cellules[j].Col
		}
		return cellules[i].Lig < cellules[j].Lig
	})
}
