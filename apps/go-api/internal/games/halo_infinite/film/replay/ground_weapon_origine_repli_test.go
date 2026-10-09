package replay

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// ground_weapon_origine_repli_test.go — l origine `dropped` d un objet au sol, decidee par la
// fenetre 200 ms / 1,5 m de fin de vie ([gwPadsClass]), est un REPLI NOMME ET COMPTE (lot J8.3 du
// plan de suite d audit, constat RB2-8 : la fenetre decidait hors registre).
//
// MUTATION : retirer le `DeclencheN` de [buildWeaponPads] — ROUGE (compte 0), et la direction (B)
// du ratchet `archlint` rougit aussi (ancre absente).
func TestLOrigineLacheeParFenetreEstUnRepliCompte(t *testing.T) {
	fam := gwTestFamily(t, 1)
	scan := WorldObjectScan{
		Scanned: true,
		Stats:   types.EquipmentCreationStats{Accepted: 3},
		Creations: []types.EquipmentCreation{
			// Deux apparitions LACHEES : une vie de joueur s acheve la, a moins de 200 ms.
			gwTestCreation(20, 0, 10_000_000, fam, 30, 30),
			gwTestCreation(21, 0, 50_000_000, fam, 30.1, 30),
			// Une apparition loin de toute fin de vie : `spawned`, hors du repli.
			gwTestCreation(22, 0, 10_000_000, fam, 90, 90),
		},
		Keyframes: grammar.WorldObjectKeyframes{TimesUS: []uint64{0, 20_000_000, 40_000_000, 60_000_000}},
	}
	pos := []grammar.BipedPosition{
		{Slot: 3, TimestampUS: 9_950_000, X: 30, Y: 30, HasWorld: true},
		{Slot: 3, TimestampUS: 49_950_000, X: 30.1, Y: 30, HasWorld: true},
	}
	clk := gwTestClock()
	clk.fb = fallback.NouveauCompteur()
	_, _, cov, _ := buildWeaponPads(PadScans{Weapons: scan}, pos, clk, padCatalogs{})
	if cov.Dropped != 2 {
		t.Fatalf("2 apparitions lachees attendues, %d : %+v", cov.Dropped, cov)
	}
	if got := clk.fb.Compte(fallback.NomOrigineAuSolLacheeParFenetre); got != 2 {
		t.Fatalf("le repli de l origine lachee compte %d declenchement(s), attendu 2 (une par "+
			"apparition classee `dropped` par la fenetre)", got)
	}
}
