// tmp_i63tags — THROWAWAY : capture les séquences de tags i63 (count1>0) des records
// biped qui DÉSYNC vs ceux qui sont CLEAN, sur 000d5950 chunk_03. Le tag juste avant un
// tag>=12 dans une séquence "bad" = la largeur de sous-deser fautive.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
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

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		var slot, ti uint32
		if _, e := fmt.Sscanf(string(tok), "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

func main() {
	filmdec.SetRecordStateParam(2)
	filmdec.SetBipedActionDebug(true)
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

	d := inflate(cache + "/chunk_03.bin")
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			w := freshWorld(reg)
			filmdec.DecodeFrameRecords(filmdec.NewBitReader(d[off+16:off+16+sz]), w, cfg)
		}
		off += 16 + sz
	}

	fmt.Printf("=== %d séquences BAD (désync, finit en tag>=12) ===\n", len(filmdec.BiBadSeqs))
	suspect := map[uint64]int{} // tag juste avant le >=12
	firstBad := 0
	for i, s := range filmdec.BiBadSeqs {
		if i < 25 {
			fmt.Printf("  bad: %v\n", s)
		}
		if len(s) >= 2 {
			suspect[s[len(s)-2]]++
		} else {
			firstBad++ // le 1er tag est déjà >=12 (désalignement amont)
		}
	}
	fmt.Printf("\n  -- tag juste AVANT le tag>=12 (= largeur suspecte) --\n")
	type kv struct {
		k uint64
		v int
	}
	var arr []kv
	for k, v := range suspect {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
	for _, e := range arr {
		fmt.Printf("    tag %d : %d fois\n", e.k, e.v)
	}
	fmt.Printf("  séquences où le 1ER tag est déjà >=12 (désalignement AMONT, pas un tag) : %d\n", firstBad)

	fmt.Printf("\n=== %d séquences OK (clean, count1>0) ===\n", len(filmdec.BiOkSeqs))
	okTags := map[uint64]int{}
	for i, s := range filmdec.BiOkSeqs {
		if i < 15 {
			fmt.Printf("  ok : %v\n", s)
		}
		for _, t := range s {
			okTags[t]++
		}
	}
	fmt.Printf("  -- distribution des tags dans les séquences OK --\n")
	var arr2 []kv
	for k, v := range okTags {
		arr2 = append(arr2, kv{k, v})
	}
	sort.Slice(arr2, func(a, b int) bool { return arr2[a].k < arr2[b].k })
	for _, e := range arr2 {
		fmt.Printf("    tag %d : %d\n", e.k, e.v)
	}
}
