package duckdb

// favorite_weapon_rule_test.go — L ARME FAVORITE DU SCOREBOARD NE PEUT PAS ETRE UN OBJET HORS
// ARSENAL (revue adverse du lot M6 des retours du rejeu, constat R1, 2026-09-24).
//
// Depuis ce lot, la bobine a fusion UNSC (`hinf_coil_kinetic`, classe `environmental`) porte les
// identifiants de film de l objet que le joueur tient : `resolveWeaponKeyDimensions` lui rend un
// identifiant numerique NON NUL. Le filtre `WeaponID == 0` ne l ecartait donc plus du classement :
// un joueur qui fait sauter des bobines sur une carte Forge aurait eu « Bobine a fusion » pour arme
// favorite. La regle passe par la CLASSE (`domain.IsFavoriteWeaponCandidate`).

import (
	"testing"

	"levelup/go-api/internal/domain"
)

func TestTopWeaponByXUIDEcarteLaBobineAFusion(t *testing.T) {
	rows := []domain.BulkWeaponKillRaw{
		{XUID: "a", WeaponID: 0x1d63a8cdfab48286, Kills: 5, Class: "environmental", FromDamageSource: true},
		{XUID: "a", WeaponID: 0x48c19d2d42c9679f, Kills: 3, Class: "shoulder", FromDamageSource: true},
		// Halo 5 (lignes de `weapon_kills`, non mesurees dans le film) : la regle d avant, inchangee.
		{XUID: "h5", WeaponID: 3168248199, Kills: 9, Class: "unattributed"},
		{XUID: "h5", WeaponID: 1, Kills: 2, Class: "shoulder"},
	}
	top := topWeaponByXUID(rows)
	if top["a"] != 0x48c19d2d42c9679f {
		t.Errorf("arme favorite de a = %#x, attendu le fusil d assaut (la bobine est un objet hors arsenal)",
			uint64(top["a"]))
	}
	if top["h5"] != 3168248199 {
		t.Errorf("arme favorite Halo 5 = %d : le second titre ne devait pas bouger", top["h5"])
	}
}

func TestFavoriteWeaponFromScopeEcarteLaBobineAFusion(t *testing.T) {
	rows := []weaponScopeRow{
		{weaponID: 0x1d63a8cdfab48286, weaponKey: "hinf_coil_kinetic", class: "environmental", kills: 5},
		{weaponID: 0x48c19d2d42c9679f, weaponKey: "hinf_ma40_ar", class: "shoulder", kills: 3},
	}
	w, ok := premiereArmeFavorite(rows)
	if !ok || w.weaponKey != "hinf_ma40_ar" {
		t.Errorf("arme favorite = %+v (%v), attendu le fusil d assaut", w, ok)
	}
	if got := armesDeLArsenal(rows, 5); len(got) != 1 || got[0].weaponKey != "hinf_ma40_ar" {
		t.Errorf("top armes = %+v, attendu le seul fusil d assaut", got)
	}
}
