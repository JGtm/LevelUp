// tmp_archcomps — dump la liste ordonnee des composants (Components[]) des archetypes
// donnes, pour cibler le grind de portage des desers. Usage : go run ./cmd/tmp_archcomps
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	// archetypes presents au keyframe (handoff L3) + blocants observes
	arches := []int{0, 2, 5, 6, 9, 12, 13, 14, 15, 17, 18, 19, 21, 22, 25, 26, 27, 29, 30, 34, 35, 37, 38, 40, 41, 42, 43, 45, 47}
	for _, ai := range arches {
		a, ok := reg.Archetype(ai)
		if !ok {
			continue
		}
		fmt.Printf("=== archetype %d : %d composants ===\n", ai, len(a.Components))
		for i, c := range a.Components {
			fmt.Printf("  i%-2d %s\n", i, c)
		}
	}
}
