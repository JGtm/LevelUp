// tmp_i63debug — THROWAWAY : capture les séquences de tags i63 (biped-action-component) des records
// qui DÉSYNC (count1>0, finit sur tag>5) vs ceux qui sont CLEAN, pour identifier le tag/forme à corriger.
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

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

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
func listFrames(d []byte) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
}
func freshWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

func main() {
	filmdec.SetRecordStateParam(2)
	filmdec.PositionCalibratedSkip = true
	for n, w := range map[string]int{
		"object-forward-and-up-component": 9, "object-angular-velocity-component": 1,
		"object-shield-vitality-component": 29, "object-region-state-component": 358,
		"object-multiplayer-properties-component": 334,
	} {
		filmdec.SetCalibratedWidth(n, w)
	}
	filmdec.SetBipedActionDebug(true)
	reg, _ := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))

	for idx := 2; idx <= 8; idx++ { // quelques chunks suffisent
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			filmdec.DecodeFrameRecords(filmdec.NewBitReader(fr), w, calCfg)
		}
	}

	fmt.Printf("=== %d séquences BAD (désync) ; %d séquences OK (clean count1>0) ===\n\n",
		len(filmdec.BiBadSeqs), len(filmdec.BiOkSeqs))

	// distribution du DERNIER tag des séquences BAD (= celui qui désync)
	lastBad := map[uint64]int{}
	for _, s := range filmdec.BiBadSeqs {
		if len(s) > 0 {
			lastBad[s[len(s)-1]]++
		}
	}
	fmt.Println("=== dernier tag des séquences BAD (celui sur lequel ça désync) ===")
	printTagDist(lastBad)

	// distribution de TOUS les tags dans OK (tags valides observés)
	allOk := map[uint64]int{}
	for _, s := range filmdec.BiOkSeqs {
		for _, t := range s {
			allOk[t]++
		}
	}
	fmt.Println("\n=== tous les tags des séquences OK (tags valides 0-5 qui marchent) ===")
	printTagDist(allOk)

	// échantillons de séquences BAD
	fmt.Println("\n=== échantillons de séquences BAD (jusqu'au désync) ===")
	for i, s := range filmdec.BiBadSeqs {
		if i >= 15 {
			break
		}
		fmt.Printf("  %v\n", s)
	}
}

func printTagDist(m map[uint64]int) {
	type kv struct {
		t uint64
		c int
	}
	var kvs []kv
	for t, c := range m {
		kvs = append(kvs, kv{t, c})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].c > kvs[j].c })
	for _, e := range kvs {
		flag := ""
		if e.t > 5 {
			flag = " <<< INVALIDE (>5, dispatch erreur) = lecture désalignée"
		}
		fmt.Printf("  tag=%-3d : %d%s\n", e.t, e.c, flag)
	}
}
