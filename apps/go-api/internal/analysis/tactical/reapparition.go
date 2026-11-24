package tactical

// reapparition.go — LE MODELE DE REAPPARITION : ou le jeu fait reapparaitre un joueur, d'apres
// la position des autres.
//
// # LA QUESTION ET LE MODELE
//
// Le jeu choisit, parmi les points de reapparition d'une carte, celui qui convient a la
// position des joueurs ; ce choix se prevoit. Le modele est un CHOIX DISCRET (logit
// conditionnel) : chaque point candidat recoit un score, somme ponderee de variables, et la
// probabilite d'un point est le softmax des scores. Les poids se mesurent par maximum de
// vraisemblance sur les reapparitions observees.
//
// # LES POINTS CANDIDATS SE DEDUISENT DU JEU OBSERVE
//
// Aucun catalogue : un point est candidat s'il a ete observe dans au moins MatchsMinCandidat
// matchs distincts, a la maille de 0,5 m en plan et d'un metre en hauteur. Pour prevoir (ou
// evaluer) un match, ces matchs sont d'AUTRES matchs : le point reel du match prevu ne devient
// pas candidat parce que ce match l'a vu. Une reapparition qui tombe hors des candidats ne peut
// pas etre predite exactement : elle se compte (hors candidats) plutot que de se perdre.
//
// # LES VARIABLES
//
// Six variables, dans l'ordre de NomsVariables : des distances ou des comptes de la position des
// autres joueurs. Les distances sont plafonnees (DistanceMaxM, DistanceMaxMortM) et exprimees en
// dizaines de metres pour que les poids restent lisibles.
//
// LA FREQUENCE (VarFrequence) SE LIT HORS DU MATCH EVALUE : un point observe dans ce match ne
// peut pas servir a le predire lui-meme.
//
// PUR : aucune I/O.

import (
	"math"
	"sort"
)

// Point3 est une position en metres monde.
type Point3 struct{ X, Y, Z float64 }

// Distance rend la distance euclidienne en 3D.
func (p Point3) Distance(q Point3) float64 {
	return math.Sqrt((p.X-q.X)*(p.X-q.X) + (p.Y-q.Y)*(p.Y-q.Y) + (p.Z-q.Z)*(p.Z-q.Z))
}

// Variables du modele, par indice.
const (
	VarEnnemiProche = iota
	VarEnnemisProches
	VarEquipierProche
	VarDistanceMort
	VarFrequence
	VarBarycentreEnnemi
	NbVariables
)

// NomsVariables : le libelle technique de chaque variable (rapports, fichier de poids).
var NomsVariables = [NbVariables]string{
	"ennemi_le_plus_proche_10m",
	"ennemis_a_moins_de_20m",
	"equipier_le_plus_proche_10m",
	"distance_au_lieu_de_la_mort_10m",
	"frequence_log",
	"distance_au_barycentre_ennemi_10m",
}

// Seuils du modele. TOUTES LES DISTANCES DE CE MODELE SONT EN UNITES DU JEU (les positions du film, 1
// unite = 3,048 m) : le suffixe « M » de ces noms, et le « m » des commentaires et des noms de variables
// (`..._10m`), disent des unites du jeu, pas des metres. Les valeurs ont ete reglees sur ces unites ;
// les convertir changerait le modele.
const (
	// MatchsMinCandidat : matchs distincts qui ont vu un point pour qu'il soit candidat.
	MatchsMinCandidat = 2
	// DistanceMaxM plafonne la distance a l'ennemi ou a l'equipier le plus proche quand il n'y
	// en a aucun : « personne » vaut 40 unites.
	DistanceMaxM = 40.0
	// DistanceMaxMortM plafonne la distance au lieu de la mort et au barycentre ennemi.
	DistanceMaxMortM = 60.0
	// RayonEnnemisProchesM : le rayon du compte d'ennemis proches.
	RayonEnnemisProchesM = 20.0
	// DizaineM convertit des unites du jeu en unites de variable (dizaines).
	DizaineM = 10.0
)

// EvenementReapparition est une reapparition observee, avec l'etat du jeu qui la determine.
type EvenementReapparition struct {
	MatchID string
	CarteID string
	// Reel est le point ou le joueur est reapparu.
	Reel Point3
	// Mort est le lieu de la mort qui precede.
	Mort Point3
	// Ennemis et Equipiers sont les joueurs vivants a l'instant considere (la reapparition, ou
	// la mort selon la mesure), hors le joueur lui-meme.
	Ennemis, Equipiers []Point3
}

// CleCandidat est l'adresse d'un point candidat : 0,5 m en plan, 1 m en hauteur.
type CleCandidat struct{ X, Y, Z int }

// CleDe rend la cle d'un point.
func CleDe(p Point3) CleCandidat {
	return CleCandidat{X: int(math.Round(p.X * 2)), Y: int(math.Round(p.Y * 2)), Z: int(math.Round(p.Z))}
}

// Candidat est un point de reapparition connu d'une carte.
type Candidat struct {
	Cle   CleCandidat
	Pos   Point3
	Total int
	// ParMatch : le nombre de reapparitions vues en ce point, par match.
	ParMatch map[string]int
}

// MatchsHors rend le nombre de matchs distincts qui ont vu le point, sans compter `match`.
func (c Candidat) MatchsHors(match string) int {
	n := len(c.ParMatch)
	if _, vu := c.ParMatch[match]; vu {
		n--
	}
	return n
}

// CandidatPour dit si le point est candidat pour une reapparition du match `match` : vu dans
// au moins MatchsMinCandidat AUTRES matchs. Un point que ce match est seul a completer n'est
// pas candidat : une prevision ne voit jamais le match qu'elle prevoit.
func (c Candidat) CandidatPour(match string) bool {
	return c.MatchsHors(match) >= MatchsMinCandidat
}

// PointsCandidats deduit les candidats d'une carte : les points vus dans au moins
// MatchsMinCandidat matchs distincts, du plus frequent au moins frequent (a egalite, ordre des
// cles : le resultat est deterministe). C'est l'ENSEMBLE MAXIMAL : celui d'un match precis se
// restreint par CandidatPour.
func PointsCandidats(evts []EvenementReapparition) []Candidat {
	par := map[CleCandidat]*Candidat{}
	for _, e := range evts {
		k := CleDe(e.Reel)
		c := par[k]
		if c == nil {
			c = &Candidat{Cle: k, Pos: e.Reel, ParMatch: map[string]int{}}
			par[k] = c
		}
		c.ParMatch[e.MatchID]++
		c.Total++
	}
	out := make([]Candidat, 0, len(par))
	for _, c := range par {
		if len(c.ParMatch) >= MatchsMinCandidat {
			out = append(out, *c)
		}
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Total != out[b].Total {
			return out[a].Total > out[b].Total
		}
		x, y := out[a].Cle, out[b].Cle
		if x.X != y.X {
			return x.X < y.X
		}
		if x.Y != y.Y {
			return x.Y < y.Y
		}
		return x.Z < y.Z
	})
	return out
}

// Variables calcule les variables d'un point candidat pour un evenement. `freq` est le compte de
// reapparitions vues en ce point HORS du match de l'evenement.
func Variables(e EvenementReapparition, c Candidat, freq int) [NbVariables]float64 {
	minE, minM, proches := DistanceMaxM, DistanceMaxM, 0.0
	var cx, cy, cz float64
	for _, p := range e.Ennemis {
		d := p.Distance(c.Pos)
		minE = math.Min(minE, d)
		if d < RayonEnnemisProchesM {
			proches++
		}
		cx, cy, cz = cx+p.X, cy+p.Y, cz+p.Z
	}
	bary := 0.0
	if n := float64(len(e.Ennemis)); n > 0 {
		bary = math.Min(Point3{cx / n, cy / n, cz / n}.Distance(c.Pos), DistanceMaxMortM) / DizaineM
	}
	for _, p := range e.Equipiers {
		minM = math.Min(minM, p.Distance(c.Pos))
	}
	var v [NbVariables]float64
	v[VarEnnemiProche] = minE / DizaineM
	v[VarEnnemisProches] = proches
	v[VarEquipierProche] = minM / DizaineM
	v[VarDistanceMort] = math.Min(e.Mort.Distance(c.Pos), DistanceMaxMortM) / DizaineM
	v[VarFrequence] = math.Log1p(float64(freq))
	v[VarBarycentreEnnemi] = bary
	return v
}

// Score est la somme ponderee des variables.
func Score(v [NbVariables]float64, w [NbVariables]float64) float64 {
	s := 0.0
	for j := range v {
		s += w[j] * v[j]
	}
	return s
}

// Probabilites rend le softmax des scores (somme 1). Une liste vide rend nil.
func Probabilites(scores []float64) []float64 {
	if len(scores) == 0 {
		return nil
	}
	mx := scores[0]
	for _, s := range scores {
		mx = math.Max(mx, s)
	}
	out := make([]float64, len(scores))
	z := 0.0
	for i, s := range scores {
		out[i] = math.Exp(s - mx)
		z += out[i]
	}
	for i := range out {
		out[i] /= z
	}
	return out
}
