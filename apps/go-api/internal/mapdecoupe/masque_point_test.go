package mapdecoupe

import (
	"math/rand"
	"os"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/testutil"
)

// masque_point_test.go — LA FERMETURE EN UN POINT EST LA FERMETURE (lot M7 des retours du rejeu,
// correctif RR-M7-01). `PraticableComble` doit rendre, cellule pour cellule, ce que rend
// `Comble(r).Praticable` : sinon le cache compact du decor de carte deciderait autre chose que la
// regle qu il remplace.

// grilleAleatoire : des taches de matiere, des trous, des bords touches — de quoi exercer la
// fermeture sur ses trois cas (trou bouche, vide exterieur garde, cadre compte plein).
func grilleAleatoire(rng *rand.Rand, nx, ny int) []bool {
	dur := make([]bool, nx*ny)
	for n := 0; n < 6; n++ {
		ci, cj, rr := rng.Intn(nx), rng.Intn(ny), 2+rng.Intn(nx/3)
		for j := 0; j < ny; j++ {
			for i := 0; i < nx; i++ {
				if (i-ci)*(i-ci)+(j-cj)*(j-cj) <= rr*rr {
					dur[j*nx+i] = true
				}
			}
		}
	}
	for n := 0; n < nx*ny/12; n++ { // trous de reconstruction, isoles
		dur[rng.Intn(nx*ny)] = false
	}
	return dur
}

func TestPraticableCombleEgaleLaFermetureGlobale_Synthetique(t *testing.T) {
	rng := rand.New(rand.NewSource(20260924))
	for essai := 0; essai < 12; essai++ {
		nx, ny := 18+rng.Intn(20), 18+rng.Intn(20)
		m := masqueTest(t, nx, ny, grilleAleatoire(rng, nx, ny))
		c := m.Compacte()
		for _, rayon := range []float64{0, 0.7, 1, 1.5, 2.3, 3, 4.6} {
			ferme := m.Comble(rayon)
			for j := 0; j < ny; j++ {
				for i := 0; i < nx; i++ {
					x, y := float64(i)+0.5, float64(ny-j)-0.5
					if got, want := c.PraticableComble(x, y, rayon), ferme.Praticable(x, y); got != want {
						t.Fatalf("essai %d, %dx%d, rayon %.1f, cellule (%d,%d) : point %v, global %v",
							essai, nx, ny, rayon, i, j, got, want)
					}
					if got, want := c.Praticable(x, y), m.Praticable(x, y); got != want {
						t.Fatalf("masque brut : cellule (%d,%d) point %v, global %v", i, j, got, want)
					}
				}
			}
		}
	}
}

// Hors du cadre, jamais praticable — comme le masque dont il est la forme compacte.
func TestPraticableCombleHorsCadre(t *testing.T) {
	dur := make([]bool, 10*10)
	for k := range dur {
		dur[k] = true
	}
	c := masqueTest(t, 10, 10, dur).Compacte()
	if c.PraticableComble(-3, 5, 4) || c.PraticableComble(5, 40, 4) {
		t.Error("un point hors du cadre est declare praticable")
	}
	if !c.PraticableComble(5, 5, 4) {
		t.Error("le centre d une grille pleine n est pas praticable")
	}
}

// Sur un fond REEL du depot (Goliath, la carte du Wasp de decor), au rayon canonique : points
// tires dans le cadre, plus le Wasp de d8b13ec2.
func TestPraticableCombleEgaleLaFermetureGlobale_FondReel(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	const cle = "504ebf22-12b6-46c3-a9c1-ea20ca5bf03c" // Goliath
	res := title.NewPathResolver(root)
	metaPath := res.MapBackgroundMetaPath(title.DefaultSlug, cle)
	meta, err := replay.LoadMapBackground(metaPath)
	if err != nil {
		t.Fatalf("sidecar de Goliath : %v", err)
	}
	imgPath := res.MapBackgroundImageFilePath(title.DefaultSlug, meta.Image)
	if _, err := os.Stat(imgPath); err != nil {
		t.Fatalf("fond de Goliath absent du depot : %v", err)
	}
	m, err := ChargeMasque(imgPath, metaPath)
	if err != nil {
		t.Fatal(err)
	}
	ferme, c := m.Comble(ToleranceParDefaut), m.Compacte()
	cal := m.Calage
	rng := rand.New(rand.NewSource(768))
	points := [][2]float64{{-0.44, -8.16}}
	for n := 0; n < 400; n++ {
		points = append(points, [2]float64{
			cal.OriginX + rng.Float64()*float64(cal.WidthPx)*cal.MetersPerPixel,
			cal.OriginY - rng.Float64()*float64(cal.HeightPx)*cal.MetersPerPixel,
		})
	}
	for _, p := range points {
		if got, want := c.PraticableComble(p[0], p[1], ToleranceParDefaut), ferme.Praticable(p[0], p[1]); got != want {
			t.Fatalf("Goliath (%.2f, %.2f) : point %v, global %v", p[0], p[1], got, want)
		}
	}
	if c.Octets() > (m.NX*m.NY+63)/64*8 {
		t.Errorf("forme compacte : %d octets pour %d cellules", c.Octets(), m.NX*m.NY)
	}
}
