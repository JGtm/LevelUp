// Package himap — zone_jouable.go : separer l'aire de jeu du decor qui l'entoure.
//
// LE PROBLEME. Une carte reconstruite depuis les modules porte bien plus que l'aire de jeu :
// le relief alentour vient des memes bsp et des memes modules globaux. Mesure sur Cliffhanger
// contre la carte validee — sans tri, on dessine 149 % de matiere en trop pour 4 % de manque.
//
// CE QUI NE SEPARE PAS, mesure et REFUTE le 2026-08-09 (ne pas rejouer) :
//   - le MODULE : aucun n'est « le decor ». Le module de la carte couvre 50,3 % de la
//     reference, `common` 60,2 %, `multiplayer` 47,5 % — ils se completent.
//   - l'EMPRISE bornee au voisinage des ancres : 100 x 100 m sur cette carte, 92 instances
//     ecartees, exces inchange.
//   - la TRANCHE d'altitude seule : indispensable (cf. `TrancheDeJeuMin`), mais l'exces se
//     repartit sur toute la tranche.
//   - la PENTE : deja refutee par l'oracle des positions de joueur (handoff §1 ter).
//
// CE QUI SEPARE : la FINESSE DU MAILLAGE. Les surfaces jouees sont modelisees au decimetre,
// le decor lointain non. L'aire projetee du triangle MEDIAN, par instance, s'etale sur quatre
// ordres de grandeur — p50 0,0001 m2, p90 0,0161, p99 1,8739 — et le decor est tout entier
// dans la queue.
//
// C'est une propriete du maillage, pas un reglage d'image : elle ne demande ni reference
// validee ni calibration par carte. C'est ce qui la distingue d'un masque.
package himap

import (
	"math"
	"sort"
)

// AireMaxTriangleJouable : aire projetee au sol, en m2, du triangle MEDIAN au-dela de laquelle
// une instance est tenue pour du decor.
//
// 0,005 m2 = un triangle de ~10 cm de cote. VALIDE par l'utilisateur le 2026-08-09 sur le
// rendu de Cliffhanger.
//
// POURQUOI CETTE VALEUR ET PAS L'OPTIMUM STRICT. Score d'accord (intersection / union avec la
// silhouette de la carte validee) en fonction du seuil :
//
//	seuil (m2)   manquants   exces    accord
//	aucun            4,0 %   149,3 %   38,5 %
//	0,002           26,6 %    15,2 %   63,7 %
//	0,003           18,3 %    21,2 %   67,4 %   <- optimum
//	0,005           10,8 %    33,8 %   66,7 %   <- retenu
//	0,012            7,9 %    59,9 %   57,6 %
//
// A 0,7 point d'accord pres, 0,005 garde 89,2 % de la zone jouable contre 81,7 %. Perdre de la
// structure jouee est le defaut que l'utilisateur signale depuis le debut ; l'exces, non.
//
// CE QUI RESTE A PROUVER : le seuil est calibre sur la SEULE carte qui possede une reference
// validee. La regle doit etre confrontee aux 26 autres par l'oracle faible des ancres (methode
// du handoff §1 ter). Si le seuil doit se retoucher carte par carte, c'est la regle qui est
// mauvaise — pas la valeur. Inscrit au registre des reports.
const AireMaxTriangleJouable = 0.005

// AireMedianeProjetee rend l'aire projetee au sol du triangle MEDIAN d'un maillage place par
// son instance.
//
// La MEDIANE, pas la moyenne : un maillage fin qui porte deux grandes faces de socle aurait
// une moyenne de decor. C'est la finesse TYPIQUE qu'on veut lire, pas la plus grande face.
func AireMedianeProjetee(m *Mesh, in Instance) float64 {
	if m == nil || len(m.Triangles) == 0 {
		return 0
	}
	monde := make([][3]float64, len(m.Vertices))
	for i, s := range m.Vertices {
		monde[i] = in.LocalToWorld(s)
	}
	aires := make([]float64, 0, len(m.Triangles))
	for _, t := range m.Triangles {
		a, b, c := monde[t[0]], monde[t[1]], monde[t[2]]
		aires = append(aires, math.Abs((b[0]-a[0])*(c[1]-a[1])-(b[1]-a[1])*(c[0]-a[0]))/2)
	}
	sort.Float64s(aires)
	return aires[len(aires)/2]
}

// EstDecorGrossier dit si une instance est du decor plutot que de l'aire de jeu.
//
// `aireMax <= 0` desactive le tri — reserve aux temoins qui comparent les deux lectures.
func EstDecorGrossier(m *Mesh, in Instance, aireMax float64) bool {
	if aireMax <= 0 {
		return false
	}
	return AireMedianeProjetee(m, in) > aireMax
}

// MarcheMaxMetres : denivele qu'un Spartan franchit entre deux cellules voisines.
//
// PHYSIQUE, pas esthetique : le pas d'escalade automatique de Halo Infinite est de l'ordre du
// demi-metre, le clamber porte a ~1,2 m mais demande une action volontaire. On retient 0,85 m
// — au-dessus, deux surfaces voisines ne communiquent plus en marchant.
//
// Il joue le meme role que `SeuilAreteMetres` (0,5 m) mais ne s'y confond pas : l'un decide ce
// qu'on DESSINE comme bord, l'autre ce qui est ATTEIGNABLE.
const MarcheMaxMetres = 0.85

// ComposanteAccessible rend, cellule par cellule, la partie du rendu ATTEIGNABLE EN MARCHANT
// depuis des points de depart donnes.
//
// REFUTE COMME CRITERE DE ZONE JOUABLE (2026-08-10). Mesure sur Cliffhanger, contre la carte
// validee et les 29 221 positions de joueur :
//
//	marche (m)   manquants   exces    ACCORD   positions jouees gardees
//	aucune           4,0 %   149,3 %   38,5 %          95,21 %
//	0,50            16,2 %   108,8 %   40,2 %          89,21 %
//	0,85             8,3 %   136,9 %   38,7 %          90,96 %
//	1,20             7,2 %   137,7 %   39,0 %          91,89 %
//	0,85 + grain    33,8 %    16,0 %   57,1 %          86,58 %
//
// Toutes les variantes PERDENT des positions reellement jouees — critere disqualifiant annonce
// d'avance — et aucune ne fait mieux que 40,2 % d'accord.
//
// DEUX RAISONS STRUCTURELLES, a retenir avant de retenter :
//  1. le rendu est un CHAMP D'ALTITUDE (surface la plus haute par pixel), et sur un champ
//     d'altitude la connexite est presque gratuite — le relief alentour rejoint l'arene par
//     des pentes douces. Un vrai test demanderait le VOLUME avec degagement, pas le z-buffer.
//  2. marcher sous-estime l'accessibilite reelle : un joueur tombe, saute, prend un ascenseur.
//     La connexite pietonne ne peut donc pas capturer la zone jouee sans en perdre.
//
// Conserve comme INSTRUMENT (le banc `TestSondePhysiqueRendu` le mesure) et parce qu'une
// reprise sur le volume le reutiliserait. NE PAS le brancher en production en l'etat.
//
// POURQUOI CE CRITERE, apres quatre echecs. Le module ne separe rien, l'emprise par les ancres
// non plus, la tranche d'altitude seule non plus, le grain du maillage ne vaut que sur les
// cartes de roche, et la collision est portee par le decor autant que par l'arene (une falaise
// arrete un Spartan comme le sol de l'arene). Le seul trait qui reste propre a l'aire de jeu
// est la CONNEXITE : on y accede depuis les points d'objectif en marchant, le decor non.
//
// Ce n'est pas le filtre de PENTE refute au handoff §1 ter : celui-la jugeait chaque surface
// isolement sur son inclinaison. Ici deux cellules communiquent si leur denivele se franchit,
// et une pente raide reste accessible si on peut y monter marche apres marche.
//
// `depuis` porte des positions monde (les ancres d'objectifs). Une cellule de depart sans
// matiere est cherchee dans son voisinage immediat — une ancre flotte au-dessus du sol.
func (r *Rendu) ComposanteAccessible(depuis [][3]float64, marcheMax float64) []bool {
	vu := make([]bool, r.NX*r.NY)
	var file [][2]int
	for _, p := range depuis {
		i := int((p[0] - r.Min[0]) / r.Cell)
		j := int((p[1] - r.Min[1]) / r.Cell)
		if d, ok := r.cellulePleineProche(i, j); ok && !vu[d[1]*r.NX+d[0]] {
			vu[d[1]*r.NX+d[0]] = true
			file = append(file, d)
		}
	}
	for len(file) > 0 {
		c := file[len(file)-1]
		file = file[:len(file)-1]
		z, _ := r.Altitude(c[0], c[1])
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			i, j := c[0]+d[0], c[1]+d[1]
			if i < 0 || i >= r.NX || j < 0 || j >= r.NY || vu[j*r.NX+i] {
				continue
			}
			zv, ok := r.Altitude(i, j)
			if !ok || math.Abs(zv-z) > marcheMax {
				continue
			}
			vu[j*r.NX+i] = true
			file = append(file, [2]int{i, j})
		}
	}
	return vu
}

// Restreindre efface du rendu toute cellule hors du masque. C'est l'application d'une zone
// jouable a une carte deja rasterisee.
func (r *Rendu) Restreindre(masque []bool) {
	if len(masque) != len(r.z) {
		return
	}
	for k, garde := range masque {
		if !garde {
			r.z[k] = math.Inf(-1)
		}
	}
}

// cellulePleineProche cherche une cellule portant de la matiere autour de (i,j).
//
// Une ancre d'objectif est donnee EN L'AIR, au centre de sa zone : sa cellule exacte peut etre
// vide (un trou de reconstruction, une passerelle ajouree). Sans ce rattrapage, une seule ancre
// mal placee ferait rendre une composante vide.
func (r *Rendu) cellulePleineProche(i, j int) ([2]int, bool) {
	const rayon = 12 // ~1,1 m a 9 cm par cellule
	for d := 0; d <= rayon; d++ {
		for dj := -d; dj <= d; dj++ {
			for di := -d; di <= d; di++ {
				if max(abs(di), abs(dj)) != d {
					continue
				}
				if _, ok := r.Altitude(i+di, j+dj); ok {
					return [2]int{i + di, j + dj}, true
				}
			}
		}
	}
	return [2]int{}, false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
