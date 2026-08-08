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

// TestCarteCliffhanger produit l ARTEFACT DE REVUE de la carte (gate visuel de la piste B).
//
// Ce n est pas un test au sens strict : il ECRIT une image et ne juge rien — le juge est
// l utilisateur (PLAN_BELLE_CARTE_TRIANGLES, gate humain). Il vit ici parce que sans lui la
// recette du rendu n existe nulle part, et elle a coute une journee a retrouver.
//
// Variables : PROBE_CARTE, PROBE_Z (zmin,zmax), PROBE_MULTI, PROBE_NIVEAU,
// PROBE_TOUS_MODULES, PROBE_PNG.
//
// Recette du prototype — volume borne en Z, praticabilite par
// degagement, COUPE horizontale.
// optionsCarte : les choix qui font une reconstruction. Ils sont explicites parce que ce
// sont EUX qu'on compare — l'oracle (`carte_oracle_gamefiles_test.go`) rejoue le meme
// volume sous plusieurs jeux d'options pour les departager sur donnees reelles.
type optionsCarte struct {
	Carte      string
	ZMin, ZMax float64
	CarteSeule bool // ne garder que les instances du module de la carte
	Dequant    Dequant
	// BorneAABB : marge du bornage a la boite monde de l'instance, en metres. Negatif =
	// pas de bornage (cf. AddMeshBorne).
	BorneAABB float64
}

// construitVolume rasterise la carte dans un volume, selon les options donnees.
func construitVolume(t *testing.T, o optionsCarte) *Volume {
	t.Helper()
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	if o.Carte == "" {
		o.Carte = "ridgeline"
	}
	if o.Dequant == nil {
		o.Dequant = DequantBrut
	}
	modCarte := moduleDuJeu(t, "pc", o.Carte)
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
	vol := NewVolumeZ(lo, hi, o.ZMin, o.ZMax)
	t.Logf("volume %d x %d x %d (cellule %.2f m, bande %.1f m, tranche [%.0f ; %.0f])",
		vol.NX, vol.NY, vol.NZ, vol.Cell, vol.ZBand, vol.ZMin, vol.ZMin+float64(vol.NZ)*vol.ZBand)

	assets := map[uint32]*RuntimeGeoAsset{}
	rendues, ecartees := 0, 0
	for _, in := range bsp.Instances {
		if in.QuickDeleted() {
			continue
		}
		id := in.RuntimeGeoID()
		g, mod, ok := idx.Lookup(id)
		if !ok || g != "rtgo" {
			continue
		}
		if o.CarteSeule && filepath.Base(mod) != filepath.Base(modCarte) {
			ecartees++
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
		if m := a.MeshDequant(in.MeshIndex, o.Dequant); m != nil {
			rendues++
			if o.BorneAABB < 0 {
				vol.AddMesh(m, in)
			} else {
				vol.AddMeshBorne(m, in, o.BorneAABB)
			}
		}
	}
	t.Logf("%d instances rasterisees · %d ecartees (modules globaux)", rendues, ecartees)
	return vol
}

func TestCarteCliffhanger(t *testing.T) {
	zmin, zmax := VolumeZMin, VolumeZMax
	if v := os.Getenv("PROBE_Z"); v != "" {
		fmt.Sscanf(v, "%f,%f", &zmin, &zmax)
	}
	vol := construitVolume(t, optionsCarte{
		Carte:      os.Getenv("PROBE_CARTE"),
		ZMin:       zmin,
		ZMax:       zmax,
		CarteSeule: os.Getenv("PROBE_TOUS_MODULES") == "",
		BorneAABB:  0.5,
	})

	degagement := HeadroomBands
	if v := os.Getenv("PROBE_DEGAGEMENT"); v != "" {
		fmt.Sscanf(v, "%d", &degagement)
	}
	sol := vol.Floors(degagement)
	niveau, cellules := sol.NiveauLePlusPeuple()
	if v := os.Getenv("PROBE_NIVEAU"); v != "" {
		fmt.Sscanf(v, "%f", &niveau)
	}
	t.Logf("bande praticable la plus peuplee : z = %+.2f m (%d cellules)", niveau, cellules)

	coupe := sol.Slice(niveau, SliceTolerance)
	n := 0
	for _, b := range coupe {
		if b {
			n++
		}
	}
	t.Logf("coupe a z = %+.2f ± %.2f m : %d cellules (%.1f %% de l'emprise)",
		niveau, SliceTolerance, n, 100*float64(n)/float64(len(coupe)))

	// Carte MULTI-NIVEAUX : pour chaque cellule, la bande praticable la plus HAUTE, teintee
	// par son altitude. Cette fois « le plus haut » est bien un sol : le volume est borne a
	// la tranche de jeu et le degagement de 2 m a deja ecarte les recoins ecrases.
	hautes := make([]int, vol.NX*vol.NY)
	for k := range hautes {
		hautes[k] = -1
	}
	for iz := 0; iz < sol.NZ; iz++ {
		for j := 0; j < sol.NY; j++ {
			for i := 0; i < sol.NX; i++ {
				if sol.Get(iz, j, i) {
					hautes[j*sol.NX+i] = iz
				}
			}
		}
	}
	nn := 0
	for _, z := range hautes {
		if z >= 0 {
			nn++
		}
	}
	t.Logf("carte multi-niveaux : %d cellules praticables (%.1f %% de l'emprise)",
		nn, 100*float64(nn)/float64(len(hautes)))

	img := image.NewRGBA(image.Rect(0, 0, vol.NX, vol.NY))
	multi := os.Getenv("PROBE_MULTI") != ""
	for j := 0; j < vol.NY; j++ {
		for i := 0; i < vol.NX; i++ {
			c := color.RGBA{247, 248, 250, 255}
			if multi {
				if iz := hautes[j*vol.NX+i]; iz >= 0 {
					f := float64(iz) / float64(sol.NZ-1)
					c = color.RGBA{uint8(28 + 200*f), uint8(46 + 175*f), uint8(84 + 130*f), 255}
				}
			} else if coupe[j*vol.NX+i] {
				c = color.RGBA{61, 86, 115, 255}
			}
			img.Set(i, vol.NY-1-j, c)
		}
	}
	sortie := os.Getenv("PROBE_PNG")
	if sortie == "" {
		sortie = "coupe.png"
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
