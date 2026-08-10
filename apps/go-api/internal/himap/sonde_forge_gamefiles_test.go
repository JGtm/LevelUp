package himap

// SONDE (2026-08-10) — F1 du plan PLAN_PORT_TRIANGLES_GO.md §4 : le tag `food`
// (definition d'objet Forge) porte-t-il, ou reference-t-il, le MODELE a dessiner ?
//
// Mesure sur les objets de VAGABOND (vagabond_map.mvar du dump, 4 709 objets) :
// chaque type_id se resout-il en tag food, et les dependances de ce tag menent-elles
// a un groupe de geometrie (rtgo, mode, hlmt...) ? On publie l'histogramme des groupes
// de dependance — on ne devine pas le groupe porteur, on le lit.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himodule"
)

func fourCCDe(v uint32) string {
	b := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
	for _, c := range b {
		if c < 0x20 || c > 0x7e {
			return fmt.Sprintf("%08x", v)
		}
	}
	return string(b)
}

// indexTousModules indexe GlobalID -> groupe sur les modules globaux connus.
func indexTousModules(t *testing.T, root string) map[uint32]string {
	t.Helper()
	byID := map[uint32]string{}
	for _, rel := range []string{
		"any/globals/forge/forge_objects-rtx-new.module",
		"any/globals/globals-rtx-new.module",
		"any/globals/multiplayer-rtx-new.module",
		"any/globals/common-rtx-new.module",
		"any/globals/levels-rtx-new.module",
		"pc/globals/forge/forge_objects-rtx-new.module",
		"pc/globals/globals-rtx-new.module",
		"pc/globals/multiplayer-rtx-new.module",
		"pc/globals/common-rtx-new.module",
		"pc/globals/levels-rtx-new.module",
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
	return byID
}

func TestSondeForgeF1VagabondModele(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	chemin, err := cheminDepuisDepot(".ai/re_dump/mapvar/vagabond_map.mvar")
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatal(err)
	}
	v, err := mapvar.Parse(brut)
	if err != nil {
		t.Fatal(err)
	}
	types := map[int32]int{}
	for _, o := range v.Objects {
		types[o.TypeID]++
	}
	t.Logf("vagabond : %d objets, %d type_id distincts", len(v.Objects), len(types))

	forge, err := himodule.Open(filepath.Join(root, "any", "globals", "forge", "forge_objects-rtx-new.module"))
	if err != nil {
		t.Fatal(err)
	}
	foodParID := map[uint32]himodule.File{}
	for _, f := range forge.Files("food") {
		foodParID[f.GlobalID] = f
	}
	byID := indexTousModules(t, root)
	t.Logf("%d tags food au module forge · %d tags indexes pour la resolution", len(foodParID), len(byID))

	// Pour chaque type_id : food trouve ? deps ? groupes des deps ?
	resolus, sansFood, sansDep := 0, 0, 0
	groupesTypes := map[string]int{}  // groupe -> nb de type_id ayant >= 1 dep de ce groupe
	groupesObjets := map[string]int{} // idem pondere par le nombre d'objets places
	exemples := map[string][]int32{}
	for id, nObj := range types {
		f, ok := foodParID[uint32(id)]
		if !ok {
			sansFood++
			t.Logf("  type_id %d : AUCUN tag food (x%d objets)", id, nObj)
			continue
		}
		resolus++
		tag, err := forge.Extract(f)
		if err != nil {
			t.Errorf("food %d : extraction : %v", id, err)
			continue
		}
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			t.Errorf("food %d : %v", id, err)
			continue
		}
		if ti.deps == 0 {
			sansDep++
			continue
		}
		depsTab := ti.blockTab - ti.deps*depEntrySize
		vus := map[string]bool{}
		for i := 0; i < ti.deps; i++ {
			e := depsTab + i*depEntrySize
			grp := fourCCDe(uint32(u32(tag, e)))
			depID := uint32(u32(tag, e+8))
			cible, connu := byID[depID]
			if connu {
				grp = grp + "->" + cible
			}
			if !vus[grp] {
				vus[grp] = true
				groupesTypes[grp]++
				groupesObjets[grp] += nObj
				if len(exemples[grp]) < 3 {
					exemples[grp] = append(exemples[grp], id)
				}
			}
		}
	}
	t.Logf("type_id resolus en food : %d/%d · sans tag food : %d · food SANS dependance : %d",
		resolus, len(types), sansFood, sansDep)
	var noms []string
	for g := range groupesTypes {
		noms = append(noms, g)
	}
	sort.Slice(noms, func(i, j int) bool { return groupesTypes[noms[i]] > groupesTypes[noms[j]] })
	for _, g := range noms {
		t.Logf("  dep %-14s : %4d type_id · %5d objets places · ex. %v", g, groupesTypes[g], groupesObjets[g], exemples[g])
	}
}

// TestSondeForgeF1RootHash — les deps sont vides (457/467) : le modele est-il reference par
// le champ a root+0x08 (le « hash non craque » de l'investigation) ? On le resout contre
// TOUS les modules de l'installation, pas seulement les globals.
func TestSondeForgeF1RootHash(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	// Index COMPLET : tous les .module de l'installation.
	byID := map[uint32]string{}
	modules := 0
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".module" {
			return nil
		}
		m, err := himodule.Open(p)
		if err != nil {
			return nil
		}
		modules++
		base := filepath.Base(p)
		for _, f := range m.Files("") {
			if _, ok := byID[f.GlobalID]; !ok {
				byID[f.GlobalID] = f.Group + "@" + base
			}
		}
		return nil
	})
	t.Logf("%d modules · %d tags indexes", modules, len(byID))

	// Inventaire des groupes du module forge : ou une table food -> modele pourrait-elle vivre ?
	forge, err := himodule.Open(filepath.Join(root, "any", "globals", "forge", "forge_objects-rtx-new.module"))
	if err != nil {
		t.Fatal(err)
	}
	groupes := map[string]int{}
	for _, f := range forge.Files("") {
		groupes[f.Group]++
	}
	var noms []string
	for g := range groupes {
		noms = append(noms, g)
	}
	sort.Slice(noms, func(i, j int) bool { return groupes[noms[i]] > groupes[noms[j]] })
	for _, g := range noms {
		t.Logf("  groupe %-6q : %5d tags au module forge", g, groupes[g])
	}

	// Le champ root+0x08 des food de Vagabond : se resout-il quelque part ?
	chemin, err := cheminDepuisDepot(".ai/re_dump/mapvar/vagabond_map.mvar")
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatal(err)
	}
	v, err := mapvar.Parse(brut)
	if err != nil {
		t.Fatal(err)
	}
	types := map[int32]bool{}
	for _, o := range v.Objects {
		types[o.TypeID] = true
	}
	foodParID := map[uint32]himodule.File{}
	for _, f := range forge.Files("food") {
		foodParID[f.GlobalID] = f
	}
	resolus, testes := map[string]int{}, 0
	exemples := map[string][]string{}
	for id := range types {
		f, ok := foodParID[uint32(id)]
		if !ok {
			continue
		}
		tag, err := forge.Extract(f)
		if err != nil {
			continue
		}
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			continue
		}
		abs, size := ti.blockAbs(mustRoot(ti))
		if size < 12 {
			continue
		}
		testes++
		h := uint32(u32(tag, abs+8))
		if cible, ok := byID[h]; ok {
			resolus[cible]++
			if len(exemples[cible]) < 3 {
				exemples[cible] = append(exemples[cible], fmt.Sprintf("%d->%08x", id, h))
			}
		}
	}
	total := 0
	for cible, n := range resolus {
		total += n
		t.Logf("  root+0x08 -> %-28s : %d type_id · ex. %v", cible, n, exemples[cible])
	}
	t.Logf("root+0x08 : %d/%d food de Vagabond se resolvent dans l'installation", total, testes)
}

// TestSondeForgeF1ScanInline — les deps sont vides et root+0x08 est l'auto-reference :
// le lien vers le modele est-il un champ INLINE quelque part dans le tag food ?
// Methode : scanner chaque u32 de chaque tag food de Vagabond contre l'index complet de
// l'installation, et publier par OFFSET (dans le bloc porteur) la part de tags qui resolvent,
// et vers quel groupe. Un offset constant qui resout massivement = le champ de jointure —
// c'est la methode qui a etabli type_id -> food.
func TestSondeForgeF1ScanInline(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	byID := map[uint32]string{}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".module" {
			return nil
		}
		m, err := himodule.Open(p)
		if err != nil {
			return nil
		}
		for _, f := range m.Files("") {
			if f.Group == "" {
				continue
			}
			if _, ok := byID[f.GlobalID]; !ok {
				byID[f.GlobalID] = f.Group
			}
		}
		return nil
	})
	chemin, err := cheminDepuisDepot(".ai/re_dump/mapvar/vagabond_map.mvar")
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatal(err)
	}
	v, err := mapvar.Parse(brut)
	if err != nil {
		t.Fatal(err)
	}
	types := map[int32]bool{}
	for _, o := range v.Objects {
		types[o.TypeID] = true
	}
	forge, err := himodule.Open(filepath.Join(root, "any", "globals", "forge", "forge_objects-rtx-new.module"))
	if err != nil {
		t.Fatal(err)
	}
	foodParID := map[uint32]himodule.File{}
	for _, f := range forge.Files("food") {
		foodParID[f.GlobalID] = f
	}
	// offset (dans le TAG complet, pas par bloc — le format est stable a 1 388 o) ->
	// groupe -> compte. On ignore l'auto-reference.
	type cle struct {
		off int
		grp string
	}
	compte := map[cle]int{}
	testes := 0
	for id := range types {
		f, ok := foodParID[uint32(id)]
		if !ok {
			continue
		}
		tag, err := forge.Extract(f)
		if err != nil {
			continue
		}
		testes++
		for o := 0; o+4 <= len(tag); o += 4 {
			h := uint32(u32(tag, o))
			if h == f.GlobalID || h == 0 || h == 0xffffffff || h == 0xbcbcbcbc {
				continue
			}
			if g, ok := byID[h]; ok {
				compte[cle{o, g}]++
			}
		}
	}
	var cles []cle
	for k := range compte {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return compte[cles[i]] > compte[cles[j]] })
	t.Logf("%d food testes · %d couples (offset, groupe) resolvants", testes, len(cles))
	for i, k := range cles {
		if i >= 25 {
			break
		}
		t.Logf("  off 0x%04x -> %-6q : %4d/%d (%.0f %%)", k.off, k.grp, compte[k], testes, 100*float64(compte[k])/float64(testes))
	}
}

// TestSondeForgeF1CouvertureRtgo — LA mesure de F1 : combien d'objets PLACES de Vagabond
// ont un food qui reference au moins un tag de geometrie (rtgo), n'importe ou dans ses
// octets ? Ponderation par le nombre d'objets, et separation forme/sans-forme (les volumes
// de mode et de mort n'ont legitimement PAS de modele).
func TestSondeForgeF1CouvertureRtgo(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	parGroupe := map[string]map[uint32]bool{}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(p) != ".module" {
			return nil
		}
		m, err := himodule.Open(p)
		if err != nil {
			return nil
		}
		for _, f := range m.Files("") {
			if f.Group == "" {
				continue
			}
			if parGroupe[f.Group] == nil {
				parGroupe[f.Group] = map[uint32]bool{}
			}
			parGroupe[f.Group][f.GlobalID] = true
		}
		return nil
	})
	chemin, err := cheminDepuisDepot(".ai/re_dump/mapvar/vagabond_map.mvar")
	if err != nil {
		t.Skip(err)
	}
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Fatal(err)
	}
	v, err := mapvar.Parse(brut)
	if err != nil {
		t.Fatal(err)
	}
	nParType := map[int32]int{}
	avecForme := map[int32]bool{}
	for _, o := range v.Objects {
		nParType[o.TypeID]++
		if o.ShapeRaw != nil {
			avecForme[o.TypeID] = true
		}
	}
	forge, err := himodule.Open(filepath.Join(root, "any", "globals", "forge", "forge_objects-rtx-new.module"))
	if err != nil {
		t.Fatal(err)
	}
	foodParID := map[uint32]himodule.File{}
	for _, f := range forge.Files("food") {
		foodParID[f.GlobalID] = f
	}
	refsDe := func(tag []byte, groupe string) []uint32 {
		var out []uint32
		vus := map[uint32]bool{}
		for o := 0; o+4 <= len(tag); o += 4 {
			h := uint32(u32(tag, o))
			if !vus[h] && parGroupe[groupe][h] {
				vus[h] = true
				out = append(out, h)
			}
		}
		return out
	}
	typesAvecRtgo, typesSans := 0, 0
	objetsAvecRtgo, objetsSans := 0, 0
	var exemplesSans []int32
	tailleTags := map[int]int{}
	for id, n := range nParType {
		f, ok := foodParID[uint32(id)]
		if !ok {
			continue
		}
		tag, err := forge.Extract(f)
		if err != nil {
			continue
		}
		tailleTags[len(tag)]++
		if len(refsDe(tag, GroupeRtgo)) > 0 {
			typesAvecRtgo++
			objetsAvecRtgo += n
		} else {
			typesSans++
			objetsSans += n
			if len(exemplesSans) < 12 {
				exemplesSans = append(exemplesSans, id)
			}
		}
	}
	t.Logf("type_id avec ref rtgo : %d (couvre %d objets) · sans : %d (couvre %d objets)",
		typesAvecRtgo, objetsAvecRtgo, typesSans, objetsSans)
	t.Logf("exemples SANS rtgo : %v", exemplesSans)
	sansEtForme := 0
	for _, id := range exemplesSans {
		if avecForme[id] {
			sansEtForme++
		}
	}
	t.Logf("parmi les exemples sans rtgo, %d portent une FORME (volume logique, pas de modele attendu)", sansEtForme)
	tailles := make([]int, 0, len(tailleTags))
	for taille := range tailleTags {
		tailles = append(tailles, taille)
	}
	sort.Ints(tailles)
	for _, taille := range tailles {
		if tailleTags[taille] > 5 {
			t.Logf("  taille de tag %5d o : %d tags", taille, tailleTags[taille])
		}
	}

	// Les 93 types SANS rtgo : vers quels groupes leurs food pointent-ils ?
	groupesSans := map[string]int{}
	objetsParGroupe := map[string]int{}
	for id, n := range nParType {
		f, ok := foodParID[uint32(id)]
		if !ok {
			continue
		}
		tag, err := forge.Extract(f)
		if err != nil || len(refsDe(tag, GroupeRtgo)) > 0 {
			continue
		}
		vus := map[string]bool{}
		for o := 0; o+4 <= len(tag); o += 4 {
			h := uint32(u32(tag, o))
			for _, g := range []string{"rtmp", "hlmt", "bloc", "scen", "mach", "effe", "ligh", "sndc", "prt3", "cust"} {
				if parGroupe[g][h] && !vus[g] {
					vus[g] = true
					groupesSans[g]++
					objetsParGroupe[g] += n
				}
			}
		}
	}
	t.Logf("types SANS rtgo, groupes references : %v (objets : %v)", groupesSans, objetsParGroupe)
}
