package replaylabels

import (
	"testing"

	"levelup/go-api/internal/games/weapons"
)

// Chaque arme dont le projectile explose (`[shot_impact]`) est une arme du registre canonique :
// une faute de frappe la laisserait sans explosion en silence.
func TestLoad_ArmesQuiExplosentExistentAuRegistre(t *testing.T) {
	cat, err := Load(repoRoot(t), "halo_infinite")
	if err != nil {
		t.Fatalf("chargement du catalogue : %v", err)
	}
	if len(cat.Impacts) == 0 {
		t.Fatal("aucune arme a pied ne declare d'explosion a l'impact")
	}
	connues := map[string]bool{}
	for _, key := range weapons.FilmshellWeaponKeysByFamily() {
		connues[key] = true
	}
	for key := range cat.Impacts {
		if !connues[key] {
			t.Errorf("[shot_impact] %q : weapon_key inconnu du registre des armes", key)
		}
	}
}
