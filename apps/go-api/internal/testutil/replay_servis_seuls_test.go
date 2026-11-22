package testutil

import (
	"strings"
	"testing"
)

// avecServisSeuls pose une entree datee dans chacune des deux listes le temps d'un test : les listes livrees
// sont vides, le verificateur se prouve sur des entrees de test.
func avecServisSeuls(t *testing.T) (champs, types map[string]bool) {
	t.Helper()
	ReplayChampsServisSeuls["Doc.Extra"] = "2026-01-01 : champ de test"
	ReplayTypesServisSeuls["Lot"] = "2026-01-01 : type de test"
	t.Cleanup(func() {
		delete(ReplayChampsServisSeuls, "Doc.Extra")
		delete(ReplayTypesServisSeuls, "Lot")
	})
	return map[string]bool{"Doc.Extra": true}, map[string]bool{"Lot": true}
}

// TestReplayServisSeulsIncoherents : le verificateur partage refuse ce qu'il doit refuser.
func TestReplayServisSeulsIncoherents(t *testing.T) {
	if pbs := ReplayServisSeulsIncoherents(nil, nil); len(pbs) != 0 {
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
