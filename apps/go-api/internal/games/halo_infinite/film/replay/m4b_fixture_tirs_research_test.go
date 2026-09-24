//go:build research

package replay

// m4b_fixture_tirs_research_test.go — LOT M4b : les tirs d un fixture d entrees fige, par build
// (M4B_BUILD = identifiant court). Mesure seule, AUCUN octet de film : le fixture est deja decode.

import (
	"fmt"
	"os"
	"sort"
	"testing"
)

func TestM4bFixtureTirs(t *testing.T) {
	id := os.Getenv("M4B_BUILD")
	if id == "" {
		t.Skip("M4B_BUILD absent")
	}
	for _, b := range goldenBuilds() {
		if b.Short8 != id {
			continue
		}
		g, _ := chargerGoldenBuild(t, b)
		par := map[string]int{}
		for _, e := range g.Fire {
			par[fmt.Sprintf("tireur %v index %2d sonde %v court %v bloc %v", e.HasShooter, e.FilmIndex, e.Unit.Probe, e.Short, e.Bloc)]++
		}
		var l []string
		for k, v := range par {
			l = append(l, fmt.Sprintf("%5d %s", v, k))
		}
		sort.Strings(l)
		for _, s := range l {
			t.Log(s)
		}
		if out := os.Getenv("M4B_DUMP"); out != "" {
			var lignes []string
			for _, e := range g.Fire {
				lignes = append(lignes, fmt.Sprintf("%d %d %d %v", e.Chunk, e.PacketIndex, e.FilmIndex, e.HasShooter))
			}
			sort.Strings(lignes)
			f, _ := os.Create(out)
			for _, s := range lignes {
				fmt.Fprintln(f, s)
			}
			_ = f.Close()
		}
	}
}

// TestM4bUniteContrePlace mesure, sur chaque fixture de build, ce que la REFERENCE 0 (le slot de
// l unite tireuse) rattacherait a pied contre le rattachement par l index de tireur.
func TestM4bUniteContrePlace(t *testing.T) {
	for _, b := range goldenBuilds() {
		g, _ := chargerGoldenBuild(t, b)
		tracks := indexBySlot(g.Positions)
		owner := map[uint32]int{}
		for _, p := range g.Positions {
			_ = p
		}
		_ = owner
		parUnite, parUniteBiped, sansUnite := 0, 0, 0
		for _, e := range g.Fire {
			if !e.Unit.Present {
				sansUnite++
				continue
			}
			tr, ok := tracks[e.Unit.Slot]
			if !ok {
				continue
			}
			parUniteBiped++
			if p, d := tr.at(e.TimestampUS); d <= shotPosToleranceUS && p.HasWorld {
				parUnite++
			}
		}
		t.Logf("%s : %d tirs · unite presente %d · unite = slot de bipede %d · posable par l unite %d (%.1f %%)",
			b.Short8, len(g.Fire), len(g.Fire)-sansUnite, parUniteBiped, parUnite,
			100*float64(parUnite)/float64(max(1, len(g.Fire))))
	}
}

// TestM4bAccordIndexUnite mesure, sur chaque fixture de build, l ACCORD entre l index de tireur BRUT
// du record 36 et l index du joueur dont le bipede est l unite tireuse (reference 0), lu dans le
// document assemble (piste du slot -> xuid -> entree du roster).
func TestM4bAccordIndexUnite(t *testing.T) {
	for _, b := range goldenBuilds() {
		g, entry := chargerGoldenBuild(t, b)
		doc := assemblerGoldenBuild(t, b, g, entry)
		idxParXUID := map[string]int{}
		for _, r := range doc.Roster {
			idxParXUID[r.XUID] = r.FilmIndex
		}
		accord, desaccord := 0, 0
		for _, e := range g.Fire {
			if !e.Unit.Present || !e.HasShooter {
				continue
			}
			x := ""
			for _, tr := range doc.Tracks {
				if tr.Slot == e.Unit.Slot && tr.XUID != "" {
					x = tr.XUID
					break
				}
			}
			idx, ok := idxParXUID[x]
			if !ok {
				continue
			}
			if idx == e.FilmIndex {
				accord++
			} else {
				desaccord++
			}
		}
		t.Logf("%s : accord index brut / unite %d, desaccord %d (%.1f %% d accord)", b.Short8, accord, desaccord,
			100*float64(accord)/float64(max(1, accord+desaccord)))
	}
}
