// Package archlint — garde-fous d'architecture vérifiés en test (ratchet).
//
// no_weapon_family_label_literal_test.go : garde-rail complémentaire (lot M5 L4,
// 2026-09-08, .ai/PLAN_LIBELLES_EN_DUR_GO_2026-09-07.md §2.C/§4), frère de
// TestNoNewModePlaylistLabelLiteral (no_bare_resolve_mode_ui_test.go, lot L3).
//
// `games/weapons/registry.go` portait `weaponFamilyRow struct{ key, en, fr string }` —
// une famille d'arme (« battle_rifle ») associée à un NOM EN et un NOM FR en dur dans le
// slice Go `weaponRegistryFamilies`. Vérifié sur pièces (grep Go ET web) : ces libellés
// n'avaient AUCUN lecteur (le sunburst « Frags par arme »,
// apps/web/src/lib/i18n/manifests/frags.toml, ne localise que les niveaux classe/rôle,
// jamais le niveau famille ; le nom PAR ARME est une source distincte et déjà correcte,
// weapon_name_labels/weapon_names.toml, V72-06) — 0 code mort plutôt qu'une migration
// TOML qui aurait recopié du contenu mort (CLAUDE.md règle 7). Colonnes purgées de
// weapon_families par la migration purge_weapon_families_labels_columns ; le struct est
// retombé à `struct{ key string }`.
//
// Un futur retour en arrière (nouveau champ EN/FR littéral sur ce même modèle — la
// tentation naturelle serait de rajouter un libellé « pour l'affichage ») recréerait
// exactement le problème sans qu'aucun test ne le retienne (no_french_label_literal_test.go
// ne détecte QUE les littéraux accentués, pas un couple EN/FR non accentué comme
// {"smg", "SMG", "SMG"}). Volontairement étroit (un seul motif, comme les autres
// ratchets du dossier) : il cible le motif exact déjà vu dans ce paquet, pas toute forme
// possible de libellé en dur ailleurs.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// weaponFamilyLabelFieldRE matche la déclaration de champs de struct « key, en, fr
// string » (dans l'ordre ou avec des noms de champs adjacents `en string` / `fr string`)
// — le motif exact retiré de weaponFamilyRow le 2026-09-08.
var weaponFamilyLabelFieldRE = regexp.MustCompile(`\ben,\s*fr\s+string\b|\bfr,\s*en\s+string\b`)

// weaponFamilyLabelFieldAllowlist : aucun site toléré au 2026-09-08 (allowlist vide,
// modèle no_mojibake_test.go) — le motif vient d'être éradiqué du seul fichier qui le
// portait ; toute occurrence future est une régression, pas une dette héritée.
var weaponFamilyLabelFieldAllowlist = map[string]bool{}

func TestNoNewWeaponFamilyLabelLiteral(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	weaponsRoot := filepath.Join(filepath.Dir(filepath.Dir(thisFile)), "games", "weapons") // .../internal/games/weapons

	var violations []string
	err := filepath.WalkDir(weaponsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(weaponsRoot, path)
		rel = filepath.ToSlash(rel)
		if weaponFamilyLabelFieldAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			if weaponFamilyLabelFieldRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+" → "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("nouveau champ de struct « en, fr string » dans games/weapons (%d) — un nom "+
			"d'arme ou de famille d'arme se déclare dans weapon_names.toml "+
			"(config/titles/{slug}/mappings/, source unique depuis V72-06), jamais un couple "+
			"EN/FR littéral en Go (lot M5 L4, 2026-09-08) :\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}
