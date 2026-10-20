package main

// backfill_child_test.go — ce qui reste en propre a ce paquet apres la migration du lanceur.
//
// LE PROTOCOLE PARENT/ENFANT N'EST PLUS TESTE ICI, ET C'EST LA CONSEQUENCE DE L'ITEM 5.4 DE
// PLAN_CUISSON_PERF : la traduction des codes en categories, la reconnaissance du marqueur de
// pic memoire, le relais du journal et l'imposition de la racine du depot vivent desormais dans
// `internal/filmproc`, avec leurs tests (`filmproc_test.go`, `runner_child_test.go` — ce dernier
// exerce le protocole sur un VRAI processus, ce que la copie locale ne faisait pas). Les
// dupliquer ici testerait deux fois la meme chose et laisserait croire qu'il existe encore deux
// protocoles.
//
// Ce que ce paquet garde en propre : la traduction d'une issue en CATEGORIE DE RECAP
// (cmd_backfill_replay_passe_test.go) et le drapeau repetable des identites de carte.

import "testing"

func TestListeDrapeau_Repetable(t *testing.T) {
	var l listeDrapeau
	for _, v := range []string{"Live Fire", "Aquarius, la carte"} {
		if err := l.Set(v); err != nil {
			t.Fatalf("Set(%q): %v", v, err)
		}
	}
	if len(l) != 2 || l[0] != "Live Fire" || l[1] != "Aquarius, la carte" {
		t.Fatalf("liste = %v — la repetition doit preserver les libelles a virgule", l)
	}
	if l.String() != "Live Fire, Aquarius, la carte" {
		t.Fatalf("String() = %q", l.String())
	}
}
