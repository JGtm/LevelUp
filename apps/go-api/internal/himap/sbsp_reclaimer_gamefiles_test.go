package himap

import (
	"math"
	"path/filepath"
	"sort"
	"testing"
)

// Ce fichier solde la DETTE DE METHODE du handoff §9.3 : « le code de Reclaimer n'a jamais
// ete lu dans ce chantier ». Il ne suppose rien — il confronte ce que
// `Reclaimer.Blam/Blam/HaloInfinite/ScenarioStructureBspTag.cs` lit dans un sbsp a ce que
// notre chaine lit, et MESURE l'ecart sur les octets du jeu.
//
// Ce que Reclaimer lit et que nous ne lisions pas, etabli par lecture croisee de la source
// C# et du plugin `sbsp.xml` (les deux concordent a l'octet sur les 320 o d'une instance) :
//
//	Clusters            @300 du root  -> bloc `clusters` du plugin : la geometrie NON
//	                                    INSTANCIEE, indexee par un `mesh index`.
//	FlagsOverride       @0x110        -> `mesh flags override` ; Reclaimer SAUTE l'instance
//	                                    quand `mesh is custom shadow caster` est pose.
//	TransformScale      @0x00         -> lu chez nous, mais JAMAIS APPLIQUE.

// TestReclaimerBlocsRacine inventorie les tag-blocks racine du sbsp principal. Il repond a
// une seule question : le bloc `clusters` porte-t-il quelque chose ?
func TestReclaimerBlocsRacine(t *testing.T) {
	blocs, err := DescribeRootBlocks(moduleDuJeu(t, "pc", "ridgeline"))
	if err != nil {
		t.Fatal(err)
	}
	// Les blocs VIDES sont affiches aussi : c'est un bloc a zero qui refute une hypothese,
	// et le filtrer rendrait le diagnostic muet la ou il est le plus utile.
	for _, b := range blocs {
		t.Logf("rang %2d  offset %#06x  %-45s count=%-7d taille=%d",
			b.Rank, b.FieldOffset, b.PluginName, b.Count, b.BlockSize)
	}
}

// TestReclaimerChampsInstance recense les deux champs que Reclaimer lit et que nous
// ignorions : `scale` (@0x00) et `mesh flags override` (@0x110).
func TestReclaimerChampsInstance(t *testing.T) {
	toutes := instancesDeCliffhanger(t)

	unite, nonUnite, ombres := 0, 0, 0
	minS, maxS := math.Inf(1), math.Inf(-1)
	for _, in := range toutes {
		if in.ProjecteurOmbre() {
			ombres++
		}
		egal := true
		for a := 0; a < 3; a++ {
			minS, maxS = math.Min(minS, in.Scale[a]), math.Max(maxS, in.Scale[a])
			if math.Abs(in.Scale[a]-1) > 1e-4 {
				egal = false
			}
		}
		if egal {
			unite++
		} else {
			nonUnite++
		}
	}
	t.Logf("scale : %d instances a (1,1,1) · %d differentes · min %.4f max %.4f",
		unite, nonUnite, minS, maxS)
	t.Logf("mesh flags override : %d instances `custom shadow caster` sur %d",
		ombres, len(toutes))
	histogrammeMeshFlags(t, toutes)
}

// histogrammeMeshFlags dit si @0x110 est un vrai champ de bits ou du bruit.
//
// Un offset faux etale les valeurs sur 65 536 possibilites ; un champ de bits de 11 drapeaux
// se concentre sur une poignee de valeurs, toutes sous 1<<11. C'est ce contraste qui fait
// la verification — pas le fait que la lecture « donne des chiffres ».
func histogrammeMeshFlags(t *testing.T, toutes []Instance) {
	t.Helper()
	hist := map[uint16]int{}
	horsPortee := 0
	for _, in := range toutes {
		hist[in.MeshFlags]++
		if in.MeshFlags >= 1<<11 {
			horsPortee++
		}
	}
	type vc struct {
		v uint16
		n int
	}
	vs := make([]vc, 0, len(hist))
	for v, n := range hist {
		vs = append(vs, vc{v, n})
	}
	sort.Slice(vs, func(i, j int) bool { return vs[i].n > vs[j].n })
	t.Logf("mesh flags override : %d valeurs distinctes · %d instances hors des 11 drapeaux declares",
		len(hist), horsPortee)
	for i, e := range vs {
		if i >= 8 {
			break
		}
		t.Logf("   %#06b  %d instances", e.v, e.n)
	}
}

// TestReclaimerEchelleInstance tranche la question laissee ouverte par le commentaire
// « Scale : champ @0x00. Repute vestigial ».
//
// LE TEMOIN QUI SEPARE. La boite monde declaree de l'instance (@0x7C du sbsp) et le
// maillage (tag rtgo) viennent de deux sources independantes : la boite ne peut pas
// « donner raison » a une lecture par construction. On transforme donc le maillage des
// DEUX facons et on regarde laquelle rentre dans la boite.
func TestReclaimerEchelleInstance(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
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

	assets := map[uint32]*RuntimeGeoAsset{}
	var sans, avec []float64
	var sansNU, avecNU []float64 // restreint aux instances dont le scale n'est pas (1,1,1)
	for _, in := range instancesDeCliffhanger(t) {
		if in.QuickDeleted() {
			continue
		}
		id := in.RuntimeGeoID()
		g, mod, ok := idx.Lookup(id)
		if !ok || g != "rtgo" || filepath.Base(mod) != filepath.Base(modCarte) {
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
		diag := diagonaleBoite(in)
		if m == nil || len(m.Vertices) == 0 || !(diag > 1e-6) {
			continue
		}
		eSans := ecartBoite(m, in, false) / diag
		eAvec := ecartBoite(m, in, true) / diag
		sans, avec = append(sans, eSans), append(avec, eAvec)
		if scaleNonUnitaire(in) {
			sansNU, avecNU = append(sansNU, eSans), append(avecNU, eAvec)
		}
	}
	t.Logf("%d instances mesurees", len(sans))
	t.Logf("ecart a la boite declaree (fraction de la diagonale) — TOUTES :")
	t.Logf("   sans echelle : median %.4f   p90 %.4f", centile(sans, 0.5), centile(sans, 0.90))
	t.Logf("   avec echelle : median %.4f   p90 %.4f", centile(avec, 0.5), centile(avec, 0.90))
	if len(sansNU) == 0 {
		t.Log("aucune instance a scale != (1,1,1) : le champ est bien vestigial ici")
		return
	}
	t.Logf("ecart — seulement les %d instances a scale != (1,1,1) :", len(sansNU))
	t.Logf("   sans echelle : median %.4f   p90 %.4f", centile(sansNU, 0.5), centile(sansNU, 0.90))
	t.Logf("   avec echelle : median %.4f   p90 %.4f", centile(avecNU, 0.5), centile(avecNU, 0.90))
}

// instancesDeCliffhanger rend les instances de TOUS les bsp du module de la carte. Le
// recensement porte sur le module entier, pas sur le bsp retenu pour le dessin : un champ
// mal lu le serait partout.
func instancesDeCliffhanger(t *testing.T) []Instance {
	t.Helper()
	bsps, err := ReadModuleInstances(moduleDuJeu(t, "pc", "ridgeline"))
	if err != nil {
		t.Fatal(err)
	}
	var toutes []Instance
	for _, b := range bsps {
		toutes = append(toutes, b.Instances...)
	}
	return toutes
}

func scaleNonUnitaire(in Instance) bool {
	for ax := 0; ax < 3; ax++ {
		if math.Abs(in.Scale[ax]-1) > 1e-4 {
			return true
		}
	}
	return false
}

func diagonaleBoite(in Instance) float64 {
	d2 := 0.0
	for ax := 0; ax < 3; ax++ {
		d := in.AABBMax[ax] - in.AABBMin[ax]
		d2 += d * d
	}
	return math.Sqrt(d2)
}

// ecartBoite rend l'ecart maximal, axe par axe, entre la boite du maillage transforme et la
// boite monde declaree de l'instance.
func ecartBoite(m *Mesh, in Instance, echelle bool) float64 {
	mn := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	mx := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, s := range m.Vertices {
		var w [3]float64
		if echelle {
			w = in.LocalToWorld(s)
		} else {
			w = in.LocalToWorldSansEchelle(s)
		}
		for ax := 0; ax < 3; ax++ {
			mn[ax], mx[ax] = math.Min(mn[ax], w[ax]), math.Max(mx[ax], w[ax])
		}
	}
	pire := 0.0
	for ax := 0; ax < 3; ax++ {
		pire = math.Max(pire, math.Abs(mn[ax]-in.AABBMin[ax]))
		pire = math.Max(pire, math.Abs(mx[ax]-in.AABBMax[ax]))
	}
	return pire
}

// centile rend le centile p d'un echantillon, sans le modifier. (La `mediane` du package de
// test existe deja dans geometry_gamefiles_test.go, mais elle panique sur un echantillon
// vide — ici les echantillons peuvent l'etre si l'installation ne resout aucun tag.)
func centile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	i := int(p * float64(len(c)-1))
	return c[i]
}
