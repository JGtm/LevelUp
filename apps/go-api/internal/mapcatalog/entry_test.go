package mapcatalog

// entry_test.go — LE VERROU DE DERIVE, teste la ou il vit desormais.
//
// Ces cas ont suivi `SamePads` lors de son extraction du `package main` de `mapopads-build` :
// une fonction qui change de maison sans ses tests laisse un trou que personne ne voit.

import (
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

func mcSocle(x float64, typeID, family string) replay.MapWeaponPadSpot {
	return replay.MapWeaponPadSpot{
		Pos: mapvar.Vec3{X: x, Y: 0, Z: 0}, TypeID: typeID, Family: family, Objects: 1,
	}
}

// TestMemesSoclesDetecteToutEcart — le verrou lui-meme.
func TestMemesSoclesDetecteToutEcart(t *testing.T) {
	ref := []replay.MapWeaponPadSpot{
		mcSocle(1, "0x5F379533", "power"),
		mcSocle(2, "0x6253CFC0", "rack"),
	}
	cas := []struct {
		nom   string
		autre []replay.MapWeaponPadSpot
		egaux bool
	}{
		{"identiques", []replay.MapWeaponPadSpot{
			mcSocle(1, "0x5F379533", "power"), mcSocle(2, "0x6253CFC0", "rack")}, true},
		{"un socle en moins", []replay.MapWeaponPadSpot{
			mcSocle(1, "0x5F379533", "power")}, false},
		{"un socle en plus", []replay.MapWeaponPadSpot{
			mcSocle(1, "0x5F379533", "power"), mcSocle(2, "0x6253CFC0", "rack"),
			mcSocle(3, "0x5E86D110", "powerup")}, false},
		// LE CAS QUI COMPTE : meme nombre, meme ordre, une position qui a bouge de 5 cm.
		// C'est exactement la forme que prend une derive de source, et c'est celle qu'un
		// simple `len()` laisserait passer.
		{"une position deplacee de 5 cm", []replay.MapWeaponPadSpot{
			mcSocle(1.05, "0x5F379533", "power"), mcSocle(2, "0x6253CFC0", "rack")}, false},
		{"une famille changee", []replay.MapWeaponPadSpot{
			mcSocle(1, "0x5F379533", "rack"), mcSocle(2, "0x6253CFC0", "rack")}, false},
		{"l ordre inverse", []replay.MapWeaponPadSpot{
			mcSocle(2, "0x6253CFC0", "rack"), mcSocle(1, "0x5F379533", "power")}, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := SamePads(ref, c.autre); got != c.egaux {
				t.Errorf("memesSocles = %v, attendu %v — un ecart non detecte fait ecrire des "+
					"points d'apparition sur une carte dont les socles decrivent une autre "+
					"version du fichier", got, c.egaux)
			}
		})
	}
}

// TestMemesSoclesAccepteDeuxListesVides : une carte sans socle n'est pas un ecart.
func TestMemesSoclesAccepteDeuxListesVides(t *testing.T) {
	if !SamePads(nil, nil) {
		t.Error("deux absences de socle sont egales — une carte sans socle doit pouvoir " +
			"recevoir des points d'apparition")
	}
	if SamePads(nil, []replay.MapWeaponPadSpot{mcSocle(1, "0x5F379533", "power")}) {
		t.Error("une liste vide et une liste pleine ne sont pas egales")
	}
}
