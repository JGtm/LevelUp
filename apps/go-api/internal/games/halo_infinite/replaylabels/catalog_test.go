package replaylabels

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/weaponv3"
	"levelup/go-api/internal/games/weapons"
)

// repoRoot remonte jusqu'au répertoire qui porte config/titles.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("répertoire courant : %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "config", "titles")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("racine du dépôt introuvable")
	return ""
}

func TestLoad_CatalogueReelDuTitre(t *testing.T) {
	cat, err := Load(repoRoot(t), "halo_infinite")
	if err != nil {
		t.Fatalf("chargement du catalogue : %v", err)
	}
	if cat.Empty() {
		t.Fatal("catalogue vide : les mappings du titre ne sont plus lus")
	}
	if len(cat.Grenades) != 4 {
		t.Errorf("%d rang(s) de grenade, attendu 4", len(cat.Grenades))
	}
	// Les deux langues sont obligatoires PARTOUT : un artefact est construit une fois et
	// servi aux deux locales.
	for i, g := range cat.Grenades {
		if g.En == "" || g.Fr == "" {
			t.Errorf("rang de grenade %d incomplet : %+v", i, g)
		}
	}
	for idx, a := range cat.Abilities {
		if a.En == "" || a.Fr == "" {
			t.Errorf("capacité %d incomplète : %+v", idx, a)
		}
	}
	for fam, w := range cat.Weapons {
		if w.En == "" || w.Fr == "" {
			t.Errorf("famille %08X : libellé incomplet %+v", fam, w)
		}
	}
}

// TestLoad_TitreSansMappings — un titre sans ces fichiers ÉCHOUE, il ne dégrade pas en
// silence. Un rejeu muet est indistinguable, à l'écran, d'un rejeu dont toutes les armes
// seraient inconnues : c'est à l'outil de build de refuser, pas au lecteur de deviner.
func TestLoad_TitreSansMappings(t *testing.T) {
	if _, err := Load(repoRoot(t), "halo_5"); err == nil {
		t.Error("un titre sans replay_labels.toml doit faire échouer le chargement")
	}
	if _, err := Load(t.TempDir(), "halo_infinite"); err == nil {
		t.Error("une racine sans config/titles doit faire échouer le chargement")
	}
}

// famillesSansWeaponKey — LE TROU CONNU, FIGÉ POUR QU'IL NE GROSSISSE PAS EN SILENCE.
//
// Le décodeur connaît des familles d'arme que le REGISTRE du titre ne porte pas : ce sont
// des curiosités de bac à sable (Fiesta). Elles n'ont donc pas de weapon_key, donc pas de
// libellé, donc elles gardent leur hexadécimal à l'écran — ce qui est la règle du
// chantier, mais qui doit rester une liste COURTE et VUE.
//
// Aucune n'apparaît dans les trois artefacts construits à ce jour (mesuré le 2026-08-02).
var famillesSansWeaponKey = map[string]bool{
	"Sandwich":        true,
	"Mythic Sandwich": true,
	"Mutilator":       true,
}

func TestFamillesDArmeConnuesDuDecodeurSontAuRegistre(t *testing.T) {
	registre := weapons.FilmshellWeaponKeysByFamily()
	var nouvelles []string
	for high, nom := range weaponv3.KnownWeaponHigh32 {
		if _, ok := registre[high]; ok {
			continue
		}
		if famillesSansWeaponKey[nom] {
			continue
		}
		nouvelles = append(nouvelles, nom)
	}
	sort.Strings(nouvelles)
	if len(nouvelles) > 0 {
		t.Errorf("famille(s) d'arme connues du décodeur mais ABSENTES du registre du titre : %v — "+
			"elles n'auront aucun libellé dans le rejeu. Les ajouter au registre, ou les inscrire "+
			"dans famillesSansWeaponKey avec la raison", nouvelles)
	}
	// Le garde-rail doit pouvoir échouer : si l'allowlist ne décrit plus la réalité,
	// c'est elle qui est périmée (leçon J4.0 — un garde qui ne peut pas échouer ne garde rien).
	for nom := range famillesSansWeaponKey {
		trouvee := false
		for high, n := range weaponv3.KnownWeaponHigh32 {
			if n != nom {
				continue
			}
			trouvee = true
			if _, ok := registre[high]; ok {
				t.Errorf("%q est désormais AU REGISTRE : la retirer de famillesSansWeaponKey", nom)
			}
		}
		if !trouvee {
			t.Errorf("%q n'est plus connue du décodeur : entrée périmée dans famillesSansWeaponKey", nom)
		}
	}
}
