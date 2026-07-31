package main

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis"
)

// dumpCatalog prints the weapon catalogue, grouped by low32 suffix, to understand
// which low32 is the canonical "real weapon" marker and whether Rushdown Hammer
// (low32 0xd8d07ca1) is a genuine weapon family or a skin/variant collision.
func dumpCatalog() {
	fmt.Printf("\n================ CATALOGUE analysis.WeaponIDToName ================\n")
	type e struct {
		id   uint64
		name string
	}
	var es []e
	bySuffix := map[uint32]int{}
	for id, n := range analysis.WeaponIDToName {
		es = append(es, e{id, n})
		bySuffix[uint32(id)]++
	}
	sort.Slice(es, func(i, j int) bool {
		if uint32(es[i].id) != uint32(es[j].id) {
			return uint32(es[i].id) < uint32(es[j].id)
		}
		return es[i].name < es[j].name
	})
	fmt.Printf("%d entrées au catalogue\n", len(es))
	fmt.Printf("\nRépartition par low32 (suffixe) :\n")
	type sc struct {
		suf uint32
		n   int
	}
	var scs []sc
	for s, n := range bySuffix {
		scs = append(scs, sc{s, n})
	}
	sort.Slice(scs, func(i, j int) bool { return scs[i].n > scs[j].n })
	for _, s := range scs {
		fmt.Printf("  low32=0x%08x : %d armes\n", s.suf, s.n)
	}
	fmt.Printf("\nDétail (id64 => nom) :\n")
	for _, x := range es {
		fmt.Printf("  0x%016x  high32=0x%08x low32=0x%08x  %s\n", x.id, uint32(x.id>>32), uint32(x.id), x.name)
	}
}
