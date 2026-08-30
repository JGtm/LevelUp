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
	maxT := 0
	for _, w := range doc.WeaponChanges {
		if w.T > maxT {
			maxT = w.T
		}
	}
	t.Logf("FRAMES : derniere frame du document=%d ; plus grand instant publie=%d "+
		"(l'affichage au sol vit dans groundWeapons depuis le schema 26)", doc.FrameCount, maxT)
	for i, w := range doc.WeaponChanges {
		if i >= 8 {
			break
		}
		t.Logf("   t=%-5d vie=%-4d %-8s w=%-8s from=%-8s", w.T, w.Slot, w.Kind, w.W, w.From)
	}
}

// TestGroundWeaponsSurFilmReel — le calque des armes au sol observees (schema 26), de bout en
// bout. Meme garde que le test au-dessus : les deux se mesurent sur le meme assemblage.
func TestGroundWeaponsSurFilmReel(t *testing.T) {
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
	doc, err := BuildFromFilm("mesure-armes-au-sol", "halo_infinite", dir, Options{MapQuant: &entry})
	if err != nil {
		t.Fatalf("assemblage : %v", err)
	}
	if doc.Coverage.GroundWeaponItems == nil {
		t.Fatal("coverage.groundWeaponItems absente : elle doit etre publiee MEME vide")
	}
	cov := *doc.Coverage.GroundWeaponItems
	t.Logf("COUVERTURE : objets=%d publiees=%d auRepos=%d", cov.Objects, cov.Published, cov.AtRest)
	t.Logf("LIENS : lacheurNomme=%d ; prisesRecues=%d dont ramasseurNomme=%d",
		cov.DropperNamed, cov.TakesTotal, cov.PickupLinked)
	t.Logf("FINS : pickup=%d vue=%d ouverte=%d (somme=%d, publiees=%d)",
		cov.EndPickup, cov.EndSeen, cov.EndOpen, cov.EndPickup+cov.EndSeen+cov.EndOpen, cov.Published)
	if cov.EndPickup+cov.EndSeen+cov.EndOpen != cov.Published {
		t.Errorf("les fins ne somment pas aux publiees")
	}
	if len(doc.GroundWeapons) != cov.Published {
		t.Errorf("document=%d, couverture=%d", len(doc.GroundWeapons), cov.Published)
	}
	for _, g := range doc.GroundWeapons {
		if g.T1 < g.T0 || g.T1 >= doc.FrameCount+1 {
			t.Errorf("bornes hors axe : [%d, %d] pour %d frames", g.T0, g.T1, doc.FrameCount)
		}
		if g.End == GroundWeaponEndPickup && g.Picker < 0 {
			t.Errorf("fin pickup sans ramasseur")
		}
	}
	for i, g := range doc.GroundWeapons {
		if i >= 10 {
			break
		}
		t.Logf("   [%4d..%4d] %-7s w=%s origin=%-7s dropper=%-4d end=%-6s picker=%d",
			g.T0, g.T1, "", g.W, g.Origin, g.Dropper, g.End, g.Picker)
	}
}
