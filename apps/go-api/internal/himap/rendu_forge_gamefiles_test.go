package himap

// PROTOTYPE CARTE FORGE (F2 du plan §4) — rendre Vagabond depuis les objets du .mvar.
//
// La chaine, etablie sur pieces au lot 2 (2026-08-10, sondes sonde_forge_gamefiles_test.go) :
//
//	objet .mvar --type_id--> tag `food` (GlobalID, forge_objects-rtx-new.module)
//	           --refs inline--> tags `rtgo` (memes maillages que la chaine sbsp)
//	           --Pos/Up/Forward--> repere monde (Left = Up x Forward, base orthonormee)
//
// Mesures qui fondent chaque maillon :
//   - type_id -> food : 467/468 type_id de Vagabond se resolvent (l'investigation volumes
//     de mort avait etabli 20/20) ;
//   - food -> rtgo : les deps des tags food sont VIDES (457/467) et root+0x08 est
//     l'auto-reference ; le lien est INLINE — 374 type_id portent >= 1 ref rtgo, couvrant
//     3 558 des 4 697 objets (75,7 %). Les 93 restants passent par bloc/scen/mach
//     (963/173/9 objets) — saut supplementaire NON traite par ce prototype ;
//   - echelle : AUCUNE dans le .mvar de Vagabond — mesure, pas suppose : le champ objet [6]
//     n'existe pas et [9] est une struct VIDE sur 4 709/4 709 objets (le piege de l'echelle
//     sbsp, paye deux jours, a ete verifie ici sur pieces).
//
// CAS VAGABOND, traite explicitement : son entree catalogue porte le module generique
// `map` (comme Highpower) — la selection par nom de module est ambigue. On selectionne par
// map_id (asset UGC) et on VERIFIE la jointure par le compte d'objets du catalogue.
//
// Volumes de mort (investigation VOLUMES_MORT_MVAR) : exclus par construction — ils n'ont
// aucune ref rtgo — et COMPTES explicitement pour que l'exclusion soit un fait mesure.

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himodule"
)

// mapIDVagabond : asset UGC de Vagabond au catalogue d'objectifs.
const mapIDVagabond = "105f5d84-8de1-4908-af3a-1c4f3bf9d642"

// typesVolumesDeMort : empreinte fonctionnelle etablie sur 101 .mvar
// (INVESTIGATION_VOLUMES_MORT_MVAR_2026-08-10.md §2).
var typesVolumesDeMort = map[int32]string{
	-588988541:  "volume de mort principal",
	176825834:   "plancher de mort",
	937132837:   "volume de mort sous-sol",
	-1751270658: "mur de limite",
}

// instanceForge construit l'instance de pose d'un objet .mvar : base orthonormee
// Forward/Left/Up avec Left = Up x Forward, echelle unitaire (mesuree absente du format).
func instanceForge(o mapvar.Object) Instance {
	f := normalise([3]float64{o.Forward.X, o.Forward.Y, o.Forward.Z})
	u := normalise([3]float64{o.Up.X, o.Up.Y, o.Up.Z})
	l := [3]float64{
		u[1]*f[2] - u[2]*f[1],
		u[2]*f[0] - u[0]*f[2],
		u[0]*f[1] - u[1]*f[0],
	}
	return Instance{
		Scale:    [3]float64{1, 1, 1},
		Position: [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z},
		Forward:  f,
		Left:     l,
		Up:       u,
	}
}

// rtgoRefsInline rend les GlobalID rtgo references dans les octets d'un tag food, dans
// l'ordre du tag (le premier est la variante principale — convention du prototype).
func rtgoRefsInline(tag []byte, estRtgo func(uint32) bool) []uint32 {
	var out []uint32
	vus := map[uint32]bool{}
	for o := 0; o+4 <= len(tag); o += 4 {
		h := uint32(u32(tag, o))
		if !vus[h] && estRtgo(h) {
			vus[h] = true
			out = append(out, h)
		}
	}
	return out
}

func TestRenduForgeVagabond(t *testing.T) {
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

	// Les ancres — selection EXPLICITE par map_id (module generique `map`, cf. en-tete).
	p, err := cheminCatalogue()
	if err != nil {
		t.Skip(err)
	}
	cat, err := replay.LoadMapObjectives(p)
	if err != nil {
		t.Fatal(err)
	}
	entree, err := cat.Lookup(mapIDVagabond)
	if err != nil {
		t.Fatal(err)
	}
	if entree.Module != "map" {
		t.Fatalf("l'entree Vagabond devrait porter le module generique %q, obtenu %q", "map", entree.Module)
	}
	if entree.ObjectsN != len(v.Objects) {
		t.Fatalf("jointure catalogue<->mvar cassee : %d objets au catalogue, %d dans le .mvar",
			entree.ObjectsN, len(v.Objects))
	}
	var ancres [][3]float64
	for _, o := range entree.Objectives {
		ancres = append(ancres, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
	}
	if len(ancres) == 0 {
		t.Fatal("aucune ancre pour Vagabond — l'oracle faible est vide")
	}
	t.Logf("vagabond (%s) : %d objets, %d ancres", entree.PublicName, len(v.Objects), len(ancres))

	// L'index : forge_objects d'abord (les food et leurs rtgo), puis le canevas et les globals.
	chemins := []string{
		filepath.Join(root, "any", "globals", "forge", "forge_objects-rtx-new.module"),
	}
	if p := filepath.Join(root, "pc", "globals", "forge", "forge_objects-rtx-new.module"); existe(p) {
		chemins = append(chemins, p)
	}
	if dir, err := LevelsDir("pc"); err == nil {
		if p := filepath.Join(dir, "fo08_wetland", "fo08_wetland-rtx-new.module"); existe(p) {
			chemins = append(chemins, p)
		}
	}
	globs, _ := filepath.Glob(filepath.Join(root, "pc", "globals", "*.module"))
	chemins = append(chemins, globs...)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d modules indexes, %d tags", len(chemins), idx.Taille())

	forge, err := himodule.Open(chemins[0])
	if err != nil {
		t.Fatal(err)
	}
	foodParID := map[uint32]himodule.File{}
	for _, f := range forge.Files("food") {
		foodParID[f.GlobalID] = f
	}
	estRtgo := func(h uint32) bool {
		g, _, ok := idx.Lookup(h)
		return ok && g == "rtgo"
	}

	// rtgo de chaque type_id (premiere ref inline), etabli une fois par type.
	rtgoDuType := map[int32]uint32{}
	sansFood, sansRtgo := map[int32]int{}, map[int32]int{}
	for _, o := range v.Objects {
		if _, deja := rtgoDuType[o.TypeID]; deja {
			continue
		}
		if _, deja := sansFood[o.TypeID]; deja {
			continue
		}
		if _, deja := sansRtgo[o.TypeID]; deja {
			continue
		}
		f, ok := foodParID[uint32(o.TypeID)]
		if !ok {
			sansFood[o.TypeID] = 0
			continue
		}
		tag, err := forge.Extract(f)
		if err != nil {
			t.Errorf("food %d : %v", o.TypeID, err)
			continue
		}
		refs := rtgoRefsInline(tag, estRtgo)
		if len(refs) == 0 {
			sansRtgo[o.TypeID] = 0
			continue
		}
		rtgoDuType[o.TypeID] = refs[0]
	}

	rendu := cadreSurAncres(t, ancres)
	// LA TRANCHE, RELATIVE AU SOL DES ANCRES. `TrancheDeJeuMin/Max` sont des altitudes
	// ABSOLUES mesurees sur Cliffhanger (sol vers z=0) ; le sol de Vagabond vit vers z=52.
	// On reutilise la MEME tranche, translatee au sol deduit des ancres (mediane - decalage
	// etalonne) — une regle, pas un reglage par carte. RENDU_FORGE_SANS_TRANCHE la retire
	// pour comparaison.
	if os.Getenv("RENDU_FORGE_SANS_TRANCHE") == "" {
		sol := medianeZ(ancres) - AncrageDecalageSol
		rendu.Tranche(sol+TrancheDeJeuMin, sol+TrancheDeJeuMax)
		t.Logf("tranche relative au sol des ancres (%.2f m) : [%.2f ; %.2f]",
			sol, sol+TrancheDeJeuMin, sol+TrancheDeJeuMax)
	}
	assets := map[uint32]*RuntimeGeoAsset{}
	dessines, sautes, morts := 0, 0, 0
	for _, o := range v.Objects {
		if _, mort := typesVolumesDeMort[o.TypeID]; mort {
			morts++
			continue
		}
		id, ok := rtgoDuType[o.TypeID]
		if !ok {
			sautes++
			continue
		}
		a, deja := assets[id]
		if !deja {
			tag, blob, err := idx.ExtractWithResources(id)
			if err != nil {
				t.Logf("rtgo %08x : %v", id, err)
				assets[id] = nil
				continue
			}
			if a, err = NewRuntimeGeoAsset(tag, blob); err != nil {
				t.Logf("rtgo %08x : %v", id, err)
				assets[id] = nil
				continue
			}
			assets[id] = a
		}
		if a == nil {
			sautes++
			continue
		}
		in := instanceForge(o)
		for mi := 0; mi < a.MeshCount(); mi++ {
			if m := a.Mesh(mi); m != nil {
				rendu.AddMesh(m, in)
			}
		}
		dessines++
	}
	t.Logf("objets : %d dessines · %d sans modele rtgo (sautes) · %d volumes de mort EXCLUS",
		dessines, sautes, morts)
	if morts == 0 {
		t.Error("Vagabond devrait porter au moins un volume de mort a exclure")
	}

	jugeParLesAncres(t, rendu, ancres)
	// Detail par ancre : 4 ancres seulement, la mediane cache tout. Un ecart tres negatif
	// sur un CENTRE de zone Bastion dit « il y a un toit au-dessus », pas « le sol est faux »
	// — le z-buffer retient la surface la plus haute.
	for i, o := range entree.Objectives {
		ci := int((o.Pos.X - rendu.Min[0]) / rendu.Cell)
		cj := int((o.Pos.Y - rendu.Min[1]) / rendu.Cell)
		if z, ok := rendu.Altitude(ci, cj); ok {
			t.Logf("  ancre %d (%s) z=%.2f : surface dessinee %.2f (ecart %+.2f)",
				i, o.Role, o.Pos.Z, z, o.Pos.Z-AncrageDecalageSol-z)
		}
	}

	if sortie := os.Getenv("RENDU_PNG_FORGE"); sortie != "" {
		ecritPNG(t, sortie, carteSeulePNG(rendu, rendu.NX, rendu.NY, nil, StyleCarteParDefaut))
		fmt.Println("carte forge ecrite:", sortie)
	}
}

func existe(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
