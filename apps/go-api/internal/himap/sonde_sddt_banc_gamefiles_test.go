package himap

// SONDE D'INVESTIGATION (2026-08-10, non destinee au commit) — LE BANC : la
// coquille sddt (triangles-frontieres) confrontee a la reference validee et a
// l'oracle des positions jouees, memes reglages que TestSondePhysiqueRendu.

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// coquilleSddt est l'ensemble des demi-espaces des triangles-frontieres,
// dedoublonnes : interieur = n.p >= d pour tous (normales vers l'interieur —
// orientation etablie par l'oracle des positions jouees, voir TestSondeSddtRendu).
type coquilleSddt [][4]float64

func coquilleDepuis(ceils []sddtCeiling) coquilleSddt {
	var out coquilleSddt
	for _, c := range ceils {
		p := [4]float64{c.Normal[0], c.Normal[1], c.Normal[2], c.D}
		deja := false
		for _, q := range out {
			d := math.Abs(p[0]-q[0]) + math.Abs(p[1]-q[1]) + math.Abs(p[2]-q[2]) + math.Abs(p[3]-q[3])
			if d < 1e-4 {
				deja = true
				break
			}
		}
		if !deja {
			out = append(out, p)
		}
	}
	return out
}

func (c coquilleSddt) contient(p [3]float64, eps float64) bool {
	for _, pl := range c {
		if pl[0]*p[0]+pl[1]*p[1]+pl[2]*p[2] < pl[3]-eps {
			return false
		}
	}
	return true
}

// dansPrisme : interieur d'un volume 100 o = n.p <= d pour ses 5 plans
// (orientation opposee a la coquille — etablie par le temoin des centroides,
// joue dans TestSondeSddtRendu).
func dansPrisme(v sddtVolume, p [3]float64, eps float64) bool {
	if len(v.Plans) == 0 {
		return false
	}
	for _, pl := range v.Plans {
		if pl[0]*p[0]+pl[1]*p[1]+pl[2]*p[2] > pl[3]+eps {
			return false
		}
	}
	return true
}

func centroideVolume(v sddtVolume) [3]float64 {
	var c [3]float64
	n := 0
	for _, tri := range v.Tris {
		for s := 0; s < 3; s++ {
			for a := 0; a < 3; a++ {
				c[a] += tri[s][a]
			}
			n++
		}
	}
	if n > 0 {
		for a := 0; a < 3; a++ {
			c[a] /= float64(n)
		}
	}
	return c
}

// masqueCoquille rend le masque par cellule : la matiere de cette colonne
// est-elle dans la coquille ?
func masqueCoquille(r *Rendu, c coquilleSddt) []bool {
	m := make([]bool, r.NX*r.NY)
	for j := 0; j < r.NY; j++ {
		for i := 0; i < r.NX; i++ {
			z, ok := r.Altitude(i, j)
			if !ok {
				continue
			}
			p := [3]float64{r.Min[0] + (float64(i)+0.5)*r.Cell, r.Min[1] + (float64(j)+0.5)*r.Cell, z}
			m[j*r.NX+i] = c.contient(p, 0)
		}
	}
	return m
}

// TestSondeSddtRendu — LE JUGE sur Cliffhanger.
func TestSondeSddtRendu(t *testing.T) {
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

	vols, ceils := litSddt(t, moduleAny(t, "ridgeline"))
	coq := coquilleDepuis(ceils)
	t.Logf("coquille : %d plans distincts (sur %d triangles) · %d volumes", len(coq), len(ceils), len(vols))

	// TEMOIN d'orientation des prismes : le centroide des sommets de CHAQUE
	// volume doit etre dans son propre prisme, et le sens INVERSE doit echouer.
	dedans, sensInverse := 0, 0
	for _, v := range vols {
		c := centroideVolume(v)
		if dansPrisme(v, c, 1e-4) {
			dedans++
		}
		horsUnPlan := false
		for _, pl := range v.Plans {
			if pl[0]*c[0]+pl[1]*c[1]+pl[2]*c[2] < pl[3]-1e-4 {
				horsUnPlan = true
				break
			}
		}
		if !horsUnPlan {
			sensInverse++
		}
	}
	t.Logf("orientation prismes : centroide dans son prisme %d/%d · sens inverse accepterait %d/%d",
		dedans, len(vols), sensInverse, len(vols))

	pos := chargePositions(t)
	// L'ORACLE CONTRE LA COQUILLE : un volume de MORT ne contient aucune
	// position jouee ; une coquille de RETENUE les contient toutes.
	dansCoq, dansVol := 0, 0
	for _, p := range pos {
		q := [3]float64{p.X, p.Y, p.Z}
		if coq.contient(q, 0) {
			dansCoq++
		}
		for _, v := range vols {
			if dansPrisme(v, q, 0) {
				dansVol++
				break
			}
		}
	}
	t.Logf("ORACLE vs coquille : %d/%d positions jouees dedans (%.2f %%)",
		dansCoq, len(pos), 100*float64(dansCoq)/float64(len(pos)))
	t.Logf("ORACLE vs 233 prismes : %d/%d positions jouees dedans (%.3f %%)",
		dansVol, len(pos), 100*float64(dansVol)/float64(len(pos)))

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	bsp := choisitBSP(bsps, ancresCliffhanger(t))
	assets := map[uint32]*RuntimeGeoAsset{}
	type item struct {
		m    *Mesh
		in   Instance
		aire float64
	}
	var items []item
	for _, in := range bsp.Instances {
		if in.QuickDeleted() || in.ProjecteurOmbre() {
			continue
		}
		id := in.RuntimeGeoID()
		if g, _, ok := idx.Lookup(id); !ok || g != "rtgo" {
			continue
		}
		a, deja := assets[id]
		if !deja {
			tg, blob, err := idx.ExtractWithResources(id)
			if err != nil {
				continue
			}
			if a, err = NewRuntimeGeoAsset(tg, blob); err != nil {
				continue
			}
			assets[id] = a
		}
		m := a.Mesh(in.MeshIndex)
		if m == nil {
			continue
		}
		items = append(items, item{m: m, in: in, aire: AireMedianeProjetee(m, in)})
	}
	t.Logf("%d instances dessinables", len(items))

	configs := []struct {
		nom      string
		grain    bool
		coquille bool
	}{
		{"aucun-tri", false, false},
		{"grain-0.005", true, false},
		{"coquille-sddt", false, true},
		{"coquille-et-grain", true, true},
	}
	for _, cfg := range configs {
		t.Run(cfg.nom, func(t *testing.T) {
			rendu := NewRendu(lo, hi, gateEchelle)
			rendu.Tranche(TrancheDeJeuMin, TrancheDeJeuMax)
			n := 0
			for _, it := range items {
				if cfg.grain && it.aire > AireMaxTriangleJouable {
					continue
				}
				n++
				rendu.AddMeshBorne(it.m, it.in, 0.5)
			}
			if cfg.coquille {
				m := masqueCoquille(rendu, coq)
				garde := 0
				for _, ok := range m {
					if ok {
						garde++
					}
				}
				rendu.Restreindre(m)
				t.Logf("%s : masque coquille garde %d cellules pleines", cfg.nom, garde)
			}
			t.Logf("%s : %d/%d instances dessinees", cfg.nom, n, len(items))

			out := image.NewRGBA(image.Rect(0, 0, 3*larg+16, haut))
			for py := 0; py < haut; py++ {
				for px := 0; px < larg; px++ {
					out.Set(px, py, validee.At(bo.Min.X+px, bo.Min.Y+py))
				}
				j := haut - 1 - py
				for px := 0; px < larg; px++ {
					c := color.RGBA{0, 0, 0, 255}
					if e, ok := rendu.Eclairement(px, j); ok {
						g := uint8(255 * e)
						c = color.RGBA{g, g, g, 255}
					}
					out.Set(larg+8+px, py, c)
				}
			}
			comparePixels(t, validee, bo, rendu, out, larg, haut)

			dansStrict, dansVoisin, horsCadre := 0, 0, 0
			for _, p := range pos {
				i := int((p.X - rendu.Min[0]) / rendu.Cell)
				j := int((p.Y - rendu.Min[1]) / rendu.Cell)
				if i < 0 || i >= rendu.NX || j < 0 || j >= rendu.NY {
					horsCadre++
					continue
				}
				if _, ok := rendu.Eclairement(i, j); ok {
					dansStrict++
					dansVoisin++
					continue
				}
				trouve := false
				for dj := -1; dj <= 1 && !trouve; dj++ {
					for di := -1; di <= 1 && !trouve; di++ {
						ii, jj := i+di, j+dj
						if ii < 0 || ii >= rendu.NX || jj < 0 || jj >= rendu.NY {
							continue
						}
						if _, ok := rendu.Eclairement(ii, jj); ok {
							trouve = true
						}
					}
				}
				if trouve {
					dansVoisin++
				}
			}
			nb := len(pos) - horsCadre
			t.Logf("ORACLE : %d positions dans le cadre (%d hors) · dans la zone %d (%.2f %%) · a un pixel pres %d (%.2f %%)",
				nb, horsCadre, dansStrict, 100*float64(dansStrict)/float64(nb),
				dansVoisin, 100*float64(dansVoisin)/float64(nb))

			if dir := os.Getenv("SONDE_PNG_DIR"); dir != "" {
				// Vignette 3 : frontiere de la coquille en vert, emprise des
				// prismes en rouge, sur fond de la reference.
				for py := 0; py < haut; py++ {
					j := haut - 1 - py
					y := rendu.Min[1] + (float64(j)+0.5)*rendu.Cell
					for px := 0; px < larg; px++ {
						x := rendu.Min[0] + (float64(px)+0.5)*rendu.Cell
						out.Set(2*larg+16+px, py, validee.At(bo.Min.X+px, bo.Min.Y+py))
						in0 := coq.contient([3]float64{x, y, 0}, 0)
						bord := false
						for _, d := range [4][2]float64{{rendu.Cell, 0}, {-rendu.Cell, 0}, {0, rendu.Cell}, {0, -rendu.Cell}} {
							if coq.contient([3]float64{x + d[0], y + d[1], 0}, 0) != in0 {
								bord = true
								break
							}
						}
						if bord {
							out.Set(2*larg+16+px, py, color.RGBA{0, 200, 0, 255})
							continue
						}
						for _, v := range vols {
							if x >= v.AABBMin[0] && x <= v.AABBMax[0] && y >= v.AABBMin[1] && y <= v.AABBMax[1] {
								out.Set(2*larg+16+px, py, color.RGBA{220, 40, 40, 255})
								break
							}
						}
					}
				}
				ecritPNG(t, filepath.Join(dir, "sonde_sddt_"+cfg.nom+".png"), out)
			}
		})
	}
}

// TestSondeSddtRenduCatalyst joue la coquille de Catalyst sur le rendu carte
// quelconque (cadre des ancres), juge par l'oracle faible des ancres.
func TestSondeSddtRenduCatalyst(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	chemin, ok := chercheModule(racine, "catalyst_map")
	if !ok {
		t.Skip("catalyst absent")
	}
	var ancres [][3]float64
	for _, e := range ancresDuModule(t, "catalyst_map") {
		for _, o := range e.Objectives {
			ancres = append(ancres, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	if len(ancres) == 0 {
		t.Skip("aucune ancre catalyst")
	}
	_, ceils := litSddt(t, moduleAny(t, "catalyst"))
	coq := coquilleDepuis(ceils)
	t.Logf("coquille catalyst : %d plans distincts (sur %d triangles)", len(coq), len(ceils))
	dansCoq := 0
	for _, a := range ancres {
		if coq.contient(a, 0.5) {
			dansCoq++
		}
	}
	t.Logf("ancres dans la coquille : %d/%d", dansCoq, len(ancres))

	rendu := cadreSurAncres(t, ancres)
	rendu.Tranche(TrancheDeJeuMin, TrancheDeJeuMax)
	n, ecartees := peupleRendu(t, rendu, racine, chemin, ancres)
	t.Logf("%d instances dessinees · %d ecartees comme decor (grain actif par defaut)", n, ecartees)
	avant := 0
	for j := 0; j < rendu.NY; j++ {
		for i := 0; i < rendu.NX; i++ {
			if _, ok := rendu.Eclairement(i, j); ok {
				avant++
			}
		}
	}
	m := masqueCoquille(rendu, coq)
	rendu.Restreindre(m)
	apres := 0
	for j := 0; j < rendu.NY; j++ {
		for i := 0; i < rendu.NX; i++ {
			if _, ok := rendu.Eclairement(i, j); ok {
				apres++
			}
		}
	}
	t.Logf("pixels de matiere : %d avant coquille · %d apres (%.1f %% retires)",
		avant, apres, 100*float64(avant-apres)/float64(max(avant, 1)))
	jugeParLesAncres(t, rendu, ancres)
	if dir := os.Getenv("SONDE_PNG_DIR"); dir != "" {
		ecritPNG(t, filepath.Join(dir, "sonde_sddt_catalyst_coquille.png"),
			carteSeulePNG(rendu, rendu.NX, rendu.NY, nil, StyleCarteParDefaut))
	}
}
