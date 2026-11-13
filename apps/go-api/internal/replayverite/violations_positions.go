package replayverite

// violations_positions.go — CE QU'AUCUNE POSITION REELLE NE PEUT FAIRE.
//
// V-1 saut, V-2 objet hors emprise, V-6 trajet loin du vehicule, et V-7 (identite discordante,
// compteurs publies).

import (
	"fmt"
	"math"
	"sort"
)

// distance rend la distance 3D quand les deux cotes publient `z`, 2D sinon.
func distance(x1, y1 float64, z1 *float64, x2, y2 float64, z2 *float64) float64 {
	dz := 0.0
	if z1 != nil && z2 != nil {
		dz = *z2 - *z1
	}
	return math.Sqrt((x2-x1)*(x2-x1) + (y2-y1)*(y2-y1) + dz*dz)
}

// violationsSauts : V-1, deux points consecutifs d'une piste a plus de vitesseSautMaxMPS, hors
// translocation publiee et hors porte de carte mesuree (portes_de_carte.go).
func violationsSauts(d *Document, mapID string) Violation {
	var out Violation
	portes := portesDe(mapID)
	pas := float64(d.FrameIntervalMs) / 1000
	tolTransloc := d.images(toleranceTranslocationMS)
	for _, t := range d.Tracks {
		for i := 1; i < len(t.Points); i++ {
			p, q := t.Points[i-1], t.Points[i]
			dt := float64(q.T-p.T) * pas
			if dt <= 0 || distance(p.X, p.Y, p.Z, q.X, q.Y, q.Z)/dt <= vitesseSautMaxMPS {
				continue
			}
			if procheDUneTranslocation(d.Translocations, q.T, tolTransloc) || surUnePorte(portes, p) || surUnePorte(portes, q) {
				out.Exemptees++
				continue
			}
			out.Instances = append(out.Instances, fmt.Sprintf("slot %d @%d", t.Slot, q.T))
		}
	}
	sort.Strings(out.Instances)
	return out
}

func procheDUneTranslocation(ts []Translocation, t, tolerance int) bool {
	for _, x := range ts {
		if t-x.T >= -tolerance && t-x.T <= tolerance {
			return true
		}
	}
	return false
}

func surUnePorte(portes [][3]float64, p Point) bool {
	for _, c := range portes {
		z := c[2]
		if distance(p.X, p.Y, p.Z, c[0], c[1], &z) <= rayonPorteM {
			return true
		}
	}
	return false
}

// violationsHorsEmprise : V-2, un objet pose ou un echantillon de vehicule a plus de
// margeEmpriseObjetsM de l'enveloppe publiee des pistes. Les tirs, grenades et projectiles n'y sont
// PAS : ils volent legitimement au-dela de l'enveloppe des joueurs (mesure : des centaines sur les
// cartes BTB a 20 m).
//
// Un vehicule se compte par PASSAGE hors emprise : les echantillons consecutifs d'un meme vehicule
// hors de l'enveloppe forment une seule instance (datee du premier). Compter les echantillons
// ferait varier le total avec la densite de la grille, sans qu'aucun vehicule n'ait bouge.
func violationsHorsEmprise(d *Document) Violation {
	var out Violation
	if d.Bounds == nil {
		return out
	}
	b, m := *d.Bounds, margeEmpriseObjetsM
	hors := func(x, y float64) bool { return x < b.MinX-m || x > b.MaxX+m || y < b.MinY-m || y > b.MaxY+m }
	for _, f := range []struct {
		nom    string
		objets []Objet
	}{{"arme au sol", d.GroundWeapons}, {"equipement pose", d.EquipmentPlacements}, {"presentoir", d.WeaponPads}} {
		for _, o := range f.objets {
			if hors(o.X, o.Y) {
				out.Instances = append(out.Instances, fmt.Sprintf("%s @(%.0f,%.0f)", f.nom, o.X, o.Y))
			}
		}
	}
	for _, v := range d.Vehicles {
		dehors := false
		for _, s := range v.Samples {
			h := hors(s.X, s.Y)
			if h && !dehors {
				out.Instances = append(out.Instances, fmt.Sprintf("vehicule %d @%d", v.Slot, s.T))
			}
			dehors = h
		}
	}
	sort.Strings(out.Instances)
	return out
}

// violationsTrajets : V-6, un passager a plus de rayonTrajetM de son vehicule pendant un trajet,
// a l'instant d'un echantillon du vehicule (point du passager a ±toleranceEchantillonMS).
//
// Un trajet se compte par PASSAGE loin du vehicule : les echantillons evaluables consecutifs d'un
// meme trajet en violation forment une seule instance (datee du premier). Un echantillon evalue a
// moins de rayonTrajetM clot le passage ; un echantillon sans point de passager a portee n'est pas
// juge et ne le clot pas (aucune preuve de retour). Compter les echantillons ferait varier le total
// avec la densite de la grille (un passager dont le corps cesse de repliquer a la montee reste
// « loin » de plus d'echantillons quand le pas diminue), sans qu'aucun passager n'ait bouge.
func violationsTrajets(d *Document) Violation {
	var out Violation
	points := pointsParSlot(d)
	tolerance := d.images(toleranceEchantillonMS)
	for _, v := range d.Vehicles {
		for _, r := range v.Rides {
			out.Instances = append(out.Instances, passagesLoin(v, r, points[r.Slot], tolerance)...)
		}
	}
	sort.Strings(out.Instances)
	return out
}

// passagesLoin rend une instance par passage loin du vehicule d'un trajet, datee de son premier
// echantillon.
func passagesLoin(v Vehicule, r Trajet, pointsPassager []Point, tolerance int) []string {
	var out []string
	loin := false
	for _, s := range v.Samples {
		if s.T < r.T0 || s.T > r.T1 {
			continue
		}
		p, ok := pointA(pointsPassager, s.T, tolerance)
		if !ok {
			continue
		}
		trop := distance(p.X, p.Y, p.Z, s.X, s.Y, s.Z) > rayonTrajetM
		if trop && !loin {
			out = append(out, fmt.Sprintf("vehicule %d passager %d @%d", v.Slot, r.Slot, s.T))
		}
		loin = trop
	}
	return out
}

// pointsParSlot : tous les points publies d'un slot, tries par image.
func pointsParSlot(d *Document) map[int][]Point {
	out := map[int][]Point{}
	for _, t := range d.Tracks {
		out[t.Slot] = append(out[t.Slot], t.Points...)
	}
	for s := range out {
		pts := out[s]
		sort.SliceStable(pts, func(i, j int) bool { return pts[i].T < pts[j].T })
	}
	return out
}

// pointA rend le point le plus proche de l'image t, s'il est a ±tolerance images.
func pointA(pts []Point, t, tolerance int) (Point, bool) {
	i := sort.Search(len(pts), func(i int) bool { return pts[i].T >= t-tolerance })
	if i < len(pts) && pts[i].T <= t+tolerance {
		best := pts[i]
		for j := i + 1; j < len(pts) && pts[j].T <= t+tolerance; j++ {
			if abs(pts[j].T-t) < abs(best.T-t) {
				best = pts[j]
			}
		}
		return best, true
	}
	return Point{}, false
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// violationsIdentite : V-7, les desaccords d'identite publies (compteurs sans detail) : lecture
// directe contre pont par morts, deux joueurs dans un meme corps, sieges qui se recouvrent.
func violationsIdentite(d *Document) Violation {
	var out Violation
	if b := d.Coverage.Bridge; b != nil {
		out.Anonymes += b.Discordant + b.SlotCollisions
	}
	if s := d.Coverage.Seats; s != nil {
		out.Anonymes += s.Chevauchements
	}
	return out
}
