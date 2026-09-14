package domain

// pad_tiers_vocabulary_test.go — LE VOCABULAIRE DES NIVEAUX D ARME NE SE DEDOUBLE PAS.
//
// Constat de revue (2026-09-14) : `PadTierOrder` recopiait les valeurs de `persist.PadTier*`
// sans rien pour les tenir ensemble. Une mutation d une seule lettre cote ecriture —
// `PadTierGround` de "terrain" a "sol" — laissait TOUTE la suite verte, et faisait disparaitre
// le niveau « terrain » des trois pages : la base aurait ecrit "sol", la lecture aurait cherche
// "terrain", et le groupe serait reste vide sans qu aucun test ne bronche.
//
// `persist` REEXPORTE desormais ces constantes (il ne peut plus diverger par construction) ;
// ce fichier verrouille ce qui reste : l ordre est complet, sans doublon, et le zero mesure n y
// figure PAS.

import "testing"

func TestPadTierOrder_CompletEtSansDoublon(t *testing.T) {
	attendu := []string{PadTierBase, PadTierGround, PadTierPower, PadTierPowerup, PadTierUnclassified}
	if len(PadTierOrder) != len(attendu) {
		t.Fatalf("PadTierOrder = %v, attendu %v", PadTierOrder, attendu)
	}
	vus := map[string]bool{}
	for i, tier := range PadTierOrder {
		if tier != attendu[i] {
			t.Errorf("PadTierOrder[%d] = %q, attendu %q", i, tier, attendu[i])
		}
		if tier == "" {
			t.Errorf("PadTierOrder[%d] est vide", i)
		}
		if vus[tier] {
			t.Errorf("PadTierOrder contient %q deux fois", tier)
		}
		vus[tier] = true
	}
}

// TestPadTierOrder_SansLeZeroMesure — `aucune_prise` n est pas un niveau de controle.
//
// L y faire entrer creerait un groupe « aucune prise » dans un classement qui parle de ce que
// le joueur a CONTROLE, et son total serait toujours zero.
func TestPadTierOrder_SansLeZeroMesure(t *testing.T) {
	for _, tier := range PadTierOrder {
		if tier == PadTierNoPickup {
			t.Errorf("%q figure dans PadTierOrder : c'est le zero mesure d'un joueur, pas un "+
				"niveau de controle", PadTierNoPickup)
		}
	}
}
