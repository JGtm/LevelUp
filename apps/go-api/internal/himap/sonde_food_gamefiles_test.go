//go:build gamefiles

package himap

// SONDE (2026-08-10, non commitee) — identifier les tags `food` (definitions
// d'objets Forge) des volumes du .mvar de Dynasty : lequel est le volume de
// mort ? Discriminant cherche dans les DEPENDANCES et les champs du tag.

import (
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/himodule"
)

func foodCibles() map[uint32]string {
	g := func(v int32) uint32 { return uint32(v) }
	return map[uint32]string{
		g(937132837):   "geante-sous-carte-104x70-z66",
		g(43333489):    "geante-abords-85x96",
		g(-588988541):  "centre-sous-sol-23x43-z72",
		g(-696190206):  "frequent-36x-et-cylindre-250m",
		g(-1751270658): "murs-2.5x124",
		g(95146865):    "zone-ctf_include",
		g(1818458590):  "zone-strongholds",
		g(1020883125):  "zone-firefight",
	}
}

// TestSondeFoodDeps : dependances de chaque tag food candidat, resolues contre
// l'index des modules globals.
func TestSondeFoodDeps(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	byID := map[uint32]string{}
	for _, rel := range []string{
		"any/globals/forge/forge_objects-rtx-new.module",
		"any/globals/globals-rtx-new.module",
		"any/globals/multiplayer-rtx-new.module",
		"any/globals/common-rtx-new.module",
		"any/globals/levels-rtx-new.module",
	} {
		m, err := himodule.Open(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		for _, f := range m.Files("") {
			if _, ok := byID[f.GlobalID]; !ok {
				byID[f.GlobalID] = f.Group
			}
		}
	}
	t.Logf("%d tags indexes", len(byID))

	m, err := himodule.Open(filepath.Join(root, "any", "globals", "forge", "forge_objects-rtx-new.module"))
	if err != nil {
		t.Fatal(err)
	}
	cibles := foodCibles()
	for _, f := range m.Files("food") {
		role, ok := cibles[f.GlobalID]
		if !ok {
			continue
		}
		tag, err := m.Extract(f)
		if err != nil {
			t.Logf("food %d : %v", int32(f.GlobalID), err)
			continue
		}
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			t.Errorf("food %d (%s) : %v", int32(f.GlobalID), role, err)
			continue
		}
		depsTab := ti.blockTab - ti.deps*depEntrySize
		t.Logf("== food %d (%s) : %d o, deps=%d dataBlocks=%d structs=%d",
			int32(f.GlobalID), role, len(tag), ti.deps, ti.dataBlocks, ti.structs)
		for i := 0; i < ti.deps; i++ {
			e := depsTab + i*depEntrySize
			ligne := fmt.Sprintf("   dep[%d]", i)
			for o := 0; o < depEntrySize; o += 4 {
				v := uint32(u32(tag, e+o))
				marque := ""
				if g, ok := byID[v]; ok {
					marque = ":" + g
				}
				ligne += fmt.Sprintf(" %08x%s", v, marque)
			}
			t.Log(ligne)
		}
		// Les data-blocks du root et leurs premiers u32, pour reperer un enum de
		// comportement qui differerait entre zone de mode et volume de mort.
		root, _ := ti.rootBlockIndex()
		abs, size := ti.blockAbs(root)
		ligne := "   root u32:"
		for o := 0; o < size && o < 96; o += 4 {
			ligne += fmt.Sprintf(" %08x", uint32(u32(tag, abs+o)))
		}
		t.Log(ligne)
	}
}
