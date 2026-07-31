package main

// wf_marker_obje — localise les 319 'obje' FourCC : dans quel type de paquet,
// à quel offset, et leur ts vs ticks de kill. Teste si 'obje' est un marqueur
// d'event de kill (H4) ou juste une string de registry (keyframe).
//
// Dissèque aussi les paquets type-0 "carrier" (id-arme complet à bit~44) : que
// contient le préambule (les 44 bits avant l'arme) ? Y trouve-t-on un pi/slot ?

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
	chunk  int
	typ    uint16
	ts     uint64
	bufOff int
	data   []byte
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
		out = append(out, packet{chunk, typ, ts, off, d[off+16 : off+16+sz]})
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
	var allPkts []packet
	for i := 0; i <= 27; i++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, i))
		if len(d) == 0 {
			continue
		}
		allPkts = append(allPkts, parsePackets(i, d)...)
	}

	// ── 'obje' : dans quel type de paquet ? ──
	objePat := []byte{0x6f, 0x62, 0x6a, 0x65}
	objeByType := map[uint16]int{}
	var objeTsList []uint64
	for _, p := range allPkts {
		for i := 0; i+4 <= len(p.data); i++ {
			if bytes.Equal(p.data[i:i+4], objePat) {
				objeByType[p.typ]++
				objeTsList = append(objeTsList, p.ts)
			}
		}
	}
	fmt.Println("=== 'obje' par type de paquet ===")
	for t, c := range objeByType {
		fmt.Printf("  type%d : %d\n", t, c)
	}
	// distribution temporelle des 'obje' (combien dans type-0 par seconde) :
	sort.Slice(objeTsList, func(i, j int) bool { return objeTsList[i] < objeTsList[j] })
	if len(objeTsList) > 0 {
		fmt.Printf("  'obje' ts span: %.1fs .. %.1fs\n",
			float64(int64(objeTsList[0])-int64(t0))/1e6, float64(int64(objeTsList[len(objeTsList)-1])-int64(t0))/1e6)
	}

	// 'obje' dans type-0 : corrèlent-ils avec les kills ?
	var type0obje []uint64
	for _, p := range allPkts {
		if p.typ != 0 {
			continue
		}
		for i := 0; i+4 <= len(p.data); i++ {
			if bytes.Equal(p.data[i:i+4], objePat) {
				type0obje = append(type0obje, p.ts)
				break
			}
		}
	}
	kills := loadKills()
	fmt.Printf("\n=== %d paquets type-0 contiennent 'obje' ; corrélation aux kills ===\n", len(type0obje))
	near := 0
	for _, k := range kills {
		ts := int64(t0) + int64(k.timeMS)*usPerMs
		best := int64(1 << 62)
		for _, ot := range type0obje {
			d := int64(ot) - ts
			if d < 0 {
				d = -d
			}
			if d < best {
				best = d
			}
		}
		if best <= 50000 {
			near++
		}
	}
	fmt.Printf("  kills avec un 'obje'(type-0) à <=50ms : %d / %d\n", near, len(kills))

	// ── carriers bit~44 : préambule ──
	full := wmapFull()
	fmt.Println("\n=== préambule des carriers (id-arme à bit 44) : 44 bits avant l'arme ===")
	shown := 0
	preHist := map[string]int{}
	for _, p := range allPkts {
		if p.typ != 0 {
			continue
		}
		// 1er id complet
		tot := len(p.data) * 8
		firstBit := -1
		var firstName string
		for bp := 0; bp+64 <= tot; bp++ {
			hi := uint64(bitsU32(p.data, bp))
			lo := uint64(bitsU32(p.data, bp+32))
			if n, ok := full[(hi<<32)|lo]; ok {
				firstBit = bp
				firstName = n
				break
			}
		}
		if firstBit != 44 {
			continue
		}
		// préambule 44 bits = data[0:5]+4bits ; on dump les 6 premiers octets en hex.
		preKey := fmt.Sprintf("% x", p.data[0:6])
		preHist[preKey]++
		if shown < 16 {
			fmt.Printf("  chunk_%02d len=%3d pre6=[% x] arme=%-12s | b0..b5 bits: %08b %08b %08b %08b %08b %08b\n",
				p.chunk, len(p.data), p.data[0:6], firstName,
				p.data[0], p.data[1], p.data[2], p.data[3], p.data[4], p.data[5])
			shown++
		}
	}
	fmt.Println("  préambules distincts (6 octets) les plus fréquents:")
	type kv struct {
		k string
		v int
	}
	var arr []kv
	for k, v := range preHist {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
	for i, e := range arr {
		if i >= 10 {
			break
		}
		fmt.Printf("    [%s] : %d\n", e.k, e.v)
	}
}
