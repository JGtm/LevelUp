package himap

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
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

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	bsp := choisitBSP(bsps, ancresCliffhanger(t))
	assets := map[uint32]*RuntimeGeoAsset{}
	n := 0
	for _, in := range bsp.Instances {
		if in.QuickDeleted() {
			continue
		}
		id := in.RuntimeGeoID()
		g, mod, ok := idx.Lookup(id)
		if !ok || g != "rtgo" {
			continue
		}
		// Le module de la carte SEUL : les modules globaux apportent le decor lointain, qui
		// noie l'arene. La carte validee ne lisait qu'un module.
		if filepath.Base(mod) != filepath.Base(modCarte) {
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
			rendu.AddMesh(m, in)
		}
	}
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
