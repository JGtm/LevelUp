package replay

// hill_shapes_report_test.go — LOT C-ter VOLET 2 : CE QUE LA MESURE DES FORMES DE COLLINE PUBLIE.
//
// Scinde de `hill_shapes_measure_test.go` (revue adversariale ronde 1, R1-2 : 615 lignes, seuil
// 500). Le partage suit la responsabilite, pas la ligne : l'INSTRUMENT (definitions, seuils,
// appariement, temoins, TestHillShapesMeasure) reste dans le fichier de mesure ; ce fichier-ci
// ne porte que les RAPPORTS que le test imprime a partir d'une mesure deja faite — tableau des
// rampes, chemin de production, comparaison AVANT/APRES, bilan des manques.
//
// Aucun corps n'a change au deplacement : ces fonctions sont celles de la mesure du lot, et les
// chiffres publies dans `lotCter/volet2_*.log` doivent rester reproductibles a l'identique.

import (
	"fmt"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// hillTableau publie chaque rampe : sa forme appariee, ses votes, son occupation.
func hillTableau(t *testing.T, hills []Zone, ramps []zoneRamp, pts map[int][]Point, ser zoneSeries, doc ReplayDocument) {
	t.Helper()
	perHill := map[int]int{}
	ms := doc.FrameIntervalMS
	for i, r := range ramps {
		v := hillJuge(hills, pts, ser, r, 0, doc.FrameCount)
		perHill[v.ref]++
		t.Logf("    rampe %2d slot %d [%6.1f s ; %6.1f s] -> forme %2d votes %4d · emissions %3d occupees %5.1f %% ·"+
			" frames %5.1f %% (stricte %5.1f %%) %s",
			i, r.slot, float64(r.t0*ms)/1000, float64(r.tPeak*ms)/1000, v.ref, v.votes,
			v.steps, 100*v.stepOcc, 100*v.occupancy, 100*v.strict, hillMark(v))
	}
	refs := make([]int, 0, len(perHill))
	for r := range perHill {
		refs = append(refs, r)
	}
	sort.Ints(refs)
	for _, r := range refs {
		t.Logf("    forme %2d : %d rampe(s)", r, perHill[r])
	}
}

func hillMark(v hillVerdict) string {
	switch {
	case v.ref < 0:
		return "NON APPARIEE"
	case !v.inside:
		return "HORS"
	}
	return "dedans"
}

// hillProduction publie ce que le chemin de PRODUCTION (buildHillStates) rend avec ces formes.
func hillProduction(t *testing.T, doc ReplayDocument, hills []Zone) {
	t.Helper()
	if doc.Coverage == nil || doc.Coverage.Zones == nil {
		t.Logf("  PRODUCTION : aucune couverture de zone publiee")
		return
	}
	cv := doc.Coverage.Zones
	actives := 0
	for _, st := range doc.ZoneStates {
		for _, sp := range st.Spans {
			if sp.Active {
				actives += sp.T1 - sp.T0 + 1
			}
		}
	}
	t.Logf("  PRODUCTION (buildHillStates sur %d formes) : methode %s · periodes %d · zones %d ·"+
		" non appariees %d · frames actives %d/%d = %.1f %%",
		len(hills), cv.Method, cv.HillPeriods, len(doc.ZoneStates), cv.Unpaired, actives,
		doc.FrameCount, 100*p2aRate(actives, doc.FrameCount))
}

// hillAvantApres — sur les cartes du catalogue, compare les periodes de PRODUCTION obtenues avec
// les formes de Bastion/Extraction (AVANT, ce que l'artefact publie aujourd'hui) et avec les
// collines (APRES) : pour chaque periode AVANT, la colline qui la contient et si elle est
// distincte de la forme AVANT (centre de la colline hors de la forme AVANT).
func hillAvantApres(t *testing.T, film p2aFilm, ser zoneSeries, c zoneCtx, hills []Zone, pts map[int][]Point, frames int) {
	t.Helper()
	avant := p2aZonesOrNil(t, film.MapID, mapvar.RoleStrongholdZone, mapvar.RoleExtractionZone)
	if len(avant) == 0 {
		t.Logf("  AVANT/APRES : sans objet (aucune forme de Bastion/Extraction au catalogue)")
		return
	}
	avant = zoneCatalogOf(avant)
	covA := &ZonesCoverage{}
	statesA := buildHillStates(avant, ser, nil, c, covA)
	covB := &ZonesCoverage{}
	statesB := buildHillStates(zoneCatalogOf(hills), ser, nil, c, covB)
	t.Logf("  AVANT (Bastion+Extraction, %d formes) : %d periodes, %d zones, %d rampes non appariees",
		len(avant), covA.HillPeriods, len(statesA), covA.Unpaired)
	t.Logf("  APRES (collines, %d formes)            : %d periodes, %d zones, %d rampes non appariees",
		len(hills), covB.HillPeriods, len(statesB), covB.Unpaired)
	type per struct {
		t0, t1, ref int
	}
	var pa []per
	for _, st := range statesA {
		for _, sp := range st.Spans {
			pa = append(pa, per{sp.T0, sp.T1, st.ZoneRef})
		}
	}
	sort.Slice(pa, func(i, j int) bool { return pa[i].t0 < pa[j].t0 })
	differ := 0
	for i, p := range pa {
		ref, _ := hillPair(hills, pts, p.t0, min(p.t1, frames-1))
		za := avant[p.ref]
		desc := "colline NON appariee"
		if ref >= 0 {
			h := hills[ref]
			d := za.Volume.DistanceTo(h.Center)
			distinct := d > 0
			if distinct {
				differ++
			}
			desc = fmt.Sprintf("colline %d (%.1f ; %.1f) a %.1f m de la forme AVANT — %s",
				ref, h.Center.X, h.Center.Y, d, map[bool]string{true: "DISTINCTE", false: "confondue"}[distinct])
		}
		t.Logf("    periode %2d [%5d ; %5d] AVANT zone %d %s (%.1f ; %.1f) -> %s",
			i, p.t0, p.t1, p.ref, za.Role, za.Center.X, za.Center.Y, desc)
	}
	t.Logf("  AVANT/APRES : %d periodes AVANT, %d dont la colline est DISTINCTE de la forme AVANT",
		len(pa), differ)
}

// p2aZonesOrNil : comme p2aZones, mais rend nil (au lieu de sauter) quand la carte manque.
func p2aZonesOrNil(t *testing.T, mapID string, roles ...mapvar.Role) []Zone {
	t.Helper()
	cat, err := LoadMapObjectives(p2aRefDir(t) + "/map_objectives.json")
	if err != nil {
		t.Fatalf("catalogue d'objectifs illisible : %v", err)
	}
	e, err := cat.Lookup(mapID)
	if err != nil {
		return nil
	}
	var out []Zone
	for _, r := range roles {
		out = append(out, e.ZonesOfRole(r).Zones...)
	}
	return out
}

func hillShapeDesc(s *mapvar.Shape) string {
	if s == nil {
		return "sans forme"
	}
	switch s.Family {
	case mapvar.ShapeBox:
		return fmt.Sprintf("boite %.2f x %.2f (h +%.2f/-%.2f)", *s.HalfX, *s.HalfY, s.UpZ, s.DownZ)
	case mapvar.ShapeCylinder:
		return fmt.Sprintf("cylindre r=%.2f (h +%.2f/-%.2f)", *s.Radius, s.UpZ, s.DownZ)
	}
	return string(s.Family)
}

// hillManques publie le BILAN des periodes qui ne tombent pas « dedans » (D3, emissions
// croissantes) : combien sont NON APPARIEES (aucune position a moins de hillTolM d'aucune
// forme, avec leur duree mediane), et combien sont appariees a une forme OCCUPEE pendant la
// periode (>= 30 % des frames) mais VIDE aux instants ou la jauge monte — la signature d'une
// jauge qui monte avant l'arrivee des joueurs, pas d'une forme fausse.
func hillManques(t *testing.T, hills []Zone, ramps []zoneRamp, pts map[int][]Point, ser zoneSeries, doc ReplayDocument) {
	t.Helper()
	var durees []int
	videAuxEmissions, autres := 0, 0
	for _, r := range ramps {
		v := hillJuge(hills, pts, ser, r, 0, doc.FrameCount)
		switch {
		case v.inside:
			continue
		case v.ref < 0:
			durees = append(durees, r.tPeak-r.t0+1)
		case v.stepOcc == 0 && v.occupancy >= 0.3:
			videAuxEmissions++
		default:
			autres++
		}
	}
	sort.Ints(durees)
	med := 0
	if len(durees) > 0 {
		med = durees[len(durees)/2]
	}
	t.Logf("  MANQUES (D3 emissions) : %d non appariees (duree mediane %.1f s), %d appariees a une forme"+
		" occupee >= 30 %% des frames mais VIDE a chaque emission croissante, %d autres",
		len(durees), float64(med*doc.FrameIntervalMS)/1000, videAuxEmissions, autres)
}
