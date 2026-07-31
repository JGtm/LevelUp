// tmp_regcmp : compare le registre chunk_00 de plusieurs films (archetype biped ti=35).
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis/filmdec"
)

func main() {
	base := `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
	films := os.Args[1:]
	if len(films) == 0 {
		films = []string{"000d5950", "64e8adfa"}
	}
	type sig struct {
		name  string
		comps []string
		flags []uint32
		n     int
	}
	var sigs []sig
	for _, f := range films {
		raw, err := os.ReadFile(filepath.Join(base, f, "chunk_00.bin"))
		if err != nil {
			fmt.Printf("%s: ERR %v\n", f, err)
			continue
		}
		reg, err := filmdec.ParseRegistryChunk(raw)
		if err != nil {
			fmt.Printf("%s: parse ERR %v\n", f, err)
			continue
		}
		a, ok := reg.Archetype(filmdec.BipedTypeIndex)
		if !ok {
			fmt.Printf("%s: pas d'archetype 35 (nb=%d)\n", f, len(reg.Archetypes))
			continue
		}
		fmt.Printf("=== film %s : %d archetypes, ti=35 -> %d composants\n", f, len(reg.Archetypes), len(a.Components))
		for i := 0; i < len(a.Components) && i < 6; i++ {
			fmt.Printf("    i%-2d flags=%-10d (0x%08x) %s\n", i, a.Flags[i], a.Flags[i], a.Components[i])
		}
		sigs = append(sigs, sig{f, a.Components, a.Flags, len(reg.Archetypes)})
	}
	if len(sigs) < 2 {
		return
	}
	fmt.Println()
	ref := sigs[0]
	for _, s := range sigs[1:] {
		fmt.Printf("--- diff %s vs %s : nbArch %d vs %d, nbComp %d vs %d\n",
			ref.name, s.name, ref.n, s.n, len(ref.comps), len(s.comps))
		n := len(ref.comps)
		if len(s.comps) < n {
			n = len(s.comps)
		}
		diff := 0
		for i := 0; i < n; i++ {
			if ref.comps[i] != s.comps[i] || ref.flags[i] != s.flags[i] {
				fmt.Printf("    i%-2d %q/%d  !=  %q/%d\n", i, ref.comps[i], ref.flags[i], s.comps[i], s.flags[i])
				diff++
			}
		}
		if diff == 0 {
			fmt.Printf("    IDENTIQUE sur les %d premiers composants (noms ET flags)\n", n)
		}
	}
	// dump complet de tous les archetypes : hash des flags pour comparer globalement
	fmt.Println()
	for _, f := range films {
		raw, err := os.ReadFile(filepath.Join(base, f, "chunk_00.bin"))
		if err != nil {
			continue
		}
		reg, _ := filmdec.ParseRegistryChunk(raw)
		var h uint64 = 1469598103934665603
		tot := 0
		for _, a := range reg.Archetypes {
			for i := range a.Components {
				tot++
				for _, c := range []byte(a.Components[i]) {
					h = (h ^ uint64(c)) * 1099511628211
				}
				h = (h ^ uint64(a.Flags[i])) * 1099511628211
			}
		}
		fmt.Printf("%s : %d archetypes, %d slots non vides, FNV(noms+flags)=%016x\n", f, len(reg.Archetypes), tot, h)
	}
}
