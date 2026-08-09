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
	// Plafond : plafond de triangles par maillage (0 = MaxTrianglesPerMesh). Il decide du
	// niveau de detail retenu.
	Plafond int
	// CheminModule court-circuite la resolution par nom de carte, pour le balayage.
	CheminModule string
	// Emprise borne le volume en X/Y. Nulle = l emprise complete du sbsp, qui peut etre
	// enorme : la premiere carte du balayage a demande 11 324 x 12 308 cellules, soit 4 Go.
	EmpriseMin, EmpriseMax *[2]float64
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
	modCarte := o.CheminModule
	if modCarte == "" {
		modCarte = moduleDuJeu(t, "pc", o.Carte)
	}
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
	if o.EmpriseMin != nil && o.EmpriseMax != nil {
		for k := 0; k < 2; k++ {
			lo[k] = math.Max(lo[k], o.EmpriseMin[k])
			hi[k] = math.Min(hi[k], o.EmpriseMax[k])
		}
	}
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
			a.PlafondTriangles = o.Plafond
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

	// Carte DERIVEE : pour chaque cellule, la bande praticable la plus proche de la surface
	// d'altitude de reference construite sur les ancres d'objectifs de la carte. Aucun
	// reglage propre a la carte — cf. `reference.go`.
	//
	// Elle remplace « la bande praticable la plus HAUTE », qui ramenait le decor par-dessus
	// l'arene des que les modules globaux entraient dans le volume.
	hautes := make([]int, vol.NX*vol.NY)
	for k := range hautes {
		hautes[k] = -1
	}
	if v := os.Getenv("PROBE_TOLERANCE"); v != "" {
		fmt.Sscanf(v, "%f", &toleranceRendu)
	}
	surface := surfaceDesAncres(t, os.Getenv("PROBE_CARTE"))
	if surface.Vide() {
		t.Log("aucune ancre : repli sur la bande praticable la plus haute")
		for iz := 0; iz < sol.NZ; iz++ {
			for j := 0; j < sol.NY; j++ {
				for i := 0; i < sol.NX; i++ {
					if sol.Get(iz, j, i) {
						hautes[j*sol.NX+i] = iz
					}
				}
			}
		}
	} else {
		hautes = sol.CarteParReference(surface, toleranceRendu)
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
					// Teinte par l'ecart a la surface de REFERENCE, pas par l'altitude
					// absolue : c'est l'etage joue qui doit ressortir, pas le relief.
					x := vol.Min[0] + (float64(i)+0.5)*vol.Cell
					y := vol.Min[1] + (float64(j)+0.5)*vol.Cell
					f := 0.5
					if !surface.Vide() {
						f = (vol.Altitude(iz) - surface.At(x, y) + toleranceRendu) / (2 * toleranceRendu)
						f = math.Max(0, math.Min(1, f))
					}
					c = color.RGBA{uint8(34 + 190*f), uint8(52 + 168*f), uint8(88 + 126*f), 255}
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
