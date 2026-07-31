// tmp_archlist — THROWAWAY : dumpe les 118 archetypes du registre + leurs composants,
// pour localiser l'archetype de l'EVENEMENT KILL (kill-feed/death/damage/medal) que
// FUN_1406730c4 lit (event+0x04 victime, +0x08 tueur, +0x538 arme). On cherche un
// archetype "event" porteur d'un composant kill-feed/damage.
//
// Usage : tmp_archlist [filtre]   (filtre = sous-chaine ; defaut : kill|death|damage|event|feed|medal|report)
package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"strings"

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
	filters := []string{"kill", "death", "damage", "event", "feed", "medal", "report", "dead"}
	if len(os.Args) >= 2 {
		filters = strings.Split(os.Args[1], "|")
	}
	fmt.Printf("registre : %d archetypes\n\n", len(reg.Archetypes))
	for i := 0; i < len(reg.Archetypes); i++ {
		arch, _ := reg.Archetype(i)
		if len(arch.Components) == 0 {
			continue
		}
		// nom d'archetype = on n'a pas de nom direct ; on liste les composants et on
		// marque ceux qui matchent un filtre.
		var hits []string
		for _, c := range arch.Components {
			lc := strings.ToLower(c)
			for _, f := range filters {
				if f != "" && strings.Contains(lc, f) {
					hits = append(hits, c)
					break
				}
			}
		}
		if len(hits) == 0 {
			continue
		}
		fmt.Printf("=== archetype #%d (%d composants) — 1er composant = %s ===\n", i, len(arch.Components), arch.Components[0])
		for j, c := range arch.Components {
			mark := ""
			lc := strings.ToLower(c)
			for _, f := range filters {
				if f != "" && strings.Contains(lc, f) {
					mark = "   <<<"
					break
				}
			}
			fmt.Printf("    i%-2d %s%s\n", j, c, mark)
		}
		fmt.Println()
	}
}
