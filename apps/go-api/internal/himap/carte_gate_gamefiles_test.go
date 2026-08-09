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

// LE GATE VISUEL — la carte reconstruite contre la carte VALIDEE, au meme calage.
//
// `PLAN_BELLE_CARTE_TRIANGLES` exige un artefact cote a cote a la meme echelle. « Cote a cote »
// ne suffit pas : deux images de meme taille mais decalees de deux metres se comparent mal, et
// l'oeil comble les ecarts. On projette donc la carte reconstruite DANS la grille de pixels de
// la reference, en utilisant son calage mesure — 0,0920 m/px, X0 -43,5, Y1 61,0
// (`ETAT_DU_POC.md`). Chaque pixel des deux moities designe alors le meme point du monde.
//
// Ce test ne juge rien : il ecrit une image. Le juge est l'utilisateur.
//
// Variables : GATE_REFERENCE (chemin de la carte validee), GATE_PNG (sortie).

// Calage de `carte_validee_v1.png`, mesure sur les trajectoires de joueur.
const (
	gateEchelle = 0.0920 // metres par pixel
	gateX0      = -43.5
	gateY1      = 61.0
)

func TestGateVisuelCliffhanger(t *testing.T) {
	ref := os.Getenv("GATE_REFERENCE")
	if ref == "" {
		p, err := cheminDepuisDepot(".ai/V7.5/dumps/carte_validee_v1.png")
		if err != nil {
			t.Skip(err)
		}
		ref = p
	}
	f, err := os.Open(ref) //nolint:gosec // chemin de test, lecture seule
	if err != nil {
		t.Skip(err)
	}
	defer func() { _ = f.Close() }()
	validee, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	bornes := validee.Bounds()
	larg, haut := bornes.Dx(), bornes.Dy()
	t.Logf("carte validee : %d x %d px, soit %.0f x %.0f m au calage mesure",
		larg, haut, float64(larg)*gateEchelle, float64(haut)*gateEchelle)

	// Le volume couvre l'emprise de la reference, pas celle du sbsp : c'est la zone qu'on
	// compare, et l'allouer plus large ne ferait que rasteriser du decor.
	lo := [2]float64{gateX0, gateY1 - float64(haut)*gateEchelle}
	hi := [2]float64{gateX0 + float64(larg)*gateEchelle, gateY1}
	ancres := ancresCliffhanger(t)
	vol := construitVolume(t, optionsCarte{
		Carte: "ridgeline", ZMin: -25, ZMax: 25, BorneAABB: 0.5,
		EmpriseMin: &lo, EmpriseMax: &hi, Ancres: ancres,
	})
	sol := vol.Floors(HeadroomBands)
	surface := NewSurfaceReference(ancres)
	bandes := sol.CarteParReference(surface, toleranceRendu)

	// Cote a cote : la reference a gauche, la reconstruction a droite, meme grille de pixels.
	out := image.NewRGBA(image.Rect(0, 0, 2*larg+8, haut))
	for py := 0; py < haut; py++ {
		for px := 0; px < larg; px++ {
			out.Set(px, py, validee.At(bornes.Min.X+px, bornes.Min.Y+py))
		}
		for px := 0; px < 8; px++ {
			out.Set(larg+px, py, color.RGBA{225, 228, 233, 255})
		}
	}
	dessines := 0
	for py := 0; py < haut; py++ {
		y := gateY1 - (float64(py)+0.5)*gateEchelle
		for px := 0; px < larg; px++ {
			x := gateX0 + (float64(px)+0.5)*gateEchelle
			c := color.RGBA{247, 248, 250, 255}
			i := int((x - sol.Min[0]) / sol.Cell)
			j := int((y - sol.Min[1]) / sol.Cell)
			if i >= 0 && i < sol.NX && j >= 0 && j < sol.NY {
				if iz := bandes[j*sol.NX+i]; iz >= 0 {
					dessines++
					f := 0.5
					if !surface.Vide() {
						f = (sol.AltitudeSol(iz) - surface.At(x, y) + toleranceRendu) / (2 * toleranceRendu)
						f = math.Max(0, math.Min(1, f))
					}
					c = color.RGBA{uint8(34 + 190*f), uint8(52 + 168*f), uint8(88 + 126*f), 255}
				}
			}
			out.Set(larg+8+px, py, c)
		}
	}
	t.Logf("reconstruction : %d px dessines sur %d (%.1f %%)",
		dessines, larg*haut, 100*float64(dessines)/float64(larg*haut))

	sortie := os.Getenv("GATE_PNG")
	if sortie == "" {
		sortie = "gate.png"
	}
	g, err := os.Create(sortie)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = g.Close() }()
	if err := png.Encode(g, out); err != nil {
		t.Fatal(err)
	}
	fmt.Println("gate visuel ecrit:", sortie)
}

func ancresCliffhanger(t *testing.T) [][3]float64 {
	t.Helper()
	var pts [][3]float64
	for _, e := range ancresDuModule(t, moduleCliffhanger) {
		for _, o := range e.Objectives {
			pts = append(pts, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	return pts
}

// cheminDepuisDepot remonte depuis le repertoire de test jusqu'a trouver un chemin relatif au
// depot.
func cheminDepuisDepot(rel string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for d := wd; ; {
		c := filepath.Join(d, rel)
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("%s introuvable depuis %s", rel, wd)
		}
		d = parent
	}
}
