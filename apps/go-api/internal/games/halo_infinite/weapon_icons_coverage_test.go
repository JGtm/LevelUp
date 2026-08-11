package halo_infinite

import (
	"testing"

	"levelup/go-api/internal/games/weapons"
)

// TestChaqueArmeDuRegistreAUneIcone est le garde-rail de COUVERTURE : toute famille
// d'arme du registre canonique doit être servie, par l'atlas extrait du jeu ou par la
// table des concepts hors atlas.
//
// La famille est la clé du registre ET celle de l'icône : les 32 bits hauts d'un
// identifiant filmshell (le tag `weap`). Les deux tables se confrontent donc sans
// traduction ni nom au milieu — c'est ce qui rend ce test capable d'échouer.
//
// Une arme qui entre au registre sans icône échoue ici, au lieu de rendre une case
// vide en production sans que rien ne le signale.
func TestChaqueArmeDuRegistreAUneIcone(t *testing.T) {
	t.Parallel()
	familles := weapons.FilmshellWeaponKeysByFamily()
	if len(familles) == 0 {
		t.Fatal("registre vide : le test ne prouverait rien")
	}
	for tag, key := range familles {
		_, atlas := weaponIconFileByTag[tag]
		_, concept := weaponIconConceptFiles[tag]
		if !atlas && !concept {
			t.Errorf("%s (tag %08x) n'a aucune icône : ni atlas du jeu, ni concept déclaré", key, tag)
		}
	}
}

// TestLesConceptsSontHorsAtlas verrouille la frontière : une entrée de la table des
// concepts qui apparaîtrait AUSSI dans l'atlas serait une icône dessinée à la main
// servie à la place de celle du jeu — la régression que ce chantier corrige.
func TestLesConceptsSontHorsAtlas(t *testing.T) {
	t.Parallel()
	for tag, stem := range weaponIconConceptFiles {
		if atlasStem, ok := weaponIconFileByTag[tag]; ok {
			t.Errorf("tag %08x : concept %q masque l'icône du jeu %q", tag, stem, atlasStem)
		}
	}
}

// TestSentinellesHorsEspaceDesTags : les sentinelles sont interceptées AVANT le tag
// parce que leur identifiant tient dans les 32 bits bas — leur tag vaut donc 0. Si un
// jour l'atlas revendiquait le tag 0, l'ordre de résolution deviendrait un piège
// silencieux ; ce test le dit tout de suite.
func TestSentinellesHorsEspaceDesTags(t *testing.T) {
	t.Parallel()
	if stem, ok := weaponIconFileByTag[0]; ok {
		t.Errorf("l'atlas revendique le tag 0 (%q) : il entre en collision avec les sentinelles", stem)
	}
	if _, ok := weaponIconConceptFiles[0]; ok {
		t.Error("un concept est keyé sur le tag 0 : collision avec les sentinelles")
	}
}
