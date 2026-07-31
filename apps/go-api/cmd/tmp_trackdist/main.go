// tmp_trackdist — THROWAWAY : suit les armes mono-propriétaire (Hydra=whiteknight,
// Bulldog=VitaminA, Skewer/Ravager=akatsuki) à travers les keyframes et imprime, pour
// le record qui les porte, les champs candidats player-index (weapon1-16, weapon2-16,
// position). Si un champ est STABLE pour une arme donnée -> c'est le player-index.
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
func extractType2(d []byte) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
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

type rec struct {
	pos    int
	p1, p2 int
	w1, w2 string
}

func records(payload []byte) []rec {
	h2n := map[uint32]string{}
	for id, n := range analysis.WeaponIDToName {
		h2n[uint32(id>>32)] = n
	}
	type lit struct {
		pos  int
		name string
	}
	var lits []lit
	tot := len(payload) * 8
	for bp := 0; bp+32 <= tot; bp++ {
		if n, ok := h2n[uint32(rb(payload, bp, 32))]; ok {
			lits = append(lits, lit{bp, n})
		}
	}
	sort.Slice(lits, func(i, j int) bool { return lits[i].pos < lits[j].pos })
	var recs []rec
	pos := 0
	for i := 0; i+1 < len(lits); i++ {
		g := lits[i+1].pos - lits[i].pos
		if g > 50 && g < 500 && lits[i].pos < 350000 {
			recs = append(recs, rec{pos, lits[i].pos, lits[i+1].pos, lits[i].name, lits[i+1].name})
			pos++
			i++
		}
	}
	return recs
}

func main() {
	dist := map[string]string{
		"MLRS-2 Hydra": "whiteknight", "CQS48 Bulldog": "VitaminA",
		"Skewer": "akatsuki", "Ravager": "akatsuki", "Cindershot": "JGtm?",
	}
	fmt.Printf("%-12s %-5s %-14s %-14s | pos  w1-16 w2-16 w1-12hi w2-12hi w1+67\n", "keyframe", "~s", "w1", "w2")
	for n := 2; n <= 26; n++ {
		name := fmt.Sprintf("chunk_%02d.bin", n)
		payload := extractType2(inflate(cache + "/" + name))
		if payload == nil {
			continue
		}
		for _, r := range records(payload) {
			owner := ""
			if o, ok := dist[r.w1]; ok {
				owner = o + "(w1)"
			} else if o, ok := dist[r.w2]; ok {
				owner = o + "(w2)"
			}
			if owner == "" {
				continue
			}
			w1m16 := rb(payload, r.p1-16, 8)
			w2m16 := rb(payload, r.p2-16, 8)
			w1m12hi := rb(payload, r.p1-12, 8) >> 4
			w2m12hi := rb(payload, r.p2-12, 8) >> 4
			w1p67 := rb(payload, r.p1+67, 5)
			fmt.Printf("%-12s %4d %-14s %-14s | %3d  0x%02x  0x%02x   %2d      %2d     %2d   [%s]\n",
				name, (n-2)*20, r.w1, r.w2, r.pos, w1m16, w2m16, w1m12hi, w2m12hi, w1p67, owner)
		}
	}
}
