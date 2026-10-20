package mapcatalog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// entreeZonesTest : une entrée Forge minimale, une zone nommée.
func entreeZonesTest(en string) replay.MapCalloutsEntry {
	return replay.MapCalloutsEntry{
		Provenance: replay.CalloutsProvenanceMvar,
		Zones: []replay.CalloutZone{{
			VolumeIndex: 7, EN: en, FR: en, X: 1, Y: 2, Z: 3, ZBottom: 0, ZTop: 6,
			Polygon: [][2]float64{{0, 0}, {4, 0}, {4, 4}},
		}},
	}
}

// TestAddCalloutsOverlayEntryCreeLeCatalogueGenere — le premier rattrapage d'une instance neuve
// CRÉE le catalogue généré (dossier `generated/` compris), à la forme du catalogue versionné.
func TestAddCalloutsOverlayEntryCreeLeCatalogueGenere(t *testing.T) {
	overlay := filepath.Join(t.TempDir(), "reference", "generated", "map_callouts.json")
	if err := AddCalloutsOverlayEntry(overlay, "halo_infinite", "aaa", entreeZonesTest("Cave")); err != nil {
		t.Fatalf("premier ajout : %v", err)
	}
	cat, err := replay.LoadMapCallouts(overlay)
	if err != nil {
		t.Fatalf("relecture par le lecteur du catalogue versionné : %v", err)
	}
	e, err := cat.LookupByID("aaa")
	if err != nil || e.Zones[0].EN != "Cave" || cat.TitleSlug != "halo_infinite" {
		t.Fatalf("catalogue généré relu : %+v / %v", cat, err)
	}
	if len(cat.Maps) != 0 {
		t.Errorf("le catalogue généré ne porte que des cartes Forge : %d modules", len(cat.Maps))
	}
}

// TestAddCalloutsOverlayEntryEstEnAjoutSeul — une carte déjà rattrapée n'est jamais réécrite,
// et les autres restent intactes.
func TestAddCalloutsOverlayEntryEstEnAjoutSeul(t *testing.T) {
	overlay := filepath.Join(t.TempDir(), "map_callouts.json")
	for _, id := range []string{"aaa", "bbb"} {
		if err := AddCalloutsOverlayEntry(overlay, "halo_infinite", id, entreeZonesTest("Cave "+id)); err != nil {
			t.Fatalf("ajout %s : %v", id, err)
		}
	}
	err := AddCalloutsOverlayEntry(overlay, "halo_infinite", "aaa", entreeZonesTest("AUTRE"))
	if !errors.Is(err, ErrEntryExists) {
		t.Fatalf("second ajout de la même carte : err = %v, attendu ErrEntryExists", err)
	}
	cat, err := ChargerCatalogueGenere(overlay, "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.MapsByID) != 2 || cat.MapsByID["aaa"].Zones[0].EN != "Cave aaa" {
		t.Fatalf("catalogue généré après refus : %+v", cat.MapsByID)
	}
}

// TestAddCalloutsOverlayEntryRefuseUnCatalogueIllisible — l'écraser effacerait les cartes déjà
// rattrapées : l'ajout échoue, le fichier ne bouge pas.
func TestAddCalloutsOverlayEntryRefuseUnCatalogueIllisible(t *testing.T) {
	overlay := filepath.Join(t.TempDir(), "map_callouts.json")
	if err := os.WriteFile(overlay, []byte("{pas du json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AddCalloutsOverlayEntry(overlay, "halo_infinite", "aaa", entreeZonesTest("Cave")); err == nil {
		t.Fatal("catalogue généré illisible : erreur attendue")
	}
	blob, _ := os.ReadFile(overlay)
	if string(blob) != "{pas du json" {
		t.Errorf("catalogue illisible réécrit : %q", blob)
	}
	if err := AddCalloutsOverlayEntry(overlay, "halo_infinite", "", entreeZonesTest("Cave")); err == nil {
		t.Error("map_id vide : erreur attendue")
	}
}

// TestChargerCatalogueGenereAbsentEstVide — l'absence est l'état d'une instance neuve, pas une
// erreur.
func TestChargerCatalogueGenereAbsentEstVide(t *testing.T) {
	cat, err := ChargerCatalogueGenere(filepath.Join(t.TempDir(), "absent.json"), "halo_infinite")
	if err != nil || cat == nil || len(cat.MapsByID) != 0 {
		t.Fatalf("catalogue absent : %+v / %v, attendu vide sans erreur", cat, err)
	}
	if _, err := cat.LookupByID("aaa"); !errors.Is(err, replay.ErrCalloutsUnknownMap) {
		t.Errorf("lookup dans un catalogue vide : %v", err)
	}
}
