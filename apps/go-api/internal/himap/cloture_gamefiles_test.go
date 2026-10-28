//go:build gamefiles

package himap

import (
	"image"
	"image/png"
	"os"
	"testing"
)

// L'HYPOTHESE DE L'UTILISATEUR (2026-08-10) : « toutes les cartes natives ont un environnement
// FERME, donc peut-etre que le probleme ne se pose pas ».
//
// Elle propose un critere que les sept precedents n'avaient pas : tous cherchaient a qualifier
// la MATIERE (son module, son grain, sa collision, sa connexite depuis l'interieur). Celui-ci
// part de l'EXTERIEUR — on inonde le cadre depuis ses bords a hauteur de poitrine, et ce que
// l'eau n'atteint pas est enferme.
//
// Juge : l'oracle des 29 221 positions de joueur, et l'accord avec la carte validee. Criteres
// annonces d'avance, les memes que pour les sept precedents : garder >= 95 % des positions
// jouees ET faire mieux que 66,7 % d'accord.
func TestClotureCliffhanger(t *testing.T) {
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
	ancres := ancresCliffhanger(t)
	bsp := ChoisitBSP(bsps, ancres)

	// Le sol de reference : la mediane des ancres, comme partout ailleurs.
	zRef := MedianeZ(ancres)
	t.Logf("sol de reference (mediane des ancres) : %+.2f m", zRef)

	rendu := NewRendu(lo, hi, gateEchelle)
	rendu.Tranche(TrancheDeJeu(zRef - AncrageDecalageSol))
	vol := NewVolumeZ(lo, hi, TrancheDeJeuMin, TrancheDeJeuMax)

	assets := map[uint32]*RuntimeGeoAsset{}
	n := 0
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
			tag, blob, err := idx.ExtractWithResources(id)
			if err != nil {
				continue
			}
			if a, err = NewRuntimeGeoAsset(tag, blob); err != nil {
				continue
			}
			assets[id] = a
		}
		m := a.Mesh(in.MeshIndex)
		if m == nil || EstDecorGrossier(m, in, AireMaxTriangleJouable) {
			continue
		}
		n++
		rendu.AddMeshBorne(m, in, 0.5)
		vol.AddMeshBorne(m, in, 0.5)
	}
	t.Logf("%d instances dessinees (grain actif, comme le defaut)", n)
	pos := chargePositions(t)

	pleines := 0
	for j := 0; j < rendu.NY; j++ {
		for i := 0; i < rendu.NX; i++ {
			if _, ok := rendu.Altitude(i, j); ok {
				pleines++
			}
		}
	}

	// Plusieurs hauteurs de recherche des murs : le critere ne doit pas dependre d'un
	// reglage fin, et s'il en depend il faut le SAVOIR.
	for _, h := range []float64{0.6, HauteurClotureMetres, 2.0} {
		close := vol.ZoneClose(zRef+h-0.25, zRef+h+0.25)
		dedans := 0
		for _, ok := range close {
			if ok {
				dedans++
			}
		}
		masque := make([]bool, rendu.NX*rendu.NY)
		garde := 0
		for j := 0; j < rendu.NY; j++ {
			y := rendu.Min[1] + (float64(j)+0.5)*rendu.Cell
			for i := 0; i < rendu.NX; i++ {
				x := rendu.Min[0] + (float64(i)+0.5)*rendu.Cell
				k, dans := vol.CelluleDe(x, y)
				if dans && close[k] {
					masque[j*rendu.NX+i] = true
					garde++
				}
			}
		}
		t.Logf("hauteur %+.2f m : %d cellules de volume enfermees · masque garde %d px sur %d pleins (%.1f %%)",
			h, dedans, garde, pleines, 100*float64(garde)/float64(max(pleines, 1)))

		copie := *rendu
		copie.z = append([]float64(nil), rendu.z...)
		copie.Restreindre(masque)
		out := image.NewRGBA(image.Rect(0, 0, 3*larg+16, haut))
		comparePixels(t, validee, bo, &copie, out, larg, haut)

		// L'ORACLE, et c'est LUI le critere disqualifiant : une zone qui perd des positions
		// reellement jouees est perdante, quelle que soit son elegance.
		dansZone := 0
		for _, p := range pos {
			i := int((p.X - copie.Min[0]) / copie.Cell)
			j := int((p.Y - copie.Min[1]) / copie.Cell)
			if i < 0 || i >= copie.NX || j < 0 || j >= copie.NY {
				continue
			}
			if _, ok := copie.Altitude(i, j); ok {
				dansZone++
			}
		}
		t.Logf("hauteur %+.2f m : ORACLE %d/%d positions jouees gardees (%.2f %%)",
			h, dansZone, len(pos), 100*float64(dansZone)/float64(max(len(pos), 1)))
	}
}
