//go:build gamefiles

package himap

import (
	"math"
	"path/filepath"
	"sort"
	"testing"
)

// LES INSTANCES RESOLUES DANS UN MODULE GLOBAL SONT-ELLES BIEN RESOLUES ?
//
// Enjeu. L'oracle etablit deux faits qui se contredisent en apparence : 11,1 % des positions
// jouees n'ont AUCUNE matiere sous elles dans le module de la carte, et leur geometrie est
// couverte par des instances qui resolvent dans les modules globaux (`common`, `multiplayer`) ;
// mais integrer ces modules FAIT CHUTER la justesse des altitudes. Deux lectures possibles :
// soit ces instances sont justes et c'est la regle de praticabilite qui les digere mal, soit
// elles sont MAL RESOLUES et deversent de la matiere quelconque sur la carte.
//
// Le temoin. Chaque instance porte sa boite monde declaree (AABB @0x7C), independamment du
// maillage. On decode le maillage, on le place, et on mesure l'ecart entre l'emprise reelle
// des sommets et la boite declaree. Ce n'est PAS le critere tautologique ecarte par le
// handoff du 26/07 (« les bornes locales transformees reproduisent l'AABB ») : ici on compare
// les SOMMETS DECODES, pas la boite de quantification. Un maillage etranger a l'instance
// remplit mal sa boite.
//
// La comparaison est RELATIVE : la population « module de la carte » sert d'etalon a la
// population « modules globaux ». Un chiffre absolu ne dirait rien.
func TestGlobauxBienResolus(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	modCarte := moduleDuJeu(t, "pc", "ridgeline")
	chemins, _ := GeometrySearchPath(racine, modCarte)
	idx, err := NewModuleIndex(chemins...)
	if err != nil {
		t.Fatal(err)
	}
	bsps, _ := ReadModuleInstances(modCarte)
	var bsp BSPInstances
	for _, b := range bsps {
		if len(b.Instances) > len(bsp.Instances) {
			bsp = b
		}
	}

	ecarts := map[string][]float64{}
	assets := map[uint32]*RuntimeGeoAsset{}
	for _, in := range bsp.Instances {
		if in.QuickDeleted() {
			continue
		}
		id := in.RuntimeGeoID()
		g, mod, ok := idx.Lookup(id)
		if !ok || g != GroupeRtgo {
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
		if m == nil || len(m.Vertices) == 0 {
			continue
		}
		lo := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
		hi := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
		for _, s := range m.Vertices {
			w := in.LocalToWorld(s)
			for k := 0; k < 3; k++ {
				lo[k] = math.Min(lo[k], w[k])
				hi[k] = math.Max(hi[k], w[k])
			}
		}
		// Ecart rapporte a la diagonale de la boite declaree : sans quoi un gros rocher et
		// une caisse ne se comparent pas.
		diag := 0.0
		pire := 0.0
		for k := 0; k < 3; k++ {
			d := in.AABBMax[k] - in.AABBMin[k]
			diag += d * d
			pire = math.Max(pire, math.Abs(lo[k]-in.AABBMin[k]))
			pire = math.Max(pire, math.Abs(hi[k]-in.AABBMax[k]))
		}
		diag = math.Sqrt(diag)
		if diag <= 0 {
			continue
		}
		cle := "modules globaux"
		if baseEgale(mod, modCarte) {
			cle = "module de la carte"
		}
		ecarts[cle] = append(ecarts[cle], pire/diag)
	}

	cles := make([]string, 0, len(ecarts))
	for k := range ecarts {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	for _, k := range cles {
		v := ecarts[k]
		sort.Float64s(v)
		q := func(f float64) float64 { return v[int(f*float64(len(v)-1))] }
		t.Logf("%-20s n=%5d  ecart/diagonale  median %.3f  q90 %.3f  q99 %.3f  part>0,25 %.1f%%",
			k, len(v), q(0.5), q(0.9), q(0.99), 100*part(v, 0.25))
	}
}

func part(v []float64, seuil float64) float64 {
	n := 0
	for _, x := range v {
		if x > seuil {
			n++
		}
	}
	return float64(n) / float64(len(v))
}

// baseEgale compare deux chemins de module par leur seul nom de fichier.
func baseEgale(a, b string) bool { return filepath.Base(a) == filepath.Base(b) }
