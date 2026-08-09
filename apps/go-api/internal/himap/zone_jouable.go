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
