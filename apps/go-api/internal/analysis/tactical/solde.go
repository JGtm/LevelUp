package tactical

// solde.go — LA LECTURE « SOLDE FRAGS − MORTS » : par cellule, ce que le joueur (ou l'axe « qui »)
// y a tué moins ce qu'il y a perdu, chaque face ramenée aux matchs de l'univers.
//
// # LA MÊME MACHINERIE QUE LA LECTURE SIGNÉE, DEUX FACES AU LIEU DE DEUX ISSUES
//
// `CellulesSignees` sépare les passages par l'ISSUE du match (victoire, défaite). Le solde les
// sépare par la FACE de l'événement (le tueur sur un frag, la victime sur une mort). Le reste est
// commun : l'univers est une entrée (un match retenu sans aucun point compte au dénominateur), la
// valeur est un écart de taux par match, l'échelle est symétrique.
//
// # LE PLANCHER PORTE SUR L'UNION DES DEUX FACES
//
// Trois matchs DISTINCTS par cellule, comptés sur tous les événements de la cellule, frags et
// morts confondus : une cellule où l'on a tué dans deux matchs et où l'on est mort dans un
// troisième est observée trois fois. Exiger trois matchs PAR FACE retirerait les cellules les
// plus tranchées — celles où l'on ne fait presque que tuer, ou que mourir.
//
// # UN SEUL DÉNOMINATEUR
//
// Les deux faces se divisent par le MÊME nombre de matchs (l'univers mesuré) : la valeur vaut
// donc (frags − morts) / N, et `Brut` porte frags − morts.

import (
	"levelup/go-api/internal/domain"
)

// comptesFaces : les deux faces d'une cellule de solde.
type comptesFaces struct {
	frags, morts int
}

// RasteriseSolde compte les deux faces d'une lecture « solde » sur la grille, pour l'univers
// `matchs` (les matchs RETENUS, résultat inconnu — le solde ne lit pas l'issue).
//
// Les passages par match portent l'UNION des deux faces (le plancher en matchs distincts) ; les
// comptes par face voyagent à côté. Un point dont le match n'est pas dans l'univers rend
// ErrMatchHorsUnivers ; un point à position non finie n'entre dans aucune face et se compte dans
// PointsIgnores.
func RasteriseSolde(g Grille, matchs []string, frags, morts []domain.PositionSample) (*Raster, error) {
	tous := make([]domain.PositionSample, 0, len(frags)+len(morts))
	tous = append(append(tous, frags...), morts...)
	r, err := Rasterise(g, matchs, tous)
	if err != nil {
		return nil, err
	}
	r.faces = make(map[Cellule]comptesFaces)
	for _, p := range frags {
		if c, ok := g.Cellule(p.X, p.Y); ok {
			f := r.faces[c]
			f.frags++
			r.faces[c] = f
		}
	}
	for _, p := range morts {
		if c, ok := g.Cellule(p.X, p.Y); ok {
			f := r.faces[c]
			f.morts++
			r.faces[c] = f
		}
	}
	return r, nil
}

// CellulesSolde rend les cellules lisibles d'un raster de solde (RasteriseSolde, ou la Somme de
// tels rasters), triées pour un rendu stable : plancher sur l'union des faces, valeur
// (frags − morts) / N, `Brut` = frags − morts, `Frags` et `Morts` les deux faces.
func (r *Raster) CellulesSolde() []domain.CelluleTactique {
	total := float64(r.NbMatchs())
	out := make([]domain.CelluleTactique, 0, len(r.cellules))
	for c, parMatch := range r.cellules {
		if len(parMatch) < PlancherMatchsParCellule {
			continue
		}
		f := r.faces[c]
		cell := r.celluleDeBase(c)
		cell.Frags, cell.Morts = f.frags, f.morts
		cell.Brut = float64(f.frags - f.morts)
		cell.Matchs = len(parMatch)
		if total > 0 {
			cell.Valeur = float64(f.frags)/total - float64(f.morts)/total
		}
		out = append(out, cell)
	}
	trierCellules(out)
	return out
}

// sommerFaces ajoute les faces d'un raster de solde à celles de la somme (rien sur un raster qui
// n'en porte pas).
func sommerFaces(cible *Raster, source *Raster) {
	if source.faces == nil {
		return
	}
	if cible.faces == nil {
		cible.faces = make(map[Cellule]comptesFaces, len(source.faces))
	}
	for c, f := range source.faces {
		t := cible.faces[c]
		t.frags += f.frags
		t.morts += f.morts
		cible.faces[c] = t
	}
}
