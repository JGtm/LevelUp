package service

// replay_vehicle_weapons_test.go — le registre des armes de vehicule, pose a la requete (schema 69,
// lot M4a des retours du rejeu 2026-09-23).

import (
	"context"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// registreArmesMinimal : deux armes et un inconnu, au format du titre.
const registreArmesMinimal = `[meta]
title_slug = "halo_infinite"
schema_version = 1

[[weapons]]
tag = "C7D50912"
vehicle = "rockethog"
en = "Rocket Launcher"
fr = "Lance-roquettes"
fire = "single"
fx = "explosive"
tint = "blast"
sound = "vehicle_shot_warthog_rocket_1"
mount = { aim = "turret", ax = 0.0, ay = 0.26 }
proof = "p"

[[weapons]]
tag = "0BB6976B"
vehicle = "falcon"
en = "Grenade Launcher"
fr = "Lance-grenades"
fire = "single"
fx = "explosive"
tint = "kinetic"
silence = "aucune reconstruction"
proof = "p"

[[unknown]]
tag = "850902EF"
reason = "non identifie"
`

// artefactTirsVehicule pose un artefact dont les tirs emploient une arme du registre, une arme
// du registre SANS son, un tag inconnu et une arme de joueur.
func artefactTirsVehicule(t *testing.T, root string) {
	t.Helper()
	doc := `{"schemaVersion":69,"matchId":"m","titleSlug":"` + title.DefaultSlug + `",` +
		`"frameCount":1,"bounds":{"minX":0,"minY":0,"maxX":1,"maxY":1},"tracks":[],"shots":[` +
		`{"t":1,"slot":10,"x":0,"y":0,"w":"0xC7D5091200000000","v":700},` +
		`{"t":2,"slot":10,"x":0,"y":0,"w":"0xC7D5091200000000","v":700},` +
		`{"t":3,"slot":11,"x":0,"y":0,"w":"0x0BB6976B00000000","v":701},` +
		`{"t":4,"slot":12,"x":0,"y":0,"w":"0x850902EF00000000","v":702},` +
		`{"t":5,"slot":13,"x":0,"y":0,"w":"0x84BD29ED42C9679F"}]}`
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(title.DefaultSlug, "m"), doc)
}

// TestVehicleWeapons_PoseLesArmesEmployees — une entree par arme DU REGISTRE qu un tir emploie,
// keyee par la cle meme du tir ; ni l inconnu, ni l arme de joueur n y entrent.
func TestVehicleWeapons_PoseLesArmesEmployees(t *testing.T) {
	root := t.TempDir()
	artefactTirsVehicule(t, root)
	ecrire(t, filepath.Join(title.NewPathResolver(root).TitleMappingsDir(title.DefaultSlug),
		vehicleWeaponsFile), registreArmesMinimal)

	doc, err := NewReplayService(title.DefaultSlug, root, nil).GetReplay(context.Background(), "m")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if len(doc.VehicleWeapons) != 2 {
		t.Fatalf("%d armes servies, attendu 2 : %+v", len(doc.VehicleWeapons), doc.VehicleWeapons)
	}
	r := doc.VehicleWeapons["0xC7D5091200000000"]
	if r.Vehicle != "rockethog" || r.Fx != "explosive" || r.Tint != "blast" ||
		r.Sound != "vehicle_shot_warthog_rocket_1" || r.Mount == nil || r.Mount.Aim != "turret" ||
		r.Mount.AY != 0.26 || r.En != "Rocket Launcher" || r.Fr != "Lance-roquettes" {
		t.Errorf("roquettes = %+v", r)
	}
	if g := doc.VehicleWeapons["0x0BB6976B00000000"]; g.Sound != "" || g.Fx != "explosive" {
		t.Errorf("lance-grenades = %+v, attendu un style et un SILENCE decide", g)
	}
}

// TestVehicleWeapons_TitreSansRegistre — un titre qui n en declare pas sert le rejeu entier, sans
// table : degradation, jamais une erreur.
func TestVehicleWeapons_TitreSansRegistre(t *testing.T) {
	root := t.TempDir()
	artefactTirsVehicule(t, root)
	doc, err := NewReplayService(title.DefaultSlug, root, nil).GetReplay(context.Background(), "m")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if doc.VehicleWeapons != nil || len(doc.Shots) != 5 {
		t.Errorf("table = %v, tirs = %d : attendu aucune table et 5 tirs", doc.VehicleWeapons, len(doc.Shots))
	}
}

// TestVehicleWeaponTagOf — la cle est recomposee par l ecrivain unique du gabarit.
func TestVehicleWeaponTagOf(t *testing.T) {
	if tag, ok := vehicleWeaponTagOf(replay.VehicleWeaponKey(0xC7D50912)); !ok || tag != "C7D50912" {
		t.Errorf("tag = %q, %v", tag, ok)
	}
	for _, k := range []string{"0x84BD29ED42C9679F", "", "0xC7D50912", "0xc7d5091200000000"} {
		if _, ok := vehicleWeaponTagOf(k); ok {
			t.Errorf("%q accepte comme arme de vehicule", k)
		}
	}
}
