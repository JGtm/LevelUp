package himap

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
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

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	ancres := ancresCliffhanger(t)
	bsp := choisitBSP(bsps, ancres)

	// Le sol de reference : la mediane des ancres, comme partout ailleurs.
	zRef := medianeZ(ancres)
	t.Logf("sol de reference (mediane des ancres) : %+.2f m", zRef)

	rendu := NewRendu(lo, hi, gateEchelle)
	rendu.Tranche(TrancheDeJeuMin, TrancheDeJeuMax)
	vol := NewVolumeZ(lo, hi, TrancheDeJeuMin, TrancheDeJeuMax)

	assets := map[uint32]*RuntimeGeoAsset{}
	n := 0
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

func medianeZ(pts [][3]float64) float64 {
	zs := make([]float64, 0, len(pts))
	for _, p := range pts {
		zs = append(zs, p[2])
	}
	if len(zs) == 0 {
		return 0
	}
	return centile(zs, 0.5)
}

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

	dirAny, err := LevelsDir("any")
	if err != nil {
		t.Skip(err)
	}
	sd, err := LitSddt(filepath.Join(dirAny, "ridgeline", "ridgeline-rtx-new.module"))
	if err != nil {
		t.Fatal(err)
	}
	coq := sd.Coquille()

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	ancres := ancresCliffhanger(t)
	bsp := choisitBSP(bsps, ancres)
	zRef := medianeZ(ancres)

	rendu := NewRendu(lo, hi, gateEchelle)
	rendu.Tranche(TrancheDeJeuMin, TrancheDeJeuMax)
	assets := map[uint32]*RuntimeGeoAsset{}
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
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	dirAny, err := LevelsDir("any")
	if err != nil {
		t.Skip(err)
	}
	sd, err := LitSddt(filepath.Join(dirAny, "catalyst", "catalyst-rtx-new.module"))
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
	zRef := medianeZ(ancres)

	chemin, ok := chercheModule(racine, "catalyst_map")
	if !ok {
		t.Skip("module catalyst absent")
	}
	rendu := cadreSurAncres(t, ancres)
	rendu.Tranche(TrancheDeJeuMin, TrancheDeJeuMax)
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
		ecritPNG(t, sortie, carteSeulePNG(rendu, rendu.NX, rendu.NY, nil, StyleCarteParDefaut))
	}
}
