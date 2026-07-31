// tmp_roster — THROWAWAY : explorer le paquet type-8 (roster ~25Ko). But : trouve-t-il les 8 joueurs
// (xuid/gamertag) AVEC leur handle d'entité biped, dans l'ordre des slots ? Si oui -> binding entité→slot.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"unicode/utf16"

	"levelup/go-api/internal/analysis"
)

var h32 = map[uint32]string{}

func buildCat() {
	for id, n := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = n
	}
}
func bitsAtP(p []byte, bp, n int) uint64 {
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

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var xuidToPi = map[uint64]int{
	2535467794760703: 0, 2535437947245250: 1, 2533274823110022: 2, 2533274980284321: 3,
	2533274815845110: 4, 2535444178793711: 5, 2533274882097883: 6, 2533274826120416: 7,
}
var piName = map[int]string{0: "whiteknight2519", 1: "JAVIERLOLITO540", 2: "JGtm", 3: "LORD PEINX13", 4: "IKE ILYA", 5: "Akatsuki fire17", 6: "aldusbroncus", 7: "VitaminA1688"}

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

type pkt struct {
	typ     uint16
	payload []byte
	chunk   int
	off     int
}

func allPkts() []pkt {
	var out []pkt
	for n := 0; n <= 27; n++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, n))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			out = append(out, pkt{typ, d[off+16 : off+16+sz], n, off})
			off += 16 + sz
		}
	}
	return out
}

func utf16le(b []byte) string {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[i*2 : i*2+2])
	}
	for i, c := range u {
		if c == 0 {
			u = u[:i]
			break
		}
	}
	s := string(utf16.Decode(u))
	// nettoyer
	out := ""
	for _, r := range s {
		if r >= 0x20 && r < 0x7f {
			out += string(r)
		}
	}
	return out
}

// scan xuids (LE u64 dans [2e15,3e15]) à n'importe quel offset octet.
func scanXUIDs(d []byte) []struct {
	off  int
	xuid uint64
	pi   int
} {
	var out []struct {
		off  int
		xuid uint64
		pi   int
	}
	for i := 0; i+8 <= len(d); i++ {
		x := binary.LittleEndian.Uint64(d[i:])
		if x > 2e15 && x < 3e15 {
			pi, ok := xuidToPi[x]
			if !ok {
				pi = -1
			}
			out = append(out, struct {
				off  int
				xuid uint64
				pi   int
			}{i, x, pi})
		}
	}
	return out
}

var knownXUIDs = []uint64{
	2535467794760703, 2535437947245250, 2533274823110022, 2533274980284321,
	2533274815845110, 2535444178793711, 2533274882097883, 2533274826120416,
}

func main() {
	pkts := allPkts()

	// 0) où nos 8 xuids CONNUS apparaissent (byte-aligné) : par paquet, combien des 8 + offsets.
	known := map[uint64]int{}
	for i, x := range knownXUIDs {
		known[x] = i
	}
	fmt.Println("=== paquets contenant le PLUS de nos 8 xuids (byte-aligné) ===")
	type hit struct {
		typ   uint16
		chunk int
		off   int
		n     int
		offs  []int
		pis   []int
	}
	var hits []hit
	for _, p := range pkts {
		seen := map[int]bool{}
		var offs []int
		var pis []int
		for i := 0; i+8 <= len(p.payload); i++ {
			x := binary.LittleEndian.Uint64(p.payload[i:])
			if pi, ok := known[x]; ok && !seen[pi] {
				seen[pi] = true
				offs = append(offs, i)
				pis = append(pis, pi)
			}
		}
		if len(seen) >= 3 {
			hits = append(hits, hit{p.typ, p.chunk, p.off, len(seen), offs, pis})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].n > hits[j].n })
	for i, h := range hits {
		if i >= 8 {
			fmt.Printf("  ... (%d paquets avec >=3 xuids)\n", len(hits))
			break
		}
		fmt.Printf("  type=%-2d chunk%02d off=%d : %d/8 xuids ; pis=%v offsets=%v\n", h.typ, h.chunk, h.off, h.n, h.pis, h.offs)
	}
	fmt.Println()

	// 0b) le paquet type-9 : pour chaque xuid, scanner son bloc pour familles d'armes + handles 32b.
	buildCat()
	for _, p := range pkts {
		if p.typ != 9 {
			continue
		}
		// trouver les xuids + leurs offsets, triés
		type xo struct {
			off int
			pi  int
		}
		var xos []xo
		for i := 0; i+8 <= len(p.payload); i++ {
			if pi, ok := known[binary.LittleEndian.Uint64(p.payload[i:])]; ok {
				xos = append(xos, xo{i, pi})
			}
		}
		sort.Slice(xos, func(i, j int) bool { return xos[i].off < xos[j].off })
		fmt.Printf("=== type-9 : bloc par joueur (xuid -> familles d'armes dans [xuid, xuid_suivant]) ===\n")
		for i, x := range xos {
			end := len(p.payload)
			if i+1 < len(xos) {
				end = xos[i+1].off
			}
			fams := map[string]int{}
			for bp := x.off * 8; bp+32 <= end*8; bp++ {
				hi := uint32(bitsAtP(p.payload, bp, 32))
				if nm, ok := h32[hi]; ok {
					fams[nm]++
				}
			}
			// top familles
			type fc struct {
				n string
				c int
			}
			var fcs []fc
			for n, c := range fams {
				fcs = append(fcs, fc{n, c})
			}
			sort.Slice(fcs, func(a, b int) bool { return fcs[a].c > fcs[b].c })
			top := ""
			for j := 0; j < len(fcs) && j < 6; j++ {
				top += fmt.Sprintf(" %s×%d", fcs[j].n, fcs[j].c)
			}
			fmt.Printf("  pi%d(%-16s) bloc[%d..%d] %do : %d familles distinctes ;%s\n",
				x.pi, piName[x.pi], x.off, end, end-x.off, len(fcs), top)
		}
		break
	}
	fmt.Println()
	// distribution des types + repérer type-8
	typeCount := map[uint16]int{}
	typeSize := map[uint16]int{}
	var t8 []pkt
	for _, p := range pkts {
		typeCount[p.typ]++
		if len(p.payload) > typeSize[p.typ] {
			typeSize[p.typ] = len(p.payload)
		}
		if p.typ == 8 {
			t8 = append(t8, p)
		}
	}
	fmt.Println("=== types de paquets (count, taille max) ===")
	var types []uint16
	for t := range typeCount {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
	for _, t := range types {
		fmt.Printf("  type=%-3d count=%-6d sizeMax=%d\n", t, typeCount[t], typeSize[t])
	}

	fmt.Printf("\n=== %d paquet(s) type-8 (roster) ===\n", len(t8))
	for pi, p := range t8 {
		fmt.Printf("\n--- type-8 #%d (chunk%02d, %d octets) ---\n", pi, p.chunk, len(p.payload))
		xs := scanXUIDs(p.payload)
		fmt.Printf("  %d xuid(s) trouvés :\n", len(xs))
		for _, x := range xs {
			tag := "?"
			if x.pi >= 0 {
				tag = fmt.Sprintf("pi%d=%s", x.pi, piName[x.pi])
			}
			// contexte autour : 16 octets avant + gamertag candidat après
			gtA := utf16le(p.payload[max0(x.off-72):x.off])
			gtB := ""
			if x.off+8+64 <= len(p.payload) {
				gtB = utf16le(p.payload[x.off+8 : x.off+8+64])
			}
			// handles 32b candidats juste avant/après le xuid
			var preH, postH uint32
			if x.off >= 4 {
				preH = binary.LittleEndian.Uint32(p.payload[x.off-4:])
			}
			if x.off+8+4 <= len(p.payload) {
				postH = binary.LittleEndian.Uint32(p.payload[x.off+8:])
			}
			fmt.Printf("    off=%-6d xuid=%d (%s)  pre32=0x%08x post32=0x%08x gtPre=%q gtPost=%q\n",
				x.off, x.xuid, tag, preH, postH, trunc(gtA), trunc(gtB))
		}
		// stride entre xuids (roster = entrées régulières ?)
		if len(xs) >= 2 {
			fmt.Printf("  strides entre xuids consécutifs : ")
			for i := 1; i < len(xs); i++ {
				fmt.Printf("%d ", xs[i].off-xs[i-1].off)
			}
			fmt.Println()
		}
		// dump hex des 256 premiers octets pour inspection de structure
		fmt.Printf("  hex[0:128]: %x\n", p.payload[:min(128, len(p.payload))])
	}
}

func max0(x int) int {
	if x < 0 {
		return 0
	}
	return x
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func trunc(s string) string {
	if len(s) > 20 {
		return s[:20]
	}
	return s
}
