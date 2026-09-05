package filmdec

// vehicules_v2_items_test.go — les quatre items du lot V2, le regroupement en amas, et les
// chargeurs (corpus, bornes, pads). Complement de vehicules_v2_test.go. LECTURE SEULE.

import (
	"encoding/json"
	"math"
	"os"
	"sort"
	"strings"
	"testing"
)

// ------------------------------------------------------------------ ITEM 1 : emplacements

func v2Item1Spawns(t *testing.T, ag *v2MapAgg) {
	t.Helper()
	t.Logf("=== ITEM 1 — EMPLACEMENTS DE SPAWN ===")
	byChassis := map[uint32][][3]float64{}
	var allPts [][3]float64
	noChassis := 0
	for _, fd := range ag.films {
		for _, b := range fd.births {
			allPts = append(allPts, b.posM)
			if b.hasChassis {
				byChassis[b.chassis] = append(byChassis[b.chassis], b.posM)
			} else {
				noChassis++
			}
		}
	}
	t.Logf("  naissances totales %d (dont %d sans chassis lu) · %d chassis distincts",
		len(allPts), noChassis, len(byChassis))
	v2LogBBox(t, "  bbox naissances (m)", allPts)

	passAll := true
	for _, ch := range v2SortedChassis(byChassis) {
		pts := byChassis[ch]
		clus := v2Cluster(v2Dup(pts), v2ClusterThreshM)
		maxR, singletons := 0.0, 0
		for _, c := range clus {
			if c.radius > maxR {
				maxR = c.radius
			}
			if c.n == 1 {
				singletons++
			}
		}
		ok := len(clus) <= v2MaxClusters && maxR <= v2ClusterThreshM
		t.Logf("  chassis %#010x : %d naissances -> %d amas (dont %d isoles) · rayon max %.2f m (seuils <= %d amas, <= %.0f m) : %s",
			ch, len(pts), len(clus), singletons, maxR, v2MaxClusters, v2ClusterThreshM, v2Verdict(ok))
		for _, c := range clus {
			t.Logf("      amas centre (%.1f, %.1f, %.1f) rayon %.2f m · %d naissances",
				c.center[0], c.center[1], c.center[2], c.radius, c.n)
		}
		if !ok {
			passAll = false
		}
	}
	v2StabilityReport(t, ag, byChassis)
	t.Logf("  ITEM 1 GATE (agrege : <= %d amas/chassis, rayon <= %.0f m) : %s",
		v2MaxClusters, v2ClusterThreshM, v2Verdict(passAll))
}

// v2StabilityReport publie le nombre d'amas par chassis PAR FILM, pour juger la stabilite (+/-1).
func v2StabilityReport(t *testing.T, ag *v2MapAgg, byChassis map[uint32][][3]float64) {
	t.Helper()
	for _, ch := range v2SortedChassis(byChassis) {
		var counts []int
		for _, fd := range ag.films {
			var pts [][3]float64
			for _, b := range fd.births {
				if b.hasChassis && b.chassis == ch {
					pts = append(pts, b.posM)
				}
			}
			if len(pts) == 0 {
				continue
			}
			counts = append(counts, len(v2Cluster(v2Dup(pts), v2ClusterThreshM)))
		}
		if len(counts) == 0 {
			continue
		}
		lo, hi := counts[0], counts[0]
		for _, c := range counts {
			if c < lo {
				lo = c
			}
			if c > hi {
				hi = c
			}
		}
		t.Logf("      stabilite chassis %#010x : amas/film %v · amplitude %d (seuil +/-%d) : %s",
			ch, counts, hi-lo, v2StabilityBand, v2Verdict(hi-lo <= v2StabilityBand))
	}
}

// ------------------------------------------------------------------ ITEM 2 : confrontation .mvar

func v2Item2Mvar(t *testing.T, ag *v2MapAgg, pads *v2PadCatalog) {
	t.Helper()
	t.Logf("=== ITEM 2 — CONFRONTATION .mvar (piege Forge) ===")
	pm, ok := pads.byMapKey(ag.mapKey)
	if !ok {
		t.Logf("  aucune entree pads pour %q — confrontation impossible", ag.mapKey)
		return
	}
	fam := v2FamilyCounts(pm)
	hasVehicle := fam["vehicle"] > 0 || fam["vehi"] > 0
	t.Logf("  catalogue %s (%s) : %d pads · familles %v · famille vehicule : %v",
		pm.Module, pm.MvarFile, len(pm.Pads), fam, hasVehicle)

	v2LogBBox(t, "  bbox pads .mvar (m)", v2PadPoints(pm))
	var births [][3]float64
	for _, fd := range ag.films {
		for _, b := range fd.births {
			births = append(births, b.posM)
		}
	}
	v2LogBBox(t, "  bbox naissances (m)", births)

	clus := v2Cluster(v2Dup(births), v2ClusterThreshM)
	near, considered := 0, 0
	for _, c := range clus {
		if c.n < 3 {
			continue // amas de support faible ecarte de la confrontation
		}
		considered++
		d, famNear := v2NearestPad(c.center, pm)
		flag := ""
		if d < v2PadNearM {
			near++
			flag = "  <1m"
		}
		t.Logf("      amas (%.1f, %.1f, %.1f) n=%d -> pad le plus proche %.2f m (%s)%s",
			c.center[0], c.center[1], c.center[2], c.n, d, famNear, flag)
	}
	frac := v2SafeFrac(near, considered)
	if hasVehicle {
		t.Logf("  ITEM 2 GATE (>= 80 %% amas < 1 m d'un emplacement vehicule) : %.0f %% (%d/%d) : %s",
			100*frac, near, considered, v2Verdict(frac >= 0.8))
		return
	}
	t.Logf("  ITEM 2 : le catalogue .mvar ne porte AUCUNE famille vehicule (canevas+rack Forge : "+
		"rack/power/powerup seulement). %d/%d amas sont a < 1 m d'un pad d'ARME (%.0f %%).", near, considered, 100*frac)
	t.Logf("  ITEM 2 GATE : NON APPLICABLE — aucun emplacement vehicule declare a confronter. " +
		"Un vrai croisement exige une re-extraction du .mvar sur le type_id vehicule (himap + fichiers .mvar absents des donnees) : REPORTE.")
}

// ------------------------------------------------------------------ ITEM 3 : cooldowns

func v2Item3Cooldowns(t *testing.T, ag *v2MapAgg) {
	t.Helper()
	t.Logf("=== ITEM 3 — COOLDOWNS ===")
	var births [][3]float64
	for _, fd := range ag.films {
		for _, b := range fd.births {
			births = append(births, b.posM)
		}
	}
	var centers [][3]float64
	for _, c := range v2Cluster(v2Dup(births), v2ClusterThreshM) {
		if c.n >= 3 {
			centers = append(centers, c.center)
		}
	}
	t.Logf("  %d emplacements (amas n>=3) retenus comme reference", len(centers))

	var gaps []float64 // secondes
	overlaps := 0
	for _, fd := range ag.films {
		gaps, overlaps = v2FilmCooldowns(fd, centers, gaps, overlaps)
	}
	if len(gaps) == 0 {
		t.Logf("  aucun couple (fin bornee -> naissance suivante au meme emplacement) mesurable")
		return
	}
	sort.Float64s(gaps)
	med := v2Median(gaps)
	q1, q3 := v2Quantile(gaps, 0.25), v2Quantile(gaps, 0.75)
	iqr := q3 - q1
	ratio := iqr / med
	t.Logf("  %d cooldowns mesures (%d chevauchements ecartes) · mediane %.0f s · IQR %.0f s (Q1 %.0f, Q3 %.0f)",
		len(gaps), overlaps, med, iqr, q1, q3)
	t.Logf("  IQR/mediane = %.2f (seuil <= %.2f) : %s", ratio, v2CooldownIQRMax, v2Verdict(ratio <= v2CooldownIQRMax))
	t.Logf("  LIMITE : fins de vie bornees a +/-%.0f s ; un cooldown < ~%.0f s n'est pas resolu (resolution-limited=%v).",
		v2KeyframeStepS, v2KeyframeStepS, med < v2KeyframeStepS*1.5)
}

// v2FilmCooldowns accumule les cooldowns d'un film : par emplacement, ecart fin bornee -> naissance suivante.
func v2FilmCooldowns(fd *v2FilmData, centers [][3]float64, gaps []float64, overlaps int) ([]float64, int) {
	type ev struct {
		birthTS, goneBy uint64
		hasEnd          bool
	}
	byEmpl := map[int][]ev{}
	for _, b := range fd.births {
		ei := v2NearestCenter(b.posM, centers)
		if ei < 0 {
			continue
		}
		e := ev{birthTS: b.tsUS}
		if l, ok := fd.lives[[2]uint32{b.slot, b.gen}]; ok && l.hasGoneBy {
			e.goneBy, e.hasEnd = l.goneBy, true
		}
		byEmpl[ei] = append(byEmpl[ei], e)
	}
	for _, list := range byEmpl {
		sort.Slice(list, func(i, j int) bool { return list[i].birthTS < list[j].birthTS })
		for i := 0; i+1 < len(list); i++ {
			if !list[i].hasEnd {
				continue
			}
			gapS := (float64(list[i+1].birthTS) - float64(list[i].goneBy)) / 1e6
			if gapS <= 0 {
				overlaps++
				continue
			}
			gaps = append(gaps, gapS)
		}
	}
	return gaps, overlaps
}

// ------------------------------------------------------------------ ITEM 4 : etat detruit (i14)

func v2Item4Destroyed(t *testing.T, ag *v2MapAgg) {
	t.Helper()
	t.Logf("=== ITEM 4 — ETAT DETRUIT (i14 object-dissolver) ===")
	totRec, totI14, inWindow, evWithLife := 0, 0, 0, 0
	var fracLifePos []float64
	for _, fd := range ag.films {
		totRec += fd.i14tot
		totI14 += fd.i14rec
		for _, e := range fd.i14 {
			l, ok := fd.lives[[2]uint32{e.slot, e.gen}]
			if !ok || l.lastSeen <= l.firstSeen {
				continue
			}
			evWithLife++
			fracLifePos = append(fracLifePos,
				(float64(e.tsUS)-float64(l.firstSeen))/(float64(l.lastSeen)-float64(l.firstSeen)))
			end := l.lastSeen
			if l.hasGoneBy {
				end = l.goneBy
			}
			if math.Abs(float64(e.tsUS)-float64(end)) <= v2KeyframeStepS*1e6 {
				inWindow++
			}
		}
	}
	frac := v2SafeFrac(totI14, totRec)
	t.Logf("  i14 dans le flux delta : %d/%d records (%.4f %%) · plancher faux positifs %.2f %%",
		totI14, totRec, 100*frac, 100*v2FloorFrac)
	switch {
	case frac < v2FloorFrac:
		t.Logf("  i14 SOUS le plancher — NON CONCLUANT (regle cadrage : ne pas interpreter sous le plancher).")
	case len(fracLifePos) > 0:
		sort.Float64s(fracLifePos)
		t.Logf("  %d evenements i14 apparies a une vie · position mediane dans la vie %.2f (1.0=fin) · %d/%d dans +/-%.0f s de la fin bornee",
			evWithLife, v2Median(fracLifePos), inWindow, evWithLife, v2KeyframeStepS)
	default:
		t.Logf("  aucun evenement i14 appariable a une vie recensee.")
	}
}

func v2BonusNote(t *testing.T) {
	t.Logf("\n=== BONUS (signale, non traite) ===")
	t.Logf("  6 banques Wwise sb_008_exp_vehicle_{large,med,small}_{covenant,unsc} = explosions de")
	t.Logf("  vehicule par taille/faction (SON d'etat detruit, pour le lot sons).")
}

// ------------------------------------------------------------------ regroupement en amas

// v2Clu est un amas : centre (moyenne), rayon (dist max au centre), effectif.
type v2Clu struct {
	center [3]float64
	radius float64
	n      int
}

// v2Cluster regroupe des points par l'algorithme du meneur (leader) : un point rejoint l'amas dont
// le centre courant est a moins de thresh, sinon il ouvre un amas. DETERMINISTE (points tries).
func v2Cluster(pts [][3]float64, thresh float64) []v2Clu {
	sort.Slice(pts, func(i, j int) bool {
		for ax := 0; ax < 3; ax++ {
			if pts[i][ax] != pts[j][ax] {
				return pts[i][ax] < pts[j][ax]
			}
		}
		return false
	})
	var sums [][3]float64
	var counts []int
	var members [][]int
	for pi, p := range pts {
		best, bd := -1, thresh
		for i := range sums {
			if d := v2Dist(p, v2Centroid(sums[i], counts[i])); d <= bd {
				bd, best = d, i
			}
		}
		if best < 0 {
			sums = append(sums, p)
			counts = append(counts, 1)
			members = append(members, []int{pi})
			continue
		}
		for ax := 0; ax < 3; ax++ {
			sums[best][ax] += p[ax]
		}
		counts[best]++
		members[best] = append(members[best], pi)
	}
	out := make([]v2Clu, len(sums))
	for i := range sums {
		ctr := v2Centroid(sums[i], counts[i])
		r := 0.0
		for _, mi := range members[i] {
			if d := v2Dist(pts[mi], ctr); d > r {
				r = d
			}
		}
		out[i] = v2Clu{center: ctr, radius: r, n: counts[i]}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].n > out[j].n })
	return out
}

func v2Centroid(sum [3]float64, n int) [3]float64 {
	return [3]float64{sum[0] / float64(n), sum[1] / float64(n), sum[2] / float64(n)}
}

func v2Dist(a, b [3]float64) float64 {
	dx, dy, dz := a[0]-b[0], a[1]-b[1], a[2]-b[2]
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func v2NearestCenter(p [3]float64, centers [][3]float64) int {
	best, bd := -1, v2ClusterThreshM
	for i, c := range centers {
		if d := v2Dist(p, c); d <= bd {
			bd, best = d, i
		}
	}
	return best
}

// ------------------------------------------------------------------ pads .mvar

type v2Pad struct {
	Pos struct {
		X, Y, Z float64
	} `json:"pos"`
	TypeID string `json:"type_id"`
	Family string `json:"family"`
}

type v2PadMap struct {
	Module   string  `json:"module"`
	MvarFile string  `json:"mvar_file"`
	Pads     []v2Pad `json:"pads"`
}

type v2PadCatalog struct {
	Maps map[string]v2PadMap `json:"maps"`
}

// byMapKey retrouve l'entree pads dont le module correspond a la carte (module = "<carte>_<module>").
func (c *v2PadCatalog) byMapKey(mapKey string) (v2PadMap, bool) {
	want := strings.ReplaceAll(NormalizeMapName(mapKey), " ", "_") // "launch site" -> "launch_site"
	for _, m := range c.Maps {
		if strings.HasPrefix(strings.ToLower(m.Module), want) {
			return m, true
		}
	}
	return v2PadMap{}, false
}

func v2FamilyCounts(m v2PadMap) map[string]int {
	out := map[string]int{}
	for _, p := range m.Pads {
		out[p.Family]++
	}
	return out
}

func v2PadPoints(m v2PadMap) [][3]float64 {
	out := make([][3]float64, len(m.Pads))
	for i, p := range m.Pads {
		out[i] = [3]float64{p.Pos.X, p.Pos.Y, p.Pos.Z}
	}
	return out
}

func v2NearestPad(p [3]float64, m v2PadMap) (float64, string) {
	best, fam := math.Inf(1), ""
	for _, pad := range m.Pads {
		if d := v2Dist(p, [3]float64{pad.Pos.X, pad.Pos.Y, pad.Pos.Z}); d < best {
			best, fam = d, pad.Family
		}
	}
	return best, fam
}

// ------------------------------------------------------------------ chargeurs / utilitaires

type v2FilmSpec struct {
	short8, mapKey string
}

func v2ParseFilms(t *testing.T) []v2FilmSpec {
	t.Helper()
	raw := os.Getenv("V2_FILMS")
	if raw == "" {
		t.Skipf("V2_FILMS absent : instrument V2 saute")
	}
	var out []v2FilmSpec
	for _, tok := range strings.Split(raw, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		i := strings.Index(tok, ":")
		if i < 0 {
			t.Fatalf("V2_FILMS : entree %q sans ':' (attendu short8:carte)", tok)
		}
		out = append(out, v2FilmSpec{strings.TrimSpace(tok[:i]), strings.TrimSpace(tok[i+1:])})
	}
	if len(out) == 0 {
		t.Skipf("V2_FILMS vide apres analyse")
	}
	return out
}

func v2Root() string {
	if r := os.Getenv("V2_FILM_ROOT"); r != "" {
		return r
	}
	return `C:\Users\Guillaume\Projects\LevelUp\data\cache`
}

func v2LoadBounds(t *testing.T) *MapQuantCatalog {
	t.Helper()
	path := os.Getenv("V2_BOUNDS")
	if path == "" {
		path = `C:\Users\Guillaume\Projects\LevelUp\data\titles\halo_infinite\reference\map_quant_bounds.json`
	}
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible : %v", err)
	}
	return cat
}

func v2LoadPads(t *testing.T) *v2PadCatalog {
	t.Helper()
	path := os.Getenv("V2_PADS")
	if path == "" {
		path = `C:\Users\Guillaume\Projects\LevelUp\data\titles\halo_infinite\reference\map_weapon_pads.json`
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("catalogue de pads illisible : %v", err)
	}
	var c v2PadCatalog
	if err := json.Unmarshal(blob, &c); err != nil {
		t.Fatalf("catalogue de pads invalide : %v", err)
	}
	return &c
}

func v2SortedChassis(m map[uint32][][3]float64) []uint32 {
	out := make([]uint32, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return len(m[out[i]]) > len(m[out[j]]) })
	return out
}

func v2LogBBox(t *testing.T, label string, pts [][3]float64) {
	t.Helper()
	if len(pts) == 0 {
		t.Logf("%s : vide", label)
		return
	}
	lo, hi := pts[0], pts[0]
	for _, p := range pts {
		for ax := 0; ax < 3; ax++ {
			if p[ax] < lo[ax] {
				lo[ax] = p[ax]
			}
			if p[ax] > hi[ax] {
				hi[ax] = p[ax]
			}
		}
	}
	t.Logf("%s : x %.1f..%.1f · y %.1f..%.1f · z %.1f..%.1f", label,
		lo[0], hi[0], lo[1], hi[1], lo[2], hi[2])
}

func v2Median(sorted []float64) float64 { return v2Quantile(sorted, 0.5) }

func v2Quantile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(q * float64(len(sorted)-1))
	return sorted[idx]
}

func v2SafeFrac(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func v2Verdict(pass bool) string {
	if pass {
		return "PASS"
	}
	return "ECHEC"
}

func v2Dup(pts [][3]float64) [][3]float64 {
	out := make([][3]float64, len(pts))
	copy(out, pts)
	return out
}
