package replaybuild

// spawnpoints_test.go — LA TRADUCTION CATALOGUE -> ETAT, cablee.
//
// Ce chemin n'avait AUCUN test : replier « cle absente » sur « points etablis » ne faisait rien
// tomber, alors que c'est exactement le defaut que le lot corrige — les neuf cartes sautees se
// lisaient « carte connue, aucun point » au lieu de « points non etablis ».

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

// spEcrireCatalogue depose un catalogue minimal dans l'arborescence attendue et rend sa racine.
func spEcrireCatalogue(t *testing.T, cat *replay.MapWeaponPadsCatalog) string {
	t.Helper()
	racine := t.TempDir()
	dir := filepath.Join(racine, "data", "titles", "halo_infinite", "reference")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	blob, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "map_weapon_pads.json"), blob, 0o600); err != nil {
		t.Fatal(err)
	}
	return racine
}

// TestSpawnPointsRendLesTroisEtats — le coeur du correctif.
func TestSpawnPointsRendLesTroisEtats(t *testing.T) {
	avecPoints := []replay.MapSpawnPointSpot{
		{Pos: mapvar.Vec3{X: 1, Y: 2, Z: 3}, TypeID: "0xADEEE6D8", Kind: "grenade", Objects: 1},
	}
	sansPoint := []replay.MapSpawnPointSpot{}
	cat := &replay.MapWeaponPadsCatalog{
		SchemaVersion: replay.MapWeaponPadsSchemaVersion,
		TitleSlug:     "halo_infinite",
		Maps: map[string]replay.MapWeaponPadsEntry{
			// Cle PRESENTE et pleine : points etablis.
			"pleine": {MapID: "pleine", SpawnPoints: &avecPoints},
			// Cle PRESENTE et vide : points etablis, la carte n'en porte aucun.
			"vide": {MapID: "vide", SpawnPoints: &sansPoint},
			// Cle ABSENTE : points NON ETABLIS (carte sautee pour derive de source).
			"sautee": {MapID: "sautee"},
		},
	}
	b := &Builder{repoRoot: spEcrireCatalogue(t, cat), titleSlug: "halo_infinite"}
	// Les attendus sont des LITTERAUX : compares aux constantes, ces cas seraient tautologiques.
	cas := []struct {
		mapID   string
		attendu string
		points  int
	}{
		{"pleine", "established", 1},
		{"vide", "established", 0},
		{"sautee", "not_established", 0},
		{"inconnue-du-catalogue", "map_absent", 0},
		{"", "map_absent", 0},
	}
	for _, c := range cas {
		t.Run(c.mapID+"->"+c.attendu, func(t *testing.T) {
			pts, etat := b.spawnPoints("match", c.mapID, nil)
			if etat != c.attendu {
				t.Errorf("etat = %q, attendu %q", etat, c.attendu)
			}
			if len(pts) != c.points {
				t.Errorf("%d point(s), attendu %d", len(pts), c.points)
			}
		})
	}
}

// TestSpawnPointsReplieSurLeNomPublic — le chemin de la CLI, qui n'a pas de map_id.
func TestSpawnPointsReplieSurLeNomPublic(t *testing.T) {
	pts := []replay.MapSpawnPointSpot{
		{Pos: mapvar.Vec3{X: 1}, TypeID: "0xE42158DF", Kind: "equipment", Objects: 1},
	}
	cat := &replay.MapWeaponPadsCatalog{
		SchemaVersion: replay.MapWeaponPadsSchemaVersion, TitleSlug: "halo_infinite",
		Maps: map[string]replay.MapWeaponPadsEntry{
			"uuid-catalyst": {MapID: "uuid-catalyst", PublicName: "Catalyst", SpawnPoints: &pts},
		},
	}
	b := &Builder{repoRoot: spEcrireCatalogue(t, cat), titleSlug: "halo_infinite"}
	got, etat := b.spawnPoints("match", "", []string{"Catalyst"})
	if etat != "established" || len(got) != 1 {
		t.Fatalf("repli par nom public : etat %q, %d point(s) — attendu established et 1",
			etat, len(got))
	}
	// Un nom qui ne correspond a rien reste une carte ABSENTE.
	if _, etat := b.spawnPoints("match", "", []string{"Carte Inexistante"}); etat != "map_absent" {
		t.Errorf("nom inconnu : etat %q, attendu map_absent", etat)
	}
}
