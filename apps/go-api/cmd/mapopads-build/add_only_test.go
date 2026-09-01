package main

// add_only_test.go — LA PROMESSE CENTRALE DU MODE « AJOUT SEUL », tenue par un test.
//
// `--only-add-spawn-points` existe pour une seule raison : ne JAMAIS reecrire un socle d'arme.
// La garantie est structurelle (le code n'ecrit que `SpawnPoints`), mais la partie qui peut se
// perdre a une refactorisation est le VERROU : sauter une carte dont les socles recalcules ne
// retombent pas a l'identique. Sans ce verrou, une carte dont le `.mvar` a derive en amont
// recevrait des points d'apparition decrivant une AUTRE version de la carte que ses socles.
//
// Neuf des 72 cartes sont dans ce cas au 2026-09-01. Ce n'est donc pas un cas theorique.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

// aoSocle fabrique un socle de fixture.
func aoSocle(x float64, typeID, family string) replay.MapWeaponPadSpot {
	return replay.MapWeaponPadSpot{
		Pos: mapvar.Vec3{X: x, Y: 0, Z: 0}, TypeID: typeID, Family: family, Objects: 1,
	}
}

// ---------------------------------------------------------------------------------------------
// LES TESTS DE CABLAGE. Les cas ci-dessus valident `memesSocles` ; supprimer son APPEL dans
// `addSpawnPointsOnly` ne faisait rien tomber. Ceux-ci partent du catalogue et du depot.
// ---------------------------------------------------------------------------------------------

func aoCatalogue(t *testing.T, dir string, cat *replay.MapWeaponPadsCatalog) string {
	t.Helper()
	chemin := filepath.Join(dir, "map_weapon_pads.json")
	blob, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		t.Fatalf("serialisation du catalogue : %v", err)
	}
	if err := os.WriteFile(chemin, blob, 0o600); err != nil {
		t.Fatalf("ecriture du catalogue : %v", err)
	}
	return chemin
}

// aoRelire relit le catalogue ecrit par le mode ajout-seul.
func aoRelire(t *testing.T, chemin string) *replay.MapWeaponPadsCatalog {
	t.Helper()
	cat, err := replay.LoadMapWeaponPads(chemin)
	if err != nil {
		t.Fatalf("relecture du catalogue : %v", err)
	}
	return cat
}

// aoAvecIngestFactice remplace l'ingestion par une fonction controlee, le temps d'un test.
func aoAvecIngestFactice(t *testing.T, f func(string, replay.MapObjectivesEntry, string, string,
) (replay.MapWeaponPadsEntry, int, error),
) {
	t.Helper()
	ancien := ingestFn
	ingestFn = f
	t.Cleanup(func() { ingestFn = ancien })
}

// aoDepotAvec fabrique un depot contenant les fichiers nommes, et son index.
func aoDepotAvec(t *testing.T, objectifs *replay.MapObjectivesCatalog, noms ...string) *dumpIndex {
	t.Helper()
	depot := t.TempDir()
	for _, n := range noms {
		if err := os.WriteFile(filepath.Join(depot, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	d, err := newDumpIndex(depot, objectifs)
	if err != nil {
		t.Fatalf("index du depot : %v", err)
	}
	return d
}

// TestAddOnlyRetireLesPointsPerimesQuandLaSourceDerive — LE CORRECTIF P1-C, cable.
//
// Une carte acceptee hier, derivee aujourd'hui, gardait ses `spawn_points` de la passe
// precedente : le catalogue publiait des points issus d'une source que le generateur venait de
// declarer NON CONCORDANTE, et la note comptait cette carte parmi les « sans points » alors
// qu'elle en portait. Le mensonge etait double.
//
// LES QUATRE CAS EXERCENT LES TROIS TERMES DU VERROU. Supprimer n'importe lequel des trois
// (objects_n, level_id, socles) fait tomber le sous-test correspondant : jusqu'ici, supprimer
// l'appel a `memesSocles` ne faisait rien tomber du tout.
func TestAddOnlyRetireLesPointsPerimesQuandLaSourceDerive(t *testing.T) {
	anciens := []replay.MapSpawnPointSpot{
		{Pos: mapvar.Vec3{X: 1}, TypeID: "0xADEEE6D8", Kind: "grenade", Objects: 1},
	}
	entree := func() replay.MapWeaponPadsEntry {
		cp := append([]replay.MapSpawnPointSpot(nil), anciens...)
		return replay.MapWeaponPadsEntry{
			MapID: "m", MvarFile: "m.mvar", ObjectsN: 462, LevelID: 7,
			Pads:        []replay.MapWeaponPadSpot{aoSocle(1, "0x5F379533", "power")},
			SpawnPoints: &cp,
		}
	}
	objectifs := &replay.MapObjectivesCatalog{
		Maps: map[string]replay.MapObjectivesEntry{"m": {MapID: "m", MvarFile: "m.mvar"}},
	}
	cas := []struct {
		nom    string
		neuf   replay.MapWeaponPadsEntry
		efface bool
	}{
		{"objects_n a derive (le signal qui a detecte Deadlock)", replay.MapWeaponPadsEntry{
			ObjectsN: 410, LevelID: 7,
			Pads: []replay.MapWeaponPadSpot{aoSocle(1, "0x5F379533", "power")}}, true},
		{"level_id a derive", replay.MapWeaponPadsEntry{
			ObjectsN: 462, LevelID: 9,
			Pads: []replay.MapWeaponPadSpot{aoSocle(1, "0x5F379533", "power")}}, true},
		{"un socle a bouge de 5 cm", replay.MapWeaponPadsEntry{
			ObjectsN: 462, LevelID: 7,
			Pads: []replay.MapWeaponPadSpot{aoSocle(1.05, "0x5F379533", "power")}}, true},
		{"tout concorde — les points sont remplaces, pas effaces", replay.MapWeaponPadsEntry{
			ObjectsN: 462, LevelID: 7,
			Pads: []replay.MapWeaponPadSpot{aoSocle(1, "0x5F379533", "power")}}, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			neuf := c.neuf
			vide := []replay.MapSpawnPointSpot{}
			neuf.SpawnPoints = &vide
			aoAvecIngestFactice(t, func(string, replay.MapObjectivesEntry, string, string,
			) (replay.MapWeaponPadsEntry, int, error) {
				return neuf, 0, nil
			})
			cat := &replay.MapWeaponPadsCatalog{
				SchemaVersion: replay.MapWeaponPadsSchemaVersion, TitleSlug: "halo_infinite",
				Maps: map[string]replay.MapWeaponPadsEntry{"m": entree()},
			}
			chemin := aoCatalogue(t, t.TempDir(), cat)
			addSpawnPointsOnly(context.Background(), objectifs,
				aoDepotAvec(t, objectifs, "m.mvar"), chemin, false)
			e := aoRelire(t, chemin).Maps["m"]
			if c.efface {
				if e.SpawnPoints != nil {
					t.Errorf("source derivee : la cle `spawn_points` doit DISPARAITRE, "+
						"%d point(s) subsistent — le catalogue publierait des points d'une "+
						"source declaree non concordante", len(*e.SpawnPoints))
				}
			} else if e.SpawnPoints == nil {
				t.Error("source concordante : la cle doit rester PRESENTE (meme vide), sinon " +
					"la carte se lit « points non etablis »")
			}
			// DANS TOUS LES CAS, les socles ne bougent pas : la promesse du mode.
			if len(e.Pads) != 1 || e.Pads[0].Family != "power" || e.Pads[0].Pos.X != 1 {
				t.Errorf("les socles ont bouge : %+v", e.Pads)
			}
		})
	}
}

// TestAddOnlySansDumpNEffacePasLesPointsEtablis — le contre-cas, explicite.
func TestAddOnlySansDumpNEffacePasLesPointsEtablis(t *testing.T) {
	anciens := []replay.MapSpawnPointSpot{
		{Pos: mapvar.Vec3{X: 1}, TypeID: "0xADEEE6D8", Kind: "grenade", Objects: 1},
	}
	objectifs := &replay.MapObjectivesCatalog{
		Maps: map[string]replay.MapObjectivesEntry{"m": {MapID: "m", MvarFile: "m.mvar"}},
	}
	cat := &replay.MapWeaponPadsCatalog{
		SchemaVersion: replay.MapWeaponPadsSchemaVersion, TitleSlug: "halo_infinite",
		Maps: map[string]replay.MapWeaponPadsEntry{
			"m": {MapID: "m", MvarFile: "m.mvar", SpawnPoints: &anciens},
		},
	}
	chemin := aoCatalogue(t, t.TempDir(), cat)
	addSpawnPointsOnly(context.Background(), objectifs,
		aoDepotAvec(t, objectifs, "autre.mvar"), chemin, false)
	e := aoRelire(t, chemin).Maps["m"]
	if e.SpawnPoints == nil || len(*e.SpawnPoints) != 1 {
		t.Error("sans dump, les points etablis d'une passe precedente doivent SURVIVRE : " +
			"l'absence de fichier ne contredit rien, et les effacer detruirait des donnees " +
			"valides des qu'on relance sur un depot partiel")
	}
}

// TestResolveRefuseUnNomPartageParPlusieursCartes — le second cablage exige.
//
// CINQUANTE-HUIT cartes declarent `map.mvar`. Un fichier de ce nom depose a plat serait reclame
// par les 58, et chacune recevrait les socles d'une carte etrangere. C'est arrive pendant ce
// chantier : 65 cartes sur 72 sont sorties avec des socles qui n'etaient pas les leurs.
// Reintroduire la regle du nom brut sur un nom ambigu fait tomber ce test.
func TestResolveRefuseUnNomPartageParPlusieursCartes(t *testing.T) {
	depot := t.TempDir()
	for _, n := range []string{"map.mvar", "isolation_map.mvar"} {
		if err := os.WriteFile(filepath.Join(depot, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	objectifs := &replay.MapObjectivesCatalog{
		Maps: map[string]replay.MapObjectivesEntry{
			"a": {MapID: "a", MvarFile: "map.mvar", PublicName: "Isolation"},
			"b": {MapID: "b", MvarFile: "map.mvar", PublicName: "Streets"},
			"c": {MapID: "c", MvarFile: "solo.mvar", PublicName: "Solo"},
		},
	}
	dumps, err := newDumpIndex(depot, objectifs)
	if err != nil {
		t.Fatalf("index du depot : %v", err)
	}
	// La carte `b` n'a AUCUN fichier prefixe au depot : elle ne doit RIEN resoudre, surtout
	// pas le `map.mvar` litteral, qui appartient a n'importe laquelle des deux.
	if _, base, ok := dumps.resolve("b", objectifs.Maps["b"]); ok {
		t.Errorf("`map.mvar` est partage par 2 cartes et a pourtant ete resolu pour `b` (%s) — "+
			"c'est exactement le bogue qui a donne a 65 cartes les socles d'une autre", base)
	}
	// La carte `a` a son fichier PREFIXE : elle doit resoudre, par la regle du prefixe.
	if _, base, ok := dumps.resolve("a", objectifs.Maps["a"]); !ok || base != "isolation_map.mvar" {
		t.Errorf("la carte prefixee doit resoudre sur isolation_map.mvar, obtenu %q (ok=%v)",
			base, ok)
	}
	// Un nom NON ambigu reste resolu par la regle 1 — le durcissement ne doit pas tout casser.
	if err := os.WriteFile(filepath.Join(depot, "solo.mvar"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	dumps2, err := newDumpIndex(depot, objectifs)
	if err != nil {
		t.Fatal(err)
	}
	if _, base, ok := dumps2.resolve("c", objectifs.Maps["c"]); !ok || base != "solo.mvar" {
		t.Errorf("un nom non ambigu doit rester resolu par la regle 1, obtenu %q (ok=%v)",
			base, ok)
	}
}
