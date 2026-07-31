package main

// wf_marker_h2 — teste H2 : au tick EXACT du kill, le paquet type-0 le plus
// proche contient-il un record "new" (R2=01) qui serait l'entité kill-feed ?
//
// Le brief donne le header FRAME : [R1 more][R2 type:1=new/2=del/3=delta][R7 id]
// (MSB-first). On parse en boucle ces en-têtes au début du payload (sans décoder
// le body — on s'arrête au 1er désync) pour COMPTER les records new/del/delta par
// paquet, et comparer la densité de "new" au tick de kill vs au tick de contrôle.
//
// Si "new" est un marqueur de kill, son taux d'apparition au tick de kill doit
// largement dépasser le contrôle.

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

var xuidToPi = map[uint64]int{
	2535467794760703: 0, 2535437947245250: 1, 2533274823110022: 2, 2533274980284321: 3,
	2533274815845110: 4, 2535444178793711: 5, 2533274882097883: 6, 2533274826120416: 7,
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
func bitsN(d []byte, bit, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(d, bit+i)
	}
	return v
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

// countRecordHeaders : parse [R1 more][R2 type 2b][R7 id 7b] = 10 bits/record en
// boucle ; compte new(type=1)/del(2)/delta(3). S'arrête quand more=0 ou bits
// épuisés. (heuristique : on ne décode pas le body, on suppose un stride fixe ;
// si désync probable, on borne à 64 records.) Renvoie (nNew,nDel,nDelta,total).
func countRecordHeaders(d []byte, strideBits int) (nNew, nDel, nDelta, total int) {
	bit := 0
	tot := len(d) * 8
	for total < 200 && bit+10 <= tot {
		more := bitsN(d, bit, 1)
		typ := bitsN(d, bit+1, 2)
		switch typ {
		case 1:
			nNew++
		case 2:
			nDel++
		case 3:
			nDelta++
		}
		total++
		bit += strideBits
		if more == 0 {
			break
		}
	}
	return
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

	nearestPkt := func(tms int) *packet {
		ts := int64(t0) + int64(tms)*usPerMs
		var best *packet
		bd := int64(1 << 62)
		for i := range type0 {
			d := int64(type0[i].ts) - ts
			if d < 0 {
				d = -d
			}
			if d < bd {
				bd, best = d, &type0[i]
			}
		}
		return best
	}

	kills := loadKills()

	// Approche simple et honnête : on regarde juste le 1er octet (header) du
	// paquet type-0 le plus proche du tick. Si un type de header particulier
	// (ex 0xd2=weapon-entity, ou un autre) domine aux ticks de kill vs contrôle.
	fmt.Println("=== 1er octet (header de paquet) au tick de kill vs contrôle ===")
	killHdr := map[byte]int{}
	for _, k := range kills {
		p := nearestPkt(k.timeMS)
		if p != nil && len(p.data) > 0 {
			killHdr[p.data[0]]++
		}
	}
	rng := rand.New(rand.NewSource(7))
	ctrlHdr := map[byte]int{}
	for i := 0; i < len(kills); i++ {
		tms := rng.Intn(490000) + 5000
		p := nearestPkt(tms)
		if p != nil && len(p.data) > 0 {
			ctrlHdr[p.data[0]]++
		}
	}
	allHdr := map[byte]bool{}
	for b := range killHdr {
		allHdr[b] = true
	}
	for b := range ctrlHdr {
		allHdr[b] = true
	}
	var hdrs []byte
	for b := range allHdr {
		hdrs = append(hdrs, b)
	}
	sort.Slice(hdrs, func(i, j int) bool { return killHdr[hdrs[i]] > killHdr[hdrs[j]] })
	for _, b := range hdrs {
		fmt.Printf("  header=0x%02x : kill=%2d  contrôle=%2d\n", b, killHdr[b], ctrlHdr[b])
	}

	// H2 : densité new/del/delta dans le paquet du tick (stride 10 bits — heuristique).
	fmt.Println("\n=== H2 densité record-headers (stride=10b heuristique) ===")
	var kNew, kDel, kDelta, kTot int
	for _, k := range kills {
		p := nearestPkt(k.timeMS)
		if p == nil {
			continue
		}
		nn, nd, ndl, tt := countRecordHeaders(p.data, 10)
		kNew += nn
		kDel += nd
		kDelta += ndl
		kTot += tt
	}
	var cNew, cDel, cDelta, cTot int
	for i := 0; i < len(kills); i++ {
		tms := rng.Intn(490000) + 5000
		p := nearestPkt(tms)
		if p == nil {
			continue
		}
		nn, nd, ndl, tt := countRecordHeaders(p.data, 10)
		cNew += nn
		cDel += nd
		cDelta += ndl
		cTot += tt
	}
	fmt.Printf("  KILL    : new=%d del=%d delta=%d total=%d\n", kNew, kDel, kDelta, kTot)
	fmt.Printf("  CONTRÔLE: new=%d del=%d delta=%d total=%d\n", cNew, cDel, cDelta, cTot)
	fmt.Println("  (si new(KILL) >> new(CONTRÔLE), H2 a une piste ; sinon non-discriminant)")
}
