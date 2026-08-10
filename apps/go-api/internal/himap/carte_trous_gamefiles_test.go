package himap

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

// DIAGNOSTIC DES TROUS — d'ou vient la geometrie qui manque sous les joueurs.
//
// L'oracle (`carte_oracle_gamefiles_test.go`) COMPTE les positions sans sol ; il ne dit pas
// pourquoi. Ce diagnostic prend ces positions et remonte aux instances dont la boite monde
// les contient, en publiant pour chacune : son module de resolution, et si son maillage a
// pu etre decode. Trois causes se distinguent alors, la ou un pourcentage les confond :
//
//  1. l'instance est dans un module GLOBAL -> le filtre de module la supprime ;
//  2. l'instance est dans le module de la carte mais son maillage rend nil -> LOD ou
//     descripteur ;
//  3. aucune instance ne couvre le point -> la geometrie n'est pas dans le sbsp du tout.
//
// Variable : ORACLE_REPLAY. Les autres reglages suivent l'oracle.
func TestDiagnosticTrousDeCarte(t *testing.T) {
	racine, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	pos := chargePositions(t)

	vol := construitVolume(t, optionsCarte{ZMin: VolumeZMin, ZMax: VolumeZMax, CarteSeule: true})
	sol := vol.Floors(HeadroomBands)
	var trous []pointRejeu
	for _, p := range pos {
		if _, _, ok := sol.SolLePlusProche(p.X, p.Y, p.Z); !ok {
			trous = append(trous, p)
		}
	}
	t.Logf("%d positions sans aucun sol sous elles", len(trous))
	if len(trous) == 0 {
		return
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

	// couvre : l'instance contient le point en X/Y et un sol plausible sous ses pieds.
	couvre := func(in Instance, p pointRejeu) bool {
		return p.X >= in.AABBMin[0] && p.X <= in.AABBMax[0] &&
			p.Y >= in.AABBMin[1] && p.Y <= in.AABBMax[1] &&
			in.AABBMin[2] <= p.Z+0.5 && in.AABBMax[2] >= p.Z-2.5
	}

	type cause struct {
		module   string
		maillage string
	}
	tally := map[cause]int{}
	assets := map[uint32]*RuntimeGeoAsset{}
	orphelins := 0

	// Echantillon : un point sur dix suffit a etablir la cause, et la boucle est en O(n x m).
	for k := 0; k < len(trous); k += 10 {
		p := trous[k]
		trouve := false
		for _, in := range bsp.Instances {
			if in.QuickDeleted() || !couvre(in, p) {
				continue
			}
			id := in.RuntimeGeoID()
			g, mod, ok := idx.Lookup(id)
			if !ok || g != GroupeRtgo {
				tally[cause{"(non resolu)", "-"}]++
				trouve = true
				continue
			}
			a, deja := assets[id]
			if !deja {
				tag, blob, err := idx.ExtractWithResources(id)
				if err != nil {
					tally[cause{filepath.Base(mod), "extraction KO"}]++
					trouve = true
					continue
				}
				if a, err = NewRuntimeGeoAsset(tag, blob); err != nil {
					tally[cause{filepath.Base(mod), "asset KO"}]++
					trouve = true
					continue
				}
				assets[id] = a
			}
			etat := "maillage OK"
			if a.Mesh(in.MeshIndex) == nil {
				etat = "maillage NIL"
			}
			tally[cause{filepath.Base(mod), etat}]++
			trouve = true
		}
		if !trouve {
			orphelins++
		}
	}

	type ligne struct {
		c cause
		n int
	}
	var lignes []ligne
	for c, n := range tally {
		lignes = append(lignes, ligne{c, n})
	}
	sort.Slice(lignes, func(a, b int) bool { return lignes[a].n > lignes[b].n })
	t.Logf("echantillon de %d trous · %d sans AUCUNE instance qui les couvre",
		(len(trous)+9)/10, orphelins)
	for _, l := range lignes {
		t.Logf("  %-32s %-14s %d", l.c.module, l.c.maillage, l.n)
	}
	fmt.Println("diagnostic termine")

}
