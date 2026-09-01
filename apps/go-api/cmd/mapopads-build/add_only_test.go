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
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

func aoSocle(x float64, typeID, family string) replay.MapWeaponPadSpot {
	return replay.MapWeaponPadSpot{
		Pos: mapvar.Vec3{X: x, Y: 0, Z: 0}, TypeID: typeID, Family: family, Objects: 1,
	}
}

// TestMemesSoclesDetecteToutEcart — le verrou lui-meme.
func TestMemesSoclesDetecteToutEcart(t *testing.T) {
	ref := []replay.MapWeaponPadSpot{
		aoSocle(1, "0x5F379533", "power"),
		aoSocle(2, "0x6253CFC0", "rack"),
	}
	cas := []struct {
		nom   string
		autre []replay.MapWeaponPadSpot
		egaux bool
	}{
		{"identiques", []replay.MapWeaponPadSpot{
			aoSocle(1, "0x5F379533", "power"), aoSocle(2, "0x6253CFC0", "rack")}, true},
		{"un socle en moins", []replay.MapWeaponPadSpot{
			aoSocle(1, "0x5F379533", "power")}, false},
		{"un socle en plus", []replay.MapWeaponPadSpot{
			aoSocle(1, "0x5F379533", "power"), aoSocle(2, "0x6253CFC0", "rack"),
			aoSocle(3, "0x5E86D110", "powerup")}, false},
		// LE CAS QUI COMPTE : meme nombre, meme ordre, une position qui a bouge de 5 cm.
		// C'est exactement la forme que prend une derive de source, et c'est celle qu'un
		// simple `len()` laisserait passer.
		{"une position deplacee de 5 cm", []replay.MapWeaponPadSpot{
			aoSocle(1.05, "0x5F379533", "power"), aoSocle(2, "0x6253CFC0", "rack")}, false},
		{"une famille changee", []replay.MapWeaponPadSpot{
			aoSocle(1, "0x5F379533", "rack"), aoSocle(2, "0x6253CFC0", "rack")}, false},
		{"l ordre inverse", []replay.MapWeaponPadSpot{
			aoSocle(2, "0x6253CFC0", "rack"), aoSocle(1, "0x5F379533", "power")}, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := memesSocles(ref, c.autre); got != c.egaux {
				t.Errorf("memesSocles = %v, attendu %v — un ecart non detecte fait ecrire des "+
					"points d'apparition sur une carte dont les socles decrivent une autre "+
					"version du fichier", got, c.egaux)
			}
		})
	}
}

// TestMemesSoclesAccepteDeuxListesVides : une carte sans socle n'est pas un ecart.
func TestMemesSoclesAccepteDeuxListesVides(t *testing.T) {
	if !memesSocles(nil, nil) {
		t.Error("deux absences de socle sont egales — une carte sans socle doit pouvoir " +
			"recevoir des points d'apparition")
	}
	if memesSocles(nil, []replay.MapWeaponPadSpot{aoSocle(1, "0x5F379533", "power")}) {
		t.Error("une liste vide et une liste pleine ne sont pas egales")
	}
}
