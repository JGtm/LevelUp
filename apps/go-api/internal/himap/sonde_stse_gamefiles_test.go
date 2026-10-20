//go:build gamefiles

package himap

// SONDE (2026-08-10, non commitee) — les polygones du tag stse : trace complet
// et dessin sur la carte validee (Cliffhanger). Chaque bloc de records de 12 o
// est une liste de sommets vec3 ; les blocs de 36 o et 16 o restent a
// interpreter (dump inclus).

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/himodule"
)

func litStsePolygones(t *testing.T, chemin string) ([][][3]float64, tagInfo) {
	t.Helper()
	m, err := himodule.Open(chemin)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range m.Files("stse") {
		tag, err := m.Extract(f)
		if err != nil {
			continue
		}
		ti, err := meilleurTagInfo(tag)
		if err != nil {
			continue
		}
		var polys [][][3]float64
		for _, l := range liensBlocs(ti) {
			abs, size := ti.blockAbs(l.target)
			c := compteChamp(ti, l)
			if c < 2 || size != c*12 {
				continue
			}
			var poly [][3]float64
			for i := 0; i < c; i++ {
				b := abs + i*12
				poly = append(poly, [3]float64{f32(ti.tag, b), f32(ti.tag, b+4), f32(ti.tag, b+8)})
			}
			polys = append(polys, poly)
		}
		return polys, ti
	}
	t.Fatal("aucun stse lisible")
	return nil, tagInfo{}
}

func TestSondeStseDump(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, carte := range []string{"ridgeline", "catalyst"} {
		t.Run(carte, func(t *testing.T) {
			polys, ti := litStsePolygones(t, moduleAny(t, carte))
			t.Logf("%s : %d polygones", carte, len(polys))
			for i, p := range polys {
				zmin, zmax := p[0][2], p[0][2]
				for _, v := range p {
					if v[2] < zmin {
						zmin = v[2]
					}
					if v[2] > zmax {
						zmax = v[2]
					}
				}
				t.Logf("  poly[%d] : %d sommets, z [%.1f ; %.1f], sommets %v", i, len(p), zmin, zmax, p)
			}
			// Blocs 36 o et 16 o : dump brut.
			for _, l := range liensBlocs(ti) {
				abs, size := ti.blockAbs(l.target)
				c := compteChamp(ti, l)
				if c < 2 || (size != c*36 && size != c*16) {
					continue
				}
				stride := size / c
				t.Logf("  bloc %d : %d x %d o", l.target, c, stride)
				for i := 0; i < c && i < 10; i++ {
					dumpRecord(t, ti.tag, abs+i*stride, stride, "   ")
				}
			}
		})
	}
}

// TestSondeStseDessin superpose les polygones stse a la carte validee.
func TestSondeStseDessin(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
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

	polys, _ := litStsePolygones(t, moduleAny(t, "ridgeline"))
	out := image.NewRGBA(image.Rect(0, 0, larg, haut))
	for py := 0; py < haut; py++ {
		for px := 0; px < larg; px++ {
			out.Set(px, py, validee.At(bo.Min.X+px, bo.Min.Y+py))
		}
	}
	// Trace des aretes de chaque polygone (ferme), en couleur par index.
	couleurs := []color.RGBA{
		{230, 40, 40, 255}, {40, 160, 230, 255}, {40, 200, 80, 255}, {230, 160, 40, 255},
		{200, 60, 200, 255}, {60, 220, 220, 255}, {240, 240, 60, 255}, {150, 90, 40, 255},
		{240, 120, 180, 255}, {120, 240, 120, 255}, {180, 180, 255, 255}, {255, 255, 255, 255},
	}
	versPx := func(x, y float64) (int, int) {
		px := int((x - gateX0) / gateEchelle)
		py := haut - 1 - int((y-(gateY1-float64(haut)*gateEchelle))/gateEchelle)
		return px, py
	}
	for pi, p := range polys {
		c := couleurs[pi%len(couleurs)]
		for i := range p {
			a, b := p[i], p[(i+1)%len(p)]
			ax, ay := versPx(a[0], a[1])
			bx, by := versPx(b[0], b[1])
			n := max(abs(bx-ax), abs(by-ay))
			if n == 0 {
				n = 1
			}
			for k := 0; k <= n; k++ {
				x := ax + (bx-ax)*k/n
				y := ay + (by-ay)*k/n
				for dj := -1; dj <= 1; dj++ {
					for di := -1; di <= 1; di++ {
						if x+di >= 0 && x+di < larg && y+dj >= 0 && y+dj < haut {
							out.Set(x+di, y+dj, c)
						}
					}
				}
			}
		}
	}
	if dir := os.Getenv("SONDE_PNG_DIR"); dir != "" {
		ecritPNG(t, filepath.Join(dir, "sonde_stse_polygones.png"), out)
	}
}
