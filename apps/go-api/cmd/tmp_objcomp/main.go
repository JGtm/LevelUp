// tmp_objcomp — liste des composants des archetypes d objets au sol (ti=42 armes,
// ti=37 equipement, ti=38, ti=41, ti=43). THROWAWAY.
package main

import (
	"fmt"
	"os"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	raw, err := os.ReadFile(dir + "/chunk_00.bin")
	if err != nil {
		fmt.Println("chunk_00:", err)
		return
	}
	reg, err := filmdec.ParseRegistryChunk(raw)
	if err != nil {
		fmt.Println("registre:", err)
		return
	}
	for _, ti := range []int{37, 38, 41, 42, 43} {
		a, ok := reg.Archetype(ti)
		if !ok {
			fmt.Printf("ti=%d absent\n", ti)
			continue
		}
		fmt.Printf("\n=== ti=%d : %d composants ===\n", ti, len(a.Components))
		for i, c := range a.Components {
			fmt.Printf("  i%-2d flags=%d  %s\n", i, a.Level(i), c)
		}
	}
}
