package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"math"
	"os"
	"sort"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const liveDir = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce`
const sfx = uint32(0x42c9679f)

var h32 = map[uint32]string{}

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}
func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}
func u64LE(d []byte, bp int) uint64 {
	var b [8]byte
	for i := 0; i < 8; i++ {
		var by byte
		for j := 0; j < 8; j++ {
			q := bp + i*8 + j
			if q>>3 < len(d) {
				by |= ((d[q>>3] >> uint(7-(q&7))) & 1) << uint(7-j)
			}
		}
		b[i] = by
	}
	return binary.LittleEndian.Uint64(b[:])
}
func idxD(h uint32) int {
	if h < 0xEC500000 || h > 0xEC600000 {
		return -1
	}
	return int((h - 0xEC500000) / 0x10002)
}
func idxK(h uint32) int {
	if h < 0xE1500000 || h > 0xE1600000 {
		return -1
	}
	return int((h - 0xE1500000) / 0x10002)
}

type dmg struct {
	slot int
	fam  string
	ts   uint64
}

func main() {
	m := "000d5950"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	cache := root + "/" + m
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	// dégâts 0xd2
	var dmgs []dmg
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			bp := 41
			if bitsAt(pl, bp, 1) == 1 {
				bp++
			} else {
				bp += 3
			}
			f := uint32(bitsAt(pl, bp, 32))
			if nm, ok := h32[f]; ok && uint32(bitsAt(pl, bp+32, 32)) == sfx {
				dmgs = append(dmgs, dmg{int(bitsAt(pl, 36, 5)) >> 1, nm, ts})
			}
		}
	}
	sort.Slice(dmgs, func(i, j int) bool { return dmgs[i].ts < dmgs[j].ts })
	// kill-feed (killer xuid, TimeMS)
	var kills, deaths []struct {
		x uint64
		t int
	}
	for ch := 41; ch >= 18; ch-- {
		b := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(b) == 0 {
			continue
		}
		ev, _ := analysis.ParseHighlightEvents(b, 0)
		var kk, dd []struct {
			x uint64
			t int
		}
		for _, e := range ev {
			switch e.EventType {
			case analysis.EventTypeKill:
				kk = append(kk, struct {
					x uint64
					t int
				}{e.XUID, e.TimeMS})
			case analysis.EventTypeDeath:
				dd = append(dd, struct {
					x uint64
					t int
				}{e.XUID, e.TimeMS})
			}
		}
		if len(kk) > len(kills) {
			kills, deaths = kk, dd
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	// roster type-8
	var t8 []byte
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ == 8 && len(pl) > len(t8) {
				t8 = pl
			}
		}
	}
	want := map[uint64]bool{}
	for _, k := range kills {
		want[k.x] = true
	}
	for _, d := range deaths {
		want[d.x] = true
	}
	fb := map[uint64]int{}
	for bp := 0; bp <= len(t8)*8-64; bp++ {
		if v := u64LE(t8, bp); want[v] {
			if _, ok := fb[v]; !ok {
				fb[v] = bp
			}
		}
	}
	type xb struct {
		x  uint64
		bp int
	}
	var xbs []xb
	for x, bp := range fb {
		xbs = append(xbs, xb{x, bp})
	}
	sort.Slice(xbs, func(i, j int) bool { return xbs[i].bp < xbs[j].bp })
	slotOf := map[uint64]int{}
	for s, e := range xbs {
		slotOf[e.x] = s
	}
	// pair kills<->deaths pour la victime (comme offgen)
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	used := make([]bool, len(deaths))
	type fk struct {
		kx, vx uint64
		t      int
	}
	var feed []fk
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if used[i] || d.x == k.x {
				continue
			}
			dt := k.t - d.t
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd, best = dt, i
			}
		}
		vx := uint64(0)
		if best >= 0 {
			used[best] = true
			vx = deaths[best].x
		}
		feed = append(feed, fk{k.x, vx, k.t})
	}
	// kills avec slot (+ victime slot)
	type kk struct{ slot, vslot, t int }
	var ks []kk
	for _, k := range feed {
		if s, ok := slotOf[k.kx]; ok {
			vs := -1
			if v, ok2 := slotOf[k.vx]; ok2 {
				vs = v
			}
			ks = append(ks, kk{s, vs, k.t})
		}
	}
	if len(ks) == 0 || len(dmgs) == 0 {
		fmt.Println("données insuffisantes")
		return
	}
	// WARP ICP : ts -> TimeMS, derive offline
	var tsMin, tsMax uint64 = dmgs[0].ts, dmgs[len(dmgs)-1].ts
	tMin, tMax := ks[0].t, ks[0].t
	for _, k := range ks {
		if k.t < tMin {
			tMin = k.t
		}
		if k.t > tMax {
			tMax = k.t
		}
	}
	a := float64(tMax-tMin) / float64(tsMax-tsMin)
	b := float64(tMin) - a*float64(tsMin)
	warp := func(ts uint64) float64 { return a*float64(ts) + b }
	// raffinage : 3 iters "nearest" (bootstrap robuste) puis 3 iters "dernier-avant" (anchors letaux precis)
	fit := func(lastBefore bool) {
		var sx, sy, sxx, sxy float64
		n := 0.0
		for _, k := range ks {
			var bts uint64
			bd := 1e18
			found := false
			for _, d := range dmgs {
				if d.slot != k.slot {
					continue
				}
				wt := warp(d.ts)
				if lastBefore {
					if wt <= float64(k.t) && d.ts > bts {
						bts = d.ts
						found = true
						bd = float64(k.t) - wt
					}
				} else {
					di := wt - float64(k.t)
					if di < 0 {
						di = -di
					}
					if di < bd {
						bd = di
						bts = d.ts
						found = true
					}
				}
			}
			if found && bd < 8000 {
				x := float64(bts)
				y := float64(k.t)
				sx += x
				sy += y
				sxx += x * x
				sxy += x * y
				n++
			}
		}
		if n > 2 {
			na := (n*sxy - sx*sy) / (n*sxx - sx*sx)
			nb := (sy - na*sx) / n
			if na > 0 {
				a, b = na, nb
			}
		}
	}
	for it := 0; it < 3; it++ {
		fit(false)
	}
	for it := 0; it < 3; it++ {
		fit(true)
	}
	// R2 du warp final (anchors last-before)
	var sx, sy, sxx, syy, sxy, nn float64
	for _, k := range ks {
		var bts uint64
		found := false
		for _, d := range dmgs {
			if d.slot != k.slot {
				continue
			}
			if warp(d.ts) <= float64(k.t) && d.ts > bts {
				bts = d.ts
				found = true
			}
		}
		if found {
			x := float64(bts)
			y := float64(k.t)
			sx += x
			sy += y
			sxx += x * x
			syy += y * y
			sxy += x * y
			nn++
		}
	}
	r2 := 0.0
	if nn > 2 {
		num := nn*sxy - sx*sy
		den := (nn*sxx - sx*sx) * (nn*syy - sy*sy)
		if den > 0 {
			r2 = num * num / den
		}
	}
	// ATTRIBUTION offline. Stratégie sélectionnable via env ATTRIB :
	//   "lastbefore" (défaut) = dernier 0xd2 du tueur <= temps kill (en temps warpé)
	//   "majw"               = arme MAJORITAIRE des 0xd2 du tueur dans [k.t-W, k.t+W] (W=env WIN)
	//   "nearest"            = 0xd2 du tueur le plus proche du kill
	// majw est robuste au lag variable du kill-feed TimeMS (couvre les 2 côtés).
	strat := os.Getenv("ATTRIB")
	if strat == "" {
		strat = "lastbefore"
	}
	win := 1000.0
	if v := os.Getenv("WIN"); v != "" {
		fmt.Sscanf(v, "%f", &win)
	}
	type out struct {
		slot, vslot, t int
		w              string
	}
	var outs []out
	for _, k := range ks {
		best := ""
		switch strat {
		case "majw":
			tally := map[string]int{}
			bestN := 0
			for _, d := range dmgs {
				if d.slot != k.slot {
					continue
				}
				wt := warp(d.ts)
				if wt >= float64(k.t)-win && wt <= float64(k.t)+win {
					tally[d.fam]++
					if tally[d.fam] > bestN {
						bestN = tally[d.fam]
						best = d.fam
					}
				}
			}
		case "nearest":
			bd := 1e18
			for _, d := range dmgs {
				if d.slot != k.slot {
					continue
				}
				di := warp(d.ts) - float64(k.t)
				if di < 0 {
					di = -di
				}
				if di < bd {
					bd = di
					best = d.fam
				}
			}
		default: // lastbefore
			var bts uint64
			for _, d := range dmgs {
				if d.slot != k.slot {
					continue
				}
				if warp(d.ts) <= float64(k.t) && d.ts >= bts {
					bts = d.ts
					best = d.fam
				}
			}
		}
		outs = append(outs, out{k.slot, k.vslot, k.t, best})
	}
	attrib := 0
	for _, o := range outs {
		if o.w != "" {
			attrib++
		}
	}
	fmt.Printf("=== %s OFFLINE PUR : %d dégâts 0xd2, %d kills, warp a=%.3e b=%.0f ===\n", m, len(dmgs), len(ks), a, b)
	fmt.Printf("  arme attribuee : %d/%d (%.0f%%) ; warp R2=%.4f\n", attrib, len(ks), 100*float64(attrib)/float64(len(ks)), r2)
	// breakdown arme par joueur (produit)
	perP := map[int]map[string]int{}
	for _, o := range outs {
		w := o.w
		if w == "" {
			w = "(non attribué)"
		}
		if perP[o.slot] == nil {
			perP[o.slot] = map[string]int{}
		}
		perP[o.slot][w]++
	}
	fmt.Printf("--- arme par kill, par joueur (slot) ---\n")
	for s := 0; s < 8; s++ {
		if perP[s] == nil {
			continue
		}
		var ws []string
		for w := range perP[s] {
			ws = append(ws, w)
		}
		sort.Slice(ws, func(i, j int) bool { return perP[s][ws[i]] > perP[s][ws[j]] })
		line := ""
		for _, w := range ws {
			line += fmt.Sprintf("%s:%d ", w, perP[s][w])
		}
		fmt.Printf("  slot%d : %s\n", s, line)
	}
	// VALIDATION vs live (capture dual-hook ground-truth, sans pont d'horloge)
	dd, e1 := os.ReadFile(liveDir + "/" + m + "_dmg.bin")
	kc, e2 := os.ReadFile(liveDir + "/" + m + "_kill.bin")
	if e1 != nil || e2 != nil {
		dd, e1 = os.ReadFile(liveDir + "/dmgcapture_run2.bin")
		kc, e2 = os.ReadFile(liveDir + "/killcapture.bin")
	}
	if e1 == nil && e2 == nil {
		// dégâts live : atk@0, fam@8, tsc=([20]<<32)|[16]
		type ld struct {
			atk int
			w   string
			tsc uint64
		}
		var lds []ld
		for o := 0; o+32 <= len(dd); o += 32 {
			at := idxD(binary.LittleEndian.Uint32(dd[o:]))
			if at < 0 {
				continue
			}
			tsc := uint64(binary.LittleEndian.Uint32(dd[o+20:]))<<32 | uint64(binary.LittleEndian.Uint32(dd[o+16:]))
			lds = append(lds, ld{at, h32[binary.LittleEndian.Uint32(dd[o+8:])], tsc})
		}
		// kills live : vic@0, kil@4, tsc=([12]<<32)|[8] ; dédup tueur+victime <1ms
		type lk struct {
			kil, vic int
			tsc      uint64
		}
		var lkAll []lk
		for o := 0; o+16 <= len(kc); o += 16 {
			vi := idxK(binary.LittleEndian.Uint32(kc[o:]))
			ki := idxK(binary.LittleEndian.Uint32(kc[o+4:]))
			if ki < 0 || vi < 0 {
				continue
			}
			tsc := uint64(binary.LittleEndian.Uint32(kc[o+12:]))<<32 | uint64(binary.LittleEndian.Uint32(kc[o+8:]))
			lkAll = append(lkAll, lk{ki, vi, tsc})
		}
		sort.Slice(lkAll, func(i, j int) bool { return lkAll[i].tsc < lkAll[j].tsc })
		var lks []lk
		for _, k := range lkAll {
			dup := false
			for j := len(lks) - 1; j >= 0 && k.tsc-lks[j].tsc < 3000000; j-- {
				if lks[j].kil == k.kil && lks[j].vic == k.vic {
					dup = true
					break
				}
			}
			if !dup {
				lks = append(lks, k)
			}
		}
		// arme vérité = dernier dégât du tueur (atk==kil) à tsc<=tsc_kill
		type lw struct {
			kil, vic int
			w        string
		}
		var lwk []lw
		for _, k := range lks {
			w := ""
			var bt uint64
			for _, d := range lds {
				if d.atk == k.kil && d.tsc <= k.tsc && d.tsc >= bt {
					bt = d.tsc
					w = d.w
				}
			}
			lwk = append(lwk, lw{k.kil, k.vic, w})
		}
		// (1) distribution par joueur (robuste, sans appariement)
		liveDist := map[int]map[string]int{}
		for _, l := range lwk {
			if l.w == "" {
				continue
			}
			if liveDist[l.kil] == nil {
				liveDist[l.kil] = map[string]int{}
			}
			liveDist[l.kil][l.w]++
		}
		offDist := map[int]map[string]int{}
		for _, o := range outs {
			if o.w == "" {
				continue
			}
			if offDist[o.slot] == nil {
				offDist[o.slot] = map[string]int{}
			}
			offDist[o.slot][o.w]++
		}
		dtot, dmatch := 0, 0
		for s := 0; s < 16; s++ {
			for w, c := range liveDist[s] {
				dtot += c
				if m2 := offDist[s][w]; m2 < c {
					dmatch += m2
				} else {
					dmatch += c
				}
			}
		}
		// (2) par (tueur,victime,rang)
		liveByKV := map[[2]int][]string{}
		for _, l := range lwk {
			liveByKV[[2]int{l.kil, l.vic}] = append(liveByKV[[2]int{l.kil, l.vic}], l.w)
		}
		offByKV := map[[2]int][]string{}
		{
			type ot struct {
				t int
				w string
			}
			tmp := map[[2]int][]ot{}
			for _, o := range outs {
				if o.w == "" {
					continue
				}
				tmp[[2]int{o.slot, o.vslot}] = append(tmp[[2]int{o.slot, o.vslot}], ot{o.t, o.w})
			}
			for kv, os_ := range tmp {
				sort.Slice(os_, func(i, j int) bool { return os_[i].t < os_[j].t })
				for _, o := range os_ {
					offByKV[kv] = append(offByKV[kv], o.w)
				}
			}
		}
		ptot, pmatch := 0, 0
		for kv, ll := range liveByKV {
			lo := offByKV[kv]
			n := len(ll)
			if len(lo) < n {
				n = len(lo)
			}
			for i := 0; i < n; i++ {
				ptot++
				if ll[i] == lo[i] {
					pmatch++
				}
			}
		}
		// (4) par (tueur, rang temporel) — sans victime (le plus propre si slot==idx)
		liveByK := map[int][]string{}
		for _, l := range lwk {
			if l.w != "" {
				liveByK[l.kil] = append(liveByK[l.kil], l.w) // lwk suit l'ordre tsc de lks
			}
		}
		offByK := map[int][]string{}
		{
			type ot struct {
				t int
				w string
			}
			tmp := map[int][]ot{}
			for _, o := range outs {
				if o.w == "" {
					continue
				}
				tmp[o.slot] = append(tmp[o.slot], ot{o.t, o.w})
			}
			for s, os_ := range tmp {
				sort.Slice(os_, func(i, j int) bool { return os_[i].t < os_[j].t })
				for _, o := range os_ {
					offByK[s] = append(offByK[s], o.w)
				}
			}
		}
		rtot, rmatch := 0, 0
		for s := 0; s < 16; s++ {
			ll := liveByK[s]
			lo := offByK[s]
			n := len(ll)
			if len(lo) < n {
				n = len(lo)
			}
			for i := 0; i < n; i++ {
				rtot++
				if ll[i] == lo[i] {
					rmatch++
				}
			}
		}
		// (3) distribution GLOBALE (sans identité joueur) = les armes matchent-elles ?
		gLive := map[string]int{}
		gOff := map[string]int{}
		for _, l := range lwk {
			if l.w != "" {
				gLive[l.w]++
			}
		}
		for _, o := range outs {
			if o.w != "" {
				gOff[o.w]++
			}
		}
		gtot, gmatch := 0, 0
		for w, c := range gLive {
			gtot += c
			if m2 := gOff[w]; m2 < c {
				gmatch += m2
			} else {
				gmatch += c
			}
		}
		// (5) PAR KILL via pont d'horloge tsc (métrique propre, = celle des 96% sur 000d5950)
		// warp offline packet_ts -> live tsc, dérivé en appariant (slot/idx, arme, ordre).
		offBy := map[string][]uint64{}
		for _, d := range dmgs {
			offBy[fmt.Sprintf("%d|%s", d.slot, d.fam)] = append(offBy[fmt.Sprintf("%d|%s", d.slot, d.fam)], d.ts)
		}
		liveBy := map[string][]uint64{}
		for _, d := range lds {
			liveBy[fmt.Sprintf("%d|%s", d.atk, d.w)] = append(liveBy[fmt.Sprintf("%d|%s", d.atk, d.w)], d.tsc)
		}
		var sX, sY, sXX, sXY, nA float64
		for key, ot := range offBy {
			lt := liveBy[key]
			if len(lt) == 0 {
				continue
			}
			sort.Slice(ot, func(i, j int) bool { return ot[i] < ot[j] })
			sort.Slice(lt, func(i, j int) bool { return lt[i] < lt[j] })
			n := len(ot)
			if len(lt) < n {
				n = len(lt)
			}
			for i := 0; i < n; i++ {
				x := float64(ot[i])
				y := float64(lt[i])
				sX += x
				sY += y
				sXX += x * x
				sXY += x * y
				nA++
			}
		}
		k5tot, k5match := 0, 0
		mism := map[string]int{}
		if nA > 2 {
			aL := (nA*sXY - sX*sY) / (nA*sXX - sX*sX)
			bL := (sY - aL*sX) / nA
			toTsc := func(tms int) float64 { ts := (float64(tms) - b) / a; return aL*ts + bL }
			toMs := func(tsc float64) float64 { return (tsc-bL)/aL*a + b } // tsc -> TimeMS offline (pour lisibilité)
			traced := 0
			for _, o := range outs {
				if o.w == "" {
					continue
				}
				tk := toTsc(o.t)
				truth := ""
				var bt float64
				for _, d := range lds {
					if d.atk == o.slot && float64(d.tsc) <= tk+1 && float64(d.tsc) >= bt {
						bt = float64(d.tsc)
						truth = d.w
					}
				}
				if truth == "" {
					continue
				}
				k5tot++
				if o.w == truth {
					k5match++
				} else {
					mism[fmt.Sprintf("offline=%s vs live=%s", o.w, truth)]++
					// TRACE détaillée des désaccords BR/MA40
					if traced < 5 && (o.w == "BR75" || o.w == "MA40 AR") && (truth == "BR75" || truth == "MA40 AR") {
						traced++
						fmt.Printf("  --- KILL tueur slot%d : offline=%s vs live=%s (TimeMS=%d) ---\n", o.slot, o.w, truth, o.t)
						fmt.Printf("      0xd2 OFFLINE du tueur (warpé en TimeMS) autour du kill:\n")
						type rc struct {
							t float64
							w string
						}
						var oc []rc
						for _, d := range dmgs {
							if d.slot == o.slot && warp(d.ts) >= float64(o.t)-2500 && warp(d.ts) <= float64(o.t)+2500 {
								oc = append(oc, rc{warp(d.ts), d.fam})
							}
						}
						sort.Slice(oc, func(i, j int) bool { return oc[i].t < oc[j].t })
						for _, c := range oc {
							mk := " "
							if c.t <= float64(o.t) {
								mk = "<"
							}
							fmt.Printf("        %s t=%.0f %s\n", mk, c.t, c.w)
						}
						fmt.Printf("      LIVE du tueur (tsc->TimeMS) autour du kill:\n")
						var lc []rc
						for _, d := range lds {
							if d.atk == o.slot && toMs(float64(d.tsc)) >= float64(o.t)-2500 && toMs(float64(d.tsc)) <= float64(o.t)+2500 {
								lc = append(lc, rc{toMs(float64(d.tsc)), d.w})
							}
						}
						sort.Slice(lc, func(i, j int) bool { return lc[i].t < lc[j].t })
						for _, c := range lc {
							mk := " "
							if float64(c.t) <= toMs(tk) {
								mk = "<"
							}
							fmt.Printf("        %s t=%.0f %s\n", mk, c.t, c.w)
						}
					}
				}
			}
			fmt.Printf("  [pont] anchors=%d ; aL=%.4g bL=%.4g\n", int(nA), aL, bL)
			type mc struct {
				s string
				c int
			}
			var ms []mc
			for s, c := range mism {
				ms = append(ms, mc{s, c})
			}
			sort.Slice(ms, func(i, j int) bool { return ms[i].c > ms[j].c })
			line := ""
			for i, x := range ms {
				if i >= 8 {
					break
				}
				line += fmt.Sprintf("[%s ×%d] ", x.s, x.c)
			}
			fmt.Printf("  [erreurs top] %s\n", line)
		}
		k5pct := 0.0
		if k5tot > 0 {
			k5pct = 100 * float64(k5match) / float64(k5tot)
		}
		// (7) mapping slot->idx par assignation optimale (Hungarian brute-force 8!) sur les
		// histogrammes de dégât complets, puis per-kill avec le bon mapping.
		offH := make([]map[string]float64, 8)
		liveH := make([]map[string]float64, 8)
		for s := 0; s < 8; s++ {
			offH[s] = map[string]float64{}
			liveH[s] = map[string]float64{}
		}
		for _, d := range dmgs {
			if d.slot >= 0 && d.slot < 8 {
				offH[d.slot][d.fam]++
			}
		}
		for _, d := range lds {
			if d.atk >= 0 && d.atk < 8 {
				liveH[d.atk][d.w]++
			}
		}
		cos := func(a, b map[string]float64) float64 {
			var dot, na, nb float64
			for k, v := range a {
				dot += v * b[k]
				na += v * v
			}
			for _, v := range b {
				nb += v * v
			}
			if na == 0 || nb == 0 {
				return 0
			}
			return dot / (math.Sqrt(na) * math.Sqrt(nb))
		}
		var cm [8][8]float64
		for s := 0; s < 8; s++ {
			for i := 0; i < 8; i++ {
				cm[s][i] = cos(offH[s], liveH[i])
			}
		}
		bestPerm := []int{0, 1, 2, 3, 4, 5, 6, 7}
		bestScore := -1.0
		perm := []int{0, 1, 2, 3, 4, 5, 6, 7}
		var rec func(k int)
		rec = func(k int) {
			if k == 8 {
				sc := 0.0
				for s := 0; s < 8; s++ {
					sc += cm[s][perm[s]]
				}
				if sc > bestScore {
					bestScore = sc
					bestPerm = append([]int{}, perm...)
				}
				return
			}
			for i := k; i < 8; i++ {
				perm[k], perm[i] = perm[i], perm[k]
				rec(k + 1)
				perm[k], perm[i] = perm[i], perm[k]
			}
		}
		rec(0)
		fmt.Printf("  [mapping optimal slot->idx] %v (score %.2f)\n", bestPerm, bestScore)
		k7tot, k7match := 0, 0
		if nA > 2 {
			aL := (nA*sXY - sX*sY) / (nA*sXX - sX*sX)
			bL := (sY - aL*sX) / nA
			toTsc := func(tms int) float64 { ts := (float64(tms) - b) / a; return aL*ts + bL }
			for _, o := range outs {
				if o.w == "" || o.slot < 0 || o.slot >= 8 {
					continue
				}
				want := bestPerm[o.slot]
				tk := toTsc(o.t)
				truth := ""
				var bt float64
				for _, d := range lds {
					if d.atk == want && float64(d.tsc) <= tk+1 && float64(d.tsc) >= bt {
						bt = float64(d.tsc)
						truth = d.w
					}
				}
				if truth == "" {
					continue
				}
				k7tot++
				if o.w == truth {
					k7match++
				}
			}
		}
		k7pct := 0.0
		if k7tot > 0 {
			k7pct = 100 * float64(k7match) / float64(k7tot)
		}
		fmt.Printf("  PAR-KILL (mapping optimal) : %d/%d (%.0f%%)\n", k7match, k7tot, k7pct)
		fmt.Printf("  VALIDATION live %d kills (dedup) | PAR-KILL(pont tsc) %d/%d (%.0f%%) | GLOBALE %d/%d (%.0f%%) | (tueur,rang) %d/%d (%.0f%%)\n",
			len(lks), k5match, k5tot, k5pct, gmatch, gtot, 100*float64(gmatch)/float64(gtot), rmatch, rtot, 100*float64(rmatch)/float64(rtot))
		fmt.Println("--- LIVE par idx ---")
		for s := 0; s < 8; s++ {
			if liveDist[s] == nil {
				continue
			}
			line := ""
			for w, c := range liveDist[s] {
				line += fmt.Sprintf("%s:%d ", w, c)
			}
			fmt.Printf("  idx%d : %s\n", s, line)
		}
	}
}
