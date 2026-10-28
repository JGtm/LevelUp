//go:build gamefiles

package himap

import (
	"encoding/binary"
	"errors"
	"testing"
)

// TestRuntimeGeoResolutionMultiModule — le témoin de T1, en trois volets falsifiables.
//
//  1. L'OFFSET de la référence est le bon. Piège rencontré le 2026-08-08 : les offsets 8 ET
//     12 résolvent tous deux vers des tags `rtgo` valides (98,6 % des instances y portent
//     la même valeur), et le taux de résolution seul ne les départage pas — pas plus que la
//     borne de `MeshIndex`, satisfaite à 100 % par les deux. Ce qui les départage : à
//     l'offset 8 la résolution est un SUR-ENSEMBLE STRICT (147 instances de plus, et jamais
//     l'inverse). Le test exige donc que `refOffGlobalID` soit l'argmax STRICT.
//  2. La RÉSOLUTION est multi-module. Le module de la carte seul ne couvre que 26 % des
//     instances : une régression vers le mono-module fait tomber le seuil.
//  3. Le PAS DE 144. Chaque tag `rtgo` atteint doit porter un bloc `Per Mesh Data` de
//     taille multiple entier de 144, et le `MeshIndex` de chaque instance doit tomber DANS
//     le nombre de maillages lu. Un offset de champ faux ne donne pas un multiple entier.
//
// Mutations qui doivent le faire rougir : `refOffGlobalID` de 8 à 12 (volet 1),
// `rtgoOffPerMeshData` de 16 à autre chose (volet 3).
func TestRuntimeGeoResolutionMultiModule(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skipf("installation introuvable: %v", err)
	}
	modCarte := moduleDuJeu(t, "pc", "ridgeline")

	chemins, err := GeometrySearchPath(racine, modCarte)
	if err != nil {
		t.Fatalf("chemin de recherche: %v", err)
	}
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	t.Logf("index : %d modules, %d entrées", len(chemins), idx.Taille())

	bsps, err := ReadModuleInstances(modCarte)
	if err != nil {
		t.Fatal(err)
	}
	var ins []Instance
	for _, b := range bsps {
		if len(b.Instances) > len(ins) {
			ins = b.Instances
		}
	}
	if len(ins) < minInstances {
		t.Fatalf("instances = %d, attendu >= %d", len(ins), minInstances)
	}

	// Volet 1 — l'offset retenu doit être l'argmax STRICT de la résolution.
	resolutionA := func(off int) int {
		n := 0
		for _, in := range ins {
			id := binary.LittleEndian.Uint32(in.MeshRef[off:])
			if g, _, ok := idx.Lookup(id); ok && g == GroupeRtgo {
				n++
			}
		}
		return n
	}
	retenu := resolutionA(refOffGlobalID)
	for off := 0; off+4 <= insSizeMeshRef; off += 4 {
		if off == refOffGlobalID {
			continue
		}
		if n := resolutionA(off); n >= retenu {
			t.Errorf("offset %d résout %d instances, l'offset retenu %d n'en résout que %d — "+
				"la référence n'est pas là où on croit", off, n, refOffGlobalID, retenu)
		}
	}

	// Volet 2 — la résolution vient des modules globaux autant que de celui de la carte.
	taux := 100 * float64(retenu) / float64(len(ins))
	t.Logf("instances résolues : %d/%d (%.1f %%)", retenu, len(ins), taux)
	if taux < 80 {
		t.Errorf("résolution %.1f %%, attendu >= 80 %% — l'index multi-module ne joue plus "+
			"(le module de la carte seul en couvre 26 %%)", taux)
	}

	// Volet 3 — le pas de 144, et la borne de MeshIndex, sur TOUS les tags atteints.
	nMesh := map[uint32]int{}
	meshes, horsBorne := 0, 0
	for _, in := range ins {
		id := in.RuntimeGeoID()
		g, _, ok := idx.Lookup(id)
		if !ok || g != GroupeRtgo {
			continue
		}
		n, vu := nMesh[id]
		if !vu {
			blob, err := idx.Extract(id)
			if err != nil {
				t.Fatalf("extraction du tag %#x: %v", id, err)
			}
			rg, err := ReadRuntimeGeoTag(blob)
			if err != nil {
				if errors.Is(err, ErrPerMeshStride) {
					t.Fatalf("tag %#x : %v — l'offset de Per Mesh Data ne désigne pas un tableau de 144", id, err)
				}
				t.Fatalf("tag %#x : %v", id, err)
			}
			n = rg.MeshCount
			nMesh[id] = n
			meshes += n
		}
		if in.MeshIndex >= n {
			horsBorne++
		}
	}
	t.Logf("tags rtgo ouverts : %d, %d maillages au total", len(nMesh), meshes)
	if horsBorne > 0 {
		t.Errorf("%d instances ont un MeshIndex hors du nombre de maillages de leur tag", horsBorne)
	}
	if meshes == 0 {
		t.Error("aucun maillage lu — la chaîne s'arrête avant T2")
	}
}
