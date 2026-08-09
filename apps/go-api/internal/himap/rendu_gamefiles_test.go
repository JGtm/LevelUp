package himap

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestRenduCliffhanger produit le rendu du maillage vu du dessus, au calage de la carte
// validee, et l'ecrit cote a cote avec elle. Il ne juge rien : le juge est l'utilisateur.
func TestRenduCliffhanger(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	ref, err := cheminDepuisDepot(".ai/V7.5/dumps/carte_validee_v1.png")
	if err != nil {
		t.Skip(err)
	}
	f, err := os.Open(ref)
	if err != nil {
		t.Skip(err)
	}
	defer func() { _ = f.Close() }()
	validee, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	bo := validee.Bounds()
	larg, haut := bo.Dx(), bo.Dy()

	lo := [2]float64{gateX0, gateY1 - float64(haut)*gateEchelle}
	hi := [2]float64{gateX0 + float64(larg)*gateEchelle, gateY1}
	rendu := NewRendu(lo, hi, gateEchelle)
	// Plafond DEDUIT des ancres : la plus haute, plus une hauteur de Spartan et sa marge de
	// saut. Aucun reglage par carte.
	zmax := math.Inf(-1)
	for _, a := range ancresCliffhanger(t) {
		zmax = math.Max(zmax, a[2])
	}
	rendu.Plafond(zmax + 6)
	t.Logf("plafond deduit des ancres : %+.2f m", zmax+6)

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	// TOUS les bsp du module : une carte en declare plusieurs, et rien ne dit que les grandes
	// dalles de terrain vivent dans celui qui porte les objectifs.
	var toutes []Instance
	for _, b := range bsps {
		toutes = append(toutes, b.Instances...)
	}
	bsp := BSPInstances{Instances: toutes}
	assets := map[uint32]*RuntimeGeoAsset{}
	n, globaux, nonResolues, maillageNil, supprimees := 0, 0, 0, 0, 0
	var inconnues []Instance
	for _, in := range bsp.Instances {
		if in.QuickDeleted() {
			supprimees++
			continue
		}
		id := in.RuntimeGeoID()
		g, mod, ok := idx.Lookup(id)
		if !ok || g != "rtgo" {
			nonResolues++
			inconnues = append(inconnues, in)
			continue
		}
		// Le module de la carte SEUL : les modules globaux apportent le decor lointain, qui
		// noie l'arene. La carte validee ne lisait qu'un module.
		if filepath.Base(mod) != filepath.Base(modCarte) {
			globaux++
			continue
		}
		a, deja := assets[id]
		if !deja {
			tag, blob, err := idx.ExtractWithResources(id)
			if err != nil {
				continue
			}
			if a, err = NewRuntimeGeoAsset(tag, blob); err != nil {
				continue
			}
			assets[id] = a
		}
		if m := a.Mesh(in.MeshIndex); m != nil {
			n++
			rendu.AddMeshBorne(m, in, 0.5)
		} else {
			maillageNil++
		}
	}
	t.Logf("bsp %d instances · %d rendues · %d globales · %d non resolues · %d maillage nil · %d supprimees",
		len(bsp.Instances), n, globaux, nonResolues, maillageNil, supprimees)
	t.Logf("%d bsp dans le module", len(bsps))
	t.Logf("%d instances rendues sur une grille %d x %d a %.4f m/px", n, rendu.NX, rendu.NY, rendu.Cell)

	out := image.NewRGBA(image.Rect(0, 0, 2*larg+8, haut))
	couverts := 0
	for py := 0; py < haut; py++ {
		for px := 0; px < larg; px++ {
			out.Set(px, py, validee.At(bo.Min.X+px, bo.Min.Y+py))
		}
		for px := 0; px < 8; px++ {
			out.Set(larg+px, py, color.RGBA{40, 40, 40, 255})
		}
		j := haut - 1 - py
		for px := 0; px < larg; px++ {
			c := color.RGBA{0, 0, 0, 255}
			if e, ok := rendu.Eclairement(px, j); ok {
				couverts++
				g := uint8(255 * e)
				c = color.RGBA{g, g, g, 255}
			}
			out.Set(larg+8+px, py, c)
		}
	}
	// Empreinte des instances NON RESOLUES, en rouge : ou manque-t-il de la geometrie ?
	for _, in := range inconnues {
		for y := in.AABBMin[1]; y <= in.AABBMax[1]; y += gateEchelle {
			for x := in.AABBMin[0]; x <= in.AABBMax[0]; x += gateEchelle {
				px := int((x - gateX0) / gateEchelle)
				py := int((gateY1 - y) / gateEchelle)
				if px >= 0 && px < larg && py >= 0 && py < haut {
					out.Set(larg+8+px, py, color.RGBA{210, 40, 40, 255})
				}
			}
		}
	}
	t.Logf("%d px couverts sur %d (%.1f %%)", couverts, larg*haut, 100*float64(couverts)/float64(larg*haut))

	sortie := os.Getenv("RENDU_PNG")
	if sortie == "" {
		sortie = filepath.Join(os.TempDir(), "rendu.png")
	}
	g, err := os.Create(sortie)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = g.Close() }()
	if err := png.Encode(g, out); err != nil {
		t.Fatal(err)
	}
	fmt.Println("rendu ecrit:", sortie)
}
