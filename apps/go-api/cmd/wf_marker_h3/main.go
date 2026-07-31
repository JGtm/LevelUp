package main

// wf_marker_h3 — derniers tests :
//  (A) Combien de slots d2 distincts (byte1,byte2hi) ? ~ nb d'armes sur la carte
//      ou ~ 8 joueurs ? + byte3 = quoi (compteur monotone par slot = tick) ?
//  (B) H3 ordering : à chaque kill, dans le paquet type-0 le plus proche, la
//      séquence des records. Y a-t-il un délimiteur récurrent juste avant
//      l'arme du killer ? On mesure si l'arme du KILLER (déduite par cohérence)
//      est systématiquement la 1re/dernière d'une fenêtre.
//  (C) Test décisif : le d2 carrier porte-t-il une RÉFÉRENCE au holder ? On
//      cherche, dans le payload du carrier, un pi (5 bits) ou xuid court qui
//      serait STABLE pour un slot donné (=> le slot a un propriétaire).

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
	full := wmapFull()

	type carrier struct {
		ts     uint64
		data   []byte
		byte1  byte
		byte2  byte
		byte3  byte
		arme   string
		wepBit int
	}
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
		carriers = append(carriers, carrier{p.ts, p.data, p.data[1], p.data[2], p.data[3], n, 44})
	}
	sort.Slice(carriers, func(i, j int) bool { return carriers[i].ts < carriers[j].ts })

	// (A) slots distincts
	type slot struct{ b1, b2hi byte }
	slotSet := map[slot]bool{}
	for _, c := range carriers {
		slotSet[slot{c.byte1, c.byte2 >> 4}] = true
	}
	fmt.Printf("=== (A) %d carriers ; %d slots (byte1,byte2hi) distincts ===\n", len(carriers), len(slotSet))
	// byte3 monotone par slot ?
	fmt.Println("  byte3 par slot (échantillon 6 slots) :")
	perSlot := map[slot][]byte{}
	for _, c := range carriers {
		s := slot{c.byte1, c.byte2 >> 4}
		perSlot[s] = append(perSlot[s], c.byte3)
	}
	shown := 0
	for s, b3s := range perSlot {
		if len(b3s) < 5 {
			continue
		}
		fmt.Printf("    slot b1=0x%02x b2hi=%x : byte3=%v\n", s.b1, s.b2hi, b3s[:min(12, len(b3s))])
		shown++
		if shown >= 6 {
			break
		}
	}

	// (C) le carrier porte-t-il un holder stable ? Pour un slot donné, on lit
	// plusieurs champs candidats (octets/bits à divers offsets) et on regarde
	// lesquels sont CONSTANTS sur toutes les occurrences du slot (=> identité
	// d'entité), vs ceux qui varient (=> état). On teste les 3 bits/octets juste
	// après l'id-arme (bit 108..) et le byte3.
	fmt.Println("\n=== (C) champs constants par slot (recherche du holder) ===")
	// Pour chaque slot avec >=8 occurrences, lister les valeurs de byte4, byte5,
	// et 5-bit à bit 108 (juste après l'arme 64b @44..108).
	type slotStat struct {
		s        slot
		n        int
		b4set    map[byte]bool
		afterArm map[uint32]bool // 8 bits après l'arme
		preB1lo  byte
	}
	stats := map[slot]*slotStat{}
	for _, c := range carriers {
		s := slot{c.byte1, c.byte2 >> 4}
		st := stats[s]
		if st == nil {
			st = &slotStat{s: s, b4set: map[byte]bool{}, afterArm: map[uint32]bool{}}
			stats[s] = st
		}
		st.n++
		if len(c.data) > 13 {
			st.b4set[c.data[13]] = true // octet ~après l'arme (bit 104+)
		}
		st.afterArm[bitsN(c.data, 108, 8)] = true
	}
	var ordered []*slotStat
	for _, st := range stats {
		ordered = append(ordered, st)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].n > ordered[j].n })
	for i, st := range ordered {
		if i >= 14 {
			break
		}
		fmt.Printf("  slot b1=0x%02x b2hi=%x n=%3d : #valeurs byte13=%d  #valeurs after-arm(8b)=%d\n",
			st.s.b1, st.s.b2hi, st.n, len(st.b4set), len(st.afterArm))
	}

	// CONCLUSION quantifiée sur l'hypothèse "slot = arme-instance, pas joueur" :
	// pour chaque slot, l'arme dominante couvre quel % ?
	slotArme := map[slot]map[string]int{}
	for _, c := range carriers {
		s := slot{c.byte1, c.byte2 >> 4}
		if slotArme[s] == nil {
			slotArme[s] = map[string]int{}
		}
		slotArme[s][c.arme]++
	}
	pure := 0
	for _, m := range slotArme {
		tot, top := 0, 0
		for _, v := range m {
			tot += v
			if v > top {
				top = v
			}
		}
		if float64(top)/float64(tot) >= 0.9 {
			pure++
		}
	}
	fmt.Printf("\n  slots dont >=90%% des occurrences = 1 seule arme : %d / %d\n", pure, len(slotArme))
	fmt.Println("  => si proche de 100%, le slot identifie une INSTANCE d'arme (entité monde), pas un joueur.")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
