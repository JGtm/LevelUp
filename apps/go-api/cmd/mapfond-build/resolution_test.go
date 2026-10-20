package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLiensDuDepotPrendLeSuffixeLePlusLong — le témoin qui départage.
//
// `oasis_sentry_defense_btb_exiled.mvar` contient plusieurs `_`, et une lecture naïve (couper
// au premier) rendrait `sentry_defense_btb_exiled` comme module. La règle coupe à gauche mais
// ne retient que ce qui désigne un DOSSIER INSTALLÉ — c'est l'installation qui tranche, pas la
// forme du nom.
//
// MUTATION JOUÉE : couper au DERNIER `_` au lieu du premier dossier reconnu fait rendre
// `exiled` (qui n'est pas un dossier) et le test rougit sur les deux cas à suffixe long.
func TestLiensDuDepotPrendLeSuffixeLePlusLong(t *testing.T) {
	repo := t.TempDir()
	depot := filepath.Join(repo, filepath.FromSlash(".ai/re_dump/mapvar"))
	if err := os.MkdirAll(depot, 0o755); err != nil {
		t.Fatal(err)
	}
	levels := t.TempDir()
	for _, m := range []string{"btb_exiled", "btb_drydock", "ridgeline"} {
		if err := os.MkdirAll(filepath.Join(levels, m), 0o755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(levels, m, m+"-rtx-new.module"))
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
	for _, n := range []string{
		"oasis_btb_exiled", "oasis_sentry_defense_btb_exiled", "oasis_map",
		"deadlock_btb_drydock", "deadlock_map",
		"cliffhanger_ridgeline",
		"absolution_fo09_academy", // canevas non installé : doit être ignoré
	} {
		f, err := os.Create(filepath.Join(depot, n+".mvar"))
		if err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}

	liens := liensDuDepot(repo, levels)
	attendu := map[string]string{
		"oasis":                "btb_exiled",
		"oasis_sentry_defense": "btb_exiled",
		"deadlock":             "btb_drydock",
		"cliffhanger":          "ridgeline",
	}
	for carte, module := range attendu {
		if got := liens[carte]; got != module {
			t.Errorf("liensDuDepot[%q] = %q, attendu %q", carte, got, module)
		}
	}
	if _, ok := liens["absolution"]; ok {
		t.Error("un .mvar dont le suffixe n'est pas installé ne doit produire aucun lien")
	}

	// La résolution depuis le nom du CATALOGUE, qui porte le suffixe de variante par défaut.
	for _, c := range []struct{ catalogue, attendu string }{
		{"oasis_map", "btb_exiled"},
		{"oasis_sentry_defense_map", "btb_exiled"},
		{"deadlock_map", "btb_drydock"},
		{"cliffhanger_ridgeline", "ridgeline"},
		{"inconnue_map", ""},
	} {
		if got := moduleDuCatalogue(liens, levels, c.catalogue); got != c.attendu {
			t.Errorf("moduleDuCatalogue(%q) = %q, attendu %q", c.catalogue, got, c.attendu)
		}
	}
}
