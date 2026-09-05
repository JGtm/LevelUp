//go:build gamefiles

package himap

// LA PREUVE DES REGIONS DE LIVE FIRE, MATERIALISEE EN TEST DURABLE (lot C catalogues,
// 2026-08-27).
//
// CE QUE CE TEST REJOUE. Live Fire est la premiere carte du catalogue de bornes dont le
// module (`sgh_interlock`, prouve level_id par TestPreuveLevelIDCartes) ne porte AUCUN tag
// sbsp : ses regions de compression sont resolues a travers `ds/globals` par
// `RegionsBSPExternes` (sbsp_region_externe.go), et la region JOUEE n'est pas la premiere
// de l'ordre — l'entree de catalogue declare region=1 sur un index de 2 bits. Ce test
// rejoue les preuves STATIQUES de cette declaration a chaque passage sur le jeu installe :
//
//  1. l'ordre des regions est STABLE et vaut [7047b96f, d88e1d88, a59f5052, 91c336c1]
//     (deux blocs du levl le portent, la mecanique de sbsp_region.go exige leur accord) ;
//  2. les largeurs de la region 1 valent [12 12 11] — la valeur de l'entree de catalogue ;
//  3. les ancres d'objectifs de la carte (catalogue map_objectives.json, entrees choisies
//     par LEVEL_ID, la meme cle que la preuve de module) tombent TOUTES dans la region 1,
//     et parmi les regions qui les contiennent toutes, la region 1 est la PLUS PETITE —
//     le controle croise historique du catalogue (plus petit AABB).
//
// La preuve DYNAMIQUE (l'index 01 porte par 59 376/59 377 records i0 des 2 films Oddball
// 60ae07c4/c88ec007, et le decoupage lu [13 12 11] au gate 5 = [12 12 11] a l'index pres)
// est jouee par les instruments filmdec (TestWorldObjectPrecisionLayout,
// TestControleBornesFilms) — elle ne se rejoue pas ici : ce test est sans film.

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// liveFireLevelID est le level_id de Live Fire (preuve de module, TestPreuveLevelIDCartes).
const liveFireLevelID = 1253388187

// liveFireOrdreRegions est l'ordre attendu du bloc structure-BSP du levl de sgh_interlock.
var liveFireOrdreRegions = []uint32{0x7047b96f, 0xd88e1d88, 0xa59f5052, 0x91c336c1}

// liveFireRegionJouee / liveFireLargeurs : la declaration de l'entree de catalogue.
const liveFireRegionJouee = 1

var liveFireLargeurs = [3]int{12, 12, 11}

func TestPreuveRegionsLiveFire(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	module := filepath.Join(root, "ds", "levels", "multi", "sgh_interlock", "sgh_interlock-rtx-new.module")
	porteurs, err := filepath.Glob(filepath.Join(root, "ds", "globals", "*.module"))
	if err != nil || len(porteurs) == 0 {
		t.Fatalf("globals ds introuvables (%v, %d porteur(s))", err, len(porteurs))
	}
	regions, err := RegionsBSPExternes(module, porteurs)
	if err != nil {
		t.Fatal(err)
	}

	// (1) L'ordre des regions.
	if len(regions) != len(liveFireOrdreRegions) {
		t.Fatalf("%d region(s) resolue(s), %d attendue(s)", len(regions), len(liveFireOrdreRegions))
	}
	for i, gid := range liveFireOrdreRegions {
		if regions[i].BSP.GlobalID != gid {
			t.Errorf("region %d : %08x, attendu %08x", i, regions[i].BSP.GlobalID, gid)
		}
	}

	// (2) Les largeurs de la region jouee.
	jouee := regions[liveFireRegionJouee].BSP
	if w := jouee.Bounds.AxisWidths(); w != liveFireLargeurs {
		t.Errorf("largeurs de la region jouee : %v, attendu %v (bornes %+v)", w, liveFireLargeurs, jouee.Bounds)
	}
	t.Logf("region jouee %08x (%s) : X[%.3f;%.3f] Y[%.3f;%.3f] Z[%.3f;%.3f]",
		jouee.GlobalID, regions[liveFireRegionJouee].Module,
		jouee.Bounds.Min[0], jouee.Bounds.Max[0], jouee.Bounds.Min[1], jouee.Bounds.Max[1],
		jouee.Bounds.Min[2], jouee.Bounds.Max[2])

	// (3) Les ancres d'objectifs, par level_id.
	ancres := ancresParLevelID(t, liveFireLevelID)
	if len(ancres) == 0 {
		t.Fatalf("aucune ancre d'objectif au level_id %d — la preuve des ancres ne peut pas se jouer", liveFireLevelID)
	}
	var plusPetite = -1
	for i, r := range regions {
		if !contientToutes(r.BSP.Bounds, ancres) {
			continue
		}
		if plusPetite < 0 || volume(r.BSP.Bounds) < volume(regions[plusPetite].BSP.Bounds) {
			plusPetite = i
		}
	}
	if plusPetite != liveFireRegionJouee {
		t.Errorf("la plus petite region contenant les %d ancres est la %d, attendu %d",
			len(ancres), plusPetite, liveFireRegionJouee)
	}
	t.Logf("%d ancres d'objectifs, toutes dans la region %d — plus petite region englobante : %d",
		len(ancres), liveFireRegionJouee, plusPetite)
}

// ancresParLevelID rend les positions d'objectifs des entrees du catalogue au level_id donne.
func ancresParLevelID(t *testing.T, levelID int64) [][3]float64 {
	t.Helper()
	p, err := cheminCatalogue()
	if err != nil {
		t.Skip(err)
	}
	cat, err := replay.LoadMapObjectives(p)
	if err != nil {
		t.Fatal(err)
	}
	var out [][3]float64
	for _, e := range cat.Maps {
		if e.LevelID != levelID {
			continue
		}
		for _, o := range e.Objectives {
			out = append(out, [3]float64{o.Pos.X, o.Pos.Y, o.Pos.Z})
		}
	}
	return out
}

func contientToutes(b Bounds, pts [][3]float64) bool {
	for _, p := range pts {
		for ax := 0; ax < 3; ax++ {
			if p[ax] < b.Min[ax] || p[ax] > b.Max[ax] {
				return false
			}
		}
	}
	return true
}

func volume(b Bounds) float64 { return b.Extent(0) * b.Extent(1) * b.Extent(2) }
