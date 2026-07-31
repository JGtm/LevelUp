// tmp_offgen — pipeline arme-par-kill 100% OFFLINE GÉNÉRIQUE (sans CE/Ghidra).
// Décode dégâts 0xd2 (slot R5>>1 + arme + temps) + kill-feed (xuid+gamertag) + roster type-8
// (bit-scan LE des xuids du feed -> ordre = slots). Attribue. Usage: tmp_offgen <matchID>
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/weaponv3"
	"os"
	"sort"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
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
func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }
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

type dmg struct {
	slot int
	fam  string
	tms  int
}
type kf struct {
	killer, victim uint64
	t              int
}

func main() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
	m := os.Args[1]
	cache := root + "/" + m
	// dégâts
	type rr struct {
		slot int
		fam  string
		ts   uint64
	}
	var raw []rr
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(d) == 0 {
			continue
		}
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
			r5 := int(bitsAt(pl, 36, 5))
			bp := 41
			if bitsAt(pl, bp, 1) == 1 {
				bp++
			} else {
				bp += 3
			}
			fam32 := uint32(bitsAt(pl, bp, 32))
			low := uint32(bitsAt(pl, bp+32, 32))
			if nm, ok := h32[fam32]; ok && low == sfx {
				raw = append(raw, rr{r5 >> 1, nm, ts})
			}
		}
	}
	if len(raw) == 0 {
		fmt.Println("aucun 0xd2")
		return
	}
	sort.Slice(raw, func(i, j int) bool { return raw[i].ts < raw[j].ts })
	t0 := raw[0].ts
	var dmgs []dmg
	for _, r := range raw {
		dmgs = append(dmgs, dmg{r.slot, r.fam, int((r.ts - t0) / 1000)})
	}
	// scan MÊLÉE (même t0 ; PI = slot) — comble les frags non-firearm
	type mh struct {
		pi  int
		fam string
		tms int
	}
	var melees []mh
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(d) == 0 {
			continue
		}
		type seg struct {
			start int
			ts    uint64
		}
		var segs []seg
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			if typ == 0 {
				segs = append(segs, seg{off + 16, ts})
			}
			off += 16 + sz
		}
		est := func(bp int) float64 {
			var tt uint64
			for _, s := range segs {
				if s.start <= bp {
					tt = s.ts
				} else {
					break
				}
			}
			return float64(int64(tt)-int64(t0)) / 1000.0
		}
		for _, h := range weaponv3.ScanMeleeHits(d, est) {
			if !weaponv3.MeleeHitLethal(h.HitType) {
				continue
			}
			fam := h32[uint32(h.WeaponID>>32)]
			if fam == "" {
				fam = "Mêlée"
			}
			melees = append(melees, mh{h.PI, fam, h.TimeMS})
		}
	}
	// kill-feed + gamertags
	var kills, deaths []struct {
		x uint64
		t int
	}
	gt := map[uint64]string{}
	for ch := 41; ch >= 18; ch-- {
		b := mustRead(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(b) == 0 {
			continue
		}
		ev, _ := analysis.ParseHighlightEvents(b, 0)
		nk := 0
		var kk, dd []struct {
			x uint64
			t int
		}
		for _, e := range ev {
			if e.Gamertag != "" {
				gt[e.XUID] = e.Gamertag
			}
			switch e.EventType {
			case analysis.EventTypeKill:
				kk = append(kk, struct {
					x uint64
					t int
				}{e.XUID, e.TimeMS})
				nk++
			case analysis.EventTypeDeath:
				dd = append(dd, struct {
					x uint64
					t int
				}{e.XUID, e.TimeMS})
			}
		}
		if nk > len(kills) {
			kills, deaths = kk, dd
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	used := make([]bool, len(deaths))
	var feed []kf
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
		if best >= 0 {
			used[best] = true
			feed = append(feed, kf{k.x, deaths[best].x, k.t})
		}
	}
	// roster slot->xuid via type-8 (bit-scan LE, ordre = slots)
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
	firstBit := map[uint64]int{}
	for bp := 0; bp <= len(t8)*8-64; bp++ {
		if v := u64LE(t8, bp); want[v] {
			if _, ok := firstBit[v]; !ok {
				firstBit[v] = bp
			}
		}
	}
	type xb struct {
		x  uint64
		bp int
	}
	var xbs []xb
	for x, bp := range firstBit {
		xbs = append(xbs, xb{x, bp})
	}
	sort.Slice(xbs, func(i, j int) bool { return xbs[i].bp < xbs[j].bp })
	slotOf := map[uint64]int{}
	for s, e := range xbs {
		slotOf[e.x] = s
	}
	fmt.Printf("=== %s : %d dégâts ; %d kills ; roster type-8 = %d/%d joueurs localisés ===\n", m, len(dmgs), len(feed), len(xbs), len(want))
	// offset scan (roster connu) : couverture max
	cov := func(B int) int {
		c := 0
		for _, k := range feed {
			s, ok := slotOf[k.killer]
			if !ok {
				continue
			}
			bd := 400
			for _, d := range dmgs {
				if d.slot != s {
					continue
				}
				dd := d.tms - (k.t - B)
				if dd < 0 {
					dd = -dd
				}
				if dd < 6000 && dd < bd {
					bd = dd
					c++
					break
				}
			}
		}
		return c
	}
	bestB, bestC := 0, -1
	for B := -10000; B <= 60000; B += 200 {
		if c := cov(B); c > bestC {
			bestC, bestB = c, B
		}
	}
	// attribution + breakdown
	att, fA := 0, 0
	mA := 0
	_ = mA
	byG := map[string]map[string]int{}
	kc := map[string]int{}
	for _, k := range feed {
		s, ok := slotOf[k.killer]
		g := gt[k.killer]
		if g == "" {
			g = fmt.Sprintf("xuid:%d", k.killer)
		}
		kc[g]++
		if byG[g] == nil {
			byG[g] = map[string]int{}
		}
		fam := "?"
		if ok {
			bd := 1500
			for _, d := range dmgs {
				if d.slot != s {
					continue
				}
				dd := d.tms - (k.t - bestB)
				if dd < 0 {
					dd = -dd
				}
				if dd < bd {
					bd, fam = dd, d.fam
				}
			}
			if fam != "?" {
				fA++
			}
		}
		_ = melees // mêlée désactivée : le scanner détecte des SWINGS, pas des kills (réfuté vs live 000d5950)
		if fam != "?" {
			att++
		}
		byG[g][fam]++
	}
	fmt.Printf("offset=%dms ; COUVERTURE = %d/%d (%.0f%%)\n", bestB, att, len(feed), 100*float64(att)/float64(len(feed)))
	// DIAG slot N : ses kills (temps) vs ses ticks de dégât (temps+arme) pour voir l'alignement réel
	if len(os.Args) > 2 {
		var sl int
		fmt.Sscanf(os.Args[2], "%d", &sl)
		fmt.Printf("\n=== DIAG slot %d : KILLS (temps) ===\n", sl)
		for _, k := range feed {
			if s, ok := slotOf[k.killer]; ok && s == sl {
				fmt.Printf("  kill t=%6.1fs\n", float64(k.t)/1000)
			}
		}
		fmt.Printf("=== slot %d : TICKS dégât (temps, arme) ===\n", sl)
		for _, d := range dmgs {
			if d.slot == sl {
				fmt.Printf("  tick t=%6.1fs  %s\n", float64(d.tms)/1000, d.fam)
			}
		}
		return
	}
	// === TEST EQUIPE : les attribués sont-ils biaisés vers une équipe (tueur ou victime) ? ===
	enemy := map[uint64]map[uint64]int{}
	for _, k := range feed {
		if enemy[k.killer] == nil {
			enemy[k.killer] = map[uint64]int{}
		}
		if enemy[k.victim] == nil {
			enemy[k.victim] = map[uint64]int{}
		}
		enemy[k.killer][k.victim]++
		enemy[k.victim][k.killer]++
	}
	team := map[uint64]int{}
	var seed uint64
	for x := range enemy {
		seed = x
		break
	}
	team[seed] = 0
	q := []uint64{seed}
	for len(q) > 0 {
		p := q[0]
		q = q[1:]
		for e := range enemy[p] {
			if _, ok := team[e]; !ok {
				team[e] = 1 - team[p]
				q = append(q, e)
			}
		}
	}
	type tc struct{ att, non int }
	byKT := map[int]*tc{0: {}, 1: {}}
	byVT := map[int]*tc{0: {}, 1: {}}
	for _, k := range feed {
		s, ok := slotOf[k.killer]
		fam := "?"
		if ok {
			bd := 1500
			for _, d := range dmgs {
				if d.slot != s {
					continue
				}
				dd := d.tms - (k.t - bestB)
				if dd < 0 {
					dd = -dd
				}
				if dd < bd {
					bd, fam = dd, d.fam
				}
			}
		}
		kt := team[k.killer]
		vt := team[k.victim]
		if fam != "?" {
			byKT[kt].att++
			byVT[vt].att++
		} else {
			byKT[kt].non++
			byVT[vt].non++
		}
	}
	// DIAG dérive : pour chaque kill, écart (tick - kill) du tick le plus proche du tueur (tout offset)
	fmt.Printf("\n=== DIAG DÉRIVE TEMPS (kill_time, écart au tick le plus proche du tueur) ===\n")
	type kd struct{ kt, delta int }
	var kds []kd
	for _, k := range feed {
		s, ok := slotOf[k.killer]
		if !ok {
			continue
		}
		bd, bdelta, found := 1<<30, 0, false
		for _, d := range dmgs {
			if d.slot != s {
				continue
			}
			dd := d.tms - k.t
			ad := dd
			if ad < 0 {
				ad = -ad
			}
			if ad < bd {
				bd, bdelta, found = ad, dd, true
			}
		}
		if found {
			kds = append(kds, kd{k.t, bdelta})
		}
	}
	sort.Slice(kds, func(i, j int) bool { return kds[i].kt < kds[j].kt })
	for i, e := range kds {
		if i%5 == 0 {
			fmt.Printf("  kill t=%6.1fs  écart=%+6.1fs\n", float64(e.kt)/1000, float64(e.delta)/1000)
		}
	}
	fmt.Printf("\n=== TEST BIAIS EQUIPE ===\n")
	for t := 0; t < 2; t++ {
		fmt.Printf("  équipe %c comme TUEUR  : attribués=%d non=%d\n", 'A'+byte(t), byKT[t].att, byKT[t].non)
	}
	for t := 0; t < 2; t++ {
		fmt.Printf("  équipe %c comme VICTIME : attribués=%d non=%d\n", 'A'+byte(t), byVT[t].att, byVT[t].non)
	}
	fmt.Println("=== breakdown armes par joueur (offline) ===")
	var gs []string
	for g := range byG {
		gs = append(gs, g)
	}
	sort.Slice(gs, func(i, j int) bool { return kc[gs[i]] > kc[gs[j]] })
	for _, g := range gs {
		var fs []string
		for f := range byG[g] {
			fs = append(fs, f)
		}
		sort.Slice(fs, func(i, j int) bool { return byG[g][fs[i]] > byG[g][fs[j]] })
		line := ""
		for _, f := range fs {
			line += fmt.Sprintf("%s:%d ", f, byG[g][f])
		}
		fmt.Printf("  %-18s (%2d) : %s\n", g, kc[g], line)
	}
}
