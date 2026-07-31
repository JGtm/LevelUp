package main

// wf_marker_smallpkt — caractérise les paquets type-0 qui portent un littéral
// d'arme COMPLET (high-32 suivi de 0x42c9679f) au début du payload (bit ~44),
// et teste s'ils coïncident avec les ticks de kill.
//
// Hypothèse : ces petits paquets sont des events discrets « arme X assignée à
// l'entité E » (kill-feed / pickup). Si à chaque kill il en existe un au tick
// exact (±quelques ms) ET que son arme est plausible pour le killer, c'est le
// marqueur.
//
// On dump aussi le payload brut (hex) des paquets porteurs d'arme proches d'un
// kill pour repérer killer/victim pi et un FourCC 'obje'.

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
	b2    byte
	b3    byte
	ts    uint64
	data  []byte
}

func parsePackets(chunk int, d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		b2, b3 := d[off+2], d[off+3]
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, packet{chunk, typ, b2, b3, ts, d[off+16 : off+16+sz]})
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

// full weapon map : id 64-bit -> nom (pour valider le suffixe complet).
func wmapFull() (map[uint32]string, map[uint64]string) {
	hi := map[uint32]string{}
	full := map[uint64]string{}
	for id, n := range analysis.WeaponIDToName {
		if c, ok := analysis.WeaponFusionMap[n]; ok {
			n = c
		}
		hi[uint32(id>>32)] = n
		full[id] = n
	}
	return hi, full
}

// scanFullWeapon : cherche un id 64-bit COMPLET (high32 + low32) MSB-first.
type fwHit struct {
	bit  int
	id   uint64
	name string
}

func scanFullWeapon(d []byte, full map[uint64]string) []fwHit {
	var out []fwHit
	tot := len(d) * 8
	for bp := 0; bp+64 <= tot; bp++ {
		hi := uint64(bitsU32(d, bp))
		lo := uint64(bitsU32(d, bp+32))
		id := (hi << 32) | lo
		if n, ok := full[id]; ok {
			out = append(out, fwHit{bp, id, n})
		}
	}
	return out
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
	_, full := wmapFull()

	// Phase A : caractériser TOUS les paquets type-0 portant un id arme complet.
	// distribution de leur taille, de b2/b3, et de la position bit du 1er id.
	sizeHist := map[int]int{}
	bitHist := map[int]int{}
	var carriers []int // index dans type0
	for i := range type0 {
		hits := scanFullWeapon(type0[i].data, full)
		if len(hits) == 0 {
			continue
		}
		carriers = append(carriers, i)
		sizeHist[len(type0[i].data)]++
		bitHist[hits[0].bit]++
	}
	fmt.Printf("=== %d/%d paquets type-0 portent >=1 id-arme COMPLET ===\n", len(carriers), len(type0))
	fmt.Println("  tailles (octets) des porteurs:")
	{
		type kv struct{ k, v int }
		var arr []kv
		for k, v := range sizeHist {
			arr = append(arr, kv{k, v})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
		for i, kv := range arr {
			if i >= 12 {
				break
			}
			fmt.Printf("    len=%4d : %d\n", kv.k, kv.v)
		}
	}
	fmt.Println("  position bit du 1er id-arme:")
	{
		type kv struct{ k, v int }
		var arr []kv
		for k, v := range bitHist {
			arr = append(arr, kv{k, v})
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
		for i, kv := range arr {
			if i >= 12 {
				break
			}
			fmt.Printf("    bit=%5d : %d\n", kv.k, kv.v)
		}
	}

	// Phase B : pour chaque kill, le porteur d'arme le plus proche en temps.
	kills := loadKills()
	fmt.Printf("\n=== pour chaque kill: porteur d'arme type-0 le plus proche ===\n")
	matchAt := map[string]int{} // bucket de |Δ| -> count
	for ki, k := range kills {
		ts := int64(t0) + int64(k.timeMS)*usPerMs
		bestIdx, bestD := -1, int64(1<<62)
		for _, ci := range carriers {
			d := int64(type0[ci].ts) - ts
			if d < 0 {
				d = -d
			}
			if d < bestD {
				bestD, bestIdx = d, ci
			}
		}
		if bestIdx < 0 {
			continue
		}
		p := type0[bestIdx]
		hits := scanFullWeapon(p.data, full)
		names := map[string]bool{}
		for _, h := range hits {
			names[h.name] = true
		}
		var ns []string
		for n := range names {
			ns = append(ns, n)
		}
		dms := float64(bestD) / 1000.0
		bucket := ">100ms"
		switch {
		case dms <= 5:
			bucket = "<=5ms"
		case dms <= 20:
			bucket = "<=20ms"
		case dms <= 50:
			bucket = "<=50ms"
		case dms <= 100:
			bucket = "<=100ms"
		}
		matchAt[bucket]++
		if ki < 20 {
			fmt.Printf("  kill[%02d] t=%6dms killer=%-5s victim=%-5s | nearest carrier Δ=%+7.1fms chunk_%02d len=%3d b2=%02x b3=%02x armes=%v\n",
				ki, k.timeMS, piLabel(k.killerPi), piLabel(k.victimPi), float64(int64(p.ts)-ts)/1000.0, p.chunk, len(p.data), p.b2, p.b3, ns)
		}
	}
	fmt.Println("\n  distribution |Δ| kill->porteur le plus proche:")
	for _, b := range []string{"<=5ms", "<=20ms", "<=50ms", "<=100ms", ">100ms"} {
		fmt.Printf("    %-8s : %d / %d\n", b, matchAt[b], len(kills))
	}
}
