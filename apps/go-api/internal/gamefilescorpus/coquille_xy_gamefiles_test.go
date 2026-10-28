//go:build gamefiles

package gamefilescorpus

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/testfixtures"

	"github.com/JGtm/LevelUp/libs/himap/himap"
)

// TestCoquilleEnXY — la coquille testee a l'altitude DE JEU et non a l'altitude DESSINEE.
//
// D'OU VIENT L'IDEE. `masqueCoquille` (sonde) teste chaque pixel au z de la surface dessinee :
// un rocher haut au-dessus de l'arene sort de la coquille verticalement, la cellule est
// effacee, et la position jouee EN DESSOUS perd son sol. C'est ce qui coutait 10 points de
// positions. Or les 29 221 positions sont toutes DANS la coquille : tester a leur altitude
// devrait donc n'en perdre aucune, par construction.
//
// C'est la difference entre « ce pixel est-il dans la coquille » et « ce pixel est-il au-dessus
// d'un endroit ou l'on joue ».
func TestCoquilleEnXY(t *testing.T) {
	racine, err := himap.DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	ref := testfixtures.CarteValidee()
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

	dirAny, err := himap.LevelsDir("any")
	if err != nil {
		t.Skip(err)
	}
	sd, err := himap.LitSddt(filepath.Join(dirAny, "ridgeline", "ridgeline-rtx-new.module"))
	if err != nil {
		t.Fatal(err)
	}
	coq := sd.Coquille()

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := himap.GeometrySearchPath(racine, modCarte)
	idx, err := himap.NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := himap.ReadModuleInstances(modCarte)
	ancres := ancresCliffhanger(t)
	bsp := himap.ChoisitBSP(bsps, ancres)
	zRef := himap.MedianeZ(ancres)

	rendu := himap.NewRendu(lo, hi, gateEchelle)
	rendu.Tranche(himap.TrancheDeJeu(zRef - himap.AncrageDecalageSol))
	assets := map[uint32]*himap.RuntimeGeoAsset{}
	for _, in := range bsp.Instances {
		if in.QuickDeleted() || in.ProjecteurOmbre() {
			continue
		}
		id := in.RuntimeGeoID()
		if g, _, ok := idx.Lookup(id); !ok || g != himap.GroupeRtgo {
			continue
		}
		a, deja := assets[id]
		if !deja {
			tag, blob, err := idx.ExtractWithResources(id)
			if err != nil {
				continue
			}
			if a, err = himap.NewRuntimeGeoAsset(tag, blob); err != nil {
				continue
			}
			assets[id] = a
		}
		m := a.Mesh(in.MeshIndex)
		if m == nil || himap.EstDecorGrossier(m, in, himap.AireMaxTriangleJouable) {
			continue
		}
		rendu.AddMeshBorne(m, in, 0.5)
	}
	pos := chargePositions(t)

	masque := make([]bool, rendu.NX*rendu.NY)
	garde := 0
	for j := 0; j < rendu.NY; j++ {
		y := rendu.Min[1] + (float64(j)+0.5)*rendu.Cell
		for i := 0; i < rendu.NX; i++ {
			x := rendu.Min[0] + (float64(i)+0.5)*rendu.Cell
			if coq.Contient([3]float64{x, y, zRef}, 0) {
				masque[j*rendu.NX+i] = true
				garde++
			}
		}
	}
	t.Logf("coquille a l'altitude de jeu (%+.2f m) : %d cellules gardees sur %d",
		zRef, garde, rendu.NX*rendu.NY)

	rendu.Restreindre(masque)
	out := image.NewRGBA(image.Rect(0, 0, 3*larg+16, haut))
	comparePixels(t, validee, bo, rendu, out, larg, haut)

	dansZone := 0
	for _, p := range pos {
		i := int((p.X - rendu.Min[0]) / rendu.Cell)
		j := int((p.Y - rendu.Min[1]) / rendu.Cell)
		if i < 0 || i >= rendu.NX || j < 0 || j >= rendu.NY {
			continue
		}
		if _, ok := rendu.Altitude(i, j); ok {
			dansZone++
		}
	}
	t.Logf("ORACLE : %d/%d positions jouees gardees (%.2f %%)",
		dansZone, len(pos), 100*float64(dansZone)/float64(max(len(pos), 1)))
}

// TestCoquilleEnXYCatalyst — la meme regle sur la carte ou le grain echoue.
//
// Si la coquille a l'altitude de jeu coupe ici sans percer d'ancre, la regle est GRATUITE la
// ou elle ne sert pas (Cliffhanger : 99 % du cadre garde) et UTILE la ou le grain est muet.
func TestCoquilleEnXYCatalyst(t *testing.T) {
	racine, err := himap.DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	dirAny, err := himap.LevelsDir("any")
	if err != nil {
		t.Skip(err)
	}
	sd, err := himap.LitSddt(filepath.Join(dirAny, "catalyst", "catalyst-rtx-new.module"))
	if err != nil {
		t.Fatal(err)
	}
	coq := sd.Coquille()

	var ancres [][3]float64
	for _, e := range ancresDuModule(t, "catalyst_map") {
		for _, o := range e.Objectives {
			ancres = append(ancres, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	if len(ancres) == 0 {
		t.Skip("aucune ancre pour catalyst_map")
	}
	zRef := himap.MedianeZ(ancres)

	chemin, ok := himap.ChercheModuleInstalle("catalyst_map")
	if !ok {
		t.Skip("module catalyst absent")
	}
	rendu := cadreSurAncres(t, ancres)
	rendu.Tranche(himap.TrancheDeJeu(zRef - himap.AncrageDecalageSol))
	rendu.NiveauDeJeu(zRef)
	n, ecartees := peupleRendu(t, rendu, racine, chemin, ancres)
	t.Logf("%d instances dessinees · %d ecartees par le grain", n, ecartees)

	avant := 0
	for j := 0; j < rendu.NY; j++ {
		for i := 0; i < rendu.NX; i++ {
			if _, ok := rendu.Altitude(i, j); ok {
				avant++
			}
		}
	}
	masque := make([]bool, rendu.NX*rendu.NY)
	for j := 0; j < rendu.NY; j++ {
		y := rendu.Min[1] + (float64(j)+0.5)*rendu.Cell
		for i := 0; i < rendu.NX; i++ {
			x := rendu.Min[0] + (float64(i)+0.5)*rendu.Cell
			masque[j*rendu.NX+i] = coq.Contient([3]float64{x, y, zRef}, 0)
		}
	}
	rendu.Restreindre(masque)
	apres := 0
	for j := 0; j < rendu.NY; j++ {
		for i := 0; i < rendu.NX; i++ {
			if _, ok := rendu.Altitude(i, j); ok {
				apres++
			}
		}
	}
	t.Logf("coquille a l'altitude de jeu (%+.2f m) : %d px avant · %d apres (%.1f %% retires)",
		zRef, avant, apres, 100*float64(avant-apres)/float64(max(avant, 1)))
	jugeParLesAncres(t, rendu, ancres)

	if sortie := os.Getenv("CLOTURE_PNG"); sortie != "" {
		style := himap.StyleFond(os.Getenv("RENDU_STYLE"))
		if style == "" {
			style = himap.StyleFondParDefaut
		}
		ecritPNG(t, sortie, himap.FondPNG(rendu, rendu.NX, rendu.NY, nil, style))
	}
}
