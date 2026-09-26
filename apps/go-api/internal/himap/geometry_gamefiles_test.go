//go:build gamefiles

package himap

import (
	"encoding/binary"
	"math"
	"sort"
	"testing"
)

// TestGeometrieCliffhanger — les témoins T2 et T3 du portage, sur ridgeline. T4 y est
// MESURÉ mais non tranché : voir le bloc T4 en fin de test.
//
//	T2 : chaque maillage atteint livre son bloc de LOD render data (multiple de 148).
//	T3 : aucun descripteur retenu ne pointe hors du blob de ressources.
//
// La règle du chantier tient en une ligne : un témoin qui ne départage pas ne teste rien.
func TestGeometrieCliffhanger(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skipf("installation introuvable: %v", err)
	}
	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, err := GeometrySearchPath(racine, modCarte)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, err := ReadModuleInstances(modCarte)
	if err != nil {
		t.Fatal(err)
	}
	var bsp BSPInstances
	for _, b := range bsps {
		if len(b.Instances) > len(bsp.Instances) {
			bsp = b
		}
	}

	assets := map[uint32]*RuntimeGeoAsset{}
	var ecarts, ecartsFaux []float64
	rendues, sansGeo, maillagesNuls, triangles := 0, 0, 0, 0
	for _, in := range bsp.Instances {
		if in.QuickDeleted() {
			continue
		}
		id := in.RuntimeGeoID()
		if g, _, ok := idx.Lookup(id); !ok || g != GroupeRtgo {
			sansGeo++
			continue
		}
		a, deja := assets[id]
		if !deja {
			tag, blob, err := idx.ExtractWithResources(id)
			if err != nil {
				t.Fatalf("extraction %#x: %v", id, err)
			}
			if a, err = NewRuntimeGeoAsset(tag, blob); err != nil {
				t.Fatalf("asset %#x: %v", id, err)
			}
			assets[id] = a
			// T3 — aucun descripteur retenu ne sort du blob.
			for k, d := range a.vertexDescs {
				if d.Offset < 0 || d.Offset+d.Size > len(blob) {
					t.Errorf("tag %#x : descripteur de sommets %d hors du blob (%d+%d > %d)",
						id, k, d.Offset, d.Size, len(blob))
				}
			}
			for k, d := range a.indexDescs {
				if d.Offset < 0 || d.Offset+d.Size > len(blob) {
					t.Errorf("tag %#x : descripteur d'indices %d hors du blob (%d+%d > %d)",
						id, k, d.Offset, d.Size, len(blob))
				}
			}
		}
		m := a.Mesh(in.MeshIndex)
		if m == nil {
			maillagesNuls++
			continue
		}
		rendues++
		triangles += len(m.Triangles)

		// T4 — les DEUX lectures, sur les memes octets.
		mn, mx, okB := a.bounds(in.MeshIndex)
		vd, id2, okV := a.MeshBuffers(in.MeshIndex)
		if !okB || !okV {
			t.Errorf("tag %#x maillage %d : bornes ou tampon illisibles", id, in.MeshIndex)
			continue
		}
		if r := coherenceAretes(a.Blob(), vd, id2, mn, mx, false); r > 0 {
			ecarts = append(ecarts, r)
		}
		if r := coherenceAretes(a.Blob(), vd, id2, mn, mx, true); r > 0 {
			ecartsFaux = append(ecartsFaux, r)
		}
	}

	t.Logf("instances rendues %d · sans géométrie %d · maillages nuls %d · %d tags · %d triangles",
		rendues, sansGeo, maillagesNuls, len(assets), triangles)

	// T2 — un maillage atteint doit livrer sa géométrie. Zéro nul est le seuil : un
	// maillage nul signifie un LOD introuvable ou un descripteur incohérent.
	if maillagesNuls > 0 {
		t.Errorf("%d maillages n'ont livré aucune géométrie", maillagesNuls)
	}
	if triangles == 0 {
		t.Fatal("aucun triangle produit")
	}

	// T4 — LA DÉQUANTIFICATION N'EST PAS DÉPARTAGEABLE PAR UNE STATISTIQUE DE MAILLAGE.
	//
	// Trois mesures ont été tentées le 2026-08-08 pour choisir entre le `u16` brut et le
	// `i16 + 32768` du prototype Python, sur les MÊMES octets :
	//
	//	écart aux bornes déclarées   : 16,9 mm contre 2,1 mm — donne raison à la FAUSSE
	//	longueur médiane d'arête     : 0,0189 contre 0,0196 — ne sépare pas
	//	part d'arêtes > 1/4 de boîte : 0,0821 contre 0,0897 — ne sépare pas
	//
	// La première est BIAISÉE et il ne faut pas la réintroduire : une rotation du quantum
	// disperse les valeurs vers les deux extrêmes, donc elle épouse mieux les bornes par
	// construction. Les deux autres ne séparent pas, et leur échec est instructif : la
	// rotation ne DÉCHIRE pas les maillages, elle les DÉCALE chacun d'une demi-boîte. La
	// forme de chaque maillage reste donc intacte — c'est leur REGISTRE MUTUEL qui casse,
	// ce qu'aucune statistique interne à un maillage ne peut voir.
	//
	// Le juge est donc un oracle EXTERNE, et il existe : les positions de joueurs du film.
	// Un joueur se tient sur le sol ; une géométrie décalée le fait flotter ou s'enfoncer.
	// C'est le témoin de T6, et c'est là que la lecture sera tranchée. En attendant, le
	// `u16` brut est retenu sur la foi du handoff §2.1 et du rendu comparé — pas sur celle
	// d'une mesure de ce test, qui ne le dit pas.
	if len(ecarts) == 0 || len(ecartsFaux) == 0 {
		t.Fatal("aucune mesure de cohérence")
	}
	t.Logf("part d'arêtes traversant plus du quart de la boîte : u16 brut %.4f · i16+32768 %.4f "+
		"(les deux lectures ne se séparent pas ici — cf. T6)",
		mediane(ecarts), mediane(ecartsFaux))
}

// coherenceAretes rend la PART des aretes plus longues que le quart de la diagonale de la
// boite du maillage. `faux` rejoue la lecture `i16 + 32768` du prototype Python.
func coherenceAretes(blob []byte, vd, id bufferDesc, mn, mx [3]float64, faux bool) float64 {
	if vd.Usage != vertexUsagePosition || vd.Stride != vertexStridePosition ||
		vd.Offset < 0 || vd.Offset+vd.Size > len(blob) ||
		id.Offset < 0 || id.Offset+id.Size > len(blob) {
		return 0
	}
	diag := math.Sqrt((mx[0]-mn[0])*(mx[0]-mn[0]) + (mx[1]-mn[1])*(mx[1]-mn[1]) + (mx[2]-mn[2])*(mx[2]-mn[2]))
	if diag <= 0 {
		return 0
	}
	verts := make([][3]float64, vd.Count)
	for i := 0; i < vd.Count; i++ {
		r := blob[vd.Offset+i*vertexStridePosition:]
		for ax := 0; ax < 3; ax++ {
			u := binary.LittleEndian.Uint16(r[ax*2:])
			q := float64(u)
			if faux {
				q = float64(int16(u)) + 32768
			}
			verts[i][ax] = mn[ax] + q/quantMax*(mx[ax]-mn[ax])
		}
	}
	var long []float64
	for i := 0; i+2 < id.Count && len(long) < 3000; i += 3 {
		var t [3]int
		hors := false
		for k := 0; k < 3; k++ {
			var v int
			if id.Stride == 2 {
				v = int(binary.LittleEndian.Uint16(blob[id.Offset+(i+k)*2:]))
			} else {
				v = int(binary.LittleEndian.Uint32(blob[id.Offset+(i+k)*4:]))
			}
			if v >= len(verts) {
				hors = true
				break
			}
			t[k] = v
		}
		if hors {
			continue
		}
		for k := 0; k < 3; k++ {
			a, b := verts[t[k]], verts[t[(k+1)%3]]
			d := math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
			long = append(long, d/diag)
		}
	}
	if len(long) == 0 {
		return 0
	}
	// La PART d'arêtes anormalement longues, pas leur médiane. Mesuré le 2026-08-08 : la
	// médiane ne sépare pas les deux lectures (0,0189 contre 0,0196) parce que la
	// déchirure ne touche qu'une minorité d'arêtes — celles qui enjambent la frontière de
	// rotation. C'est leur proportion qui doit exploser, pas la tendance centrale.
	longues := 0
	for _, d := range long {
		if d > 0.25 {
			longues++
		}
	}
	return float64(longues) / float64(len(long))
}

func mediane(v []float64) float64 {
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[len(c)/2]
}
