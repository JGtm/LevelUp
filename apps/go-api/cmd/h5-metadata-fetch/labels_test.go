package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/testutil"
)

// realAssetLabels résout le fichier versionné depuis la racine du dépôt.
func realAssetLabels(t *testing.T) string {
	t.Helper()
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt : %v", err)
	}
	return filepath.Join(root, "config", "titles", "halo_5", "mappings", "asset_labels.toml")
}

func TestLoadAssetLabels_RealFile(t *testing.T) {
	labels, err := loadAssetLabels(realAssetLabels(t))
	if err != nil {
		t.Fatalf("asset_labels.toml : %v", err)
	}
	if got, want := countLang(labels.Medals, i18n.LocaleFR), 184; got != want {
		t.Errorf("médailles `fr` = %d (want %d)", got, want)
	}
	if got, want := countLang(labels.MedalDescriptions, i18n.LocaleFR), 194; got != want {
		t.Errorf("descriptions `fr` = %d (want %d)", got, want)
	}
	if got := labels.Medals["Splatter"][i18n.LocaleFR]; got != "Écrasement" {
		t.Errorf("Medals[Splatter] = %q", got)
	}
	if len(labels.MapsByID) != 1 || labels.MapsByID[0].Name[i18n.Fallback] != "Tidal" || labels.MapsByID[0].Name[i18n.LocaleFR] != "Tidal" {
		t.Errorf("MapsByID = %+v", labels.MapsByID)
	}
}

// countLang compte les entrées d'une table de noms qui portent un texte dans la langue `l`.
func countLang(table map[string]i18n.Text, l i18n.Locale) int {
	n := 0
	for _, text := range table {
		if strings.TrimSpace(text[l]) != "" {
			n++
		}
	}
	return n
}

// Une langue sans entrée dans le fichier ne fabrique rien : médailles et descriptions n'en ont
// aucune dans la langue de l'API, et un nom absent se lit vide (les seeds retombent alors sur la
// donnée de l'API, medalTexts / commendationTexts).
func TestAssetLabels_LangueDeLAPISansEntree(t *testing.T) {
	labels, err := loadAssetLabels(realAssetLabels(t))
	if err != nil {
		t.Fatal(err)
	}
	if n, d := countLang(labels.Medals, i18n.Fallback), countLang(labels.MedalDescriptions, i18n.Fallback); n != 0 || d != 0 {
		t.Errorf("aucun nom dans la langue de l'API dans les tables de médailles, got %d / %d", n, d)
	}
	if got := labels.Medals["Splatter"][i18n.Fallback]; got != "" {
		t.Errorf("Splatter dans la langue de l'API = %q, want vide", got)
	}
}

func TestLoadAssetLabels_AbsentFileIsEmpty(t *testing.T) {
	labels, err := loadAssetLabels(filepath.Join(t.TempDir(), "absent.toml"))
	if err != nil {
		t.Fatalf("fichier absent : dégradation propre attendue, got %v", err)
	}
	if len(labels.Medals) != 0 || len(labels.MapsByID) != 0 {
		t.Errorf("catalogue non vide : %+v", labels)
	}
}

func TestLoadAssetLabels_Refus(t *testing.T) {
	cases := map[string]string{
		"langue inconnue":  "[medals]\n\"Kill\" = { zz = \"Frag\" }\n",
		"toml illisible":   "[medals\n",
		"carte langue zz":  "[[maps_by_id]]\nid = \"x\"\nname = { en = \"A\", zz = \"B\" }\n",
		"valeur non table": "[medals]\n\"Kill\" = \"Frag\"\n",
	}
	for name, raw := range cases {
		path := filepath.Join(t.TempDir(), "asset_labels.toml")
		if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := loadAssetLabels(path)
		if err == nil {
			t.Errorf("%s : accepté alors qu'il devrait être refusé", name)
			continue
		}
		if name == "langue inconnue" && !strings.Contains(err.Error(), `"zz"`) {
			t.Errorf("%s : l'erreur doit nommer la langue : %v", name, err)
		}
	}
}
