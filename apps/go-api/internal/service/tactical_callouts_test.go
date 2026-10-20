package service

// tactical_callouts_test.go — la FORME et la TRANCHE des zones nommées voyagent jusqu'à la règle du
// nom en jeu d'une cellule, et les deux témoins du plan Tactique v2 (D26) tiennent sur le catalogue
// RÉEL versionné.

import (
	"math"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/testutil"
)

func TestZonesNommees_FormeEtTranche(t *testing.T) {
	poly := [][2]float64{{0, 0}, {4, 0}, {4, 4}}
	part := [][][2]float64{{{10, 10}, {12, 10}, {12, 12}}}
	trou := [][][2]float64{{{1, 1}, {2, 1}, {2, 2}}}
	out := zonesNommees([]replay.CalloutZone{{
		VolumeIndex: 7, FR: "Salle", EN: "Room", X: 1, Y: 2,
		ZBottom: -1.5, ZTop: 3.25, Polygon: poly, Parts: part, Holes: trou,
	}})
	if len(out) != 1 {
		t.Fatalf("zones = %+v, attendu 1", out)
	}
	z := out[0]
	if len(z.Polygone) != 3 || len(z.Parties) != 1 || len(z.Trous) != 1 {
		t.Fatalf("forme non projetée : %+v", z)
	}
	if z.ZBas != -1.5 || z.ZHaut != 3.25 || z.VolumeIndex != 7 {
		t.Fatalf("tranche / index non projetés : %+v", z)
	}
	pures := zonesPures(out)
	if len(pures[0].Polygone) != 3 || len(pures[0].Parties) != 1 || len(pures[0].Trous) != 1 ||
		pures[0].ZBas != -1.5 || pures[0].ZHaut != 3.25 || pures[0].VolumeIndex != 7 {
		t.Fatalf("jumeau pur incomplet : %+v", pures[0])
	}
}

// zonesDuCatalogue rend les zones d'une carte intégrée du catalogue réel, projetées comme le
// magasin de l'onglet les projette.
func zonesDuCatalogue(t *testing.T, module string) []tactical.ZoneNommee {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	cat, err := replay.LoadMapCallouts(filepath.Join(root, "data", "titles", "halo_infinite", "reference", "map_callouts.json"))
	if err != nil {
		t.Fatal(err)
	}
	entry, err := cat.Lookup(module)
	if err != nil {
		t.Fatalf("%s : %v", module, err)
	}
	return zonesPures(zonesNommees(entry.Zones))
}

// TestNommerZone_TemoinsDuCatalogueReel — les deux témoins du plan (D26). Les hauteurs sont celles
// des morts de la cellule la plus chaude (« Morts », « Moi ») relevées dans les données de la
// maquette (`.ai/V7.5/MAQUETTE_TACTIQUE_2026-10-06.html`, `VM.witness`, face victime), arrondies
// au centimètre par la maquette.
func TestNommerZone_TemoinsDuCatalogueReel(t *testing.T) {
	// Illusion, cellule (−7, 2) à 2 m : centre (−13, 5), 11 morts, z médian 2,90 m. Aucun
	// polygone ne contient le centre : « Nid blindé », le plus proche, à 0,81 m (règle (c)).
	illusion := []float64{2.9, 2.9, 2.89, 2.9, 2.9, 3.67, 2.9, 2.9, 2.9, 2.9, 2.9}
	got, ok := tactical.NommerZone(-13, 5, illusion, zonesDuCatalogue(t, "ctf_illusion"))
	if !ok || got.Zone.NomFR != "Nid blindé" || got.Regle != tactical.RegleZoneProche ||
		math.Abs(got.DistanceM-0.81) > 0.01 {
		t.Fatalf("Illusion : %q par %q à %.3f m, attendu « Nid blindé » par (c) à 0,81 m",
			got.Zone.NomFR, got.Regle, got.DistanceM)
	}

	// Bazaar, cellule (−4, −1) à 2 m : centre (−7, −1), 9 morts, z médian 3,18 m. Trois zones
	// empilées contiennent le centre : la règle (b) tranche — la tranche la plus étroite parmi
	// celles qui contiennent la majorité des morts, « Pont du marché ouest » (nom fixé au lot L2 du
	// plan ; la maquette, sur une autre règle — la zone la plus fréquente, sans marge —, disait
	// « Grande cour ouest »).
	bazaar := []float64{3.18, 0.38, 3.18, 0.36, 3.18, 3.21, 3.22, 3.21, 3.24}
	zones := zonesDuCatalogue(t, "ctf_bazaar")
	empilees := 0
	for _, z := range zones {
		// Une zone seule, sans hauteur, est nommée par (a) si et seulement si sa forme contient le centre.
		if r, ok := tactical.NommerZone(-7, -1, nil, []tactical.ZoneNommee{z}); ok && r.Regle == tactical.RegleZonePolygone {
			empilees++
		}
	}
	if empilees != 3 {
		t.Errorf("Bazaar : %d zones contiennent le centre, attendu 3", empilees)
	}
	got, ok = tactical.NommerZone(-7, -1, bazaar, zones)
	if !ok || got.Regle != tactical.RegleZoneEmpilee || got.Zone.NomFR != "Pont du marché ouest" ||
		got.Zone.NomEN != "West Market Bridge" {
		t.Fatalf("Bazaar : %q / %q par %q, attendu « Pont du marché ouest » / « West Market Bridge » par (b)",
			got.Zone.NomFR, got.Zone.NomEN, got.Regle)
	}
}
