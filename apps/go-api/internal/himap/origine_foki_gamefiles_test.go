//go:build gamefiles

package himap

// origine_foki_gamefiles_test.go — DESCENDRE D'UN CRAN : ce que le `foki` référence.
//
// POURQUOI. La recette « tag `food` qui référence un `foki` » sépare parfaitement sur les huit
// types étalons, mais appliquée au corpus elle SUR-RETIENT : elle classe 61,5 % des objets de
// Highpower et 60,5 % de Scarr comme points d'apparition. Deux types portent l'anomalie —
// 0x8413E9BA (jusqu'à 178 par carte) et 0xA4EE54ED (83). Le seul comptage ne suffit pas à les
// écarter : le `rack` PROUVÉ monte lui-même à 52 sur Fragmentation Heavies.
//
// CE QUE CETTE SONDE MESURE : pour chaque type retenu par la recette, on extrait les `foki`
// qu'il référence, et on publie les GROUPES que CES `foki` référencent à leur tour. Un point
// d'apparition d'objet ramassable doit mener à un `weap` (arme) ou un `eqip` (équipement) ;
// s'il ne mène qu'à de la géométrie, il n'en est pas un.
//
// LECTURE SEULE sur les fichiers du jeu installé.

import (
	"path/filepath"
	"sort"
	"testing"
)

// origFokiDe rend les GlobalID des `foki` référencés par un type.
func origFokiDe(idxForge, idxRef *ModuleIndex, typeID uint32) []uint32 {
	tag, err := idxForge.Extract(typeID)
	if err != nil {
		return nil
	}
	var out []uint32
	vus := map[uint32]bool{}
	RefsInline(tag, func(h uint32) bool {
		if g, _, ok := idxRef.Lookup(h); ok && g == GroupeRamassable && !vus[h] {
			vus[h] = true
			out = append(out, h)
		}
		return false
	})
	return out
}

// TestOrigineFokiVersQuoi publie, par type retenu, les groupes atteints via ses `foki`.
func TestOrigineFokiVersQuoi(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	idxForge := origIndexForge(t, racine)
	modCarte := moduleDuJeu(t, "pc", "catalyst")
	geo, _ := GeometrySearchPath(racine, modCarte)
	idxRef, err := NewModuleIndex(append(append([]string{}, geo...),
		filepath.Join(racine, "any", "globals", "forge", "forge_objects-rtx-new.module"))...)
	if err != nil {
		t.Skipf("index de référence indisponible : %v", err)
	}
	// Les types mesurés sur le corpus, du plus suspect au plus étalonné.
	sondes := []struct {
		id   uint32
		note string
	}{
		{0x8413E9BA, "ABERRANT — jusqu'a 178 par carte (Highpower)"},
		{0xA4EE54ED, "ABERRANT — jusqu'a 83 par carte (Breaker)"},
		{0x6253CFC0, "PROUVE rack — jusqu'a 52 (Fragmentation)"},
		{0x5F379533, "PROUVE power — max 5"},
		{0x5E86D110, "PROUVE powerup — max 3"},
		{0xADEEE6D8, "CANDIDAT mesure — max 16"},
		{0xE42158DF, "CANDIDAT mesure — max 26"},
		{0x0CD504B0, "retenu — max 16 (Forest)"},
		{0x76110919, "retenu — present 15/15"},
		{0x11CBFF52, "retenu — 1 par carte, 15/15"},
	}
	t.Logf("== CE QUE LE `foki` MENE · index de reference %d entrees ==", idxRef.Taille())
	for _, s := range sondes {
		fokis := origFokiDe(idxForge, idxRef, s.id)
		compte := map[string]int{}
		for _, f := range fokis {
			tag, err := idxRef.Extract(f)
			if err != nil {
				compte["(inextractible)"]++
				continue
			}
			RefsInline(tag, func(h uint32) bool {
				if g, _, ok := idxRef.Lookup(h); ok {
					compte[g]++
				}
				return false
			})
		}
		keys := make([]string, 0, len(compte))
		for g := range compte {
			keys = append(keys, g)
		}
		sort.Slice(keys, func(i, j int) bool { return compte[keys[i]] > compte[keys[j]] })
		s2 := ""
		for i, g := range keys {
			if i >= 10 {
				break
			}
			if i > 0 {
				s2 += " "
			}
			s2 += g + ":" + itoaSimple(compte[g])
		}
		t.Logf("0x%08X  %d foki -> %s", s.id, len(fokis), s2)
		t.Logf("            %s", s.note)
	}
	t.Log("LECTURE : si les types ABERRANTS ne menent pas a `weap`/`eqip` la ou les PROUVES y " +
		"menent, la recette gagne un second crible et cesse de sur-retenir.")
}
