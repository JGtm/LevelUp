package himap

// SONDE (2026-08-10, non commitee) — variante laterale de la coquille sddt et
// inventaire pfnd dans les trois variantes de depot.

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/himodule"
)

// TestSondeSddtRenduXY : la coquille reduite a ses plans lateraux (|nz| < 0,9),
// evaluee au z du toit de matiere — ferme la question « le plafond a tout gache ».
func TestSondeSddtRenduXY(t *testing.T) {
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

	_, ceils := litSddt(t, moduleAny(t, "ridgeline"))
	var lateraux coquilleSddt
	for _, p := range coquilleDepuis(ceils) {
		if p[2] > -0.9 && p[2] < 0.9 {
			lateraux = append(lateraux, p)
		}
	}
	t.Logf("coquille laterale : %d plans", len(lateraux))

	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	bsp := choisitBSP(bsps, ancresCliffhanger(t))
	assets := map[uint32]*RuntimeGeoAsset{}
	rendu := NewRendu(lo, hi, gateEchelle)
	rendu.Tranche(TrancheDeJeuMin, TrancheDeJeuMax)
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
		rendu.AddMeshBorne(m, in, 0.5)
	}
	rendu.Restreindre(masqueCoquille(rendu, lateraux))
	out := image.NewRGBA(image.Rect(0, 0, 3*larg+16, haut))
	comparePixels(t, validee, bo, rendu, out, larg, haut)

	pos := chargePositions(t)
	dans := 0
	for _, p := range pos {
		i := int((p.X - rendu.Min[0]) / rendu.Cell)
		j := int((p.Y - rendu.Min[1]) / rendu.Cell)
		if i < 0 || i >= rendu.NX || j < 0 || j >= rendu.NY {
			continue
		}
		if _, ok := rendu.Eclairement(i, j); ok {
			dans++
		}
	}
	t.Logf("ORACLE : %d/%d positions dans la zone (%.2f %%)", dans, len(pos), 100*float64(dans)/float64(len(pos)))
}

// TestSondePfndPartout : ou vivent les tags pfnd, et quelle taille font-ils ?
func TestSondePfndPartout(t *testing.T) {
	if _, err := DeployRoot(); err != nil {
		t.Skip(err)
	}
	for _, variante := range []string{"any", "pc", "ds"} {
		dir, err := LevelsDir(variante)
		if err != nil {
			continue
		}
		for _, carte := range []string{"ridgeline", "catalyst"} {
			p := filepath.Join(dir, carte, carte+"-rtx-new.module")
			if _, err := os.Stat(p); err != nil {
				// Les modules ds n'ont pas toujours le suffixe rtx.
				p = filepath.Join(dir, carte, carte+".module")
				if _, err := os.Stat(p); err != nil {
					t.Logf("%s/%s : module absent", variante, carte)
					continue
				}
			}
			m, err := himodule.Open(p)
			if err != nil {
				t.Logf("%s/%s : %v", variante, carte, err)
				continue
			}
			for _, grp := range []string{"pfnd", "sddt"} {
				for _, f := range m.Files(grp) {
					t.Logf("%s/%s : %s#%d — %d o decompresse", variante, carte, grp, f.Index, f.UncompSize)
				}
			}
		}
	}
}
