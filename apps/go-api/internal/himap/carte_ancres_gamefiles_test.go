package himap

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// LES ANCRES — l'oracle FAIBLE, mais disponible sur les 37 cartes.
//
// Pourquoi ce fichier existe. L'oracle des positions de joueur ne vaut que pour Cliffhanger :
// c'est la seule carte dont un film est decode. Regler la chaine carte par carte est donc
// impossible en principe, pas seulement penible. Il faut des ancres presentes PARTOUT.
//
// `data/titles/halo_infinite/reference/map_objectives.json` en porte : 37 cartes, chacune
// avec les positions x/y/z de ses objectifs (zones d'extraction, apparitions de drapeau,
// collines). Ce sont des points poses dans l'espace de JEU, par construction.
//
// Mais leur `z` n'est pas forcement l'altitude du SOL : une zone est un volume, sa position
// peut en designer le centre. Cliffhanger possede les DEUX oracles a la fois — c'est donc la
// qu'on etalonne ce decalage, une seule fois, et l'ancre devient utilisable sur les 36 autres.
//
// L'etalonnage ne passe PAS par la geometrie reconstruite : il compare l'ancre aux positions
// de joueur voisines. Donnee contre donnee, sans intermediaire faillible.

const moduleCliffhanger = "cliffhanger_ridgeline"

// rayonVoisinage : demi-largeur, en metres, de la fenetre horizontale dans laquelle on
// cherche des positions de joueur au pied d'une ancre.
const rayonVoisinage = 3.0

// cheminCatalogue trouve le catalogue d'objectifs par remontee depuis le repertoire de test.
func cheminCatalogue() (string, error) {
	const rel = "data/titles/halo_infinite/reference/map_objectives.json"
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for d := wd; ; {
		c := filepath.Join(d, rel)
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("catalogue d'objectifs introuvable depuis %s", wd)
		}
		d = parent
	}
}

// ancresDuModule rend les objectifs de la carte designee par son nom de module.
func ancresDuModule(t *testing.T, module string) []replay.MapObjectivesEntry {
	t.Helper()
	p, err := cheminCatalogue()
	if err != nil {
		t.Skip(err)
	}
	cat, err := replay.LoadMapObjectives(p)
	if err != nil {
		t.Fatal(err)
	}
	var out []replay.MapObjectivesEntry
	for _, e := range cat.Maps {
		if e.Module == module {
			out = append(out, e)
		}
	}
	return out
}

// toleranceRendu : ecart maximal, en metres, entre une bande praticable et la surface de
// reference pour qu'elle soit dessinee. Vient de la dispersion mesuree de l'etalonnage
// (interquartile 0,46 m sur Cliffhanger), elargie pour couvrir les etages intermediaires
// legitimes. Constante UNIVERSELLE : elle ne se regle pas par carte.
var toleranceRendu = 6.0

// surfaceDesAncres construit la surface de reference d'une carte depuis le catalogue
// d'objectifs. Rend une surface VIDE si la carte n'y figure pas — l'appelant doit alors se
// replier explicitement, jamais silencieusement.
func surfaceDesAncres(t *testing.T, module string) *SurfaceReference {
	t.Helper()
	if module == "" {
		module = moduleCliffhanger
	}
	var pts [][3]float64
	for _, e := range ancresDuModule(t, module) {
		for _, o := range e.Objectives {
			pts = append(pts, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	t.Logf("surface de reference : %d ancres pour le module %s", len(pts), module)
	return NewSurfaceReference(pts)
}

// TestAncresEtalonnageCliffhanger mesure le decalage entre le z d'une ancre et l'altitude des
// pieds des joueurs a son aplomb.
func TestAncresEtalonnageCliffhanger(t *testing.T) {
	entrees := ancresDuModule(t, moduleCliffhanger)
	if len(entrees) == 0 {
		t.Skipf("aucune entree pour le module %s", moduleCliffhanger)
	}
	pos := chargePositions(t)

	type mesure struct {
		role    string
		z       float64
		basZone float64
		sol     float64
		n       int
	}
	var mesures []mesure
	for _, e := range entrees {
		t.Logf("carte %s · %d objectifs", e.Module, len(e.Objectives))
		for _, o := range e.Objectives {
			var zs []float64
			for _, p := range pos {
				if math.Abs(p.X-o.Pos.X) <= rayonVoisinage && math.Abs(p.Y-o.Pos.Y) <= rayonVoisinage {
					zs = append(zs, p.Z)
				}
			}
			if len(zs) < 5 {
				continue
			}
			sort.Float64s(zs)
			m := mesure{role: string(o.Role), z: o.Pos.Z, sol: zs[len(zs)/2], n: len(zs)}
			if o.Shape != nil {
				m.basZone = o.Pos.Z - o.Shape.DownZ
			} else {
				m.basZone = math.NaN()
			}
			mesures = append(mesures, m)
		}
	}
	if len(mesures) == 0 {
		t.Skip("aucune ancre n'a de positions de joueur a son aplomb")
	}

	var dCentre, dBas []float64
	t.Log("role                  |  z ancre | bas zone |  sol joueur | ecart centre | ecart bas | n")
	for _, m := range mesures {
		t.Logf("%-21s | %+8.2f | %+8.2f | %+11.2f | %+12.2f | %+9.2f | %d",
			m.role, m.z, m.basZone, m.sol, m.z-m.sol, m.basZone-m.sol, m.n)
		dCentre = append(dCentre, m.z-m.sol)
		if !math.IsNaN(m.basZone) {
			dBas = append(dBas, m.basZone-m.sol)
		}
	}
	t.Logf("decalage CENTRE de l'ancre au sol : median %+.2f m  (dispersion %.2f m, n=%d)",
		mediane(dCentre), dispersion(dCentre), len(dCentre))
	t.Logf("decalage BAS de la zone au sol    : median %+.2f m  (dispersion %.2f m, n=%d)",
		mediane(dBas), dispersion(dBas), len(dBas))
}

// dispersion rend l'ecart interquartile — moins sensible qu'un ecart-type aux quelques ancres
// posees en hauteur, qui ne sont pas des erreurs mais des cas legitimes.
func dispersion(v []float64) float64 {
	if len(v) < 4 {
		return math.NaN()
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[3*len(c)/4] - c[len(c)/4]
}
