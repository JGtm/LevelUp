package main

// wf_marker_d2 — dissèque le record "carrier" type-0 commençant par 0xd2 et
// portant un id-arme COMPLET à bit 44 :
//
//   byte0 = 0xd2 (constant)
//   byte1 = 0x60 + e   (e = sélecteur ?)
//   byte2 = (s<<4)|0x4 (s = sélecteur arme/slot ?)
//   byte3 = compteur (tick/seq)
//   ... id-arme 64 bits à bit 44 ...
//
// On veut savoir si (byte1,byte2hi) identifie un JOUEUR (entité). Méthode :
//  1. Construire la timeline de tous les carriers d2 : (ts, byte1, byte2hi, arme).
//  2. Regarder, pour chaque "slot" = (byte1,byte2hi), quelles armes il porte au
//     fil du temps et combien de fois.
//  3. CROISER avec la vérité kill : pour chaque kill, le carrier d2 le PLUS
//     RÉCENT (ts <= tick) — son slot doit être stable = le killer.
//     On valide via les kills de JGtm (Cindershot/BR75 attendus).
//  4. Tester si le slot peut être relié au pi par cohérence d'armes (chaque
//     joueur a un profil d'armes ; on aligne slot->pi par fréquence).

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

func wmapFull() map[uint64]string {
	full := map[uint64]string{}
	for id, n := range analysis.WeaponIDToName {
		if c, ok := analysis.WeaponFusionMap[n]; ok {
			n = c
		}
		full[id] = n
	}
	return full
}

type carrier struct {
	ts     uint64
	timeMS float64
	byte1  byte
	byte2  byte
	byte3  byte
	arme   string
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
	full := wmapFull()

	// collecter tous les carriers d2 (id-arme à bit 44, byte0==0xd2).
	var carriers []carrier
	for _, p := range type0 {
		if len(p.data) < 12 || p.data[0] != 0xd2 {
			continue
		}
		hi := uint64(bitsU32(p.data, 44))
		lo := uint64(bitsU32(p.data, 76))
		n, ok := full[(hi<<32)|lo]
		if !ok {
			continue
		}
		carriers = append(carriers, carrier{
			ts: p.ts, timeMS: float64(int64(p.ts)-int64(t0)) / 1000.0,
			byte1: p.data[1], byte2: p.data[2], byte3: p.data[3], arme: n,
		})
	}
	sort.Slice(carriers, func(i, j int) bool { return carriers[i].ts < carriers[j].ts })
	fmt.Printf("=== %d carriers d2 (byte0=0xd2, arme@bit44) ===\n", len(carriers))

	// distribution par "slot" = (byte1, byte2hi). Quelles armes par slot ?
	type slot struct{ b1, b2hi byte }
	slotArmes := map[slot]map[string]int{}
	slotCount := map[slot]int{}
	for _, c := range carriers {
		s := slot{c.byte1, c.byte2 >> 4}
		if slotArmes[s] == nil {
			slotArmes[s] = map[string]int{}
		}
		slotArmes[s][c.arme]++
		slotCount[s]++
	}
	fmt.Printf("\n=== profil par slot (byte1, byte2hi) -> armes ===\n")
	type sk struct {
		s slot
		c int
	}
	var sks []sk
	for s, c := range slotCount {
		sks = append(sks, sk{s, c})
	}
	sort.Slice(sks, func(i, j int) bool { return sks[i].c > sks[j].c })
	for i, e := range sks {
		if i >= 24 {
			break
		}
		// top 3 armes
		type av struct {
			a string
			v int
		}
		var avs []av
		for a, v := range slotArmes[e.s] {
			avs = append(avs, av{a, v})
		}
		sort.Slice(avs, func(i, j int) bool { return avs[i].v > avs[j].v })
		top := ""
		for k := 0; k < len(avs) && k < 4; k++ {
			top += fmt.Sprintf("%s×%d ", avs[k].a, avs[k].v)
		}
		fmt.Printf("  slot b1=0x%02x b2hi=%x  cnt=%4d : %s\n", e.s.b1, e.s.b2hi, e.c, top)
	}

	// Idée alternative : byte2hi seul = identité d'arme (Disruptor=4? Shock=6?),
	// byte1 = entité joueur. Vérifions byte1 seul.
	fmt.Printf("\n=== profil par byte1 seul -> armes ===\n")
	b1Armes := map[byte]map[string]int{}
	for _, c := range carriers {
		if b1Armes[c.byte1] == nil {
			b1Armes[c.byte1] = map[string]int{}
		}
		b1Armes[c.byte1][c.arme]++
	}
	var b1s []byte
	for b := range b1Armes {
		b1s = append(b1s, b)
	}
	sort.Slice(b1s, func(i, j int) bool { return b1s[i] < b1s[j] })
	for _, b := range b1s {
		type av struct {
			a string
			v int
		}
		var avs []av
		tot := 0
		for a, v := range b1Armes[b] {
			avs = append(avs, av{a, v})
			tot += v
		}
		sort.Slice(avs, func(i, j int) bool { return avs[i].v > avs[j].v })
		top := ""
		for k := 0; k < len(avs) && k < 5; k++ {
			top += fmt.Sprintf("%s×%d ", avs[k].a, avs[k].v)
		}
		fmt.Printf("  byte1=0x%02x (tot=%d, %d armes distinctes): %s\n", b, tot, len(avs), top)
	}

	// CROISEMENT KILL : pour chaque kill, le carrier d2 le plus récent (ts<=tick),
	// son slot. Si byte1 identifie le killer, alors le même byte1 doit revenir
	// pour les kills d'un même joueur. On regroupe par killerPi.
	kills := loadKills()
	fmt.Printf("\n=== kill -> dernier carrier d2 avant le tick (byte1/byte2hi/arme) ===\n")
	// par killerPi : histogramme des byte1 du dernier carrier
	byKiller := map[int]map[byte]int{}
	for _, k := range kills {
		ts := int64(t0) + int64(k.timeMS)*usPerMs
		var last *carrier
		for i := range carriers {
			if int64(carriers[i].ts) <= ts {
				last = &carriers[i]
			} else {
				break
			}
		}
		if last == nil {
			continue
		}
		if byKiller[k.killerPi] == nil {
			byKiller[k.killerPi] = map[byte]int{}
		}
		byKiller[k.killerPi][last.byte1]++
	}
	var kps []int
	for p := range byKiller {
		kps = append(kps, p)
	}
	sort.Ints(kps)
	for _, p := range kps {
		// top byte1
		type bv struct {
			b byte
			v int
		}
		var bvs []bv
		for b, v := range byKiller[p] {
			bvs = append(bvs, bv{b, v})
		}
		sort.Slice(bvs, func(i, j int) bool { return bvs[i].v > bvs[j].v })
		s := ""
		for _, e := range bvs {
			s += fmt.Sprintf("0x%02x×%d ", e.b, e.v)
		}
		fmt.Printf("  killer=%-5s : byte1 du dernier carrier = %s\n", piLabel(p), s)
	}

	// JGtm focus : ses kills, et le dernier carrier d2.
	fmt.Printf("\n=== JGtm (pi2) kills : dernier carrier d2 avant tick ===\n")
	for _, k := range kills {
		if k.killerPi != 2 {
			continue
		}
		ts := int64(t0) + int64(k.timeMS)*usPerMs
		var last *carrier
		for i := range carriers {
			if int64(carriers[i].ts) <= ts {
				last = &carriers[i]
			} else {
				break
			}
		}
		if last == nil {
			continue
		}
		fmt.Printf("  t=%6dms victim=%-5s | last d2: Δ=%+7.1fms b1=0x%02x b2hi=%x arme=%s\n",
			k.timeMS, piLabel(k.victimPi), last.timeMS-float64(k.timeMS), last.byte1, last.byte2>>4, last.arme)
	}
}
