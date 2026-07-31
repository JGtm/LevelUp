package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

type wpos struct {
	Pos  int
	Name string
}

func scanWeapons(d []byte) []wpos {
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		h2n[uint32(id>>32)] = n
	}
	var out []wpos
	tot := len(d) * 8
	for bp := 0; bp+32 <= tot; bp++ {
		if n, ok := h2n[uint32(rb(d, bp, 32))]; ok {
			out = append(out, wpos{bp, n})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Pos < out[j].Pos })
	return out
}

func loadKFBytes(n int) []byte {
	p := filepath.Join(cache, fmt.Sprintf("chunk_%02d.bin", n))
	d := inflate(p)
	return extractPacket(d, 2)
}

func main() {
	// Count live records (pairs spaced ~2800) per keyframe across the whole game
	// to show the count fluctuates well below 8 -> region shows a subset, not all players.
	fmt.Println("Per-keyframe live loadout-record count (region 150K-260K):")
	for n := 2; n <= 26; n++ {
		b := loadKFBytes(n)
		if b == nil {
			continue
		}
		ws := scanWeapons(b)
		cnt := 0
		prev := -100000
		for _, w := range ws {
			if w.Pos < 150000 || w.Pos > 260000 {
				continue
			}
			if w.Pos-prev > 1000 {
				cnt++
			}
			prev = w.Pos
		}
		fmt.Printf("  chunk_%02d (~%3ds): %d live records\n", n, (n-2)*20, cnt)
	}
}
