package filmdec

// lot1_sonde_precision_helpers_test.go — decodeurs et utilitaires de la sonde precision/distance
// (scinde de lot1_sonde_precision_research_test.go pour le seuil de 500 lignes). Voir l'en-tete
// de ce fichier-la pour le contexte, les seuils et les 7 mesures.

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// sondeScanDamage rend les evenements damage_aftermath (avec SOURCE) et la base d'atterrissage
// bipede. Le decodage est PRODUCTIONISE (ScanFilmWeaponDamages, Lot 2) ; cet adaptateur mappe
// simplement la sortie de production vers le type sondeDmgEvt de la sonde — une seule copie du scan.
func sondeScanDamage(t *testing.T, dir string, reg *Registry, n int) ([]sondeDmgEvt, int) {
	t.Helper()
	dmgs, base, err := ScanFilmWeaponDamages(dir, reg, n)
	if err != nil {
		t.Fatalf("collecte des degats : %v", err)
	}
	evs := make([]sondeDmgEvt, len(dmgs))
	for i, d := range dmgs {
		evs[i] = sondeDmgEvt{
			ts: d.TimestampUS, idx0: d.VictimIdx, idx1: d.ResponsibleIdx,
			src: d.Source, hasSrc: d.HasSource, neg: d.Negative,
			magClear: d.MagClear, magRaw: d.MagRaw,
		}
	}
	return evs, base
}

// sondeScanFireArme decode les tirs type 36 (0xD2) a la GRAMMAIRE DE REFERENCE (lot1_tirs) :
// l'arme est variant_name R(32). Rend le comptage par arme (records longs), la part courte,
// le nb de records longs et le comptage par index tireur. La lecture des refs d'en-tete
// reutilise lot1RefDom1 (attaquant, dom 1) ; le corps type-36 est celui de lot1_tirs.
func sondeScanFireArme(t *testing.T, dir string, n int) (map[uint64]int, int, int, map[uint64]int) {
	t.Helper()
	armes := map[uint64]int{}
	attaquants := map[uint64]int{}
	court, long := 0, 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 36 {
				continue
			}
			if a, ok := lot1RefDom1(br); ok { // ref0 = attaquant (dom 1, sonde)
				attaquants[a]++
			}
			for range 2 { // ref1 dom 8, ref2 dom 7 : R(1) ; si 1 : R(13)+R(2)
				if br.ReadBit() {
					br.Skip(15)
				}
			}
			estCourt := br.ReadBit()
			br.Skip(1) // estBloc (non exploite ici)
			br.Skip(8) // R(7)+R(1)
			if br.ReadBit() {
				br.Skip(5)
			}
			if !br.ReadBit() {
				br.Skip(2)
			}
			if br.ReadBit() {
				br.Skip(32)
			}
			variant := br.ReadBits(32) // variant_name = l'arme
			if estCourt {
				court++
				continue
			}
			long++
			armes[variant]++
		}
	}
	return armes, court, long, attaquants
}

// sondeBipedTracks decode les positions monde des bipeds et les indexe par slot, triees par
// temps. Filtres teleport/isolation desactives pour MAXIMISER la couverture (on veut savoir
// combien de degats sont resolubles, pas lisser une trajectoire).
func sondeBipedTracks(t *testing.T, dir string, wr *Vec3Range, n int) map[uint32][]sondeSample {
	t.Helper()
	opt := DefaultScanFilmOptions()
	opt.MaxSpeedMPS = 0
	opt.IsolationGapMS = 0
	opt.WorldRange = wr
	opt.Chunks = make([]int, 0, n)
	for c := 1; c <= n; c++ {
		opt.Chunks = append(opt.Chunks, c)
	}
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("balayage biped impossible : %v", err)
	}
	tr := map[uint32][]sondeSample{}
	for _, p := range pos {
		tr[p.Slot] = append(tr[p.Slot], sondeSample{ts: p.TimestampUS, x: p.X, y: p.Y, z: p.Z})
	}
	for s := range tr {
		ss := tr[s]
		sort.Slice(ss, func(i, j int) bool { return ss[i].ts < ss[j].ts })
		tr[s] = ss
	}
	return tr
}

// sondeLookup rend la position du slot la plus proche de T dans [T-tol, T+tol], et sa validite.
func sondeLookup(track []sondeSample, T, tol uint64) (sondeSample, bool) {
	if len(track) == 0 {
		return sondeSample{}, false
	}
	i := sort.Search(len(track), func(i int) bool { return track[i].ts >= T })
	best, ok := sondeSample{}, false
	// candidat avant et candidat apres T ; on garde le plus proche dans la tolerance.
	var bd uint64 = math.MaxUint64
	pick := func(s sondeSample) {
		d := T - s.ts
		if s.ts > T {
			d = s.ts - T
		}
		if d <= tol && d < bd {
			best, ok, bd = s, true, d
		}
	}
	if i-1 >= 0 {
		pick(track[i-1])
	}
	if i < len(track) {
		pick(track[i])
	}
	return best, ok
}

// sondeWorldRange auto-detecte les bornes monde de la carte par la signature de largeurs
// d'axe (controle de coherence du catalogue). Rend nil si la carte est absente/ambigue
// (les distances sont alors desactivees et l'instrument le signale).
func sondeWorldRange(t *testing.T, dir string) *Vec3Range {
	t.Helper()
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Logf("decoupage i0 illisible (%v) : distances desactivees", err)
		return nil
	}
	path := filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Logf("catalogue de bornes illisible (%v) : distances desactivees", err)
		return nil
	}
	if name := os.Getenv(sondeMapEnv); name != "" {
		e, err := cat.Lookup(name)
		if err != nil {
			t.Logf("carte %q absente du catalogue (%v)", name, err)
			return nil
		}
		r := e.Range()
		t.Logf("carte forcee : %s (largeurs %v)", name, e.AxisWidths)
		return &r
	}
	var hits []string
	var found MapQuantEntry
	for name, e := range cat.Maps {
		if e.AxisWidths == lay.AxisW {
			hits = append(hits, name)
			found = e
		}
	}
	if len(hits) != 1 {
		sort.Strings(hits)
		t.Logf("signature %v ambigue : %d cartes %v — renseigner %s ; distances desactivees",
			lay.AxisW, len(hits), hits, sondeMapEnv)
		return nil
	}
	r := found.Range()
	t.Logf("carte auto-detectee par signature %v : %s", lay.AxisW, hits[0])
	return &r
}

// sondeDist rend la distance euclidienne monde entre deux positions.
func sondeDist(a, b sondeSample) float64 {
	dx, dy, dz := float64(b.x-a.x), float64(b.y-a.y), float64(b.z-a.z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// sondeBucket rend l'index de bucket d'une distance (0..len(edges)). PRODUCTIONISE : delegue a
// WeaponHitBucket (weapon_hits.go) — une seule implementation du seuillage.
func sondeBucket(d float64) int {
	return WeaponHitBucket(d)
}

// sondeMedian rend la mediane d'un echantillon (0 si vide).
func sondeMedian(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	return s[len(s)/2]
}

// sondeBaseSweep compte, par base candidate, les degats a deux refs dont les deux positions
// sont trouvees, et rend le pic (base la plus resolvante). Calibration du decalage de bande.
func sondeBaseSweep(dmg []sondeDmgEvt, tr map[uint32][]sondeSample) (string, int, int) {
	out := ""
	peakBase, peakRes := lot1chBases[0], -1
	for _, b := range lot1chBases {
		if b < 400 {
			continue // seules les bases de la bande bipede sont pertinentes
		}
		res := 0
		for _, e := range dmg {
			if e.idx0 < 0 || e.idx1 < 0 {
				continue
			}
			_, okv := sondeLookup(tr[uint32(b+e.idx0)], e.ts, sondePosTolUS)
			_, oka := sondeLookup(tr[uint32(b+e.idx1)], e.ts, sondePosTolUS)
			if okv && oka {
				res++
			}
		}
		out += " " + itoa(b) + "=" + itoa(res)
		if res > peakRes {
			peakBase, peakRes = b, res
		}
	}
	return out, peakBase, peakRes
}

// sonde5Hist rend l'histogramme des buckets de distance en une ligne.
func sonde5Hist(bc []int) string {
	labels := make([]string, len(sondeDistEdges)+1)
	prev := 0.0
	for i, e := range sondeDistEdges {
		labels[i] = formatBucket(prev, e)
		prev = e
	}
	labels[len(sondeDistEdges)] = formatBucket(prev, math.Inf(1))
	out := "buckets"
	for i, c := range bc {
		out += " " + labels[i] + "=" + itoa(c)
	}
	return out
}

// formatBucket rend l'etiquette d'un bucket de distance (itoa vit dans lot1_visee_calib).
func formatBucket(lo, hi float64) string {
	l := itoa(int(lo))
	if math.IsInf(hi, 1) {
		return l + "+"
	}
	return l + "-" + itoa(int(hi))
}
