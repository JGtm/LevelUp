package main

// wf_b_anchorstab — les ancres de record sont-elles STABLES (même bit-position)
// d'une keyframe à l'autre ? Si oui, le record positionnel i est le MÊME slot
// joueur partout et le cross-link arme+temps est valide. Sinon, l'indexation
// positionnelle est cassée et il faut ancrer autrement.

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
func extractPacket(d []byte, want uint16) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}
func bitAt(d []byte, p int) uint64 {
	if p < 0 || p>>3 >= len(d) {
		return 0
	}
	return uint64((d[p>>3] >> uint(7-(p&7))) & 1)
}
func rb(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		v = (v << 1) | bitAt(d, bp+i)
	}
	return v
}

type wh struct {
	pos  int
	name string
}

func scanW(d []byte) []wh {
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		if c, ok := analysis.WeaponFusionMap[n]; ok {
			n = c
		}
		h2n[uint32(id>>32)] = n
	}
	var out []wh
	tot := len(d) * 8
	for bp := 0; bp+32 <= tot; bp++ {
		if n, ok := h2n[uint32(rb(d, bp, 32))]; ok {
			out = append(out, wh{bp, n})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pos < out[j].pos })
	return out
}

func main() {
	// Imprime, pour chaque keyframe, les ancres (1er littéral de chaque cluster)
	// et l'arme de tête. On regarde si les 8 premières ancres restent ~stables.
	for i := 2; i <= 22; i++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, i))
		pl := extractPacket(d, 2)
		if pl == nil {
			continue
		}
		hits := scanW(pl)
		if len(hits) == 0 {
			continue
		}
		// clusters
		type rec struct {
			anchor int
			ws     []string
		}
		var recs []rec
		cur := rec{anchor: hits[0].pos, ws: []string{hits[0].name}}
		for j := 1; j < len(hits); j++ {
			if hits[j].pos-hits[j-1].pos > 1500 {
				recs = append(recs, cur)
				cur = rec{anchor: hits[j].pos, ws: []string{hits[j].name}}
				continue
			}
			cur.ws = append(cur.ws, hits[j].name)
		}
		recs = append(recs, cur)
		fmt.Printf("chunk_%02d: ", i)
		for k := 0; k < len(recs) && k < 9; k++ {
			fmt.Printf("[%d:%s] ", recs[k].anchor, recs[k].ws[0])
		}
		fmt.Println()
	}

	// Question clé : la 1re région (anchors ~195k-215k) reste-t-elle le slot
	// joueur ? Imprime les anchors du 1er cluster pour chaque chunk.
	fmt.Println("\n=== anchor du R0 (1er cluster) par chunk ===")
	for i := 2; i <= 22; i++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, i))
		pl := extractPacket(d, 2)
		if pl == nil {
			continue
		}
		hits := scanW(pl)
		if len(hits) == 0 {
			continue
		}
		fmt.Printf("  chunk_%02d R0_anchor=%d firstW=%s totalHits=%d\n", i, hits[0].pos, hits[0].name, len(hits))
	}
}
