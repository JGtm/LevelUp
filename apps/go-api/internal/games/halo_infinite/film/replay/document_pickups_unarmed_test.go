package replay

// document_pickups_unarmed_test.go — LES MAINS NUES (retours du rejeu, lot M6.3, 2026-09-24).
//
// `00007CA9` est l'arme « mains nues » du jeu (sonde CA9 : `WeaponTags.unarmed` du script Lua
// global). Le jeu la REMET à chaque bipède au coup d'envoi et à la réapparition : le film l'écrit
// comme un ramassage natif de classe ARME, mais ce n'est pas une prise. Décision de l'utilisateur
// (2026-09-24) : la remise est classée comme telle, HORS des ramassages publiés, avec un compteur
// dédié, et elle ne compte pas comme une famille inconnue ; l'objet n'entre jamais dans une
// dotation affichée.
//
// LES IDENTIFIANTS SONT DES LITTÉRAUX (discipline de `document_pickups_naming_test.go`) : dériver
// l'attendu de la constante testée rendrait le test tautologique.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

func TestBuildPickupsRemiseMainsNuesHorsDesRamassages(t *testing.T) {
	armes := map[uint32]string{0x48c19d2d: "hinf_ma40_ar"}
	in := []types.BipedPickup{
		{TimestampUS: 900_000, Slot: 512, CatalogID: 0x00007ca9, Class: 0},   // avant l origine
		{TimestampUS: 1_000_000, Slot: 512, CatalogID: 0x00007ca9, Class: 0}, // remise, coup d envoi
		{TimestampUS: 1_000_000, Slot: 513, CatalogID: 0x00007ca9, Class: 1}, // remise, classe 1
		{TimestampUS: 1_100_000, Slot: 512, CatalogID: 0x48c19d2d, Class: 0}, // vraie prise
		{TimestampUS: 9_000_000, Slot: 514, CatalogID: 0x00007ca9, Class: 0}, // remise, réapparition
	}
	got, cov := buildPickups(in, pkClock(nil), pickupInputs{weaponKeys: armes})
	if len(got) != 1 || got[0].W != "48c19d2d" {
		t.Fatalf("publiés = %+v, attendu la seule prise du fusil d assaut", got)
	}
	if cov.UnarmedGrants != 3 {
		t.Errorf("unarmedGrants = %d, attendu 3 (coup d envoi x2 + réapparition)", cov.UnarmedGrants)
	}
	if cov.UnknownFamilies != 0 {
		t.Errorf("unknownFamilies = %d, attendu 0 : une remise nommée n est pas une famille inconnue",
			cov.UnknownFamilies)
	}
	if cov.BeforeOrigin != 1 || cov.Weapons != 1 || cov.Published != 1 {
		t.Errorf("couverture = %+v, attendu beforeOrigin 1, weapons 1, published 1", cov)
	}
	// PARTITION : tout ramassage décodé est publié, antérieur à l origine, ou une remise.
	if cov.Decoded != cov.Published+cov.BeforeOrigin+cov.UnarmedGrants {
		t.Errorf("decoded %d != published %d + beforeOrigin %d + unarmedGrants %d",
			cov.Decoded, cov.Published, cov.BeforeOrigin, cov.UnarmedGrants)
	}
}

// TestBuildPickupsMainsNuesNonArmeResteUnRamassage — la règle ne vaut que pour la classe ARME :
// l identifiant d une autre classe n est pas l objet « mains nues » (espaces disjoints, mesuré).
func TestBuildPickupsMainsNuesNonArmeResteUnRamassage(t *testing.T) {
	in := []types.BipedPickup{{TimestampUS: 1_000_000, Slot: 512, CatalogID: 0x00007ca9, Class: 3}}
	got, cov := buildPickups(in, pkClock(nil), pickupInputs{})
	if len(got) != 1 || cov.UnarmedGrants != 0 {
		t.Errorf("publiés = %d, unarmedGrants = %d : attendu 1 et 0", len(got), cov.UnarmedGrants)
	}
}

// TestNomDeDotationExclutLesMainsNuesMemeConnues — LA RÈGLE, ÉPROUVÉE (revue adverse du lot M6,
// constat R6) : le catalogue de décodage (`weaponv3`) ne connaît pas l'objet aujourd'hui, donc un
// test qui ne passe que par lui resterait vert sans la règle. Ici le catalogue le NOMME, et la
// règle doit quand même l'écarter ; une arme ordinaire garde son nom.
func TestNomDeDotationExclutLesMainsNuesMemeConnues(t *testing.T) {
	catalogue := func(fam uint32) string {
		switch fam {
		case 0x00007ca9:
			return "Unarmed"
		case 0x48c19d2d:
			return "MA40 AR"
		}
		return ""
	}
	if got := nomDeDotation(0x00007ca9, catalogue); got != "" {
		t.Errorf("mains nues nommées %q dans une dotation : la règle nommée ne s'applique plus", got)
	}
	if got := nomDeDotation(0x48c19d2d, catalogue); got != "MA40 AR" {
		t.Errorf("arme ordinaire = %q, attendu le nom du catalogue", got)
	}
}

// TestBuildLoadoutsExclutLesMainsNues — la dotation affichée ne porte jamais l objet mains nues,
// par la règle NOMMÉE (`filmshell.IsUnarmedFamily`) ; la règle elle-même est éprouvée par
// `TestNomDeDotationExclutLesMainsNuesMemeConnues`.
func TestBuildLoadoutsExclutLesMainsNues(t *testing.T) {
	raw := []types.KeyframeLoadout{{
		TimestampUS: 1_000_000, Slot: 512, Families: []uint32{0x48c19d2d, 0x00007ca9},
	}}
	got := buildLoadouts(raw, 1_000_000, 100_000)
	if len(got) != 1 || len(got[0].W) != 1 || got[0].W[0] != "0x48C19D2D" {
		t.Errorf("dotation = %+v, attendu le seul fusil d assaut", got)
	}
}

// TestBuildWeaponChangesRemiseMainsNues — LE MEME GESTE SUR LE CANAL `weaponChanges` (revue
// adverse du lot M6, constat R2) : la remise des mains nues y est ecrite comme une PRISE
// (`taken`, emplacement vide -> `00007ca9`) au premier instant d une vie. Ce n est pas une prise :
// elle sort des changements publies (donc de leur son de ramassage et du negatif des paliers de
// socle) et se compte dans `unarmedGrants`. Un ECHANGE vers les mains nues (le joueur a tout
// jete) reste publie : c est le cas « quasi impossible » que la decision du 2026-09-24 fait
// NOMMER, pas taire.
func TestBuildWeaponChangesRemiseMainsNues(t *testing.T) {
	in := []types.HeldWeaponChange{
		{TimestampUS: wcOrigin - 1, Slot: 5, Family: 0x00007ca9, Previous: grammar.NoWeaponVariant,
			Kind: types.HeldWeaponTaken}, // avant l origine
		{TimestampUS: wcOrigin, Slot: 5, Family: 0x00007ca9, Previous: grammar.NoWeaponVariant,
			Kind: types.HeldWeaponTaken}, // remise
		{TimestampUS: wcOrigin + 200_000, Slot: 5, Family: 0x48c19d2d,
			Previous: grammar.NoWeaponVariant, Kind: types.HeldWeaponTaken}, // vraie prise
		{TimestampUS: wcOrigin + 900_000, Slot: 5, Family: 0x00007ca9, Previous: 0x48c19d2d,
			Kind: types.HeldWeaponSwapped}, // passage aux mains nues en cours de vie
	}
	got, cov := buildWeaponChanges(in, wcOrigin, wcStep)
	if len(got) != 2 || got[0].W != "48c19d2d" || got[1].W != "00007ca9" || got[1].Kind != WeaponSwapped {
		t.Fatalf("publiés = %+v, attendu la prise du fusil puis l echange vers les mains nues", got)
	}
	if cov.UnarmedGrants != 1 || cov.BeforeOrigin != 1 || cov.Taken != 1 || cov.Swapped != 1 {
		t.Errorf("couverture = %+v, attendu unarmedGrants 1, beforeOrigin 1, taken 1, swapped 1", cov)
	}
	// PARTITION : tout changement décodé est publié, ré-annoncé, antérieur à l origine, ou une remise.
	if cov.Decoded != cov.Published+cov.Restated+cov.BeforeOrigin+cov.UnarmedGrants {
		t.Errorf("partition rompue : %+v", cov)
	}
}
