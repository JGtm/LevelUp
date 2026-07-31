// tmp_xuidcluster — THROWAWAY, OEIL NEUF : le paquet type-9 (chunk_27) contient-il des records riches
// (killer+victime+assistant en xuids 64-bit) au-delà des 230 highlight events ?
// Un xuid 64-bit = ancrage sans faux positif. On scanne TOUS les bit-offsets, on lit 64 bits
// (MSB-first + byte-swap LE), on compte les occurrences par xuid, et on détecte les CLUSTERS
// (>=2 xuids distincts rapprochés = candidat record killer/victim/assistant).
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
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var xuids = []uint64{
	2535467794760703, 2535437947245250, 2533274823110022, 2533274980284321,
	2533274815845110, 2535444178793711, 2533274882097883, 2533274826120416,
}
var xuidName = map[uint64]string{
	2535467794760703: "whiteknight2519", 2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm", 2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA", 2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus", 2533274826120416: "VitaminA1688",
}

func nm(x uint64) string {
	if g, ok := xuidName[x]; ok {
		return g
	}
	return fmt.Sprintf("%d", x)
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
func bswap(v uint64) uint64 {
	return v>>56 | (v&0xff000000000000)>>40 | (v&0xff0000000000)>>24 | (v&0xff00000000)>>8 |
		(v&0xff000000)<<8 | (v&0xff0000)<<24 | (v&0xff00)<<40 | v<<56
}

func main() {
	clusterBits := 400
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &clusterBits)
	}
	xset := map[uint64]bool{}
	for _, x := range xuids {
		xset[x] = true
	}

	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
	hl := map[uint64]int{}
	for _, e := range events {
		hl[e.XUID]++
	}

	d := inflate(cache + "/chunk_27.bin")
	typ := binary.LittleEndian.Uint16(d[0:])
	sz := int(binary.LittleEndian.Uint32(d[4:]))
	fmt.Printf("=== chunk_27 pkt0 type=%d size=%d ===\n", typ, sz)
	pl := d[16:]
	if sz > 0 && 16+sz <= len(d) {
		pl = d[16 : 16+sz]
	}
	total := len(pl) * 8

	type hit struct {
		bit int
		x   uint64
		end string
	}
	var hits []hit
	countBE := map[uint64]int{}
	countLE := map[uint64]int{}
	for bp := 0; bp+64 <= total; bp++ {
		var v uint64
		for i := 0; i < 64; i++ {
			q := bp + i
			v = (v << 1) | uint64((pl[q>>3]>>uint(7-(q&7)))&1)
		}
		if xset[v] {
			hits = append(hits, hit{bp, v, "BE"})
			countBE[v]++
		}
		if sw := bswap(v); xset[sw] {
			hits = append(hits, hit{bp, sw, "LE"})
			countLE[sw]++
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].bit < hits[j].bit })

	fmt.Printf("\n=== occurrences xuid (64b, MSB-first scan) : %d total ===\n", len(hits))
	fmt.Println("  xuid                 | occ BE | occ LE | highlight (K+D+medals)")
	for _, x := range xuids {
		fmt.Printf("  %-16s | %-6d | %-6d | %d\n", nm(x), countBE[x], countLE[x], hl[x])
	}

	fmt.Printf("\n=== CLUSTERS (>=2 xuids distincts dans %d bits) ===\n", clusterBits)
	nClusters := 0
	for i := 0; i < len(hits); i++ {
		distinct := map[uint64]bool{hits[i].x: true}
		var grp []hit
		grp = append(grp, hits[i])
		for j := i + 1; j < len(hits) && hits[j].bit-hits[i].bit <= clusterBits; j++ {
			grp = append(grp, hits[j])
			distinct[hits[j].x] = true
		}
		if len(distinct) >= 2 {
			nClusters++
			if nClusters <= 40 {
				s := ""
				for _, g := range grp {
					s += fmt.Sprintf(" [+%d %s/%s]", g.bit-hits[i].bit, nm(g.x), g.end)
				}
				fmt.Printf("  @bit%-9d (%d distincts):%s\n", hits[i].bit, len(distinct), s)
			}
		}
	}
	fmt.Printf("  total clusters : %d\n", nClusters)
	fmt.Println("\n>>> occ >> highlight => structures riches multi-xuid (kill-event killer/victim/assistant).")
	fmt.Println(">>> occ ~= highlight & 0 cluster => xuids = SEULEMENT les highlight events (records riches non en xuid).")
}

func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }
