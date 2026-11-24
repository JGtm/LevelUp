package testutil

import (
	"strings"
	"testing"
)

// toutVu rend les champs et les types que les deux listes designent, tels qu'un garde qui les verrait tous les
// rendrait au verificateur.
func toutVu() (champs, types map[string]bool) {
	champs, types = map[string]bool{}, map[string]bool{}
	for cle := range ReplayChampsServisSeuls {
		champs[cle] = true
	}
	for nom := range ReplayTypesServisSeuls {
		types[nom] = true
	}
	return champs, types
}

// avecServisSeuls pose une entree datee de test dans chacune des deux listes, le temps d'un test, et rend tout ce
// que les listes designent (entrees livrees comprises).
func avecServisSeuls(t *testing.T) (champs, types map[string]bool) {
	t.Helper()
	ReplayChampsServisSeuls["Doc.Extra"] = "2026-01-01 : champ de test"
	ReplayTypesServisSeuls["Lot"] = "2026-01-01 : type de test"
	t.Cleanup(func() {
		delete(ReplayChampsServisSeuls, "Doc.Extra")
		delete(ReplayTypesServisSeuls, "Lot")
	})
	return toutVu()
}

// TestReplayServisSeulsIncoherents : le verificateur partage refuse ce qu'il doit refuser.
func TestReplayServisSeulsIncoherents(t *testing.T) {
	champs, types := toutVu()
	if pbs := ReplayServisSeulsIncoherents(champs, types); len(pbs) != 0 {
		t.Fatalf("les listes livrees doivent etre coherentes : %v", pbs)
	}

	t.Run("entrees coherentes", func(t *testing.T) {
		champs, types := avecServisSeuls(t)
		if pbs := ReplayServisSeulsIncoherents(champs, types); len(pbs) != 0 {
			t.Errorf("des entrees datees qui designent un champ et un type vus sont coherentes : %v", pbs)
		}
	})

	t.Run("entree perimee", func(t *testing.T) {
		avecServisSeuls(t)
		pbs := strings.Join(ReplayServisSeulsIncoherents(map[string]bool{}, map[string]bool{}), "\n")
		for _, want := range []string{`"Doc.Extra"`, `"Lot"`} {
			if !strings.Contains(pbs, want) {
				t.Errorf("l'entree perimee %s n'est pas refusee :\n%s", want, pbs)
			}
		}
	})

	t.Run("motif vide ou sans date", func(t *testing.T) {
		champs, types := avecServisSeuls(t)
		for _, motif := range []string{"", "  ", "sans date : motif"} {
			ReplayChampsServisSeuls["Doc.Extra"] = motif
			if pbs := ReplayServisSeulsIncoherents(champs, types); len(pbs) != 1 {
				t.Errorf("motif %q : un refus attendu, obtenu %v", motif, pbs)
			}
		}
	})
}
