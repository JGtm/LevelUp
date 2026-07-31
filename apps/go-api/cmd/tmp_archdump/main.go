// tmp_archdump — liste les composants ordonnés du biped #35 (registry chunk_00).
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"

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

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	ai := 35
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &ai)
	}
	arch, ok := reg.Archetype(ai)
	if !ok {
		fmt.Printf("pas d'archetype #%d\n", ai)
		return
	}
	fmt.Printf("archetype #%d : %d composants\n", ai, len(arch.Components))
	for i, c := range arch.Components {
		fmt.Printf("  i%-2d %s\n", i, c)
	}
}
