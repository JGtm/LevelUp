// tmp_kfpos — THROWAWAY : rendement des KEYFRAMES type-2 pour les positions
// full-state. Pour chaque chunk 00..27 : extrait le payload type-2, décode les
// positions (comb-anchor, package positions), imprime nb positions + timestamp.
// But : cadence keyframe + yield par keyframe (~8 = tous les joueurs ?) → décider
// si "tous les 8 joueurs en coarse" via keyframes est viable.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfpos [filmDir]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/positions"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

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

// allBlocks renvoie (type, payload, ts) de tous les blocs d'un chunk décompressé.
func allBlocks(d []byte) []struct {
	typ uint16
	ts  uint64
	pay []byte
} {
	var out []struct {
		typ uint16
		ts  uint64
		pay []byte
	}
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		out = append(out, struct {
			typ uint16
			ts  uint64
			pay []byte
		}{typ, ts, d[off+16 : off+16+sz]})
		off += 16 + sz
	}
	return out
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	totalKF, totalPos := 0, 0
	for idx := 0; idx <= 27; idx++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx))
		if d == nil {
			continue
		}
		for _, b := range allBlocks(d) {
			if b.typ != 2 {
				continue
			}
			tms := 0
			if b.ts >= t0Us {
				tms = int((b.ts - t0Us) / 1000)
			}
			// DecodeKeyframePositions attend un ChunkInput dont Data porte un
			// bloc type-2 ; on enrobe le payload dans un mini-conteneur.
			wrap := make([]byte, 16+len(b.pay))
			binary.LittleEndian.PutUint16(wrap[0:], 2)
			binary.LittleEndian.PutUint32(wrap[4:], uint32(len(b.pay)))
			copy(wrap[16:], b.pay)
			pos := positions.DecodeKeyframePositions([]positions.ChunkInput{{Data: wrap, StartMS: tms, ChunkType: 2}})
			totalKF++
			totalPos += len(pos)
			fmt.Printf("chunk_%02d  type-2  t=%6dms  payload=%6do  positions=%d\n", idx, tms, len(b.pay), len(pos))
			for _, p := range pos {
				fmt.Printf("      (%.1f, %.1f, %.1f) team=%d\n", p.X, p.Y, p.Z, p.Team)
			}
		}
	}
	fmt.Printf("\nTOTAL : %d keyframes type-2, %d positions (moy %.1f/keyframe)\n",
		totalKF, totalPos, float64(totalPos)/float64(max(1, totalKF)))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
