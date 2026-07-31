// tmp_loadprod — rejoue le TEMOIN CROISE du loadout sur le CHEMIN DE PRODUCTION
// (filmdec.ScanFilmKeyframeLoadouts + catalogue weaponv3, 35 familles) au lieu de la sonde
// jetable (74 familles, catalogue complete par une table texte hors production).
// Mesure la perte de couverture due au catalogue reduit. THROWAWAY.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

const defFilm = "c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950"
const boundsJSON = "C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/data/titles/halo_infinite/reference/map_quant_bounds.json"

type loadout struct {
	ts      uint64
	slot    uint32
	weapons []string
}

// ---- pont slot -> index de joueur (copie de replay/shots.go, pour le temoin) ----

const posTolUS = 120000

var aimMatchDeg, aimRejectDeg = 2.0, 20.0

type track struct{ pts []filmdec.BipedPosition }

func (t track) at(ts uint64) (filmdec.BipedPosition, uint64) {
	i := sort.Search(len(t.pts), func(i int) bool { return t.pts[i].TimestampUS >= ts })
	best, bd := filmdec.BipedPosition{}, uint64(math.MaxUint64)
	for _, j := range []int{i - 1, i} {
		if j < 0 || j >= len(t.pts) {
			continue
		}
		var d uint64
		if t.pts[j].TimestampUS > ts {
			d = t.pts[j].TimestampUS - ts
		} else {
			d = ts - t.pts[j].TimestampUS
		}
		if d < bd {
			bd, best = d, t.pts[j]
		}
	}
	return best, bd
}

func angDiff(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func indexBySlot(pos []filmdec.BipedPosition) map[uint32]track {
	m := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		m[p.Slot] = append(m[p.Slot], p)
	}
	out := map[uint32]track{}
	for s, ps := range m {
		sort.Slice(ps, func(i, j int) bool { return ps[i].TimestampUS < ps[j].TimestampUS })
		out[s] = track{ps}
	}
	return out
}

func designate(tr map[uint32]track, e filmdec.FireEvent) (uint32, bool) {
	fh, ok := e.AimHeadingDeg()
	if !ok {
		return 0, false
	}
	bs, bd, sd := uint32(0), math.Inf(1), math.Inf(1)
	found := false
	for s, t := range tr {
		p, d := t.at(e.TimestampUS)
		if d > posTolUS {
			continue
		}
		h, ok := p.AimHeadingDeg()
		if !ok {
			continue
		}
		df := angDiff(fh, float64(h))
		if df < bd {
			sd, bd, bs, found = bd, df, s, true
		} else if df < sd {
			sd = df
		}
	}
	if !found || bd > aimMatchDeg || sd < aimRejectDeg {
		return 0, false
	}
	return bs, true
}

func voteOwners(tr map[uint32]track, ev []filmdec.FireEvent) map[uint32]int {
	votes := map[uint32]map[int]int{}
	for _, e := range ev {
		s, ok := designate(tr, e)
		if !ok {
			continue
		}
		if votes[s] == nil {
			votes[s] = map[int]int{}
		}
		votes[s][e.FilmIndex]++
	}
	out := map[uint32]int{}
	for s, m := range votes {
		best, bn := -1, 0
		for i, n := range m {
			if n > bn || (n == bn && i < best) {
				bn, best = n, i
			}
		}
		out[s] = best
	}
	return out
}

func uniqueSlot(tr map[uint32]track, owner map[uint32]int, pi int, ts uint64) (uint32, bool) {
	var f uint32
	n := 0
	for s, i := range owner {
		if i != pi {
			continue
		}
		if _, d := tr[s].at(ts); d <= posTolUS {
			f, n = s, n+1
		}
	}
	return f, n == 1
}

type mapEntry struct {
	Min [3]float32 `json:"min"`
	Max [3]float32 `json:"max"`
}

type shot struct {
	ts   uint64
	slot uint32
	pi   int
	fam  string
}

func main() {
	dir := flag.String("film", defFilm, "repertoire du film")
	mp := flag.String("map", "cliffhanger", "carte")
	shuf := flag.Int("shuffle", 0, "rotation des loadouts entre slots du MEME keyframe")
	flag.Parse()

	fams := map[uint32]bool{}
	for f := range weaponv3.KnownWeaponHigh32 {
		fams[f] = true
	}
	fmt.Printf("=== catalogue de PRODUCTION : %d familles, %d noms canoniques ===\n",
		len(fams), countNames())

	raw, err := filmdec.ScanFilmKeyframeLoadouts(*dir, fams)
	if err != nil {
		panic(err)
	}
	// repli des alias (meme logique que replay/loadouts.go)
	var lo []loadout
	kfOf := map[uint64]int{}
	for _, l := range raw {
		if _, ok := kfOf[l.TimestampUS]; !ok {
			kfOf[l.TimestampUS] = len(kfOf)
		}
		seen := map[string]bool{}
		var ws []string
		for _, f := range l.Families {
			n := weaponv3.WeaponName(f)
			if n == "" || seen[n] {
				continue
			}
			seen[n] = true
			ws = append(ws, n)
		}
		if len(ws) == 0 {
			continue
		}
		lo = append(lo, loadout{l.TimestampUS, l.Slot, ws})
	}
	sort.Slice(lo, func(i, j int) bool {
		if lo[i].ts != lo[j].ts {
			return lo[i].ts < lo[j].ts
		}
		return lo[i].slot < lo[j].slot
	})
	if *shuf != 0 {
		byKF := map[uint64][]int{}
		for i := range lo {
			byKF[lo[i].ts] = append(byKF[lo[i].ts], i)
		}
		for _, idx := range byKF {
			sort.Slice(idx, func(a, b int) bool { return lo[idx[a]].slot < lo[idx[b]].slot })
			n := len(idx)
			if n < 2 {
				continue
			}
			orig := make([][]string, n)
			for j, i := range idx {
				orig[j] = lo[i].weapons
			}
			for j, i := range idx {
				lo[i].weapons = orig[((j+*shuf)%n+n)%n]
			}
		}
		fmt.Printf("=== MODE TEMOIN : loadouts permutes de %d cran(s) entre slots du meme keyframe ===\n", *shuf)
	}

	slots := map[uint32]bool{}
	nW := map[int]int{}
	weap := map[string]int{}
	combos := map[string]int{}
	for _, l := range lo {
		slots[l.slot] = true
		nW[len(l.weapons)]++
		for _, w := range l.weapons {
			weap[w]++
		}
		c := append([]string{}, l.weapons...)
		sort.Strings(c)
		combos[strings.Join(c, "+")]++
	}
	fmt.Printf("=== %d loadouts / %d records biped porteurs bruts / %d keyframes / %d slots ===\n",
		len(lo), len(raw), len(kfOf), len(slots))
	fmt.Printf("=== armes par loadout : %v ; %d armes distinctes ; %d combinaisons distinctes ===\n",
		nW, len(weap), len(combos))
	var wk []string
	for k := range weap {
		wk = append(wk, k)
	}
	sort.Slice(wk, func(i, j int) bool { return weap[wk[i]] > weap[wk[j]] })
	for _, k := range wk {
		fmt.Printf("    %-18s %d\n", k, weap[k])
	}
	h := 0.0
	for _, n := range combos {
		p := float64(n) / float64(len(lo))
		h -= p * math.Log2(p)
	}
	fmt.Printf("=== entropie des combinaisons : %.2f bits ===\n", h)

	// ---- temoin croise ----
	var cat struct {
		Maps map[string]mapEntry `json:"maps"`
	}
	b, err := os.ReadFile(boundsJSON)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, &cat); err != nil {
		panic(err)
	}
	me := cat.Maps[*mp]
	wr := filmdec.Vec3Range{
		{Min: me.Min[0], Max: me.Max[0]},
		{Min: me.Min[1], Max: me.Max[1]},
		{Min: me.Min[2], Max: me.Max[2]},
	}
	scan := filmdec.DefaultScanFilmOptions()
	scan.CaptureDirs = true
	scan.WorldRange = &wr
	pos, err := filmdec.ScanFilmBipedPositions(*dir, scan)
	if err != nil {
		panic(err)
	}
	ev, err := filmdec.ScanFilmFireEvents(*dir)
	if err != nil {
		panic(err)
	}
	tr := indexBySlot(pos)
	owner := voteOwners(tr, ev)

	var shots []shot
	for _, e := range ev {
		if e.WeaponID == 0 {
			continue
		}
		hi, known := weaponv3.CanonWeaponID(e.WeaponID)
		nm := weaponv3.WeaponName(hi)
		if !known || nm == "" {
			continue
		}
		s, ok := uniqueSlot(tr, owner, e.FilmIndex, e.TimestampUS)
		if !ok {
			continue
		}
		shots = append(shots, shot{e.TimestampUS, s, e.FilmIndex, nm})
	}

	bySlot := map[uint32][]loadout{}
	for _, l := range lo {
		bySlot[l.slot] = append(bySlot[l.slot], l)
	}
	ref := func(s uint32, ts uint64) (loadout, bool) {
		ls := bySlot[s]
		if len(ls) == 0 {
			return loadout{}, false
		}
		best, ok := loadout{}, false
		for _, l := range ls {
			if l.ts <= ts {
				best, ok = l, true
			}
		}
		if !ok {
			return ls[0], true
		}
		return best, true
	}
	has := func(l loadout, w string) bool {
		for _, x := range l.weapons {
			if x == w {
				return true
			}
		}
		return false
	}

	nRef, nOK := 0, 0
	per := map[int][2]int{}
	for _, sh := range shots {
		l, ok := ref(sh.slot, sh.ts)
		if !ok {
			continue
		}
		nRef++
		c := per[sh.pi]
		c[1]++
		if has(l, sh.fam) {
			nOK++
			c[0]++
		}
		per[sh.pi] = c
	}
	fmt.Printf("\n=== POSITIF : arme du tir dans le loadout du MEME slot : %d/%d = %.1f%% ===\n",
		nOK, nRef, 100*float64(nOK)/float64(nRef))
	for pi := 0; pi < 8; pi++ {
		if c, ok := per[pi]; ok {
			fmt.Printf("    joueur %d : %d/%d = %.0f%%\n", pi, c[0], c[1], 100*float64(c[0])/float64(c[1]))
		}
	}

	nNeg, nNegOK := 0, 0
	for _, sh := range shots {
		for s2 := range bySlot {
			if s2 == sh.slot {
				continue
			}
			if _, d := tr[s2].at(sh.ts); d > posTolUS {
				continue
			}
			l2, ok := ref(s2, sh.ts)
			if !ok {
				continue
			}
			nNeg++
			if has(l2, sh.fam) {
				nNegOK++
			}
		}
	}
	fmt.Printf("=== NEGATIF A (autre slot vivant) : %d/%d = %.1f%% ===\n",
		nNegOK, nNeg, 100*float64(nNegOK)/float64(nNeg))

	sumOK, sumN := 0, 0
	for k := 1; k <= 7; k++ {
		for _, sh := range shots {
			s2, found := uniqueSlot(tr, owner, (sh.pi+k)%8, sh.ts)
			if !found {
				continue
			}
			l2, ok := ref(s2, sh.ts)
			if !ok {
				continue
			}
			sumN++
			if has(l2, sh.fam) {
				sumOK++
			}
		}
	}
	fmt.Printf("=== NEGATIF B (permutation joueur k=1..7) : %d/%d = %.1f%% ===\n",
		sumOK, sumN, 100*float64(sumOK)/float64(sumN))
}

func countNames() int {
	s := map[string]bool{}
	for _, n := range weaponv3.KnownWeaponHigh32 {
		s[n] = true
	}
	return len(s)
}
