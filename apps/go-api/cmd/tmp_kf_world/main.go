// tmp_kf_world — THROWAWAY : le keyframe type-2 d'un film neuf construit-il le World
// (slots->archétype) via DecodeFrameRecords ? Test de faisabilité du bootstrap offline
// sans world_dump CE. Sweep idLowBits ; rapporte records décodés, slots bindés,
// distribution des typeIndex (35=biped), 1er désync.
//
// Usage : tmp_kf_world <dir_chunks>
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

	"levelup/go-api/internal/analysis/filmdec"
)

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func extractByType(d []byte, want uint16) []byte {
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

func chunkNum(p string) int {
	b := filepath.Base(p)
	n, st := 0, false
	for _, c := range b {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
			st = true
		} else if st {
			break
		}
	}
	return n
}

func main() {
	dir := "internal/sync/testdata/jgtm_full_match/chunks"
	if len(os.Args) >= 2 {
		dir = os.Args[1]
	}
	files, _ := filepath.Glob(filepath.Join(dir, "chunk_*.bin"))
	sort.Slice(files, func(i, j int) bool { return chunkNum(files[i]) < chunkNum(files[j]) })
	reg, err := filmdec.ParseRegistryChunk(inflate(files[0]))
	if err != nil {
		panic(err)
	}

	// keyframe type-2 + le type-1 (gros snapshot initial) — on teste les deux
	var kf2, kf1 []byte
	for _, f := range files[1:] {
		d := inflate(f)
		if kf2 == nil {
			kf2 = extractByType(d, 2)
		}
		if kf1 == nil {
			kf1 = extractByType(d, 1)
		}
		if kf2 != nil && kf1 != nil {
			break
		}
	}
	filmdec.SetRecordStateParam(2)

	for _, tc := range []struct {
		name string
		data []byte
	}{{"type-2 keyframe", kf2}, {"type-1 snapshot", kf1}} {
		if tc.data == nil {
			fmt.Printf("\n### %s : absent\n", tc.name)
			continue
		}
		fmt.Printf("\n### %s : %d octets (%d bits) ###\n", tc.name, len(tc.data), len(tc.data)*8)
		fmt.Printf("%-8s %-8s %-8s %-8s %-8s %s\n", "idLow", "records", "bound", "bipeds", "endBit", "typeIdxHist(top)")
		for idLow := 8; idLow <= 16; idLow++ {
			w := filmdec.NewWorld(reg)
			br := filmdec.NewBitReader(tc.data)
			recs, _ := filmdec.DecodeFrameRecords(br, w, filmdec.FrameConfig{HasExtraFields: false, IDLowBits: idLow})
			tiHist := map[uint32]int{}
			bipeds := 0
			for _, r := range recs {
				tiHist[r.TypeIndex]++
				if r.TypeIndex == 35 {
					bipeds++
				}
			}
			// top typeIndex
			type kv struct {
				ti uint32
				c  int
			}
			var arr []kv
			for ti, c := range tiHist {
				arr = append(arr, kv{ti, c})
			}
			sort.Slice(arr, func(a, b int) bool { return arr[a].c > arr[b].c })
			top := ""
			for k := 0; k < len(arr) && k < 6; k++ {
				top += fmt.Sprintf("%d:%d ", arr[k].ti, arr[k].c)
			}
			fmt.Printf("%-8d %-8d %-8d %-8d %-8d %s\n", idLow, len(recs), w.Bound(), bipeds, br.BitPos(), top)
		}
	}
}
