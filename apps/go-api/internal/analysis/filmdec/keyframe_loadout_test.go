package filmdec

import "testing"

// TestRecordContaining : chaque bit appartient au DERNIER record commençant à cet endroit ou
// avant. C'est toute la jointure occurrence -> slot : une erreur d'un cran ici et le loadout
// change de propriétaire (le témoin croisé de replay/loadouts.go chute de 98 % à 7 %).
func TestRecordContaining(t *testing.T) {
	starts := []int{100, 2900, 5700}
	cases := []struct {
		bit  int
		want int
	}{
		{99, -1},   // avant le premier record
		{100, 0},   // sur le début exact
		{2899, 0},  // dernier bit du premier
		{2900, 1},  // début exact du deuxième
		{5699, 1},  //
		{5700, 2},  //
		{99999, 2}, // au-delà du dernier début : c'est encore le dernier record
	}
	for _, c := range cases {
		if got := recordContaining(starts, c.bit); got != c.want {
			t.Fatalf("bit %d : record %d attendu, %d obtenu", c.bit, c.want, got)
		}
	}
	if got := recordContaining(nil, 42); got != -1 {
		t.Fatalf("table vide : -1 attendu, %d obtenu", got)
	}
}

// TestScanFilmKeyframeLoadouts_SansCatalogue : sans familles à chercher, le balayage ne rend
// RIEN plutôt que de parcourir tout le film pour rien.
func TestScanFilmKeyframeLoadouts_SansCatalogue(t *testing.T) {
	got, err := ScanFilmKeyframeLoadouts("répertoire inexistant", nil)
	if err != nil || got != nil {
		t.Fatalf("attendu (nil, nil), obtenu (%v, %v)", got, err)
	}
}

// TestKeyframeLoadouts_PayloadVide : un payload sans record ne doit produire aucun loadout.
func TestKeyframeLoadouts_PayloadVide(t *testing.T) {
	if got := keyframeLoadouts(nil, map[uint32]bool{1: true}); got != nil {
		t.Fatalf("aucun loadout attendu, obtenu %+v", got)
	}
}
