package main

import (
	"testing"

	"github.com/JGtm/LevelUp/libs/film/decfilm"

	"github.com/JGtm/LevelUp/libs/himap/himap"
)

// regionsSynthetiques rend n régions valides, de tailles croissantes (aucune région invalide).
func regionsSynthetiques(n int) []himap.BSP {
	out := make([]himap.BSP, n)
	for i := range out {
		etendue := float64(100 * (i + 1))
		out[i].Bounds = himap.Bounds{Min: [3]float64{0, 0, 0}, Max: [3]float64{etendue, etendue, etendue}}
	}
	return out
}

// entreeDeLaRegion0 reproduit les champs historiques d'une entrée à partir de la région 0, comme le fait
// entreeRegionLocale.
func entreeDeLaRegion0(regions []himap.BSP) decfilm.MapQuantEntry {
	e := decfilm.MapQuantEntry{Module: "essai"}
	w := regions[0].Bounds.AxisWidths()
	for ax := 0; ax < 3; ax++ {
		e.Min[ax] = float32(regions[0].Bounds.Min[ax])
		e.Max[ax] = float32(regions[0].Bounds.Max[ax])
		e.AxisWidths[ax] = uint(w[ax])
	}
	return e
}

// TestLaLargeurDIndexNEstEcriteQueSiElleDiffereDeLImplicite : une ou deux régions laissent le champ
// absent (le catalogue livré reste identique octet pour octet), trois et quatre l'écrivent à 2, et la
// largeur effective égale toujours la loi du moteur sur le compte brut de régions.
func TestLaLargeurDIndexNEstEcriteQueSiElleDiffereDeLImplicite(t *testing.T) {
	cas := []struct {
		regions   int
		ecrit     uint
		effective uint
	}{
		{regions: 1, ecrit: 0, effective: 1},
		{regions: 2, ecrit: 0, effective: 1},
		{regions: 3, ecrit: 2, effective: 2},
		{regions: 4, ecrit: 2, effective: 2},
		{regions: 5, ecrit: 3, effective: 3},
	}
	for _, c := range cas {
		regions := regionsSynthetiques(c.regions)
		e := entreeDeLaRegion0(regions)
		if err := poserLesRegions(&e, regions, 0); err != nil {
			t.Fatalf("%d région(s) : %v", c.regions, err)
		}
		if e.RegionIndexBits != c.ecrit {
			t.Errorf("%d région(s) : RegionIndexBits écrit = %d, attendu %d", c.regions, e.RegionIndexBits, c.ecrit)
		}
		if got := e.EffectiveRegionIndexBits(); got != c.effective || got != decfilm.LargeurIndexDePlage(c.regions) {
			t.Errorf("%d région(s) : largeur effective %d, attendu %d (loi : %d)",
				c.regions, got, c.effective, decfilm.LargeurIndexDePlage(c.regions))
		}
		if len(e.Regions) != c.regions {
			t.Errorf("%d région(s) : %d lignes rangées", c.regions, len(e.Regions))
		}
	}
}
