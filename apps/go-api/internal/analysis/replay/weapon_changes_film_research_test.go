package replay

// weapon_changes_film_research_test.go — le calque des prises et des lachers, VU DE BOUT EN
// BOUT sur un vrai film. Le golden d'assemblage ne peut pas le faire : son fixture d'entrees a
// ete fige avant ce calque et ne porte aucun changement d'arme.
//
// GARDE : PICKUP_FILM (repertoire du film) et PICKUP_MAP (nom de carte du catalogue de bornes).

import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

func TestWeaponChangesSurFilmReel(t *testing.T) {
	dir, mapName := os.Getenv("PICKUP_FILM"), os.Getenv("PICKUP_MAP")
	if dir == "" || mapName == "" {
		t.Skip("PICKUP_FILM / PICKUP_MAP absents : instrument de mesure saute")
	}
	path := filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Fatalf("carte %q : %v", mapName, err)
	}
	doc, err := BuildFromFilm("mesure-ramassage", "halo_infinite", dir, Options{MapQuant: &entry})
	if err != nil {
		t.Fatalf("assemblage : %v", err)
	}
	if doc.Coverage.WeaponChanges == nil {
		t.Fatal("coverage.weaponChanges absente : elle doit etre publiee MEME vide, sinon un " +
			"film sans ramassage et un film qu'on n'a pas su balayer se confondent")
	}
	cov := *doc.Coverage.WeaponChanges
	t.Logf("COUVERTURE : decodes=%d publies=%d prises=%d lachers=%d echanges=%d reannonces=%d",
		cov.Decoded, cov.Published, cov.Taken, cov.Dropped, cov.Swapped, cov.Restated)
	if len(doc.WeaponChanges) != cov.Published {
		t.Errorf("document=%d entrees, couverture=%d publies : les deux doivent coincider",
			len(doc.WeaponChanges), cov.Published)
	}
	withUntil, maxT := 0, 0
	for _, w := range doc.WeaponChanges {
		if w.T > maxT {
			maxT = w.T
		}
		if w.Kind == WeaponDropped && w.Until > 0 {
			withUntil++
			if w.Until <= w.T {
				t.Errorf("borne d'affichage %d <= instant du lacher %d", w.Until, w.T)
			}
			if w.Until > doc.FrameCount {
				t.Errorf("borne d'affichage %d au-dela de la derniere frame %d", w.Until, doc.FrameCount)
			}
		}
	}
	t.Logf("FRAMES : derniere frame du document=%d ; plus grand instant publie=%d ; "+
		"lachers avec borne d'affichage=%d", doc.FrameCount, maxT, withUntil)
	for i, w := range doc.WeaponChanges {
		if i >= 8 {
			break
		}
		t.Logf("   t=%-5d vie=%-4d %-8s w=%-8s from=%-8s until=%d",
			w.T, w.Slot, w.Kind, w.W, w.From, w.Until)
	}
}
