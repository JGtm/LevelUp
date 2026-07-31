// tmp_keyframeloadout — THROWAWAY : décoder les LOADOUTS aux KEYFRAMES (paquets type-2).
// Hypothèse : chaque keyframe snapshot l'arme tenue de chaque joueur. -> timeline possession dense.
// On scanne les weapon-id (high32∈cat) dans les paquets type-2, on cherche un champ slot/player couvrant
// les 8 joueurs (1 arme par joueur par keyframe), puis on valide held-weapon@kill vs 519 records de dégât.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)
const maxMatchMs = 600000
const variantSuffix = uint32(0x42c9679f)
const deserStartBit = 36

var h32 = map[uint32]string{}

func build() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
}
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
func bitsAt(p []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		q := bp + i
		if q>>3 >= len(p) || q < 0 {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((p[q>>3]>>uint(7-(q&7)))&1)
	}
	return v
}
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

type pkt struct {
	typ     uint16
	payload []byte
	tms     int
}

func allPkts() []pkt {
	var out []pkt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			size := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if size <= 0 || off+16+size > len(d) {
				break
			}
			tms := -1
			if ts >= t0Us {
				tms = int((ts - t0Us) / 1000)
			}
			out = append(out, pkt{typ, d[off+16 : off+16+size], tms})
			off += 16 + size
		}
	}
	return out
}

type dmg struct {
	tms int
	fam string
}

func damageRecs(pkts []pkt) []dmg {
	var out []dmg
	for _, p := range pkts {
		if p.typ != 0 || p.tms < 0 || len(p.payload) == 0 || p.payload[0] != 0xd2 {
			continue
		}
		br := filmdec.NewBitReader(p.payload)
		br.Skip(deserStartBit)
		if !br.ReadBit() {
			br.ReadBits(2)
		}
		br.ReadBits(5)
		gid := uint32(br.ReadBits(32))
		if uint32(br.ReadBits(32)) != variantSuffix {
			continue
		}
		if fam, ok := h32[gid]; ok {
			out = append(out, dmg{p.tms, fam})
		}
	}
	return out
}

type woc struct {
	pl  []byte
	bp  int
	fam string
	tms int
	pi  int // packet index (keyframe id)
}

func main() {
	build()
	pkts := allPkts()
	dmgs := damageRecs(pkts)

	// keyframes = paquets type-2. Lister + compter weapon-id par keyframe.
	var kfPkts []int
	for i, p := range pkts {
		if p.typ == 2 {
			kfPkts = append(kfPkts, i)
		}
	}
	fmt.Printf("=== %d paquets type-2 (keyframes) ; %d records de dégât ===\n", len(kfPkts), len(dmgs))

	// occurrences weapon-id dans les keyframes
	var occs []woc
	perKF := map[int]int{}
	for _, idx := range kfPkts {
		p := pkts[idx]
		pl := p.payload
		total := len(pl) * 8
		for bp := 0; bp+64 <= total; bp++ {
			hi := uint32(bitsAt(pl, bp, 32))
			fam, ok := h32[hi]
			if !ok {
				continue
			}
			occs = append(occs, woc{pl, bp, fam, p.tms, idx})
			perKF[idx]++
		}
	}
	fmt.Printf("=== %d occurrences weapon-id dans les keyframes ===\n", len(occs))
	// distribution armes/keyframe
	dist := map[int]int{}
	for _, c := range perKF {
		dist[c]++
	}
	fmt.Printf("  armes par keyframe : ")
	var keys []int
	for k := range dist {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		fmt.Printf("%d→%dkf ", k, dist[k])
	}
	fmt.Println()

	// chercher un champ slot/player (w=3,4,5) couvrant 0-7, ~1 par joueur par keyframe.
	fmt.Println("\n=== champ slot/player sur les occurrences keyframe (w=3..5, 0-7 uniformes) ===")
	for w := 3; w <= 5; w++ {
		best := []string{}
		for o := -80; o <= 200; o++ {
			if o >= 0 && o < 32 {
				continue
			}
			cnt := map[int]int{}
			for _, c := range occs {
				cnt[int(bitsAt(c.pl, c.bp+o, w))]++
			}
			d07, over, mn, mx := 0, 0, 1<<30, 0
			for v := 0; v <= 7; v++ {
				if cnt[v] > 0 {
					d07++
				}
				if cnt[v] < mn {
					mn = cnt[v]
				}
				if cnt[v] > mx {
					mx = cnt[v]
				}
			}
			for v := 8; v < (1 << w); v++ {
				over += cnt[v]
			}
			if d07 == 8 && mx > 0 && float64(mn)/float64(mx) >= 0.3 && over < len(occs)/10 {
				best = append(best, fmt.Sprintf("o=%+d ratio=%.2f over=%d", o, float64(mn)/float64(mx), over))
			}
		}
		fmt.Printf("  w=%d : %d candidats : %v\n", w, len(best), firstN(best, 8))
	}

	// VALIDATION held-weapon : pour chaque kill, l'arme du tueur au keyframe <= T (slot via candidat offset),
	// corroborée par un record de dégât même famille [T-1500,T+200] ? On teste un set d'offsets candidats.
	feed := killFeed()
	fmt.Println("\n=== VALIDATION held-weapon keyframe vs 519 records (offsets candidats) ===")
	for _, off := range []int{-4, -9, -13, -20, 64, 72, 96} {
		// pour chaque kill : dernier occ (slot==killer via off) à tms<=T
		var nHeld, nCorrob int
		for _, k := range feed {
			if k.killerPi < 0 {
				continue
			}
			bi, bt := -1, -1
			for i := range occs {
				if occs[i].tms < 0 || occs[i].tms > k.t {
					continue
				}
				if int(bitsAt(occs[i].pl, occs[i].bp+off, 5)) != k.killerPi {
					continue
				}
				if occs[i].tms > bt {
					bt, bi = occs[i].tms, i
				}
			}
			if bi < 0 {
				continue
			}
			nHeld++
			for _, dr := range dmgs {
				if dr.fam == occs[bi].fam && dr.tms >= k.t-1500 && dr.tms <= k.t+200 {
					nCorrob++
					break
				}
			}
		}
		corr := 0.0
		if nHeld > 0 {
			corr = 100 * float64(nCorrob) / float64(nHeld)
		}
		fmt.Printf("  slot@bp%+d : held=%d/%d kills ; corroborés=%d (%.0f%%)\n", off, nHeld, len(feed), nCorrob, corr)
	}
}

func firstN(s []string, n int) []string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

var xuidToPi = map[uint64]int{
	2535467794760703: 0, 2535437947245250: 1, 2533274823110022: 2, 2533274980284321: 3,
	2533274815845110: 4, 2535444178793711: 5, 2533274882097883: 6, 2533274826120416: 7,
}

type kfRow struct {
	killerPi int
	t        int
}

func killFeed() []kfRow {
	raw, _ := os.ReadFile(cache + "/chunk_27.bin")
	events, _ := analysis.ParseHighlightEvents(raw, 0)
	type ev struct {
		x uint64
		t int
	}
	var kills, deaths []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.XUID, e.TimeMS})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	usedD := make([]bool, len(deaths))
	var feed []kfRow
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if usedD[i] || d.x == k.x {
				continue
			}
			if dt := abs(k.t - d.t); dt < bd {
				bd, best = dt, i
			}
		}
		if best < 0 {
			continue
		}
		usedD[best] = true
		kp, ok := xuidToPi[k.x]
		if !ok {
			kp = -1
		}
		feed = append(feed, kfRow{kp, k.t})
	}
	return feed
}
