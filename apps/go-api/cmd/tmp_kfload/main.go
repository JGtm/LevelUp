// tmp_kfload — LOADOUT keyframe joint au slot/joueur + TEMOIN CROISE contre les
// evenements de tir (record type 105). THROWAWAY.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = "c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950"
const namesJSON = "C:/Users/GUILLA~1/AppData/Local/Temp/claude/C--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration--claude-worktrees-filmdec-continuation/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad/weapon_names.json"
const csvOut = "C:/Users/GUILLA~1/AppData/Local/Temp/claude/C--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration--claude-worktrees-filmdec-continuation/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad/loadouts_000d5950.csv"
const boundsJSON = "C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/data/titles/halo_infinite/reference/map_quant_bounds.json"

// ---------- familles d'armes ----------

var famName = map[uint32]string{} // famille high-32 -> nom canonique

func normalize(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(s, "THEATER: "))
	if i := strings.Index(s, " (variante"); i > 0 {
		s = strings.TrimSpace(s[i+len(" (variante"):])
		if j := strings.Index(s, ")"); j >= 0 {
			s = strings.TrimSpace(s[j+1:])
		}
	}
	s = strings.TrimSpace(strings.TrimPrefix(s, ")"))
	switch {
	case strings.HasPrefix(s, "MELEE"):
		return "MELEE"
	case strings.HasPrefix(s, "GRENADE"):
		return "GRENADE"
	}
	for _, head := range []string{"Gravity Hammer", "Energy Sword", "Shock Rifle"} {
		if strings.Contains(s, head) {
			return head
		}
	}
	switch s {
	case "Diminisher of Hope", "Rushdown Hammer":
		return "Gravity Hammer"
	case "Duelist Energy Sword", "Elite Bloodblade", "Infected Energy Sword":
		return "Energy Sword"
	}
	return s
}

func loadFamilies() {
	best := map[uint32]uint64{}
	for id := range analysis.WeaponIDToName {
		f := uint32(id >> 32)
		cur, ok := best[f]
		if !ok || (uint32(id) == 0x42c9679f && uint32(cur) != 0x42c9679f) {
			best[f] = id
		}
	}
	for f, id := range best {
		famName[f] = normalize(analysis.WeaponIDToName[id])
	}
	if raw, err := os.ReadFile(namesJSON); err == nil {
		var m map[string]string
		if json.Unmarshal(raw, &m) == nil {
			for k, v := range m {
				var x uint32
				fmt.Sscanf(k, "%x", &x)
				if _, dup := famName[x]; !dup {
					famName[x] = normalize(v)
				}
			}
		}
	}
}

func bitAt(b []byte, p int) uint32 {
	if p < 0 || p>>3 >= len(b) {
		return 0
	}
	return uint32(b[p>>3]>>(7-uint(p&7))) & 1
}

// ---------- loadout keyframe ----------

type loadout struct {
	kf      int
	tsUS    uint64
	slot    uint32
	weapons []string
}

func scanLoadouts(dir string) []loadout {
	var out []loadout
	kfIdx := 0
	for c := 1; c <= filmdec.CountFilmChunks(dir); c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(data)
			recs := filmdec.WalkKeyframeWorld(pay)
			sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
			type h struct {
				bit int
				fam uint32
			}
			byRec := map[int][]h{}
			nb := len(pay) * 8
			var v uint32
			for b := 0; b < nb; b++ {
				v = v<<1 | bitAt(pay, b)
				if b < 31 {
					continue
				}
				if _, ok := famName[v]; !ok {
					continue
				}
				start := b - 31
				ri := sort.Search(len(recs), func(i int) bool { return recs[i].Bit > start }) - 1
				if ri >= 0 && recs[ri].TI == 35 {
					byRec[ri] = append(byRec[ri], h{start, v})
				}
			}
			for ri, hs := range byRec {
				sort.Slice(hs, func(i, j int) bool { return hs[i].bit < hs[j].bit })
				l := loadout{kf: kfIdx, tsUS: p.TimestampUS, slot: uint32(recs[ri].Slot)}
				seen := map[string]bool{}
				for _, x := range hs {
					nm := famName[x.fam]
					if nm == "" || seen[nm] {
						continue
					}
					seen[nm] = true
					l.weapons = append(l.weapons, nm)
				}
				out = append(out, l)
			}
			kfIdx++
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].kf != out[j].kf {
			return out[i].kf < out[j].kf
		}
		return out[i].slot < out[j].slot
	})
	return out
}

// ---------- pont slot -> index de joueur (meme logique que replay/shots.go) ----------

var (
	aimMatchDeg  = 2.0
	aimRejectDeg = 20.0
)

const posTolUS = 120000

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

func uniqueSlot(tr map[uint32]track, owner map[uint32]int, e filmdec.FireEvent) (uint32, bool) {
	var f uint32
	n := 0
	for s, i := range owner {
		if i != e.FilmIndex {
			continue
		}
		if _, d := tr[s].at(e.TimestampUS); d <= posTolUS {
			f, n = s, n+1
		}
	}
	return f, n == 1
}

// ---------- main ----------

type mapEntry struct {
	Module     string     `json:"module"`
	Min        [3]float32 `json:"min"`
	Max        [3]float32 `json:"max"`
	AxisWidths [3]uint    `json:"axisWidths"`
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
	am := flag.Float64("match", 2.0, "seuil de designation (deg)")
	ar := flag.Float64("reject", 20.0, "seuil d'ambiguite (deg)")
	shuf := flag.Int("shuffle", 0, "rotation des loadouts entre slots du MEME keyframe (temoin de la jointure record->slot)")
	flag.Parse()
	aimMatchDeg, aimRejectDeg = *am, *ar
	loadFamilies()

	var cat struct {
		Maps map[string]mapEntry `json:"maps"`
	}
	raw, err := os.ReadFile(boundsJSON)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(raw, &cat); err != nil {
		panic(err)
	}
	me := cat.Maps[*mp]
	wr := filmdec.Vec3Range{
		{Min: me.Min[0], Max: me.Max[0]},
		{Min: me.Min[1], Max: me.Max[1]},
		{Min: me.Min[2], Max: me.Max[2]},
	}

	lo := scanLoadouts(*dir)
	if *shuf != 0 {
		// TEMOIN DE LA JOINTURE : on garde les MEMES loadouts, les MEMES slots, le MEME
		// keyframe — on permute seulement QUI porte QUOI (rotation circulaire de k crans
		// entre les records biped d'un meme keyframe).
		byKF := map[int][]int{}
		for i := range lo {
			byKF[lo[i].kf] = append(byKF[lo[i].kf], i)
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
	fmt.Printf("=== %d loadouts de record biped sur %d keyframes ===\n", len(lo), lo[len(lo)-1].kf+1)

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
	fmt.Printf("=== %d positions, %d slots, %d events de tir ; %d slots avec proprietaire vote ===\n",
		len(pos), len(tr), len(ev), len(owner))

	slotsLoad := map[uint32]bool{}
	for _, l := range lo {
		slotsLoad[l.slot] = true
	}
	joined, players := 0, map[int]bool{}
	kfs := map[int]bool{}
	for _, l := range lo {
		if pi, ok := owner[l.slot]; ok {
			joined++
			players[pi] = true
			kfs[l.kf] = true
		}
	}
	fmt.Printf("=== JOINTURE : %d/%d loadouts rattaches a un joueur ; %d slots porteurs ; %d joueurs ; %d keyframes ===\n",
		joined, len(lo), len(slotsLoad), len(players), len(kfs))

	var shots []shot
	for _, e := range ev {
		if e.WeaponID == 0 {
			continue
		}
		nm := famName[uint32(e.WeaponID>>32)]
		if nm == "" {
			continue
		}
		s, ok := uniqueSlot(tr, owner, e)
		if !ok {
			continue
		}
		shots = append(shots, shot{e.TimestampUS, s, e.FilmIndex, nm})
	}
	fmt.Printf("=== %d tirs rattaches a un slot (arme nommee) ===\n", len(shots))

	var csv strings.Builder
	csv.WriteString("keyframe,timestampUs,slot,playerIndex,arme1,arme2,arme3\n")
	for _, l := range lo {
		pi := -1
		if v, ok := owner[l.slot]; ok {
			pi = v
		}
		w := append([]string{}, l.weapons...)
		for len(w) < 3 {
			w = append(w, "")
		}
		fmt.Fprintf(&csv, "%d,%d,%d,%d,%s,%s,%s\n", l.kf, l.tsUS, l.slot, pi, w[0], w[1], w[2])
	}
	if err := os.WriteFile(csvOut, []byte(csv.String()), 0o644); err != nil {
		fmt.Println("ecriture CSV:", err)
	}

	bySlot := map[uint32][]loadout{}
	for _, l := range lo {
		bySlot[l.slot] = append(bySlot[l.slot], l)
	}
	for s := range bySlot {
		sort.Slice(bySlot[s], func(i, j int) bool { return bySlot[s][i].tsUS < bySlot[s][j].tsUS })
	}
	ref := func(s uint32, ts uint64) (loadout, bool) {
		ls := bySlot[s]
		if len(ls) == 0 {
			return loadout{}, false
		}
		best, ok := loadout{}, false
		for _, l := range ls {
			if l.tsUS <= ts {
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

	nRef, nOK, nPos0, nPos1 := 0, 0, 0, 0
	perPlayer := map[int][2]int{}
	weaponsOf := map[int]map[string]int{}
	slotsCovered := map[uint32]bool{}
	for _, sh := range shots {
		l, ok := ref(sh.slot, sh.ts)
		if !ok {
			continue
		}
		nRef++
		slotsCovered[sh.slot] = true
		c := perPlayer[sh.pi]
		c[1]++
		if has(l, sh.fam) {
			nOK++
			c[0]++
			if len(l.weapons) > 0 && l.weapons[0] == sh.fam {
				nPos0++
			} else if len(l.weapons) > 1 && l.weapons[1] == sh.fam {
				nPos1++
			}
		} else {
			nxt := "aucun"
			for _, l2 := range bySlot[sh.slot] {
				if l2.tsUS > sh.ts {
					nxt = fmt.Sprintf("kf%d t=%.1fs [%s] contient=%v", l2.kf,
						float64(l2.tsUS-4517903407)/1e6, strings.Join(l2.weapons, "+"), has(l2, sh.fam))
					break
				}
			}
			fmt.Printf("   DESACCORD joueur %d slot%d t=%.1fs tir=%s loadout(kf%d,t=%.1fs)=[%s] | keyframe SUIVANT : %s\n",
				sh.pi, sh.slot, float64(sh.ts-4517903407)/1e6, sh.fam, l.kf,
				float64(l.tsUS-4517903407)/1e6, strings.Join(l.weapons, "+"), nxt)
		}
		perPlayer[sh.pi] = c
		if weaponsOf[sh.pi] == nil {
			weaponsOf[sh.pi] = map[string]int{}
		}
		weaponsOf[sh.pi][sh.fam]++
	}
	fmt.Printf("=== %d slots distincts couverts par le temoin ===\n", len(slotsCovered))
	fmt.Printf("\n=== TEMOIN CROISE POSITIF : arme du tir dans le loadout du MEME slot : %d/%d = %.1f%% ===\n",
		nOK, nRef, 100*float64(nOK)/float64(nRef))
	for pi := 0; pi < 8; pi++ {
		if c, ok := perPlayer[pi]; ok {
			fmt.Printf("   joueur %d : %d/%d = %.0f%%\n", pi, c[0], c[1], 100*float64(c[0])/float64(c[1]))
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
	fmt.Printf("\n=== TEMOIN NEGATIF (meme tir, loadout d'un AUTRE slot vivant au meme instant) : %d/%d = %.1f%% ===\n",
		nNegOK, nNeg, 100*float64(nNegOK)/float64(nNeg))
	fmt.Printf("    position dans le loadout des accords : primaire=%d secondaire=%d (autre=%d)\n",
		nPos0, nPos1, nOK-nPos0-nPos1)

	// TEMOIN NEGATIF 2 : appariement JOUEUR (decalage cyclique k des index de joueur).
	slotOfPlayerAt := func(pi int, ts uint64) (uint32, bool) {
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
	fmt.Println("\n=== TEMOIN NEGATIF 2 : tirs du joueur p confrontes au loadout du joueur (p+k) mod 8 ===")
	sumOK, sumN := 0, 0
	for k := 1; k <= 7; k++ {
		ok2, n2 := 0, 0
		for _, sh := range shots {
			s2, found := slotOfPlayerAt((sh.pi+k)%8, sh.ts)
			if !found {
				continue
			}
			l2, ok := ref(s2, sh.ts)
			if !ok {
				continue
			}
			n2++
			if has(l2, sh.fam) {
				ok2++
			}
		}
		sumOK, sumN = sumOK+ok2, sumN+n2
		fmt.Printf("   k=%d : %d/%d = %.1f%%\n", k, ok2, n2, 100*float64(ok2)/float64(n2))
	}
	fmt.Printf("   TOTAL k=1..7 : %d/%d = %.1f%%\n", sumOK, sumN, 100*float64(sumOK)/float64(sumN))

	fmt.Println("\n=== ENTROPIE des armes de tir par joueur (bits) ===")
	for pi := 0; pi < 8; pi++ {
		m := weaponsOf[pi]
		if len(m) == 0 {
			continue
		}
		tot := 0
		for _, n := range m {
			tot += n
		}
		h := 0.0
		var parts []string
		type kv struct {
			k string
			v int
		}
		var ks []kv
		for k, v := range m {
			ks = append(ks, kv{k, v})
		}
		sort.Slice(ks, func(i, j int) bool { return ks[i].v > ks[j].v })
		for _, e := range ks {
			p := float64(e.v) / float64(tot)
			h -= p * math.Log2(p)
			parts = append(parts, fmt.Sprintf("%s:%d", e.k, e.v))
		}
		fmt.Printf("   joueur %d : H=%.2f bits, %d armes distinctes, n=%d | %s\n",
			pi, h, len(m), tot, strings.Join(parts, " "))
	}

	fmt.Println("\n=== CHANGEMENTS D'ARME entre keyframes consecutifs (meme joueur) ===")
	byPlayer := map[int][]loadout{}
	for _, l := range lo {
		if pi, ok := owner[l.slot]; ok {
			byPlayer[pi] = append(byPlayer[pi], l)
		}
	}
	t0 := uint64(4517903407)
	nSwap, nSwapCorrob, nPair, nSame, nSameCorrob := 0, 0, 0, 0, 0
	for pi := 0; pi < 8; pi++ {
		ls := byPlayer[pi]
		sort.Slice(ls, func(i, j int) bool { return ls[i].tsUS < ls[j].tsUS })
		for i := 1; i < len(ls); i++ {
			a, b := ls[i-1], ls[i]
			nPair++
			if strings.Join(a.weapons, "|") == strings.Join(b.weapons, "|") {
				continue
			}
			nSwap++
			newW := map[string]bool{}
			for _, w := range b.weapons {
				if !has(a, w) {
					newW[w] = true
				}
			}
			corrob := false
			for _, sh := range shots {
				if sh.pi != pi || sh.ts < a.tsUS || sh.ts > b.tsUS {
					continue
				}
				if newW[sh.fam] {
					corrob = true
					break
				}
			}
			if corrob {
				nSwapCorrob++
			}
			if a.slot == b.slot {
				nSame++
				if corrob {
					nSameCorrob++
				}
			}
			fmt.Printf("   joueur %d  %6.1fs slot%d [%s]  ->  %6.1fs slot%d [%s]  nouvelles=%v corrobore=%v\n",
				pi, float64(a.tsUS-t0)/1e6, a.slot, strings.Join(a.weapons, "+"),
				float64(b.tsUS-t0)/1e6, b.slot, strings.Join(b.weapons, "+"), keys(newW), corrob)
		}
	}
	fmt.Printf("   %d changements sur %d paires consecutives ; %d corrobores par un tir de l'arme nouvelle dans l'intervalle\n",
		nSwap, nPair, nSwapCorrob)
	fmt.Printf("   dont %d a SLOT CONSTANT (ramassage en cours de vie, %d corrobores) et %d avec changement de slot (reapparition)\n",
		nSame, nSameCorrob, nSwap-nSame)
}

func keys(m map[string]bool) []string {
	var o []string
	for k := range m {
		o = append(o, k)
	}
	sort.Strings(o)
	return o
}
