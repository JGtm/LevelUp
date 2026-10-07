package tactical

// zones.go — LE NOM EN JEU D'UNE CELLULE : la zone du catalogue (callouts) où tombe une cellule
// du plan, compte tenu de la hauteur des événements qui l'ont alimentée.
//
// # LA RÈGLE, DANS L'ORDRE
//
//	(a) polygone  les zones dont la FORME contient le centre de la cellule ET dont la tranche
//	              [z_bas − MargeTrancheZM ; z_haut + MargeTrancheZM] contient le z MÉDIAN des
//	              événements : une seule, c'est elle ;
//	(b) empilée   plusieurs : la tranche la plus ÉTROITE parmi celles qui contiennent la MAJORITÉ
//	              des événements (plus de la moitié) ; si aucune ne la contient, celle qui en
//	              contient le plus, puis la plus étroite ;
//	(c) proche    aucune : la zone dont la forme est la plus proche du centre, à MOINS de
//	              RayonZoneM (distance au bord, 0 si le centre est dedans), parmi les tranches
//	              compatibles avec le z médian d'abord, parmi toutes ensuite ;
//	(d)           sinon, pas de nom — jamais un nom de repli.
//
// Toute égalité finale se départage par l'indice de volume (déterminisme).
//
// # Z INCONNU
//
// Sans aucun z (lectures tirées des pistes du film, qui n'en portent pas ; ou z absents) : (a) sans
// test de tranche quand UNE SEULE forme contient le centre ; plusieurs formes empilées ne nomment
// rien — l'étage ne se devine pas ; aucune : (c) sans test de tranche.
//
// Pur : aucune base, aucun fichier.

import (
	"math"
	"sort"
)

// MargeTrancheZM élargit la tranche d'une zone de part et d'autre, en mètres : un joueur debout
// sur la dalle d'une salle a les pieds à quelques centimètres sous son plancher déclaré.
const MargeTrancheZM = 0.25

// RayonZoneM est la distance au bord en deçà de laquelle une zone voisine nomme une cellule qui
// tombe hors de toute forme (borne stricte).
const RayonZoneM = 2.0

// Règles qui ont nommé une cellule (journal, tests).
const (
	RegleZonePolygone = "polygone"
	RegleZoneEmpilee  = "empilee"
	RegleZoneProche   = "proche"
)

// ZoneRetenue est la zone qui nomme une cellule, la règle qui l'a retenue et sa distance au
// centre (0 quand le centre est dans la forme).
type ZoneRetenue struct {
	Zone      ZoneNommee
	Regle     string
	DistanceM float64
}

// NommerZone rend la zone qui nomme le point (x, y) — le centre d'une cellule — compte tenu des z
// des événements de la cellule (`zs`, vide = inconnu). Faux : aucune zone ne le nomme.
func NommerZone(x, y float64, zs []float64, zones []ZoneNommee) (ZoneRetenue, bool) {
	formes := make([]ZoneNommee, 0, len(zones))
	for _, z := range zones {
		if aUneForme(z) {
			formes = append(formes, z)
		}
	}
	if len(zs) == 0 {
		return nommerSansHauteur(x, y, formes)
	}
	zm := mediane(zs)
	var candidates []ZoneNommee
	for _, z := range formes {
		if dansForme(x, y, z) && dansTranche(z, zm) {
			candidates = append(candidates, z)
		}
	}
	switch len(candidates) {
	case 0:
		return laPlusProche(x, y, formes, func(z ZoneNommee) bool { return dansTranche(z, zm) })
	case 1:
		return ZoneRetenue{Zone: candidates[0], Regle: RegleZonePolygone}, true
	default:
		return ZoneRetenue{Zone: departagerEmpilees(candidates, zs), Regle: RegleZoneEmpilee}, true
	}
}

// nommerSansHauteur : la règle quand aucun z n'est connu.
func nommerSansHauteur(x, y float64, formes []ZoneNommee) (ZoneRetenue, bool) {
	var contenant []ZoneNommee
	for _, z := range formes {
		if dansForme(x, y, z) {
			contenant = append(contenant, z)
		}
	}
	switch len(contenant) {
	case 0:
		return laPlusProche(x, y, formes, nil)
	case 1:
		return ZoneRetenue{Zone: contenant[0], Regle: RegleZonePolygone}, true
	default:
		return ZoneRetenue{}, false
	}
}

// departagerEmpilees applique (b) : majorité, puis la plus étroite ; à défaut de majorité, la plus
// peuplée, puis la plus étroite ; puis l'indice de volume.
func departagerEmpilees(candidates []ZoneNommee, zs []float64) ZoneNommee {
	type candidat struct {
		zone   ZoneNommee
		compte int
	}
	cs := make([]candidat, 0, len(candidates))
	for _, z := range candidates {
		n := 0
		for _, h := range zs {
			if dansTranche(z, h) {
				n++
			}
		}
		cs = append(cs, candidat{zone: z, compte: n})
	}
	majoritaire := func(c candidat) bool { return 2*c.compte > len(zs) }
	sort.SliceStable(cs, func(i, j int) bool {
		mi, mj := majoritaire(cs[i]), majoritaire(cs[j])
		if mi != mj {
			return mi
		}
		if !mi && cs[i].compte != cs[j].compte {
			return cs[i].compte > cs[j].compte
		}
		if hi, hj := epaisseur(cs[i].zone), epaisseur(cs[j].zone); hi != hj {
			return hi < hj
		}
		return cs[i].zone.VolumeIndex < cs[j].zone.VolumeIndex
	})
	return cs[0].zone
}

// laPlusProche applique (c) : à moins de RayonZoneM, parmi les zones `compatible` d'abord (nil =
// aucun premier passage), parmi toutes ensuite.
func laPlusProche(x, y float64, formes []ZoneNommee, compatible func(ZoneNommee) bool) (ZoneRetenue, bool) {
	passe := func(retenir func(ZoneNommee) bool) (ZoneRetenue, bool) {
		meilleure, best := ZoneNommee{}, math.Inf(1)
		for _, z := range formes {
			if retenir != nil && !retenir(z) {
				continue
			}
			d := distanceAZone(x, y, z)
			if d >= RayonZoneM {
				continue
			}
			if d < best || (d == best && z.VolumeIndex < meilleure.VolumeIndex) {
				meilleure, best = z, d
			}
		}
		if math.IsInf(best, 1) {
			return ZoneRetenue{}, false
		}
		return ZoneRetenue{Zone: meilleure, Regle: RegleZoneProche, DistanceM: best}, true
	}
	if compatible != nil {
		if r, ok := passe(compatible); ok {
			return r, true
		}
	}
	return passe(nil)
}

// dansTranche : le z est-il dans la tranche de la zone, marge comprise ?
func dansTranche(z ZoneNommee, h float64) bool {
	return h >= z.ZBas-MargeTrancheZM && h <= z.ZHaut+MargeTrancheZM
}

// epaisseur : la hauteur de la tranche déclarée.
func epaisseur(z ZoneNommee) float64 { return z.ZHaut - z.ZBas }

// mediane : le z médian (moyenne des deux du milieu pour un nombre pair).
func mediane(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}
