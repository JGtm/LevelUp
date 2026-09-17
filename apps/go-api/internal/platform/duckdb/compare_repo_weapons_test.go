//go:build integration

// Package duckdb — compare_repo_weapons_test.go : le scope du profil d'armes du Face-à-face
// (plan .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md, lot 2 item 2.3, amendé au lot 3-bis).
//
// Même régime de tag que les autres tests du paquet (`integration`) : ils montent de vraies
// DB DuckDB `:memory:`.
//
// # CE QUE CES TESTS VERROUILLENT, ET QUI SE TROMPE EN SILENCE SANS EUX
//
//  1. LA CAMPAGNE EST EXCLUE. Elle l'est par un fragment SQL qui est un NO-OP sur un titre
//     sans mode masqué — un test mené sur le titre par défaut serait donc vert quelle que
//     soit l'implémentation. Ces tests passent par `halo_5`, le seul titre qui déclare des
//     game_variant de Campagne, pour que l'exclusion ait quelque chose à exclure.
//  2. ZÉRO MATCH REND (nil, nil), jamais un scope vide — qui passerait ensuite le
//     `Validate()` des lecteurs d'armes et remonterait une erreur de filtres là où la vérité
//     est « ce joueur n'a aucun match ici ».
//
// DEUX TESTS ONT ÉTÉ RETIRÉS AU LOT 3-BIS (2026-09-17) avec la méthode qu'ils couvraient,
// `GetCrossWeaponScope`. Sa requête lisait la même table avec la même exclusion et n'y
// ajoutait qu'un `EXISTS` sur le joueur courant : son résultat était un SOUS-ENSEMBLE de celui
// testé ici, donc la branche de service qui l'appelait ne pouvait jamais être prise. Les tests
// passaient — c'est précisément ce qui rendait le chemin mort invisible.
package duckdb

import (
	"context"
	"testing"
)

// slugCampagne : le titre qui déclare des variants de Campagne masqués à la lecture. Sur le
// titre par défaut la clause d'exclusion est vide, et le témoin ne témoignerait de rien.
const slugCampagne = "halo_5"

// variantCampagne : un game_variant_id de Campagne, repris de la source unique
// `analysis.campaignExcludedVariantIDs`. Écrit ici en clair et non lu depuis la source : ce
// test doit rougir si la source change sans que l'exclusion suive.
const variantCampagne = "00000003-0000-0010-8000-00aa00389b71"

// seedScopeMatch pose un match au registre (avec son variant) et la ligne d'un participant.
type scopeParticipant struct {
	matchID       string
	xuid          string
	variantID     string
	kills, deaths int
	melee, grndes int
}

func seedScopeMatch(t *testing.T, pdb *PlayerDB, ctx context.Context, p scopeParticipant) {
	t.Helper()
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_registry (match_id, game_variant_id) VALUES (?, ?)
		 ON CONFLICT (match_id) DO NOTHING`,
		p.matchID, p.variantID)
	execOnSharedDBs(t, pdb, ctx,
		`INSERT INTO shared.match_participants
			(match_id, xuid, kills, deaths, melee_kills, grenade_kills)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.matchID, p.xuid, p.kills, p.deaths, p.melee, p.grndes)
}

// wipeScopeTables vide les deux tables : le seed du paquet pose déjà un m1 pour pTestXUID, et
// un scope LIFETIME le compterait.
func wipeScopeTables(t *testing.T, pdb *PlayerDB, ctx context.Context) {
	t.Helper()
	execOnSharedDBs(t, pdb, ctx, `DELETE FROM shared.match_participants`)
	execOnSharedDBs(t, pdb, ctx, `DELETE FROM shared.match_registry`)
}

// TestGetWeaponScope_LifetimeCampagneExclue : trois matchs, dont un de Campagne. Le scope en
// retient deux, et ses totaux sont ceux de ces deux-là — la Campagne ne pèse ni dans les
// match_id ni dans les compteurs.
func TestGetWeaponScope_LifetimeCampagneExclue(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	wipeScopeTables(t, pdb, ctx)

	for _, p := range []scopeParticipant{
		{matchID: "m_pvp_1", xuid: pTestXUID, kills: 10, deaths: 4, melee: 2, grndes: 1},
		{matchID: "m_pvp_2", xuid: pTestXUID, kills: 7, deaths: 6, melee: 1, grndes: 3},
		{matchID: "m_campagne", xuid: pTestXUID, variantID: variantCampagne,
			kills: 100, deaths: 0, melee: 50, grndes: 50},
	} {
		seedScopeMatch(t, pdb, ctx, p)
	}

	repo := NewCompareRepo(pdb)
	scope, err := repo.GetWeaponScope(ctx, pTestXUID, slugCampagne)
	if err != nil {
		t.Fatalf("GetWeaponScope: %v", err)
	}
	if scope == nil {
		t.Fatal("scope nil : deux matchs PvP existent")
	}
	if scope.Matches != 2 || len(scope.MatchIDs) != 2 {
		t.Fatalf("matchs = %d (%v), attendu 2 : le match de Campagne doit être exclu",
			scope.Matches, scope.MatchIDs)
	}
	for _, id := range scope.MatchIDs {
		if id == "m_campagne" {
			t.Fatalf("m_campagne présent dans le scope : %v", scope.MatchIDs)
		}
	}
	if scope.Kills != 17 || scope.Deaths != 10 || scope.MeleeKills != 3 || scope.GrenadeKills != 4 {
		t.Fatalf("totaux = k%d d%d m%d g%d, attendu k17 d10 m3 g4 (les 100 frags de Campagne "+
			"ne doivent pas peser)", scope.Kills, scope.Deaths, scope.MeleeKills, scope.GrenadeKills)
	}
}

// TestGetWeaponScope_AucunMatch : un joueur inconnu de la base rend (nil, nil).
func TestGetWeaponScope_AucunMatch(t *testing.T) {
	pdb := newTestPlayerDB(t)
	ctx := context.Background()
	wipeScopeTables(t, pdb, ctx)

	scope, err := NewCompareRepo(pdb).GetWeaponScope(ctx, "xuid_inconnu", slugCampagne)
	if err != nil {
		t.Fatalf("GetWeaponScope: %v", err)
	}
	if scope != nil {
		t.Fatalf("scope = %+v, attendu nil : un scope vide passerait les lecteurs d'armes et "+
			"remonterait une erreur de filtres", scope)
	}
}
