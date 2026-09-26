package replay

// inventory_rules_test.go — LE RENDEMENT DES REGLES D ANCRAGE DE L INVENTAIRE SUR LE BINAIRE REEL.
//
// LES REGLES R1 A R5 ET LEURS TESTS SUR FLUX FABRIQUE SONT DESCENDUS EN `grammar` AU LOT J4.2
// (2026-09-26), avec la lecture de l inventaire (`grammar/inventory_rules_test.go`). Reste ici le
// test qui dit ce qu elles RENDENT ENSEMBLE sur les images-cles du film de reference : il partage
// ses comptes figes (`wantInventoryRead`) avec `minifilm_test.go`.

import (
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestInventoryRulesOnRealBinary : LES QUATRE REGLES, SUR LE BINAIRE REEL.
//
// Les tests ci-dessus verifient la mecanique de chaque regle sur un flux construit. Celui-ci
// verifie ce qu elles RENDENT ENSEMBLE sur les images-cles du film de reference — les seuls
// chiffres qui disent si la grammaire tient encore.
//
// LES NON-LECTURES SONT PUBLIEES, ET C EST LE POINT : 184 etats pour 132 capacites et 150
// compteurs de grenade lus. Un decodeur qui remonterait a 184/184 ne serait pas « meilleur » —
// il aurait cesse de refuser, et il faudrait comprendre pourquoi avant de s en rejouir.
//
// LES GRENADES SONT PASSEES DE 120 A 150 LE 2026-08-25, et le chiffre d arrivee n est pas
// quelconque : 150 est EXACTEMENT le nombre de records dont les munitions sont lues. La voie
// positionnelle R2b (inventory_grenades_rules.go) se borne au bloc de munitions ; elle rend donc
// une lecture partout ou ce bloc existe, et nulle part ailleurs. Les 34 records restants sont
// ceux qui ne portent AUCUNE arme — les lectures vides deja etiquetees (cf. Inventory.Empty).
func TestInventoryRulesOnRealBinary(t *testing.T) {
	inv, st, err := grammar.ScanFilmKeyframeInventory(MiniFilmDir, loadoutFamilies(), 0)
	if err != nil {
		t.Fatalf("ScanFilmKeyframeInventory : %v", err)
	}
	var ability, grenades, ammo, multi, drawn, grenSel int
	for _, i := range inv {
		if i.AbilityRank >= 0 {
			ability++
		}
		if i.GrenadesRead {
			grenades++
		}
		if i.AmmoRead {
			ammo++
		}
		if i.AmmoCandidates > 1 {
			multi++
		}
		if i.DrawnSlot >= 0 {
			drawn++
		}
		if i.SelectedGrenadeRank >= 0 {
			grenSel++
			if !i.GrenadesRead || i.SelectedGrenadeRank >= types.InventorySlotCount ||
				i.Grenades[i.SelectedGrenadeRank] == 0 {
				t.Fatalf("slot %d : selection de grenade rang %d sans compteur porte — la garde "+
					"masque==i22 ne tient plus", i.Slot, i.SelectedGrenadeRank)
			}
		}
		for k := range i.Ammo {
			if i.Ammo[k].Mag != nil && i.Ammo[k].Gauge != nil {
				t.Fatalf("slot %d : chargeur ET jauge sur le meme emplacement — la largeur 22 "+
					"n existe pas dans la carte memoire", i.Slot)
			}
		}
	}
	for _, c := range []struct {
		nom       string
		got, want int
	}{
		{"etats", len(inv), wantInventoryRead},
		{"capacite lue (R1)", ability, wantInvAbility},
		{"grenades lues (R2)", grenades, wantInvGrenades},
		{"munitions lues (R3+R4)", ammo, wantInvAmmo},
		{"lectures a plusieurs candidats", multi, wantInvMultiCandidate},
		{"selection de grenade lue (R5)", grenSel, wantInvGrenadeSel},
		{"grenades par l ancre (R2a)", st.GrenadesByAnchor, wantInvGrenAnchor},
		{"grenades par la position (R2b)", st.GrenadesByPosition, wantInvGrenPosition},
	} {
		if c.got != c.want {
			t.Errorf("%s : %d, attendu %d — une regle d ancrage a change de rendement",
				c.nom, c.got, c.want)
		}
	}
	if drawn != ammo {
		t.Errorf("%d emplacement(s) degaine(s) pour %d bloc(s) de munitions lus : le selecteur "+
			"i42 fait partie du MEME parse, les deux comptes ne peuvent pas diverger", drawn, ammo)
	}
}

// Le rendement mesure de chaque regle sur le film de reference. Ces valeurs sont ECRITES, pas
// derivees : une valeur attendue qui se recalcule depuis la sortie ne teste rien.
//
// LOT M3.1 (2026-09-23) : la marche d image-cle reparee atteint HUIT records bipedes de plus
// (184 -> 192 etats). Chaque regle lit donc davantage — capacite 132 -> 136, grenades et
// munitions 150 -> 154, plusieurs candidats 51 -> 52, selection 106 -> 110, ancre 120 -> 124 —
// et la voie par la POSITION ne bouge pas (30) : les records gagnes se lisent tous par l ancre.
const (
	wantInvAbility        = 136
	wantInvGrenades       = 154
	wantInvAmmo           = 154
	wantInvMultiCandidate = 52
	wantInvGrenadeSel     = 110
	// LA REPARTITION PAR VOIE EST TENUE A PART de son total : si une regression faisait basculer
	// des lectures de l ancre vers la position (ou l inverse) sans changer la somme, seul ce
	// couple de crans le dirait. R2a garde ses 120 lectures — la voie par l ancre reste
	// prioritaire, donc AUCUNE lecture existante n a change de valeur.
	wantInvGrenAnchor   = 124
	wantInvGrenPosition = 30
)
