// tmp_cleanframe — THROWAWAY : GRADIENT pour le port de la forme prédictive (delta).
// Une frame se décode proprement jusqu'au terminateur recEnd ssi tous ses records
// sont bit-exacts. Mesure : % de frames propres (err==nil) + catégorie d'échec +
// histogramme du composant de 1er désync. À re-lancer après chaque fix : doit monter.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_cleanframe [maxChunk]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

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

type packet struct {
	ts      uint64
	payload []byte
}

func listFrames(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
	dumpPath := cache + "/world_dump.txt"
	if p := os.Getenv("WORLD_DUMP"); p != "" {
		dumpPath = p
	}
	raw, _ := os.ReadFile(dumpPath)
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
	maxChunk := 26
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	totalFrames, clean := 0, 0
	totalRecs, cleanRecsBeforeFail := 0, 0
	desyncComp := map[string]int{} // composant de 1er désync (par record desync)
	failCat := map[string]int{}    // catégorie d'échec de frame
	w := freshWorld(reg)           // PERSISTE across frames ; rollback par frame dans DecodeFrameRecords
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			totalFrames++
			br := filmdec.NewBitReader(fr.payload)
			recs, e := filmdec.DecodeFrameRecords(br, w, calCfg)
			totalRecs += len(recs)
			if e == nil {
				clean++
				cleanRecsBeforeFail += len(recs)
				continue
			}
			cleanRecsBeforeFail += len(recs) - 1 // tous sauf le dernier (qui a échoué)
			msg := e.Error()
			switch {
			case strings.Contains(msg, "invalid record type"):
				failCat["invalid-record-type (drift)"]++
			case strings.Contains(msg, "desync record"):
				failCat["desync-component"]++
			default:
				failCat["autre"]++
			}
			// le dernier record porte le composant de désync
			if len(recs) > 0 {
				last := recs[len(recs)-1]
				if last.DesyncAt >= 0 {
					if arch, ok := reg.Archetype(int(last.TypeIndex)); ok && last.DesyncAt < len(arch.Components) {
						desyncComp[fmt.Sprintf("i%d %s (ti=%d)", last.DesyncAt, arch.Components[last.DesyncAt], last.TypeIndex)]++
					} else {
						desyncComp[fmt.Sprintf("i%d (ti=%d)", last.DesyncAt, last.TypeIndex)]++
					}
				}
			}
		}
	}

	fmt.Printf("=== frames propres (jusqu'à recEnd) : %d / %d (%.2f%%) ===\n", clean, totalFrames, 100*float64(clean)/float64(totalFrames))
	fmt.Printf("records décodés avant échec : %d / %d (%.1f%%)\n\n", cleanRecsBeforeFail, totalRecs, 100*float64(cleanRecsBeforeFail)/float64(max1(totalRecs)))

	fmt.Println("catégorie d'échec de frame :")
	for k, v := range failCat {
		fmt.Printf("  %-32s %d\n", k, v)
	}
	fmt.Println("\ncomposant de 1er désync (top 20) :")
	type kv struct {
		k string
		n int
	}
	var kvs []kv
	for k, n := range desyncComp {
		kvs = append(kvs, kv{k, n})
	}
	for i := 0; i < len(kvs); i++ {
		for j := i + 1; j < len(kvs); j++ {
			if kvs[j].n > kvs[i].n {
				kvs[i], kvs[j] = kvs[j], kvs[i]
			}
		}
	}
	for i, k := range kvs {
		if i >= 20 {
			break
		}
		fmt.Printf("  %5d  %s\n", k.n, k.k)
	}
}

func max1(x int) int {
	if x == 0 {
		return 1
	}
	return x
}
