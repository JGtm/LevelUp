package himap

import (
	"math"
	"testing"
)

// grilleTest : un rendu minimal, entierement rempli de matiere, pour observer ce que le rognage
// efface. 20 x 20 cellules d'un metre a partir de l'origine.
func grilleTest() *Rendu {
	r := &Rendu{NX: 20, NY: 20, Cell: 1, Min: [2]float64{0, 0}}
	r.z = make([]float64, r.NX*r.NY)
	r.n = make([][3]float64, r.NX*r.NY)
	for k := range r.z {
		r.z[k] = 1
	}
	return r
}

func matiere(r *Rendu) int {
	n := 0
	for _, z := range r.z {
		if !math.IsInf(z, -1) {
			n++
		}
	}
	return n
}

// UN CORPUS VIDE N'EFFACE RIEN. C'est le garde-fou qui compte le plus : une carte dont aucun film
// n'a ete decode doit sortir INTACTE, jamais videe au motif que personne n'y a jamais marche.
// Meme exigence pour un rayon nul, qui vaudrait « ne garde que les cellules echantillonnees ».
func TestRognageAuxPositionsNeVidePasSansCorpus(t *testing.T) {
	for _, cas := range []struct {
		nom       string
		positions []PositionJouee
		rayon     float64
	}{
		{"corpus vide", nil, 4},
		{"corpus vide, rayon nul", nil, 0},
		{"rayon nul", []PositionJouee{{X: 10, Y: 10}}, 0},
		{"rayon negatif", []PositionJouee{{X: 10, Y: 10}}, -4},
	} {
		r := grilleTest()
		avant := matiere(r)
		if n := r.RogneAuxPositionsJouees(cas.positions, cas.rayon, 0); n != 0 {
			t.Fatalf("%s : %d cellules effacees, aucune attendue", cas.nom, n)
		}
		if apres := matiere(r); apres != avant {
			t.Fatalf("%s : matiere passee de %d a %d", cas.nom, avant, apres)
		}
	}
}

// LE MASQUE GARDE CE QU'ON A PARCOURU ET EFFACE LE RESTE, et le rayon le gouverne de facon
// monotone : plus il est large, plus il en reste.
func TestRognageAuxPositionsGardeLeParcouru(t *testing.T) {
	pos := []PositionJouee{{X: 10.5, Y: 10.5}}

	r := grilleTest()
	r.RogneAuxPositionsJouees(pos, 2, 0)
	if !math.IsInf(r.z[0], -1) {
		t.Fatal("le coin oppose a la position courue aurait du etre efface")
	}
	if math.IsInf(r.z[10*r.NX+10], -1) {
		t.Fatal("la cellule de la position courue elle-meme a ete effacee")
	}

	etroit := matiere(r)
	r2 := grilleTest()
	r2.RogneAuxPositionsJouees(pos, 5, 0)
	if large := matiere(r2); large <= etroit {
		t.Fatalf("un rayon plus large doit garder plus de matiere : %d a 5 m contre %d a 2 m",
			large, etroit)
	}
}

// UNE POSITION HORS CADRE NE MORD PAS AU BORD. Sans le test de bornes, un point a gauche de la
// grille retomberait sur la colonne 0 et y ouvrirait une fenetre qui n'a jamais ete parcourue.
func TestRognageAuxPositionsIgnoreLesPositionsHorsCadre(t *testing.T) {
	r := grilleTest()
	r.RogneAuxPositionsJouees([]PositionJouee{{X: -50, Y: 10.5}, {X: 10.5, Y: 10.5}}, 2, 0)
	if !math.IsInf(r.z[10*r.NX+0], -1) {
		t.Fatal("une position hors cadre a garde de la matiere au bord de la grille")
	}
}

// LE RECOLLEMENT RETIRE UN ELEMENT ENTIER, et NE COMPLETE JAMAIS. La premiere version completait
// les objets majoritairement gardes ; a l image, les grandes dalles du canevas revenaient
// entieres et posaient d immenses rectangles hors de l arene. Ce temoin fige le sens unique.
func TestRecollementRetireSansJamaisCompleter(t *testing.T) {
	r := grilleTest()
	r.ArmeObjetGagnant()
	for j := 0; j < r.NY; j++ {
		r.objetGagnant[j*r.NX+10] = 1
		r.objetGagnant[j*r.NX+0] = 2
	}
	garde := make([]bool, len(r.z))
	for j := 0; j < 16; j++ {
		garde[j*r.NX+10] = true // 16 sur 20 : au-dessus des trois quarts
	}
	garde[3*r.NX+0] = true // 1 sur 20 : au-dessous

	out, retires := r.recolleAuxObjets(garde, SeuilRecollement)
	if retires != 1 {
		t.Fatalf("attendu 1 objet retire, obtenu %d", retires)
	}
	if out[16*r.NX+10] {
		t.Fatal("l objet garde a ete COMPLETE : le recollement a rajoute de la matiere")
	}
	if !out[0*r.NX+10] {
		t.Fatal("l objet majoritairement garde a perdu une cellule que le masque tenait")
	}
	if out[3*r.NX+0] {
		t.Fatal("l objet minoritairement garde n a pas ete retire")
	}
}

// SANS PROVENANCE PAR INSTANCE, le recollement rend le masque tel quel — il ne vide jamais rien
// au motif qu il ne sait pas nommer les objets.
func TestRecollementInerteSansProvenance(t *testing.T) {
	r := grilleTest()
	garde := make([]bool, len(r.z))
	garde[0] = true
	out, retires := r.recolleAuxObjets(garde, SeuilRecollement)
	if retires != 0 {
		t.Fatalf("recollement actif sans provenance : %d retires", retires)
	}
	if !out[0] || out[1] {
		t.Fatal("le masque a ete modifie alors qu aucune provenance n est armee")
	}
}

// LE MASQUE EST UN DISQUE, PAS UN CARRE. C'est la correction du crenelage signale par
// l'utilisateur : une dilatation par voisinage carre garde les coins a `rayon * sqrt(2)`, une
// distance vraie les coupe a `rayon`. Le temoin verifie les deux moities de la propriete —
// l'orthogonal est garde jusqu'au rayon, la diagonale ne l'est pas au-dela.
func TestMasqueDesPositionsEstUnDisque(t *testing.T) {
	r := grilleTest()
	pos := []PositionJouee{{X: 10.5, Y: 10.5}}
	r.RogneAuxPositionsJouees(pos, 4, 0)

	if math.IsInf(r.z[10*r.NX+14], -1) {
		t.Fatal("une cellule a 4 m dans l axe a ete effacee : le rayon n est pas tenu")
	}
	// A 4 cellules en X ET en Y, la distance vaut 5,66 — au-dela du rayon. Un voisinage carre
	// l aurait gardee ; un disque ne le doit pas.
	if !math.IsInf(r.z[14*r.NX+14], -1) {
		t.Fatal("le coin diagonal a survecu : le masque est reste carre, le crenelage revient")
	}
}
