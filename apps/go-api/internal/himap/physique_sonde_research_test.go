//go:build gamefiles && research

package himap

// physique_sonde_research_test.go — INSTRUMENT de rendu physique qui lit le depot hote (D1 lot
// 2c, 2026-10-28, decision utilisateur : conserve comme instrument de recherche).
//
// Il compare les rendus aux ancres d'objectifs et aux positions jouees du depot hote, que la
// brique ne possede pas. A la demande :
// `go test -tags=gamefiles,research ./internal/himap/ -run TestSondePhysiqueRendu`.
// Les deux autres sondes physiques (decodage du bloc, resolution) restent sous `gamefiles`.

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

type itemRendu struct {
	m       *Mesh
	in      Instance
	aire    float64
	physAny bool // un homologue physique existe (guid)
	physJeu bool // ... et il porte le drapeau Play
}

func TestSondePhysiqueRendu(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	ref := fixtureCarteValidee
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

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	bsp := ChoisitBSP(bsps, ancresCliffhanger(t))

	// Le bloc physique du MEME bsp que le rendu.
	tag, bspP := bspPrincipal(t, modCarte)
	if bspP.FileIndex != bsp.FileIndex {
		t.Fatalf("bsp du rendu (%d) != bsp de la sonde (%d)", bsp.FileIndex, bspP.FileIndex)
	}
	phys, _, _, err := physiqueEtGuidsDuTag(tag, 0)
	if err != nil {
		t.Fatal(err)
	}
	physMasque := map[uint32]uint32{}
	for _, p := range phys {
		physMasque[p.Guid] |= p.TypeMask
	}
	guidRendu := guidsRendus(tag)
	if len(guidRendu) != len(bsp.Instances) {
		t.Fatalf("%d guids de rendu pour %d instances", len(guidRendu), len(bsp.Instances))
	}

	// Une seule passe d'extraction : chaque instance dessinable, avec son maillage,
	// son aire mediane et son etat physique.
	assets := map[uint32]*RuntimeGeoAsset{}
	var items []itemRendu
	for _, in := range bsp.Instances {
		if in.QuickDeleted() || in.ProjecteurOmbre() {
			continue
		}
		id := in.RuntimeGeoID()
		if g, _, ok := idx.Lookup(id); !ok || g != GroupeRtgo {
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
		masque, ok := physMasque[guidRendu[in.Index]]
		items = append(items, itemRendu{
			m: m, in: in,
			aire:    AireMedianeProjetee(m, in),
			physAny: ok,
			physJeu: ok && masque&phyMaskPlay != 0,
		})
	}
	t.Logf("%d instances dessinables (maillage resolu, hors ombres/supprimees)", len(items))

	pos := chargePositions(t)

	// `marche` > 0 applique, APRES rendu, le masque d'accessibilite depuis les ancres. Les
	// tris par instance et le masque de connexite ne sont pas de meme nature : l'un choisit
	// ce qu'on dessine, l'autre ce qui reste atteignable une fois dessine.
	configs := []struct {
		nom    string
		garde  func(itemRendu) bool
		marche float64
	}{
		{"aucun-tri", func(it itemRendu) bool { return true }, 0},
		{"grain-0.005", func(it itemRendu) bool { return it.aire <= AireMaxTriangleJouable }, 0},
		{"physique-any", func(it itemRendu) bool { return it.physAny }, 0},
		{"acces-0.85", func(it itemRendu) bool { return true }, MarcheMaxMetres},
		{"acces-0.50", func(it itemRendu) bool { return true }, 0.50},
		{"acces-1.20", func(it itemRendu) bool { return true }, 1.20},
		{"acces-et-grain", func(it itemRendu) bool { return it.aire <= AireMaxTriangleJouable }, MarcheMaxMetres},
	}
	for _, cfg := range configs {
		t.Run(cfg.nom, func(t *testing.T) {
			rendu := NewRendu(lo, hi, gateEchelle)
			rendu.Tranche(TrancheDeJeu(MedianeZ(ancresCliffhanger(t)) - AncrageDecalageSol))
			n := 0
			for _, it := range items {
				if !cfg.garde(it) {
					continue
				}
				n++
				rendu.AddMeshBorne(it.m, it.in, 0.5)
			}
			t.Logf("%s : %d/%d instances gardees", cfg.nom, n, len(items))
			if cfg.marche > 0 {
				acc := rendu.ComposanteAccessible(ancresCliffhanger(t), cfg.marche)
				garde := 0
				for _, ok := range acc {
					if ok {
						garde++
					}
				}
				rendu.Restreindre(acc)
				t.Logf("%s : composante accessible (marche %.2f m) = %d cellules", cfg.nom, cfg.marche, garde)
			}

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

			// L'ORACLE : chaque position jouee doit trouver de la matiere dans sa
			// colonne (stricte, puis avec une tolerance d'un pixel).
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
				ecritPNG(t, filepath.Join(dir, "sonde_physique_"+cfg.nom+".png"), out)
			}
		})
	}
}
