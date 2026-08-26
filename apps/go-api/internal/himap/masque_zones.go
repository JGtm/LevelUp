package himap

// masque_zones.go — ROGNER SUR LES ZONES NOMMEES DE LA CARTE.
//
// L'IDEE, ET ELLE VIENT DE L'UTILISATEUR : les callouts d'une carte decrivent ou l'on joue.
// Ce qui n'est dans aucune zone nommee n'est probablement pas du terrain — c'est du decor, un
// toit, une coquille. On pourrait donc l'effacer.
//
// CE QUE CE FICHIER FOURNIT, ET RIEN DE PLUS : le masque, sa dilatation, et le compte de ce
// qui tomberait dehors. Il ne decide pas. La decision se prend sur MESURE, carte par carte, et
// la mesure est le seul usage honnete tant que le taux n'a pas ete regarde : un masque qui
// effacerait un quart de la matiere mangerait du terrain joue.
//
// POURQUOI UNE DILATATION, ET EN METRES. Un polygone de callout borde la zone nommee, pas la
// geometrie qui la porte : un mur, une rampe, un rebord vivent JUSTE dehors. Sans marge, le
// masque decoupe la carte au ras du sol praticable et supprime ce qui la rend lisible. La marge
// est en metres pour valoir la meme chose a toutes les echelles.
//
// LIMITE CONNUE, ECRITE POUR NE PAS ETRE REDECOUVERTE : les callouts n'existent que sur les
// cartes NATIVES (22 cartes au catalogue, 0 sur Forge). Cette voie ne vaudra jamais pour les
// cartes Forge, qui sont pourtant les plus mal cadrees.

import "math"

// MargeMasqueZones : dilatation du masque des zones nommees, en metres.
//
// 4 m ≈ la largeur d'un couloir : assez pour garder le mur qui borde une zone et la rampe qui
// y monte, assez peu pour que le masque serve encore a quelque chose.
const MargeMasqueZones = 4.0

// MasqueZones rasterise l'union de polygones dans la grille d'un rendu, puis la dilate de
// `marge` metres. Rend un masque de la taille de la grille : vrai = dans une zone (ou dans sa
// marge).
//
// Remplissage PAIR-IMPAIR par balayage de lignes : c'est la regle qui convient a une union de
// polygones simples, elle ne demande aucune orientation et ne suppose pas la convexite.
func MasqueZones(r *Rendu, polys [][][2]float64, marge float64) []bool {
	m := make([]bool, r.NX*r.NY)
	for j := 0; j < r.NY; j++ {
		y := r.Min[1] + (float64(j)+0.5)*r.Cell
		var xs []float64
		for _, p := range polys {
			xs = intersectionsLigne(p, y, xs[:0])
			for k := 0; k+1 < len(xs); k += 2 {
				i0 := int((xs[k]-r.Min[0])/r.Cell + 0.5)
				i1 := int((xs[k+1]-r.Min[0])/r.Cell + 0.5)
				if i1 < 0 || i0 >= r.NX {
					continue
				}
				i0, i1 = max(i0, 0), min(i1, r.NX-1)
				for i := i0; i <= i1; i++ {
					m[j*r.NX+i] = true
				}
			}
		}
	}
	if marge > 0 && r.Cell > 0 {
		if k := int(marge/r.Cell + 0.5); k > 0 {
			m = dilate(m, r.NX, r.NY, k)
		}
	}
	return m
}

// intersectionsLigne rend, TRIEES, les abscisses ou la ligne y coupe le polygone. Le buffer est
// reutilise par l'appelant : une carte porte des dizaines de milliers de lignes.
func intersectionsLigne(p [][2]float64, y float64, xs []float64) []float64 {
	n := len(p)
	for i := 0; i < n; i++ {
		a, b := p[i], p[(i+1)%n]
		if (a[1] > y) == (b[1] > y) {
			continue // l'arete ne traverse pas cette ligne
		}
		xs = append(xs, a[0]+(y-a[1])/(b[1]-a[1])*(b[0]-a[0]))
	}
	// Tri par insertion : quelques entrees par ligne, jamais plus d'une poignee.
	for i := 1; i < len(xs); i++ {
		for k := i; k > 0 && xs[k] < xs[k-1]; k-- {
			xs[k], xs[k-1] = xs[k-1], xs[k]
		}
	}
	return xs
}

// dilate elargit un masque booleen de k cellules, en deux passes separables (lignes puis
// colonnes). Le resultat est un carre, pas un disque — a cette echelle la difference ne se voit
// pas, et une distance euclidienne exacte couterait un ordre de grandeur de plus.
func dilate(m []bool, nx, ny, k int) []bool {
	tmp := make([]bool, len(m))
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			if !m[j*nx+i] {
				continue
			}
			for d := max(i-k, 0); d <= min(i+k, nx-1); d++ {
				tmp[j*nx+d] = true
			}
		}
	}
	out := make([]bool, len(m))
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			if !tmp[j*nx+i] {
				continue
			}
			for d := max(j-k, 0); d <= min(j+k, ny-1); d++ {
				out[d*nx+i] = true
			}
		}
	}
	return out
}

// MesureHorsZones compte la matiere dessinee, et celle qui tombe HORS du masque. C'est le
// chiffre qui doit etre regarde AVANT d'appliquer quoi que ce soit.
func (r *Rendu) MesureHorsZones(masque []bool) (matiere, dehors int) {
	for k := range r.z {
		if math.IsInf(r.z[k], -1) {
			continue
		}
		matiere++
		if k < len(masque) && !masque[k] {
			dehors++
		}
	}
	return matiere, dehors
}

// EffaceHorsZones vide la matiere qui tombe hors du masque et rend le nombre de cellules
// effacees. A n'appeler qu'apres avoir regarde `MesureHorsZones`.
func (r *Rendu) EffaceHorsZones(masque []bool) int {
	efface := 0
	for k := range r.z {
		if math.IsInf(r.z[k], -1) || (k < len(masque) && masque[k]) {
			continue
		}
		r.z[k] = math.Inf(-1)
		efface++
	}
	return efface
}
