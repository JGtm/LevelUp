package himap

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
	"testing"
)

// Sonde TEMPORAIRE : dessiner la carte correctement — echelle de couleur ROBUSTE
// (centiles, pas min/max) et ombrage de relief. Aucun choix d'etage.
func TestProbeDessinCarte(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	carte := os.Getenv("PROBE_CARTE")
	if carte == "" {
		carte = "ridgeline"
	}
	modCarte := moduleDuJeu(t, "pc", carte)
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	var bsp BSPInstances
	for _, b := range bsps {
		if len(b.Instances) > len(bsp.Instances) {
			bsp = b
		}
	}
	lo := [2]float64{bsp.Bounds.Min[0], bsp.Bounds.Min[1]}
	hi := [2]float64{bsp.Bounds.Max[0], bsp.Bounds.Max[1]}
	if v := os.Getenv("PROBE_BOX"); v != "" {
		fmt.Sscanf(v, "%f,%f,%f,%f", &lo[0], &lo[1], &hi[0], &hi[1])
	}
	cell := 0.15
	fmt.Sscanf(os.Getenv("PROBE_CELL"), "%f", &cell)
	hf := NewHeightField(lo, hi, cell)

	assets := map[uint32]*RuntimeGeoAsset{}
	for _, in := range bsp.Instances {
		if in.QuickDeleted() {
			continue
		}
		id := in.RuntimeGeoID()
		if g, _, ok := idx.Lookup(id); !ok || g != "rtgo" {
			continue
		}
		a, deja := assets[id]
		if !deja {
			tag, blob, err := idx.ExtractWithResources(id)
			if err != nil {
				t.Fatal(err)
			}
			if a, err = NewRuntimeGeoAsset(tag, blob); err != nil {
				t.Fatal(err)
			}
			assets[id] = a
		}
		if m := a.Mesh(in.MeshIndex); m != nil {
			hf.AddMesh(m, in)
		}
	}
	t.Logf("%s : champ %d x %d de %.0f cm · couverture %.1f %%",
		carte, hf.NX, hf.NY, cell*100, 100*hf.Couverture())

	// Echelle ROBUSTE : les centiles 2 et 98. Avec min/max, une seule cellule a -107 m
	// ecrase toute la carte dans deux nuances de blanc — c'etait le defaut du rendu.
	var zs []float64
	for j := 0; j < hf.NY; j++ {
		for i := 0; i < hf.NX; i++ {
			if z, ok := hf.Cellule(i, j); ok {
				zs = append(zs, z)
			}
		}
	}
	if len(zs) == 0 {
		t.Fatal("champ vide")
	}
	sort.Float64s(zs)
	zlo, zhi := zs[len(zs)*2/100], zs[len(zs)*98/100]
	t.Logf("altitude : min %.1f · p2 %.1f · p98 %.1f · max %.1f m",
		zs[0], zlo, zhi, zs[len(zs)-1])

	img := image.NewRGBA(image.Rect(0, 0, hf.NX, hf.NY))
	for j := 0; j < hf.NY; j++ {
		for i := 0; i < hf.NX; i++ {
			z, ok := hf.Cellule(i, j)
			if !ok {
				img.Set(i, hf.NY-1-j, color.RGBA{14, 15, 19, 255})
				continue
			}
			f := (z - zlo) / math.Max(zhi-zlo, 1e-9)
			f = math.Max(0, math.Min(1, f))
			// Ombrage de relief : la PENTE fait le dessin. Un degrade d'altitude seul est
			// plat a l'oeil ; c'est l'ombre qui rend les aretes et les plateformes.
			ombre := 1.0
			if zx, okx := hf.Cellule(i+1, j); okx {
				if zy, oky := hf.Cellule(i, j+1); oky {
					dx, dy := (zx-z)/hf.Cell, (zy-z)/hf.Cell
					n := 1 / math.Sqrt(dx*dx+dy*dy+1)
					// Lumiere rasante au nord-ouest, convention des cartes.
					ombre = math.Max(0.25, (0.6*(-dx*n) + 0.6*(dy*n) + 0.9*n))
				}
			}
			g := uint8(math.Max(0, math.Min(255, (40+180*f)*ombre)))
			img.Set(i, hf.NY-1-j, color.RGBA{g, g, uint8(math.Min(255, float64(g)*1.06)), 255})
		}
	}
	sortie := os.Getenv("PROBE_PNG")
	if sortie == "" {
		sortie = "carte.png"
	}
	f, err := os.Create(sortie)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	fmt.Println("PNG ecrit:", sortie)
}
