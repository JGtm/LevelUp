package himap

// composantes.go — NE GARDER QUE CE QUI TIENT A LA CARTE.
//
// LE DEFAUT, RELEVE PAR L'UTILISATEUR LE 2026-08-30 sur Houseki et Megapolis : « des formes sur
// la gauche qui ne sont pas jouables, hors de la silhouette principale ». Ce sont des morceaux de
// decor poses loin de l'arene — une passerelle, un batiment de fond, un rack d'objets oublie — que
// rien ne relie a ce sur quoi on joue. Ils etirent le cadre et brouillent la lecture.
//
// LE CRITERE EST L'ANCRE D'OBJECTIF, et il ne se discute pas : une ancre est du terrain joue par
// definition. Un amas de matiere qui n'en porte aucune ET qui ne touche aucun amas qui en porte
// une n'est pas l'arene. On l'efface.
//
// POURQUOI PAS « LA PLUS GROSSE COMPOSANTE ». Showdown Arena est faite de SEPT ilots disjoints,
// tous joues, tous porteurs d'ancres : garder la plus grosse en effacerait six. Le critere doit
// suivre les ancres, jamais la taille.
//
// LEVIER PAR CARTE, JAMAIS PAR DEFAUT : une carte dont les ancres ne couvrent pas tous les ilots
// joues y perdrait du terrain. Il s'arme apres avoir regarde, comme les autres rognages.

import "math"

// RayonAncreComposante : rayon de recherche, en cellules, autour d'une ancre pour rattacher
// celle-ci a une composante.
//
// Une ancre tombe rarement au pixel exact sur de la matiere : le navmesh se retire des murs, un
// objectif est pose au centre d'un volume, l'echelle est de quelques centimetres par pixel. 24
// cellules valent 1 a 2 m aux echelles usuelles — assez pour rattraper ce decalage, trop peu pour
// sauter d'un ilot a un autre (les plus proches sont a des dizaines de metres).
const RayonAncreComposante = 24

// GardeComposantesAncrees efface toute matiere qui n'appartient pas a une composante connexe
// portant au moins une ancre. Rend le nombre de cellules effacees et le nombre de composantes
// gardees sur le total.
//
// Connexite a 4 : deux dalles qui ne se touchent que par un coin ne sont pas le meme sol.
func (r *Rendu) GardeComposantesAncrees(ancres [][3]float64) (efface, gardees, total int) {
	etiq, total := r.etiquetteComposantes()
	if total == 0 {
		return 0, 0, 0
	}

	garde := make([]bool, total+1)
	for _, a := range ancres {
		if n := r.composanteSousAncre(etiq, a); n > 0 {
			garde[n] = true
		}
	}
	for _, g := range garde[1:] {
		if g {
			gardees++
		}
	}
	// AUCUNE ancre rattachee : on n'efface RIEN. Effacer toute l'image serait la pire reponse a
	// une donnee qu'on n'a pas su lire.
	if gardees == 0 {
		return 0, 0, total
	}
	for k := range r.z {
		if etiq[k] == 0 || garde[etiq[k]] {
			continue
		}
		r.z[k] = math.Inf(-1)
		if r.solSuppose != nil {
			r.solSuppose[k] = false
		}
		efface++
	}
	return efface, gardees, total
}

// etiquetteComposantes numerote les composantes connexes a 4 de la matiere, par parcours en
// profondeur. Rend l'etiquette de chaque cellule (0 = vide ou hors matiere) et le nombre de
// composantes trouvees. Extrait de GardeComposantesAncrees, a l'identique.
func (r *Rendu) etiquetteComposantes() ([]int32, int) {
	plein := func(k int) bool {
		return !math.IsInf(r.z[k], -1) || (r.solSuppose != nil && r.solSuppose[k])
	}
	etiq := make([]int32, len(r.z)) // 0 = vide ou pas encore vu
	pile := make([]int, 0, 1024)
	total := 0
	for k := range r.z {
		if etiq[k] != 0 || !plein(k) {
			continue
		}
		total++
		n := int32(total)
		etiq[k] = n
		pile = append(pile[:0], k)
		for len(pile) > 0 {
			c := pile[len(pile)-1]
			pile = pile[:len(pile)-1]
			i, j := c%r.NX, c/r.NX
			voisin := func(vi, vj int) {
				if vi < 0 || vi >= r.NX || vj < 0 || vj >= r.NY {
					return
				}
				vk := vj*r.NX + vi
				if etiq[vk] != 0 || !plein(vk) {
					return
				}
				etiq[vk] = n
				pile = append(pile, vk)
			}
			voisin(i-1, j)
			voisin(i+1, j)
			voisin(i, j-1)
			voisin(i, j+1)
		}
	}
	return etiq, total
}

// composanteSousAncre rend l'etiquette de la composante qui porte cette ancre, en cherchant en
// spirale carree autour de sa cellule. Zero si l'ancre tombe dans le vide.
func (r *Rendu) composanteSousAncre(etiq []int32, a [3]float64) int32 {
	ci := int((a[0] - r.Min[0]) / r.Cell)
	cj := int((a[1] - r.Min[1]) / r.Cell)
	for d := 0; d <= RayonAncreComposante; d++ {
		for j := cj - d; j <= cj+d; j++ {
			for i := ci - d; i <= ci+d; i++ {
				// Seul le BORD du carre de rayon d est neuf ; l'interieur a deja ete vu.
				if d > 0 && i != ci-d && i != ci+d && j != cj-d && j != cj+d {
					continue
				}
				if i < 0 || i >= r.NX || j < 0 || j >= r.NY {
					continue
				}
				if n := etiq[j*r.NX+i]; n != 0 {
					return n
				}
			}
		}
	}
	return 0
}
