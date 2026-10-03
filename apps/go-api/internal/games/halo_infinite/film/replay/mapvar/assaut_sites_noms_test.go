package mapvar

// assaut_sites_noms_test.go — LES NOMS DES SITES D'AMORCAGE, lus dans le script du mode.
//
// La phase A3 du lot Assaut cherchait les sites d'amorcage par ANCRAGE `ti=13` et a echoue ; le
// catalogue d'objectifs ne porte aucune forme de site sur les cinq cartes d'Assaut. Il manquait
// le NOM.
//
// Le script du mode (tag `hsc* a35c6ce9`, releve le 2026-08-31) le donne en clair, dans la table
// `armzoneArgs` : **`defender_bombsite`** et **`attacker_bombsite`**. Ce test calcule leurs
// hachages de libelle — la cle par laquelle une variante de carte designe ses objets — pour que
// la chasse au catalogue reparte d'un identifiant, et non d'une hypothese spatiale.
//
// Les hachages craques le 2026-08-30 (`assault_site`, `assault_site_plate`, `assault_bomb_spawn`)
// n'existaient que sur quatre cartes SANS film. Ceux-ci viennent du script du mode lui-meme.

import "testing"

// assautSitesNoms : les noms d'objet de la table `armzoneArgs` du script de mode, plus ceux du
// meme voisinage qui designent la bombe et son socle.
var assautSitesNoms = []string{
	"defender_bombsite",
	"attacker_bombsite",
	"ball_spawn",
	"goalPlate",
	"bombTag",
	// La base d'amorcage, nommee par son propre script (`primitive_carriable_arming_base`).
	"primitive_carriable_arming_base",
	"g_primitive_carriable_arming_base",
}

// TestAssautSitesHachages imprime le hachage de libelle de chaque nom.
func TestAssautSitesHachages(t *testing.T) {
	for _, n := range assautSitesNoms {
		t.Logf("%-34s %d", n, LabelHash(n))
	}
}
