package tactical

import "levelup/go-api/internal/domain"

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

	// resultats : matchID -> code de resultat (domain.OutcomeWin / OutcomeLoss / ...).
	// Tous les matchs ayant place au moins un point y figurent, y compris ceux dont le
	// resultat est inconnu (domain.OutcomeUnknown).
	resultats map[string]int

	// ignores : les points ecartes parce que leur position n'etait pas finie. Compte
	// expose plutot qu'avale : un decodage qui derape se voit ici.
	ignores int
}

// Rasterise compte les points sur la grille. Le resultat de chaque match reste inconnu :
// c'est la forme employee par les lectures non signees (ou je meurs, ou je tue).
func Rasterise(g Grille, points []domain.PositionSample) *Raster {
	return RasteriseAvecResultats(g, points, nil)
}

// RasteriseAvecResultats compte les points sur la grille en retenant le resultat de chaque
// match (`resultats` : matchID -> domain.OutcomeWin / OutcomeLoss / ...), ce dont la
// lecture signee « ou je gagne » a besoin. Un match absent de la table est traite comme
// domain.OutcomeUnknown : il compte dans les passages, dans aucun des deux cotes.
func RasteriseAvecResultats(g Grille, points []domain.PositionSample, resultats map[string]int) *Raster {
	r := &Raster{
		grille:    g,
		cellules:  make(map[Cellule]map[string]int),
		resultats: make(map[string]int, len(resultats)),
	}
	for _, p := range points {
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
		// Un match n'est retenu que s'il a place au moins un point : un match dont toutes
		// les positions sont illisibles ne doit pas alourdir le denominateur « par match ».
		if _, connu := r.resultats[p.MatchID]; !connu {
			r.resultats[p.MatchID] = resultats[p.MatchID]
		}
	}
	return r
}

// PasM rend le pas de la grille du raster, en metres.
func (r *Raster) PasM() float64 { return r.grille.PasM() }

// NbCellules rend le nombre de cellules ALIMENTEES (avant tout plancher).
func (r *Raster) NbCellules() int { return len(r.cellules) }

// NbMatchs rend le nombre de matchs distincts ayant alimente au moins une cellule. C'est
// le denominateur de la valeur « par match » d'une lecture non signee.
func (r *Raster) NbMatchs() int { return len(r.resultats) }

// NbMatchsResultat rend le nombre de matchs distincts du raster dont le resultat vaut
// `code` (domain.OutcomeWin, domain.OutcomeLoss, ...).
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

// Bornes rend le rectangle englobant les cellules alimentees, en metres monde. Non valide
// tant qu'aucune cellule n'est alimentee.
func (r *Raster) Bornes() domain.BornesMonde {
	var b domain.BornesMonde
	for c := range r.cellules {
		b = UnionBornes(b, r.grille.BornesDe(c))
	}
	return b
}
