package mapdecoupe

// masque_point.go — LE MASQUE PRATICABLE INTERROGE EN UN POINT, sous une forme COMPACTE (retours du
// rejeu, lot M7, correctif RR-M7-01 de la revue adverse du 2026-09-24).
//
// LE DEFAUT QU IL CORRIGE. Le decor de carte (`internal/service/replay_vehicle_scenery*.go`) posait
// a chaque ouverture de rejeu la question « ce vehicule est-il sur la matiere praticable, fermee a
// `ToleranceParDefaut` ? » en fermant le masque ENTIER : deux transformees de distance en float64
// sur toute la grille, sans cache. Mesure de la revue (TotalAlloc autour de la resolution) :
// 713 Mo alloues pour Behemoth (2 451 x 2 452 cellules), 163 Mo pour Starboard, 194 Mo pour
// Goliath — A CHAQUE REQUETE, sur un VPS qui a deja connu la « bombe RAM ».
//
// CE QUE FAIT CE FICHIER, ET POURQUOI C EST EXACT. La fermeture est LOCALE : une cellule p est dans
// la fermeture de rayon r si et seulement si toute cellule q du cadre a distance <= r de p a de la
// matiere a distance <= r d elle (erosion de la dilatation, `Comble`). La matiere qui compte est
// donc a <= 2r de p : une fenetre de demi-cote ceil(2r)+1 centree sur p suffit, et la transformee
// de distance calculee sur la fenetre rend, pour chaque q du disque, la meme reponse au seuil r que
// la transformee de la grille entiere (une matiere hors fenetre est a plus de r de tout q du disque,
// donc ne change pas la comparaison). Le dehors du cadre n est ni matiere ni vide, exactement comme
// dans `Comble` (le dehors compte PLEIN a l erosion : seules les cellules du cadre sont examinees).
// L equivalence cellule a cellule est un test (masque_point_test.go), sur grilles synthetiques et
// sur un fond reel du depot.
//
// LA FORME COMPACTE. Un bit par cellule : 734 Kio pour le plus grand fond publie (Behemoth), au lieu
// des 6 Mo d un `[]bool` et des centaines de Mo de la fermeture globale. C est cette forme qu un
// appelant garde en cache ; la fermeture en un point coute une fenetre de ~390 x 390 cellules au
// rayon canonique (1,2 Mo transitoires), quelle que soit la taille de la carte.

import (
	"math"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// MasqueCompact : le masque praticable d une carte, un bit par cellule, avec son calage.
type MasqueCompact struct {
	// Module est la cle du fond (tracabilite).
	Module string
	// Calage place la grille dans le monde.
	Calage replay.MapBackgroundCalibration
	// NX, NY sont les dimensions de la grille.
	NX, NY int
	bits   []uint64
}

// Compacte rend la forme compacte du masque, cellule pour cellule.
func (m *Masque) Compacte() *MasqueCompact {
	c := &MasqueCompact{
		Module: m.Module, Calage: m.Calage, NX: m.NX, NY: m.NY,
		bits: make([]uint64, (len(m.dur)+63)/64),
	}
	for k, d := range m.dur {
		if d {
			c.bits[k>>6] |= 1 << (uint(k) & 63)
		}
	}
	return c
}

// Octets est l empreinte memoire des bits du masque — de quoi journaliser ce qu un cache garde.
func (c *MasqueCompact) Octets() int { return len(c.bits) * 8 }

// dur dit si la cellule (i, j) du cadre porte de la matiere.
func (c *MasqueCompact) dur(i, j int) bool {
	k := j*c.NX + i
	return c.bits[k>>6]&(1<<(uint(k)&63)) != 0
}

// Praticable dit si une position monde tombe sur de la matiere (masque BRUT, sans fermeture).
func (c *MasqueCompact) Praticable(x, y float64) bool {
	px, py, ok := c.Calage.MondeVersPixel(x, y)
	return ok && c.dur(px, py)
}

// PraticableComble dit si une position monde tombe sur la matiere FERMEE au rayon `rayonM` :
// exactement `m.Comble(rayonM).Praticable(x, y)`, calcule sur une fenetre locale. Un rayon nul rend
// le masque brut, comme `Comble`.
func (c *MasqueCompact) PraticableComble(x, y, rayonM float64) bool {
	px, py, ok := c.Calage.MondeVersPixel(x, y)
	if !ok {
		return false
	}
	if rayonM <= 0 {
		return c.dur(px, py)
	}
	r := rayonM / c.Calage.MetersPerPixel
	r2 := r * r
	demi := int(math.Ceil(2*r)) + 1
	i0, j0 := max(px-demi, 0), max(py-demi, 0)
	i1, j1 := min(px+demi, c.NX-1), min(py+demi, c.NY-1)
	nx, ny := i1-i0+1, j1-j0+1
	fenetre := make([]bool, nx*ny)
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			fenetre[j*nx+i] = c.dur(i0+i, j0+j)
		}
	}
	// dm : carre de la distance de chaque cellule de la fenetre a la matiere la plus proche.
	dm := distanceCarre(fenetre, nx, ny)
	rc := int(math.Floor(r))
	for dj := -rc; dj <= rc; dj++ {
		for di := -rc; di <= rc; di++ {
			if float64(di*di+dj*dj) > r2 {
				continue
			}
			qi, qj := px+di-i0, py+dj-j0
			if qi < 0 || qi >= nx || qj < 0 || qj >= ny {
				continue // hors du cadre : ni vide ni matiere, comme a l erosion de `Comble`
			}
			if dm[qj*nx+qi] > r2 {
				return false // un vide de la dilatation a <= r de p : l erosion le retire
			}
		}
	}
	return true
}
