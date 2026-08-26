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
	if r.ecrete == nil {
		r.ecrete = make([]bool, len(r.z))
	}
	// UNE CELLULE HORS ZONE EST RETIREE DE LA CARTE, qu elle porte de la matiere ou non :
	// l eau se peint meme sans matiere (c est sa raison d etre), et sans ce marquage un volume
	// d eau situe hors des zones nommees reste dessine sur une carte dont on vient de retirer
	// le toit qui le cachait. Mesure du 2026-08-26 sur Recharge : dalle bleue en travers.
	for k := range r.z {
		if k < len(masque) && masque[k] {
			continue
		}
		r.ecrete[k] = true
		if math.IsInf(r.z[k], -1) {
			continue
		}
		r.z[k] = math.Inf(-1)
		efface++
	}
	return efface
}

// CombleTrous marque, comme SOL SUPPOSE, chaque cellule sans matiere qui tombe DANS le masque
// des zones nommees. Rend le nombre de cellules marquees.
//
// CE QUE C EST, ET CE QUE CE N EST PAS. Ce n'est pas une mesure : c'est un APLAT. Une zone
// nommee est du terrain joue par construction — si le rendu n'y a dessine aucune surface, c'est
// que la geometrie de ce sol est hors de la tranche d'altitude, sous un atrium, ou simplement
// absente du maillage de rendu. Plutot que de laisser un trou noir au milieu de l'arene, on y
// pose un sol suppose, et on le PEINT AUTREMENT pour que personne ne le prenne pour du releve.
//
// Le nombre de cellules comblees est publie au sidecar (`cellsAssumedFloor`) : un aplat qu'on
// ne compte pas est un mensonge qui grandit sans qu'on le voie.
// SEULS LES TROUS FERMES SONT COMBLES, et la premiere version ne le faisait pas : comblee sur
// toute cellule vide du masque dilate, elle a pose 611 959 cellules d'aplat sur Illusion et
// noye l'arene sous des dalles grises (mesure du 2026-08-26). Un vide OUVERT sur l'exterieur
// n'est pas un trou de relevé — c'est le bord de la carte, ou une cour, ou du vide reel.
//
// La regle : on inonde le vide depuis les bords de l'image ; ce que l'inondation n'atteint
// PAS est un trou ferme, entoure de matiere. Ceux-la seulement sont combles.
func (r *Rendu) CombleTrous(masque []bool) int {
	vide := func(k int) bool { return math.IsInf(r.z[k], -1) }
	atteint := make([]bool, len(r.z))
	pile := make([]int, 0, r.NX+r.NY)
	pousse := func(i, j int) {
		if i < 0 || i >= r.NX || j < 0 || j >= r.NY {
			return
		}
		k := j*r.NX + i
		if atteint[k] || !vide(k) {
			return
		}
		atteint[k] = true
		pile = append(pile, k)
	}
	for i := 0; i < r.NX; i++ {
		pousse(i, 0)
		pousse(i, r.NY-1)
	}
	for j := 0; j < r.NY; j++ {
		pousse(0, j)
		pousse(r.NX-1, j)
	}
	for len(pile) > 0 {
		k := pile[len(pile)-1]
		pile = pile[:len(pile)-1]
		i, j := k%r.NX, k/r.NX
		pousse(i-1, j)
		pousse(i+1, j)
		pousse(i, j-1)
		pousse(i, j+1)
	}

	comble := 0
	for k := range r.z {
		if !vide(k) || atteint[k] || k >= len(masque) || !masque[k] {
			continue
		}
		if r.solSuppose == nil {
			r.solSuppose = make([]bool, len(r.z))
		}
		r.solSuppose[k] = true
		comble++
	}
	return comble
}

// SolSuppose dit si une cellule porte un sol suppose (cf. CombleTrous).
func (r *Rendu) SolSuppose(i, j int) bool {
	if r.solSuppose == nil || i < 0 || i >= r.NX || j < 0 || j >= r.NY {
		return false
	}
	return r.solSuppose[j*r.NX+i]
}
