package main

// wf_marker_recon — Reconnaissance pour la chasse au marqueur d'arme de kill.
//
// Étape 0 (avant de tester H1..H4) :
//  - extraire kills/deaths de chunk_27 via le parseur canonique du repo
//    (analysis.ParseHighlightEvents) -> (xuid sujet, time_ms).
//  - apparier kill<->death par time_ms proche (<=5ms) pour obtenir
//    (killer_xuid, victim_xuid, time_ms).
//  - mapper xuid -> pi (table vérifiée bit-exact du brief).
//  - vérifier t0 : ts du paquet type-2 de chunk_02.
//  - pour chaque kill : trouver le paquet type-0 dont le ts est le plus proche
//    de ts_cible, et lister TOUS les high-32 d'armes présents dans une fenêtre
//    ±2000 bits autour de l'offset estimé, avec leur distance bit. C'est la
//    base brute pour juger les hypothèses de marqueur.

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

// t0 : ts du paquet type-2 de chunk_02 (tick du kill time_ms=0), ~1e6 u/s.
const t0 = uint64(4537898226)
const usPerMs = 1000

// ───────── pi <-> xuid (vérifié bit-exact) ─────────
var piToXuid = map[int]uint64{
	0: 2535467794760703,
	1: 2535437947245250,
	2: 2533274823110022, // JGtm
	3: 2533274980284321,
	4: 2533274815845110,
	5: 2535444178793711,
	6: 2533274882097883,
	7: 2533274826120416,
}
var xuidToPi = func() map[uint64]int {
	m := map[uint64]int{}
	for pi, x := range piToXuid {
		m[x] = pi
	}
	return m
}()

func piLabel(pi int) string {
	if pi == 2 {
		return "JGtm"
	}
	return fmt.Sprintf("pi%d", pi)
}

// ───────── inflate / packets / bits ─────────

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

// packet — un paquet décodé du flux d'un chunk.
type packet struct {
	chunk int
	typ   uint16
	ts    uint64
	off   int    // offset du payload dans le buffer chunk
	data  []byte // payload (Size octets)
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
		out = append(out, packet{chunk: chunk, typ: typ, ts: ts, off: off + 16, data: d[off+16 : off+16+sz]})
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

// bitsU32 lit 32 bits MSB-first à partir du bit `bit`.
func bitsU32(d []byte, bit int) uint32 {
	var v uint32
	for i := 0; i < 32; i++ {
		v = (v << 1) | bitAt(d, bit+i)
	}
	return v
}

// wmap : high-32 famille -> nom canonique.
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

// ───────── kills/deaths ─────────

type kill struct {
	killerXuid uint64
	victimXuid uint64
	killerPi   int
	victimPi   int
	timeMS     int
}

func loadEvents() []analysis.HighlightEvent {
	d := inflate(fmt.Sprintf("%s/chunk_27.bin", cache))
	// version 41+ layout (cf. parser) ; on tente plusieurs versions et garde
	// celle qui donne le plus d'events kill/death.
	best := []analysis.HighlightEvent(nil)
	bestN := -1
	for _, ver := range []int{34, 39, 41, 42} {
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
	return best
}

func pairKills(ev []analysis.HighlightEvent) []kill {
	var kills, deaths []analysis.HighlightEvent
	for _, e := range ev {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, e)
		case analysis.EventTypeDeath:
			deaths = append(deaths, e)
		}
	}
	var out []kill
	usedDeath := make([]bool, len(deaths))
	for _, k := range kills {
		bestJ, bestD := -1, 6
		for j, dth := range deaths {
			if usedDeath[j] {
				continue
			}
			dd := k.TimeMS - dth.TimeMS
			if dd < 0 {
				dd = -dd
			}
			if dd <= 5 && dd < bestD {
				bestD, bestJ = dd, j
			}
		}
		ku := kill{killerXuid: k.XUID, timeMS: k.TimeMS, killerPi: -1, victimPi: -1}
		if pi, ok := xuidToPi[k.XUID]; ok {
			ku.killerPi = pi
		}
		if bestJ >= 0 {
			usedDeath[bestJ] = true
			ku.victimXuid = deaths[bestJ].XUID
			if pi, ok := xuidToPi[deaths[bestJ].XUID]; ok {
				ku.victimPi = pi
			}
		}
		out = append(out, ku)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].timeMS < out[j].timeMS })
	return out
}

func main() {
	// 1. charger tous les chunks + paquets.
	var allPkts []packet
	for i := 0; i <= 27; i++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, i))
		if len(d) == 0 {
			continue
		}
		allPkts = append(allPkts, parsePackets(i, d)...)
	}
	// vérif t0 : type-2 de chunk_02.
	for _, p := range allPkts {
		if p.chunk == 2 && p.typ == 2 {
			fmt.Printf("chunk_02 type-2 ts=%d  (t0 attendu=%d, delta=%d)\n", p.ts, t0, int64(p.ts)-int64(t0))
			break
		}
	}

	// distribution des types de paquets.
	typCount := map[uint16]int{}
	var type0 []packet
	for _, p := range allPkts {
		typCount[p.typ]++
		if p.typ == 0 {
			type0 = append(type0, p)
		}
	}
	fmt.Printf("paquets: total=%d  ", len(allPkts))
	for t, c := range typCount {
		fmt.Printf("type%d=%d ", t, c)
	}
	fmt.Println()
	sort.Slice(type0, func(i, j int) bool { return type0[i].ts < type0[j].ts })
	if len(type0) > 0 {
		fmt.Printf("type-0: %d paquets, ts [%d .. %d], span=%.1fs\n",
			len(type0), type0[0].ts, type0[len(type0)-1].ts,
			float64(type0[len(type0)-1].ts-type0[0].ts)/1e6)
	}

	// 2. kills/deaths.
	ev := loadEvents()
	kills := pairKills(ev)
	nKills := 0
	for _, e := range ev {
		if e.EventType == analysis.EventTypeKill {
			nKills++
		}
	}
	fmt.Printf("\nevents: %d total, %d kills\n", len(ev), nKills)
	fmt.Println("=== kills (triés par time_ms) ===")
	for i, k := range kills {
		fmt.Printf("  [%02d] t=%6dms  killer=%s(xuid%d) victim=%s(xuid%d)\n",
			i, k.timeMS, piLabel(k.killerPi), k.killerXuid, piLabel(k.victimPi), k.victimXuid)
	}

	// 3. pour 12 kills variés (préférer killer/victim pi connus), dump des armes
	//    autour de ts_cible.
	wm := wmap()
	var sample []kill
	for _, k := range kills {
		if k.killerPi >= 0 && k.victimPi >= 0 {
			sample = append(sample, k)
		}
		if len(sample) >= 12 {
			break
		}
	}
	fmt.Printf("\n=== échantillon : %d kills avec killer+victim pi connus ===\n", len(sample))

	for ki, k := range sample {
		tsTarget := t0 + uint64(k.timeMS)*usPerMs
		// paquet type-0 le plus proche.
		var near *packet
		bestD := uint64(1 << 62)
		for i := range type0 {
			d := type0[i].ts
			var dd uint64
			if d > tsTarget {
				dd = d - tsTarget
			} else {
				dd = tsTarget - d
			}
			if dd < bestD {
				bestD, near = dd, &type0[i]
			}
		}
		if near == nil {
			continue
		}
		fmt.Printf("\n--- kill[%02d] t=%dms killer=%s victim=%s  ts_cible=%d  | near type-0 chunk_%02d ts=%d Δ=%dus(%.1fms) len=%d ---\n",
			ki, k.timeMS, piLabel(k.killerPi), piLabel(k.victimPi), tsTarget,
			near.chunk, near.ts, int64(near.ts)-int64(tsTarget), float64(int64(near.ts)-int64(tsTarget))/1000.0, len(near.data))
		// scan armes dans le paquet entier.
		type wh struct {
			bit  int
			name string
		}
		var hits []wh
		tot := len(near.data) * 8
		for bp := 0; bp+32 <= tot; bp++ {
			if n, ok := wm[bitsU32(near.data, bp)]; ok {
				hits = append(hits, wh{bp, n})
			}
		}
		fmt.Printf("    %d littéraux d'arme dans le paquet:\n", len(hits))
		shown := 0
		for _, h := range hits {
			fmt.Printf("      bit=%6d byte=%5d  %s\n", h.bit, h.bit/8, h.name)
			shown++
			if shown >= 40 {
				fmt.Printf("      ... (%d de plus)\n", len(hits)-shown)
				break
			}
		}
	}
}
