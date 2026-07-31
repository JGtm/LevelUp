package main

// wf_marker_window — élargit la fenêtre temporelle autour de chaque kill et
// cherche les littéraux high-32 d'arme dans TOUS les paquets type-0 dont le ts
// tombe dans [ts_cible - W .. ts_cible + W]. Pour chaque hit on note :
//   - Δms du paquet au tick,
//   - les bits voisins (±40 bits) autour du littéral, pour repérer un encodage
//     pi (5 bits) avant/après (H1),
//   - le nombre de paquets dans la fenêtre et combien portent une arme.
//
// But : comprendre la densité/dispersion des armes près d'un kill, et tester
// H1 (killer_pi ... arme ... victim_pi à courte distance).

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

const t0 = uint64(4537898226)
const usPerMs = 1000

var piToXuid = map[int]uint64{
	0: 2535467794760703, 1: 2535437947245250, 2: 2533274823110022, 3: 2533274980284321,
	4: 2533274815845110, 5: 2535444178793711, 6: 2533274882097883, 7: 2533274826120416,
}
var xuidToPi = func() map[uint64]int {
	m := map[uint64]int{}
	for pi, x := range piToXuid {
		m[x] = pi
	}
	return m
}()

func piLabel(pi int) string {
	if pi < 0 {
		return "?"
	}
	if pi == 2 {
		return "JGtm"
	}
	return fmt.Sprintf("pi%d", pi)
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

type packet struct {
	chunk int
	typ   uint16
	ts    uint64
	data  []byte
}

func parsePackets(chunk int, d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, packet{chunk, typ, ts, d[off+16 : off+16+sz]})
		off += 16 + sz
	}
	return out
}

func bitAt(d []byte, p int) uint32 {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return uint32((d[p>>3] >> uint(7-(p&7))) & 1)
}
func bitsU32(d []byte, bit int) uint32 {
	var v uint32
	for i := 0; i < 32; i++ {
		v = (v << 1) | bitAt(d, bit+i)
	}
	return v
}
func bitsN(d []byte, bit, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(d, bit+i)
	}
	return v
}

func wmap() map[uint32]string {
	m := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		if c, ok := analysis.WeaponFusionMap[n]; ok {
			n = c
		}
		m[uint32(id>>32)] = n
	}
	return m
}

type kill struct {
	killerPi, victimPi int
	timeMS             int
}

func loadKills() []kill {
	d := inflate(fmt.Sprintf("%s/chunk_27.bin", cache))
	var best []analysis.HighlightEvent
	bestN := -1
	for _, ver := range []int{34, 41, 42} {
		ev, err := analysis.ParseHighlightEvents(d, ver)
		if err != nil {
			continue
		}
		n := 0
		for _, e := range ev {
			if e.EventType == analysis.EventTypeKill || e.EventType == analysis.EventTypeDeath {
				n++
			}
		}
		if n > bestN {
			bestN, best = n, ev
		}
	}
	var kills, deaths []analysis.HighlightEvent
	for _, e := range best {
		if e.EventType == analysis.EventTypeKill {
			kills = append(kills, e)
		} else if e.EventType == analysis.EventTypeDeath {
			deaths = append(deaths, e)
		}
	}
	var out []kill
	used := make([]bool, len(deaths))
	for _, k := range kills {
		bj, bd := -1, 6
		for j, dh := range deaths {
			if used[j] {
				continue
			}
			dd := k.TimeMS - dh.TimeMS
			if dd < 0 {
				dd = -dd
			}
			if dd <= 5 && dd < bd {
				bd, bj = dd, j
			}
		}
		ku := kill{killerPi: -1, victimPi: -1, timeMS: k.TimeMS}
		if pi, ok := xuidToPi[k.XUID]; ok {
			ku.killerPi = pi
		}
		if bj >= 0 {
			used[bj] = true
			if pi, ok := xuidToPi[deaths[bj].XUID]; ok {
				ku.victimPi = pi
			}
		}
		out = append(out, ku)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].timeMS < out[j].timeMS })
	return out
}

func main() {
	var type0 []packet
	for i := 0; i <= 27; i++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, i))
		if len(d) == 0 {
			continue
		}
		for _, p := range parsePackets(i, d) {
			if p.typ == 0 {
				type0 = append(type0, p)
			}
		}
	}
	sort.Slice(type0, func(i, j int) bool { return type0[i].ts < type0[j].ts })
	wm := wmap()

	kills := loadKills()
	var sample []kill
	for _, k := range kills {
		if k.killerPi >= 0 && k.victimPi >= 0 {
			sample = append(sample, k)
		}
		if len(sample) >= 14 {
			break
		}
	}

	const W = 600000 // ±600ms en us
	fmt.Printf("=== fenêtre ±%dms autour de chaque kill ===\n", W/1000)
	for ki, k := range sample {
		ts := int64(t0) + int64(k.timeMS)*usPerMs
		lo, hi := uint64(ts-W), uint64(ts+W)
		// collecte des paquets dans la fenêtre.
		nPkt := 0
		nWithW := 0
		type whit struct {
			dMs  float64
			bit  int
			name string
			// 5 bits juste avant et juste après le littéral
			before5 uint32
			after5  uint32
			beforeB byte
			afterB  byte
		}
		var hits []whit
		for i := range type0 {
			p := &type0[i]
			if p.ts < lo || p.ts > hi {
				continue
			}
			nPkt++
			tot := len(p.data) * 8
			found := false
			for bp := 0; bp+32 <= tot; bp++ {
				if n, ok := wm[bitsU32(p.data, bp)]; ok {
					found = true
					h := whit{
						dMs:     float64(int64(p.ts)-ts) / 1000.0,
						bit:     bp,
						name:    n,
						before5: bitsN(p.data, bp-5, 5),
						after5:  bitsN(p.data, bp+32, 5),
					}
					if bp >= 8 {
						h.beforeB = byte(bitsN(p.data, bp-8, 8))
					}
					h.afterB = byte(bitsN(p.data, bp+32, 8))
					hits = append(hits, h)
				}
			}
			if found {
				nWithW++
			}
		}
		sort.Slice(hits, func(i, j int) bool {
			if hits[i].dMs != hits[j].dMs {
				return absf(hits[i].dMs) < absf(hits[j].dMs)
			}
			return hits[i].bit < hits[j].bit
		})
		fmt.Printf("\n--- kill[%02d] t=%dms killer=%s victim=%s | %d pkts fenêtre, %d avec arme, %d littéraux ---\n",
			ki, k.timeMS, piLabel(k.killerPi), piLabel(k.victimPi), nPkt, nWithW, len(hits))
		// distribution par arme
		byName := map[string]int{}
		for _, h := range hits {
			byName[h.name]++
		}
		fmt.Printf("    armes vues: ")
		for n, c := range byName {
			fmt.Printf("%s×%d ", n, c)
		}
		fmt.Println()
		// 12 plus proches du tick
		show := len(hits)
		if show > 12 {
			show = 12
		}
		for i := 0; i < show; i++ {
			h := hits[i]
			fmt.Printf("    Δ=%+6.1fms bit=%5d %-14s | pre5=%2d post5=%2d preByte=0x%02x postByte=0x%02x\n",
				h.dMs, h.bit, h.name, h.before5, h.after5, h.beforeB, h.afterB)
		}
	}
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
