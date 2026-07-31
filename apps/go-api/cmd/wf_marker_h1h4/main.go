package main

// wf_marker_h1h4 — teste H1 (killer_pi ... arme ... victim_pi à courte distance)
// et H4 (FourCC 'obje'=0x6f626a65 ou autre marqueur de damage-event) au tick du
// kill, AVEC un groupe de contrôle (ticks aléatoires non-kill) pour mesurer le
// taux de faux positifs.
//
// H1 : autour de chaque littéral high-32 d'arme trouvé dans la fenêtre serrée
// (±120ms) du tick, on cherche le killer_pi ET le victim_pi encodés en 5 bits
// (toutes positions ±64 bits autour du littéral) ou en octet. On compte combien
// de kills présentent un triplet [killer_pi][arme][victim_pi] (ordre quelconque)
// pour l'arme la plus proche du tick.
//
// H4 : on scanne tout le flux pour 'obje' et des FourCC candidats, on regarde si
// leur ts coïncide avec des ticks de kill (vs contrôle).

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
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
	if bit < 0 {
		return 0xffffffff
	}
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

// hasPi5bits cherche la valeur pi codée en 5 bits dans [bit-span .. bit+span].
func hasPiNear(d []byte, centerBit, span, pi int) bool {
	target := uint32(pi)
	for off := -span; off <= span; off++ {
		if bitsN(d, centerBit+off, 5) == target {
			return true
		}
	}
	return false
}

// hasPiSlotNear : pi<<4|slot, slot 0..15, donc 9 bits.
func hasPiSlotNear(d []byte, centerBit, span, pi int) bool {
	for off := -span; off <= span; off++ {
		v := bitsN(d, centerBit+off, 9)
		if v>>4 == uint32(pi) {
			return true
		}
	}
	return false
}

func main() {
	var type0 []packet
	allDataByChunk := map[int][]byte{}
	for i := 0; i <= 27; i++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, i))
		if len(d) == 0 {
			continue
		}
		allDataByChunk[i] = d
		for _, p := range parsePackets(i, d) {
			if p.typ == 0 {
				type0 = append(type0, p)
			}
		}
	}
	sort.Slice(type0, func(i, j int) bool { return type0[i].ts < type0[j].ts })
	wm := wmap()
	kills := loadKills()

	// ───────── H4 : FourCC 'obje' et candidats ─────────
	fmt.Println("=== H4 : recherche FourCC dans tous les chunks (byte-aligné) ===")
	fourccs := map[string][]byte{
		"obje": {0x6f, 0x62, 0x6a, 0x65},
		"ejbo": {0x65, 0x6a, 0x62, 0x6f}, // little-endian
		"kill": {0x6b, 0x69, 0x6c, 0x6c},
		"dmg ": {0x64, 0x6d, 0x67, 0x20},
		"weap": {0x77, 0x65, 0x61, 0x70},
	}
	for name, pat := range fourccs {
		cnt := 0
		for _, d := range allDataByChunk {
			for i := 0; i+4 <= len(d); i++ {
				if bytes.Equal(d[i:i+4], pat) {
					cnt++
				}
			}
		}
		fmt.Printf("  '%s' (% x) : %d occurrences byte-aligné\n", name, pat, cnt)
	}

	// ───────── H1 : triplet [killer_pi][arme][victim_pi] ─────────
	// Pour chaque kill, on prend tous les littéraux d'arme dans ±W du tick, et on
	// teste si le killer_pi ET le victim_pi sont encodés (5 bits) à ±SPAN bits.
	const W = 120000 // ±120ms
	const SPAN = 80  // ±80 bits autour du littéral
	type result struct {
		kill           kill
		nLit           int
		bestArme       string
		bestDms        float64
		killerNear5    bool
		victimNear5    bool
		killerNearSlot bool
		victimNearSlot bool
	}
	var results []result
	for _, k := range kills {
		if k.killerPi < 0 || k.victimPi < 0 {
			continue
		}
		ts := int64(t0) + int64(k.timeMS)*usPerMs
		lo, hi := uint64(ts-W), uint64(ts+W)
		r := result{kill: k, bestDms: 1e9}
		for i := range type0 {
			p := &type0[i]
			if p.ts < lo || p.ts > hi {
				continue
			}
			tot := len(p.data) * 8
			for bp := 0; bp+32 <= tot; bp++ {
				if n, ok := wm[bitsU32(p.data, bp)]; ok {
					r.nLit++
					dms := float64(int64(p.ts) - ts)
					if absf(dms) < r.bestDms*1000 {
						r.bestDms = absf(dms) / 1000
						r.bestArme = n
						r.killerNear5 = hasPiNear(p.data, bp, SPAN, k.killerPi)
						r.victimNear5 = hasPiNear(p.data, bp, SPAN, k.victimPi)
						r.killerNearSlot = hasPiSlotNear(p.data, bp, SPAN, k.killerPi)
						r.victimNearSlot = hasPiSlotNear(p.data, bp, SPAN, k.victimPi)
					}
				}
			}
		}
		results = append(results, r)
	}

	// stats H1
	nKV := len(results)
	withLit := 0
	kBoth5, kKiller5, kVictim5 := 0, 0, 0
	kBothSlot := 0
	for _, r := range results {
		if r.nLit > 0 {
			withLit++
		}
		if r.killerNear5 {
			kKiller5++
		}
		if r.victimNear5 {
			kVictim5++
		}
		if r.killerNear5 && r.victimNear5 {
			kBoth5++
		}
		if r.killerNearSlot && r.victimNearSlot {
			kBothSlot++
		}
	}
	fmt.Printf("\n=== H1 : %d kills (killer+victim pi connus) ===\n", nKV)
	fmt.Printf("  avec >=1 littéral arme dans ±%dms : %d\n", W/1000, withLit)
	fmt.Printf("  killer_pi (5bit) à ±%dbits du littéral : %d\n", SPAN, kKiller5)
	fmt.Printf("  victim_pi (5bit) à ±%dbits du littéral : %d\n", SPAN, kVictim5)
	fmt.Printf("  killer ET victim (5bit) : %d\n", kBoth5)
	fmt.Printf("  killer ET victim (pi<<4|slot) : %d\n", kBothSlot)

	// ───────── contrôle : ticks aléatoires non-kill ─────────
	rng := rand.New(rand.NewSource(42))
	killSet := map[int]bool{}
	for _, k := range kills {
		killSet[k.timeMS/1000] = true
	}
	ctrlBoth5 := 0
	ctrlN := 0
	for ctrlN < nKV {
		tms := rng.Intn(490000) + 5000
		if killSet[tms/1000] {
			continue
		}
		// pi killer/victim aléatoires distincts
		kp := rng.Intn(8)
		vp := rng.Intn(8)
		for vp == kp {
			vp = rng.Intn(8)
		}
		ts := int64(t0) + int64(tms)*usPerMs
		lo, hi := uint64(ts-W), uint64(ts+W)
		kN, vN := false, false
		for i := range type0 {
			p := &type0[i]
			if p.ts < lo || p.ts > hi {
				continue
			}
			tot := len(p.data) * 8
			for bp := 0; bp+32 <= tot; bp++ {
				if _, ok := wm[bitsU32(p.data, bp)]; ok {
					if hasPiNear(p.data, bp, SPAN, kp) {
						kN = true
					}
					if hasPiNear(p.data, bp, SPAN, vp) {
						vN = true
					}
				}
			}
		}
		if kN && vN {
			ctrlBoth5++
		}
		ctrlN++
	}
	fmt.Printf("\n=== CONTRÔLE (%d ticks non-kill, pi aléatoires) ===\n", ctrlN)
	fmt.Printf("  killer ET victim (5bit) : %d  -> taux faux positif=%.0f%%\n",
		ctrlBoth5, 100*float64(ctrlBoth5)/float64(ctrlN))
	fmt.Printf("  (rappel kills: killer+victim 5bit = %d -> %.0f%%)\n",
		kBoth5, 100*float64(kBoth5)/float64(nKV))

	// détail des 12 premiers kills
	fmt.Println("\n=== détail 12 kills ===")
	for i, r := range results {
		if i >= 12 {
			break
		}
		fmt.Printf("  kill t=%6dms killer=%-5s victim=%-5s | nLit=%d bestArme=%-12s Δ=%.1fms | k5=%v v5=%v kSlot=%v vSlot=%v\n",
			r.kill.timeMS, piLabel(r.kill.killerPi), piLabel(r.kill.victimPi), r.nLit, r.bestArme, r.bestDms,
			r.killerNear5, r.victimNear5, r.killerNearSlot, r.victimNearSlot)
	}
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
