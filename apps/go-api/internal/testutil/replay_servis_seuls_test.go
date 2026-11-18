package testutil

import (
	"strings"
	"testing"
)

// TestReplayServisSeulsIncoherents : le verificateur partage refuse ce qu'il doit refuser.
func TestReplayServisSeulsIncoherents(t *testing.T) {
	tout := func() (map[string]bool, map[string]bool) {
		champs, types := map[string]bool{}, map[string]bool{}
		for cle := range ReplayChampsServisSeuls {
			champs[cle] = true
		}
		for nom := range ReplayTypesServisSeuls {
			types[nom] = true
		}
		return champs, types
	}

	champs, types := tout()
	if pbs := ReplayServisSeulsIncoherents(champs, types); len(pbs) != 0 {
		t.Fatalf("les listes livrees doivent etre coherentes : %v", pbs)
	}

	t.Run("entree perimee", func(t *testing.T) {
		champs, types := tout()
		delete(champs, "ReplayDocument.MedalLabels")
		delete(types, "MedalSprite")
		pbs := strings.Join(ReplayServisSeulsIncoherents(champs, types), "\n")
		for _, want := range []string{`"ReplayDocument.MedalLabels"`, `"MedalSprite"`} {
			if !strings.Contains(pbs, want) {
				t.Errorf("l'entree perimee %s n'est pas refusee :\n%s", want, pbs)
			}
		}
	})

	t.Run("motif vide ou sans date", func(t *testing.T) {
		const cle = "ReplayDocument.MedalLabels"
		avant := ReplayChampsServisSeuls[cle]
		defer func() { ReplayChampsServisSeuls[cle] = avant }()
		champs, types := tout()
		for _, motif := range []string{"", "  ", "sans date : motif"} {
			ReplayChampsServisSeuls[cle] = motif
			if pbs := ReplayServisSeulsIncoherents(champs, types); len(pbs) != 1 {
				t.Errorf("motif %q : un refus attendu, obtenu %v", motif, pbs)
			}
		}
	})
}
