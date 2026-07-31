// tmp_hitevent — THROWAWAY : tester l'hypothèse UNIFIÉE — le marqueur 0x534/0x535 (hit/miss) est l'event
// de DÉGÂT GÉNÉRAL (mêlée ET arme à feu), avec type@anchor+76 = classe d'arme et player_index@anchor+23
// (qui couvre les 8 joueurs en mêlée). On tabule TOUS les types, on cherche l'arme par type, et on valide
// player_index@anchor+23 contre le kill feed + 519 records de dégât.
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

func type0() []pkt {
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
			if ts >= t0Us {
				tms := int((ts - t0Us) / 1000)
				if tms >= 0 && tms <= maxMatchMs {
					out = append(out, pkt{typ, d[off+16 : off+16+size], tms})
				}
			}
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
		if p.typ != 0 || len(p.payload) == 0 || p.payload[0] != 0xd2 {
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

// hit event décodé : marqueur 0x534/0x535, type@anchor+76, pidx@anchor+23, weapon recherché.
type hit struct {
	tms  int
	typ  uint8
	pidx int
	fam  string // arme trouvée (si une famille catalogue est près de l'anchor)
	woff int    // offset où l'arme a été trouvée (relatif anchor)
	hitb bool
}

func main() {
	build()
	pkts := type0()
	dmgs := damageRecs(pkts)
	fmt.Printf("=== %d paquets ; %d records de dégât ===\n", len(pkts), len(dmgs))

	// 1) scanner tous les marqueurs 0x534/0x535 ; tabuler type@anchor+76 ; chercher l'arme.
	var hits []hit
	typeWeapon := map[uint8]map[int]int{} // type -> woff -> count (où on trouve une famille)
	typeCount := map[uint8]int{}
	for _, p := range pkts {
		pl := p.payload
		total := len(pl) * 8
		for bp := 0; bp+200 < total; bp++ {
			m := bitsAt(pl, bp, 11)
			if m != 0x534 && m != 0x535 {
				continue
			}
			anchor := bp + 3
			typ := uint8(bitsAt(pl, anchor+76, 8))
			typeCount[typ]++
			// chercher une famille catalogue dans une fenêtre autour de l'anchor (offset 60..130)
			woff, fam := -1, ""
			for o := 60; o <= 140; o++ {
				hi := uint32(bitsAt(pl, anchor+o, 32))
				if name, ok := h32[hi]; ok {
					woff, fam = o, name
					if typeWeapon[typ] == nil {
						typeWeapon[typ] = map[int]int{}
					}
					typeWeapon[typ][o]++
					break
				}
			}
			hits = append(hits, hit{p.tms, typ, int(bitsAt(pl, anchor+23, 5)), fam, woff, m == 0x534})
		}
	}
	fmt.Printf("\n=== %d marqueurs 0x534/0x535 au total ===\n", len(hits))

	// 2) distribution des types (top) + combien ont une arme trouvée
	type tc struct {
		t uint8
		c int
	}
	var tcs []tc
	for t, c := range typeCount {
		tcs = append(tcs, tc{t, c})
	}
	sort.Slice(tcs, func(i, j int) bool { return tcs[i].c > tcs[j].c })
	fmt.Println("=== type@anchor+76 (top 20) : count, offset d'arme dominant, famille ===")
	for i, x := range tcs {
		if i >= 20 {
			break
		}
		woffStr := "(aucune arme)"
		if wm := typeWeapon[x.t]; wm != nil {
			bo, bc := -1, 0
			for o, c := range wm {
				if c > bc {
					bc, bo = c, o
				}
			}
			woffStr = fmt.Sprintf("arme@+%d x%d", bo, bc)
		}
		fmt.Printf("  type=0x%02x x%-5d %s\n", x.t, x.c, woffStr)
	}

	// 3) player_index@anchor+23 : distribution globale (doit couvrir 0-7)
	pd := map[int]int{}
	for _, h := range hits {
		pd[h.pidx]++
	}
	fmt.Println("\n=== player_index@anchor+23 (tous hits 0x534/0x535) ===")
	for v := 0; v < 8; v++ {
		fmt.Printf("  pi%d x%d\n", v, pd[v])
	}

	// 4) hits AVEC une famille d'arme à feu (≠ marteau/épée) : distribution famille + validation
	gunFams := map[string]int{}
	var gunHits []hit
	for _, h := range hits {
		if h.fam == "" {
			continue
		}
		gunFams[h.fam]++
		gunHits = append(gunHits, h)
	}
	fmt.Printf("\n=== hits avec famille trouvée : %d ; familles : \n", len(gunHits))
	type fc struct {
		n string
		c int
	}
	var fcs []fc
	for n, c := range gunFams {
		fcs = append(fcs, fc{n, c})
	}
	sort.Slice(fcs, func(i, j int) bool { return fcs[i].c > fcs[j].c })
	for i, f := range fcs {
		if i >= 20 {
			break
		}
		fmt.Printf("   %-24s x%d\n", f.n, f.c)
	}

	// 5) VALIDATION : par kill (tueur K, T), hit de pidx==K à ±W ; famille corroborée par record dégât ?
	feed := killFeed()
	fmt.Println("\n=== VALIDATION hit-event (player_index@anchor+23) vs kill feed ===")
	for _, W := range []int{500, 1500} {
		var nEvt, nCorrob, nGun int
		for _, k := range feed {
			if k.killerPi < 0 {
				continue
			}
			bi, bd := -1, 1<<30
			for i := range gunHits {
				if gunHits[i].pidx != k.killerPi {
					continue
				}
				if d := abs(gunHits[i].tms - k.t); d < bd {
					bd, bi = d, i
				}
			}
			if bi < 0 || bd > W {
				continue
			}
			nEvt++
			nGun++
			for _, dr := range dmgs {
				if dr.fam == gunHits[bi].fam && dr.tms >= k.t-1500 && dr.tms <= k.t+200 {
					nCorrob++
					break
				}
			}
		}
		corr := 0.0
		if nGun > 0 {
			corr = 100 * float64(nCorrob) / float64(nGun)
		}
		fmt.Printf("  ±%4dms : %d/%d kills ont un hit-event tueur (avec arme) ; corroborés=%d (%.0f%%)\n",
			W, nEvt, len(feed), nCorrob, corr)
	}
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
